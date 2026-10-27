// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi_test

// The isolation suite (PRD §5 bis): a person never sees a project they were not given,
// through any door — project list, forms, request list, request detail, attachments.
// It runs against a real Postgres: set HUBLOT_TEST_DATABASE_URL (make test does it).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/et-technology-consulting/hublot/internal/auth"
	"github.com/et-technology-consulting/hublot/internal/config"
	"github.com/et-technology-consulting/hublot/internal/db"
	"github.com/et-technology-consulting/hublot/internal/forge"
	"github.com/et-technology-consulting/hublot/internal/httpapi"
	"github.com/et-technology-consulting/hublot/internal/mail"
	"github.com/et-technology-consulting/hublot/internal/portal"
	"github.com/et-technology-consulting/hublot/internal/storage"
)

const testConfig = `
base_url: http://portal.test
database_url: unused
secret: test-secret-0123456789abcdef0123456789
auth:
  allowed_domains: [acme.test]
  bootstrap_owners: [boss@acme.test]
mail: {driver: log}
storage: {driver: fs, dir: %s}
forge: {driver: fake}
attachments: {max_bytes: 2048}
projects:
  - {slug: shop, name: Shop, color: "#B9AEFF", vehicle: boat, repo: acme/shop}
  - {slug: app, name: App, color: "#FF9A62", vehicle: plane, repo: acme/app}
  - {slug: open, name: Open, color: "#7FD8B6", vehicle: train, repo: acme/open, reporters_see_all: true}
`

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

type env struct {
	t      *testing.T
	srv    *httptest.Server
	q      *db.Queries
	mailer *mail.Log
	forge  *forge.Fake
	cfg    *config.Config
}

func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("HUBLOT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HUBLOT_TEST_DATABASE_URL not set (make test starts a database)")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Parse(fmt.Sprintf(testConfig, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	fake := forge.NewFake(os.DirFS("../../deploy/dev/forms"))
	p := &portal.Service{Pool: pool, Q: q, Forge: fake, Store: storage.FS{Dir: cfg.Storage.Dir}, Cfg: cfg}
	if err := p.SyncProjects(ctx); err != nil {
		t.Fatal(err)
	}
	mailer := &mail.Log{}
	s := &httpapi.Server{Cfg: cfg, Auth: &auth.Service{Q: q, Mailer: mailer, Cfg: cfg}, Portal: p}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, q: q, mailer: mailer, forge: fake, cfg: cfg}
}

// grant gives a role, creating the person if needed.
func (e *env) grant(email, project, role string) {
	e.t.Helper()
	ctx := context.Background()
	u, err := e.q.CreateUser(ctx, db.CreateUserParams{Email: email, Name: strings.Split(email, "@")[0], OrgRole: db.OrgRoleMember, Lang: "fr"})
	if err != nil {
		e.t.Fatal(err)
	}
	if project == "" {
		return
	}
	p, err := e.q.GetProjectBySlug(ctx, project)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.q.UpsertUserGrant(ctx, db.UpsertUserGrantParams{
		ProjectID: p.ID, UserID: uuid.NullUUID{UUID: u.ID, Valid: true}, Role: db.ProjectRole(role),
	}); err != nil {
		e.t.Fatal(err)
	}
}

type client struct {
	e    *env
	http *http.Client
}

var tokenRe = regexp.MustCompile(`token=([A-Za-z0-9_\-%]+)`)

// login goes through the real magic-link flow and returns a signed-in client.
func (e *env) login(email string) *client {
	e.t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &client{e: e, http: &http.Client{Jar: jar}}
	if st, _ := c.do("POST", "/api/auth/magic-link", map[string]string{"email": email}); st != http.StatusAccepted {
		e.t.Fatalf("magic link for %s: %d", email, st)
	}
	sent := e.mailer.Sent()
	m := tokenRe.FindStringSubmatch(sent[len(sent)-1].Text)
	if m == nil {
		e.t.Fatalf("no link in %q", sent[len(sent)-1].Text)
	}
	token, _ := url.QueryUnescape(m[1])
	if st, body := c.do("POST", "/api/auth/verify", map[string]string{"token": token}); st != http.StatusOK {
		e.t.Fatalf("verify %s: %d %s", email, st, body)
	}
	return c
}

