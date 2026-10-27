// SPDX-License-Identifier: AGPL-3.0-or-later

// Command hublot runs the portal and administers it from the command line.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/C2-POLY/hublot/internal/auth"
	"github.com/C2-POLY/hublot/internal/authz"
	"github.com/C2-POLY/hublot/internal/config"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
	"github.com/C2-POLY/hublot/internal/httpapi"
	"github.com/C2-POLY/hublot/internal/mail"
	"github.com/C2-POLY/hublot/internal/portal"
	"github.com/C2-POLY/hublot/internal/storage"
	"github.com/C2-POLY/hublot/web"
)

const usage = `Usage: hublot [-config hublot.yaml] <command> [flags]

Commands:
  serve                                     run the portal (applies migrations first)
  migrate                                   apply database migrations
  user  -email E [-name N] [-org-role R]    create or update a person (member, admin, owner)
  group -name G [-member E]...              create a group, optionally adding people
  grant -project P -role R (-email E | -group G)
                                            give a role on a project (viewer, reporter,
                                            contributor, triager, manager)
  resync [-project P]                       re-read every portal issue from GitHub, for one
                                            project or all of them
`

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	global := flag.NewFlagSet("hublot", flag.ExitOnError)
	cfgPath := global.String("config", envOr("HUBLOT_CONFIG", "hublot.yaml"), "configuration file")
	global.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	_ = global.Parse(os.Args[1:])
	if global.NArg() == 0 {
		global.Usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *cfgPath, global.Arg(0), global.Args()[1:]); err != nil {
		slog.Error("hublot", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfgPath, cmd string, args []string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	q := db.New(pool)
	switch cmd {
	case "serve":
		return serve(ctx, cfg, pool, q)
	case "migrate":
		slog.Info("migrations applied")
		return nil
	case "user":
		return cmdUser(ctx, q, args)
	case "group":
		return cmdGroup(ctx, q, args)
	case "grant":
		return cmdGrant(ctx, cfg, pool, q, args)
	case "resync":
		return cmdResync(ctx, cfg, pool, q, args)
	}
	return fmt.Errorf("unknown command %q", cmd)
}

func serve(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, q *db.Queries) error {
	f, err := newForge(cfg, q)
	if err != nil {
		return err
	}
	store, err := newStore(cfg)
	if err != nil {
		return err
	}
	mailer := newMailer(cfg)
	p := &portal.Service{Pool: pool, Q: q, Forge: f, Store: store, Cfg: cfg, Mailer: mailer}
	if err := p.SyncProjects(ctx); err != nil {
		return err
	}
	if p.Jobs, err = portal.NewJobs(p, true); err != nil {
		return err
	}
	// Not ctx: a signal must let running jobs finish (Stop below), not cancel them.
	if err := p.Jobs.Start(context.WithoutCancel(ctx)); err != nil {
		return err
	}
	webFS, err := webFiles(cfg)
	if err != nil {
		return err
	}
	srv := &httpapi.Server{
		Cfg:    cfg,
		Auth:   &auth.Service{Q: q, Mailer: mailer, Cfg: cfg},
		Portal: p,
		Web:    webFS,
	}
	httpServer := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- httpServer.ListenAndServe() }()
	slog.Info("hublot is listening", "addr", cfg.Listen, "base_url", cfg.BaseURL, "forge", cfg.Forge.Driver)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err = httpServer.Shutdown(shutdown)
	// Running jobs get the rest of the delay; unfinished ones are retried on the next start.
	return errors.Join(err, p.Jobs.Stop(shutdown))
}

func newForge(cfg *config.Config, q *db.Queries) (forge.Forge, error) {
	if cfg.Forge.Driver == "fake" {
		slog.Warn("fake forge: issues stay in memory and never reach GitHub")
		fake := forge.NewFake(os.DirFS(cfg.Forge.FakeDir))
		fake.LastNumber = func(ctx context.Context, r forge.Repo) (int, error) {
			n, err := q.MaxIssueNumber(ctx, db.MaxIssueNumberParams{RepoOwner: r.Owner, RepoName: r.Name})
			return int(n), err
		}
		return fake, nil
	}
	g := cfg.Forge.GitHub
	return forge.NewGitHub(g.AppID, g.InstallationID, g.PrivateKeyPath, g.APIURL)
}

func newStore(cfg *config.Config) (storage.Store, error) {
	if cfg.Storage.Driver == "s3" {
		c := cfg.Storage.S3
		return storage.NewS3(c.Endpoint, c.Region, c.Bucket, c.AccessKey, c.SecretKey, c.UseSSL)
	}
	return storage.FS{Dir: cfg.Storage.Dir}, nil
}

func newMailer(cfg *config.Config) mail.Mailer {
	if cfg.Mail.Driver == "smtp" {
		m := cfg.Mail
		return mail.SMTP{From: m.From, Host: m.Host, Port: m.Port, Username: m.Username, Password: m.Password}
	}
	slog.Warn("log mail driver: e-mails are printed, not sent")
	return &mail.Log{}
}

// webFiles prefers web_dir when set, then the SPA embedded at build time.
func webFiles(cfg *config.Config) (fs.FS, error) {
	if cfg.WebDir != "" {
		return os.DirFS(cfg.WebDir), nil
	}
	return web.Dist()
}