func (c *client) do(method, path string, body any) (int, []byte) {
	c.e.t.Helper()
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hublot", "1")
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func (c *client) upload(name string, content []byte) (int, map[string]any) {
	c.e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", name)
	fw.Write(content)
	mw.Close()
	req, _ := http.NewRequest("POST", c.e.srv.URL+"/api/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Hublot", "1")
	resp, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func (c *client) file(project, title string, attachments ...string) string {
	c.e.t.Helper()
	st, body := c.do("POST", "/api/requests", map[string]any{
		"project": project, "kind": "question", "title": title,
		"fields": map[string]any{"question": "Comment faire ?"}, "attachments": attachments,
	})
	if st != http.StatusCreated {
		c.e.t.Fatalf("file %q in %s: %d %s", title, project, st, body)
	}
	var v struct{ ID string }
	json.Unmarshal(body, &v)
	return v.ID
}

func (c *client) projects() []string {
	c.e.t.Helper()
	_, body := c.do("GET", "/api/me", nil)
	var me struct{ Projects []struct{ Slug string } }
	json.Unmarshal(body, &me)
	var out []string
	for _, p := range me.Projects {
		out = append(out, p.Slug)
	}
	return out
}

func (c *client) titles(path string) []string {
	c.e.t.Helper()
	st, body := c.do("GET", path, nil)
	if st != http.StatusOK {
		return []string{fmt.Sprintf("status %d", st)}
	}
	var list struct{ Requests []struct{ Title string } }
	json.Unmarshal(body, &list)
	out := []string{}
	for _, r := range list.Requests {
		out = append(out, r.Title)
	}
	return out
}

func eq(t *testing.T, what string, got, want any) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

func TestIsolation(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("paul@acme.test", "app", "reporter")
	e.grant("vic@acme.test", "shop", "viewer")
	e.grant("tom@acme.test", "shop", "triager")
	e.grant("nobody@acme.test", "", "")
	marie, paul, vic, tom, nobody := e.login("marie@acme.test"), e.login("paul@acme.test"),
		e.login("vic@acme.test"), e.login("tom@acme.test"), e.login("nobody@acme.test")
	boss := e.login("boss@acme.test")

	st, att := marie.upload("capture.png", pngHeader)
	eq(t, "upload", st, http.StatusCreated)
	attID := att["id"].(string)
	shopReq := marie.file("shop", "Le PDF est illisible", attID)
	appReq := paul.file("app", "Notification manquante")

	t.Run("project list", func(t *testing.T) {
		eq(t, "marie", marie.projects(), []string{"shop"})
		eq(t, "paul", paul.projects(), []string{"app"})
		eq(t, "nobody", nobody.projects(), []string{})
		eq(t, "boss", boss.projects(), []string{"shop", "app", "open"})
	})

	t.Run("forms", func(t *testing.T) {
		st, _ := marie.do("GET", "/api/projects/shop/forms", nil)
		eq(t, "own project", st, http.StatusOK)
		st, _ = marie.do("GET", "/api/projects/app/forms", nil)
		eq(t, "other project answers 404, not 403", st, http.StatusNotFound)
		st, _ = vic.do("GET", "/api/projects/shop/forms", nil)
		eq(t, "viewer cannot file", st, http.StatusForbidden)
	})

	t.Run("filing elsewhere", func(t *testing.T) {
		st, _ := marie.do("POST", "/api/requests", map[string]any{
			"project": "app", "kind": "question", "title": "x", "fields": map[string]any{"question": "y"},
		})
		eq(t, "marie in app", st, http.StatusNotFound)
		st, _ = vic.do("POST", "/api/requests", map[string]any{
			"project": "shop", "kind": "question", "title": "x", "fields": map[string]any{"question": "y"},
		})
		eq(t, "viewer in shop", st, http.StatusForbidden)
	})

	t.Run("request lists", func(t *testing.T) {
		eq(t, "marie mine", marie.titles("/api/requests?scope=mine"), []string{"Le PDF est illisible"})
		eq(t, "paul mine", paul.titles("/api/requests?scope=mine"), []string{"Notification manquante"})
		eq(t, "paul lists shop", paul.titles("/api/requests?scope=all&project=shop"), []string{"status 404"})
		eq(t, "viewer sees all of shop", vic.titles("/api/requests?scope=all&project=shop"), []string{"Le PDF est illisible"})
		eq(t, "nobody lists shop", nobody.titles("/api/requests?scope=all&project=shop"), []string{"status 404"})
	})

	t.Run("request detail", func(t *testing.T) {
		for name, c := range map[string]*client{"paul": paul, "nobody": nobody} {
			st, _ := c.do("GET", "/api/requests/"+shopReq, nil)
			eq(t, name, st, http.StatusNotFound)
		}
		st, _ := marie.do("GET", "/api/requests/"+appReq, nil)
		eq(t, "marie on app", st, http.StatusNotFound)
		st, body := marie.do("GET", "/api/requests/"+shopReq, nil)
		eq(t, "author", st, http.StatusOK)
		if !strings.Contains(string(body), attID) {
			t.Errorf("author should see the attachment: %s", body)
		}
		st, _ = boss.do("GET", "/api/requests/"+appReq, nil)
		eq(t, "owner", st, http.StatusOK)
	})

	t.Run("reporters see only theirs", func(t *testing.T) {
		e.grant("zoe@acme.test", "shop", "reporter")
		zoe := e.login("zoe@acme.test")
		eq(t, "zoe lists shop", zoe.titles("/api/requests?scope=all&project=shop"), []string{})
		st, _ := zoe.do("GET", "/api/requests/"+shopReq, nil)
		eq(t, "zoe reads marie's", st, http.StatusNotFound)
	})

	t.Run("project open to reporters, but not its private requests", func(t *testing.T) {
		e.grant("ana@acme.test", "open", "reporter")
		e.grant("bob@acme.test", "open", "reporter")
		ana, bob := e.login("ana@acme.test"), e.login("bob@acme.test")
		ana.file("open", "Public pour tous")
		st, body := ana.do("POST", "/api/requests", map[string]any{
			"project": "open", "kind": "question", "title": "Données client", "private": true,
			"fields": map[string]any{"question": "?"},
		})
		eq(t, "private filed", st, http.StatusCreated)
		eq(t, "bob lists open", bob.titles("/api/requests?scope=all&project=open"), []string{"Public pour tous"})
		var v struct{ ID string }
		json.Unmarshal(body, &v)
		st, _ = bob.do("GET", "/api/requests/"+v.ID, nil)
		eq(t, "bob reads the private one", st, http.StatusNotFound)
		st, _ = ana.do("GET", "/api/requests/"+v.ID, nil)
		eq(t, "its author does", st, http.StatusOK)
	})

	t.Run("attachments", func(t *testing.T) {
		for name, c := range map[string]*client{"paul": paul, "nobody": nobody} {
			st, _ := c.do("GET", "/api/attachments/"+attID, nil)
			eq(t, name, st, http.StatusNotFound)
		}
		for name, c := range map[string]*client{"marie": marie, "viewer": vic, "triager": tom} {
			st, _ := c.do("GET", "/api/attachments/"+attID, nil)
			eq(t, name, st, http.StatusOK)
		}
		st, _ := marie.do("DELETE", "/api/attachments/"+attID, nil)
		eq(t, "a filed attachment cannot be deleted", st, http.StatusNotFound)

		_, other := paul.upload("draft.png", pngHeader)
		st, _ = marie.do("GET", "/api/attachments/"+other["id"].(string), nil)
		eq(t, "someone else's draft upload", st, http.StatusNotFound)
		st, body := marie.do("POST", "/api/requests", map[string]any{
			"project": "shop", "kind": "question", "title": "x", "fields": map[string]any{"question": "y"},
			"attachments": []string{other["id"].(string)},
		})
		eq(t, "filing with someone else's upload", st, http.StatusUnprocessableEntity)
		if !strings.Contains(string(body), `"attachments":"invalid"`) {
			t.Errorf("body: %s", body)
		}
		st, _ = nobody.upload("x.png", pngHeader)
		eq(t, "someone who can file nowhere cannot upload", st, http.StatusForbidden)
	})

	t.Run("signed links written in the issue", func(t *testing.T) {
		issues := e.forge.Created(forge.Repo{Owner: "acme", Name: "shop"})
		link := regexp.MustCompile(`http://portal\.test(/files/[^)\s]+)`).FindStringSubmatch(issues[0].Body)
		if link == nil {
			t.Fatalf("no signed link in the body:\n%s", issues[0].Body)
		}
		resp, err := http.Get(e.srv.URL + link[1])
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		eq(t, "signed link", resp.StatusCode, http.StatusOK)
		eq(t, "sandboxed", strings.Contains(resp.Header.Get("Content-Security-Policy"), "sandbox"), true)
		resp, _ = http.Get(e.srv.URL + "/files/" + attID + "?s=forged")
		resp.Body.Close()
		eq(t, "forged signature", resp.StatusCode, http.StatusNotFound)
	})
}

func TestFilingARequest(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")

	st, body := marie.do("POST", "/api/requests", map[string]any{
		"project": "shop", "kind": "problem", "title": "  ",
		"fields": map[string]any{"impact": "Pas dans la liste"},
	})
	eq(t, "invalid", st, http.StatusUnprocessableEntity)
	var verr struct {
		Error struct{ Fields map[string]string }
	}
	json.Unmarshal(body, &verr)
	eq(t, "fields", verr.Error.Fields, map[string]string{"title": "required", "where": "required", "impact": "invalid"})

	st, body = marie.do("POST", "/api/requests", map[string]any{
		"project": "shop", "kind": "problem", "title": "Le PDF est illisible", "private": true,
		"fields": map[string]any{"what": "Tableaux coupés", "where": "Rapports", "impact": "C'est gênant", "ghost": "dropped"},
	})
	eq(t, "created", st, http.StatusCreated)
	var v struct {
		Status, Kind string
		Private      bool
		Number       int
		Project      struct{ Slug string }
	}
	json.Unmarshal(body, &v)
	eq(t, "view", fmt.Sprintf("%s %s %v %d %s", v.Status, v.Kind, v.Private, v.Number, v.Project.Slug), "received problem true 1 shop")

	issue := e.forge.Created(forge.Repo{Owner: "acme", Name: "shop"})[0]
	eq(t, "labels", issue.Labels, []string{"bug", "portal", "portal:private"})
	eq(t, "type", issue.Type, "Bug")
	for _, want := range []string{"<!-- hublot:request v1 id=", "### Où ?\n\nRapports", "### Ce qui s'est passé\n\nTableaux coupés", "par marie (m***@acme.test)"} {
		if !strings.Contains(issue.Body, want) {
			t.Errorf("body misses %q:\n%s", want, issue.Body)
		}
	}
	if strings.Contains(issue.Body, "dropped") {
		t.Error("fields absent from the form must not reach GitHub")
	}

	st, _ = marie.do("POST", "/api/requests", map[string]any{"project": "shop", "kind": "rant", "title": "x"})
	eq(t, "unknown kind", st, http.StatusUnprocessableEntity)
}

func TestAttachmentChecks(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")

	// The type comes from the content: an SVG named .png is kept as inert plain text.
	st, svg := marie.upload("evil.png", []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>"))
	eq(t, "svg named .png", st, http.StatusCreated)
	eq(t, "stored as text", svg["contentType"], "text/plain; charset=utf-8")
	resp, _ := marie.http.Get(e.srv.URL + "/api/attachments/" + svg["id"].(string))
	resp.Body.Close()
	eq(t, "served as text", resp.Header.Get("Content-Type"), "text/plain; charset=utf-8")
	eq(t, "downloaded, not shown", strings.HasPrefix(resp.Header.Get("Content-Disposition"), "attachment"), true)
	st, _ = marie.upload("page.html", []byte("<!doctype html><script>alert(1)</script>"))
	eq(t, "html", st, http.StatusUnsupportedMediaType)
	st, _ = marie.upload("big.png", append(append([]byte{}, pngHeader...), make([]byte, 4096)...))
	eq(t, "over max_bytes", st, http.StatusRequestEntityTooLarge)
	st, att := marie.upload(`..\\..\\etc\\notes.txt`, []byte("plain text notes"))
	eq(t, "text", st, http.StatusCreated)
	eq(t, "cleaned name", att["name"], "notes.txt")
	st, _ = marie.do("DELETE", "/api/attachments/"+att["id"].(string), nil)
	eq(t, "delete own draft", st, http.StatusNoContent)
}

func TestMagicLink(t *testing.T) {
	e := setup(t)
	jar, _ := cookiejar.New(nil)
	c := &client{e: e, http: &http.Client{Jar: jar}}

	st, _ := c.do("POST", "/api/auth/magic-link", map[string]string{"email": "not an email"})
	eq(t, "malformed", st, http.StatusUnprocessableEntity)
	for i := 0; i < auth.MaxLinksPerEmail; i++ {
		st, _ = c.do("POST", "/api/auth/magic-link", map[string]string{"email": "mallory@evil.test"})
		eq(t, "other domain answers like any other", st, http.StatusAccepted)
	}
	st, _ = c.do("POST", "/api/auth/magic-link", map[string]string{"email": "mallory@evil.test"})
	eq(t, "and is rate limited like any other", st, http.StatusTooManyRequests)
	eq(t, "but sends nothing", len(e.mailer.Sent()), 0)

	st, _ = c.do("GET", "/api/me", nil)
	eq(t, "signed out", st, http.StatusUnauthorized)

	for i := 0; i < auth.MaxLinksPerEmail; i++ {
		st, _ = c.do("POST", "/api/auth/magic-link", map[string]string{"email": "Marie@ACME.test", "lang": "en"})
		eq(t, "link", st, http.StatusAccepted)
	}
	st, _ = c.do("POST", "/api/auth/magic-link", map[string]string{"email": "marie@acme.test"})
	eq(t, "rate limited", st, http.StatusTooManyRequests)
	sent := e.mailer.Sent()
	eq(t, "lowercased recipient", sent[0].To, "marie@acme.test")
	eq(t, "english", sent[0].Subject, "Your sign-in link")

	token, _ := url.QueryUnescape(tokenRe.FindStringSubmatch(sent[0].Text)[1])
	st, _ = c.do("POST", "/api/auth/verify", map[string]string{"token": token})
	eq(t, "verify", st, http.StatusOK)
	st, body := c.do("GET", "/api/me", nil)
	eq(t, "signed in", st, http.StatusOK)
	eq(t, "deny by default", strings.Contains(string(body), `"projects":[]`), true)

	other := &client{e: e, http: &http.Client{}}
	st, _ = other.do("POST", "/api/auth/verify", map[string]string{"token": token})
	eq(t, "a link works once", st, http.StatusBadRequest)

	req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/logout", nil)
	resp, _ := c.http.Do(req)
	resp.Body.Close()
	eq(t, "no CSRF header", resp.StatusCode, http.StatusForbidden)

	st, _ = c.do("POST", "/api/auth/logout", nil)
	eq(t, "logout", st, http.StatusNoContent)
	st, _ = c.do("GET", "/api/me", nil)
	eq(t, "signed out again", st, http.StatusUnauthorized)

	boss := e.login("boss@acme.test")
	_, body = boss.do("GET", "/api/me", nil)
	eq(t, "bootstrap owner", strings.Contains(string(body), `"orgRole":"owner"`), true)
}