func cmdUser(ctx context.Context, q *db.Queries, args []string) error {
	fl := flag.NewFlagSet("user", flag.ExitOnError)
	email := fl.String("email", "", "e-mail address")
	name := fl.String("name", "", "display name")
	role := fl.String("org-role", "member", "member, admin or owner")
	lang := fl.String("lang", "fr", "fr or en")
	_ = fl.Parse(args)
	addr, err := auth.NormalizeEmail(*email)
	if err != nil {
		return err
	}
	org := db.OrgRole(*role)
	if !org.Valid() {
		return fmt.Errorf("unknown org role %q", *role)
	}
	if *name == "" {
		*name = strings.Split(addr, "@")[0]
	}
	u, err := q.CreateUser(ctx, db.CreateUserParams{Email: addr, Name: *name, OrgRole: org, Lang: *lang})
	if err != nil {
		return err
	}
	if err := audit(ctx, q, "user.upsert", "user", u.ID.String(), map[string]any{"email": addr, "org_role": org}); err != nil {
		return err
	}
	fmt.Printf("%s %s (%s)\n", u.ID, u.Email, u.OrgRole)
	return nil
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func cmdGroup(ctx context.Context, q *db.Queries, args []string) error {
	fl := flag.NewFlagSet("group", flag.ExitOnError)
	name := fl.String("name", "", "group name")
	var members multiFlag
	fl.Var(&members, "member", "e-mail of a person to add (repeatable)")
	_ = fl.Parse(args)
	if strings.TrimSpace(*name) == "" {
		return errors.New("-name is required")
	}
	g, err := q.CreateGroup(ctx, strings.TrimSpace(*name))
	if err != nil {
		return err
	}
	for _, m := range members {
		addr, err := auth.NormalizeEmail(m)
		if err != nil {
			return fmt.Errorf("%s: %w", m, err)
		}
		u, err := q.GetUserByEmail(ctx, addr)
		if err != nil {
			return fmt.Errorf("%s: unknown person, create them with `hublot user` first", addr)
		}
		if err := q.AddGroupMember(ctx, db.AddGroupMemberParams{GroupID: g.ID, UserID: u.ID}); err != nil {
			return err
		}
		if err := audit(ctx, q, "group.member_add", "group", g.ID.String(), map[string]any{"user": u.Email}); err != nil {
			return err
		}
	}
	fmt.Printf("%s %s\n", g.ID, g.Name)
	return nil
}

func cmdGrant(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, q *db.Queries, args []string) error {
	fl := flag.NewFlagSet("grant", flag.ExitOnError)
	project := fl.String("project", "", "project slug")
	role := fl.String("role", "", "viewer, reporter, contributor, triager or manager")
	email := fl.String("email", "", "person to grant")
	group := fl.String("group", "", "group to grant")
	_ = fl.Parse(args)
	if authz.ParseRole(*role) == authz.RoleNone {
		return fmt.Errorf("unknown role %q", *role)
	}
	if (*email == "") == (*group == "") {
		return errors.New("give exactly one of -email or -group")
	}
	// Projects come from the configuration: sync them so a fresh install can grant right away.
	p := &portal.Service{Pool: pool, Q: q, Cfg: cfg}
	if err := p.SyncProjects(ctx); err != nil {
		return err
	}
	proj, err := q.GetProjectBySlug(ctx, *project)
	if err != nil {
		return fmt.Errorf("unknown project %q", *project)
	}
	pr := db.ProjectRole(*role)
	var target string
	if *email != "" {
		addr, err := auth.NormalizeEmail(*email)
		if err != nil {
			return err
		}
		u, err := q.GetUserByEmail(ctx, addr)
		if err != nil {
			return fmt.Errorf("%s: unknown person, create them with `hublot user` first", addr)
		}
		if _, err := q.UpsertUserGrant(ctx, db.UpsertUserGrantParams{
			ProjectID: proj.ID, UserID: uuid.NullUUID{UUID: u.ID, Valid: true}, Role: pr,
		}); err != nil {
			return err
		}
		target = "user " + u.Email
	} else {
		g, err := q.GetGroupByName(ctx, *group)
		if err != nil {
			return fmt.Errorf("unknown group %q", *group)
		}
		if _, err := q.UpsertGroupGrant(ctx, db.UpsertGroupGrantParams{
			ProjectID: proj.ID, GroupID: uuid.NullUUID{UUID: g.ID, Valid: true}, Role: pr,
		}); err != nil {
			return err
		}
		target = "group " + g.Name
	}
	if err := audit(ctx, q, "grant.upsert", "project", proj.Slug, map[string]any{"to": target, "role": pr}); err != nil {
		return err
	}
	fmt.Printf("%s is now %s on %s\n", target, pr, proj.Slug)
	return nil
}

func cmdResync(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, q *db.Queries, args []string) error {
	fl := flag.NewFlagSet("resync", flag.ExitOnError)
	project := fl.String("project", "", "project slug (default: every project)")
	_ = fl.Parse(args)
	f, err := newForge(cfg, q)
	if err != nil {
		return err
	}
	p := &portal.Service{Pool: pool, Q: q, Forge: f, Cfg: cfg}
	if err := p.SyncProjects(ctx); err != nil {
		return err
	}
	// Status changes queue their notifications for the running server to send.
	if p.Jobs, err = portal.NewJobs(p, false); err != nil {
		return err
	}
	slugs := []string{*project}
	if *project == "" {
		slugs = slugs[:0]
		for _, cp := range cfg.Projects {
			slugs = append(slugs, cp.Slug)
		}
	}
	for _, slug := range slugs {
		n, err := p.ResyncProject(ctx, slug, true)
		if errors.Is(err, portal.ErrNotFound) {
			return fmt.Errorf("unknown project %q", slug)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", slug, err)
		}
		fmt.Printf("%s: %d issues read\n", slug, n)
	}
	return nil
}

// audit records command-line administration. There is no portal actor: via says where it came from.
func audit(ctx context.Context, q *db.Queries, action, targetType, targetID string, details map[string]any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	return q.InsertAudit(ctx, db.InsertAuditParams{
		Via: "cli", Action: action, TargetType: targetType, TargetID: targetID, Details: raw,
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
