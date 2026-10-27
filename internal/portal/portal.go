// SPDX-License-Identifier: AGPL-3.0-or-later

// Package portal is the portal's domain: projects, requests and attachments.
// Every read and write goes through authz.Can, on the server, at each call.
package portal

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/C2-POLY/hublot/internal/authz"
	"github.com/C2-POLY/hublot/internal/config"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
	"github.com/C2-POLY/hublot/internal/storage"
)

var (
	// ErrNotFound also answers for what exists but may not be seen, so nothing leaks.
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrTooLarge    = errors.New("file too large")
	ErrUnsupported = errors.New("unsupported file type")
)

// ValidationError lists the fields to fix, by id, with a short reason code.
type ValidationError struct{ Fields map[string]string }

func (e *ValidationError) Error() string { return fmt.Sprintf("validation failed: %v", e.Fields) }

type Service struct {
	Pool  *pgxpool.Pool
	Q     *db.Queries
	Forge forge.Forge
	Store storage.Store
	Cfg   *config.Config
	// Jobs queues background work (NewJobs); following GitHub needs it.
	Jobs *river.Client[pgx.Tx]

	mu       sync.RWMutex
	projects []db.Project
}

// SyncProjects writes the configured projects to the database and archives the ones
// removed from the configuration (never deleted: their requests stay linked).
func (s *Service) SyncProjects(ctx context.Context) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.Q.WithTx(tx)
	slugs := make([]string, 0, len(s.Cfg.Projects))
	for i, p := range s.Cfg.Projects {
		owner, name := p.RepoParts()
		if _, err := q.UpsertProject(ctx, db.UpsertProjectParams{
			Slug: p.Slug, Name: p.Name, Hint: p.Hint, Color: p.Color, Vehicle: p.Vehicle,
			RepoOwner: owner, RepoName: name, ReportersSeeAll: p.ReportersSeeAll, Inbox: p.Inbox,
			Position: int32(i),
		}); err != nil {
			return fmt.Errorf("sync project %s: %w", p.Slug, err)
		}
		slugs = append(slugs, p.Slug)
	}
	if err := q.ArchiveProjectsNotIn(ctx, slugs); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return s.reloadProjects(ctx)
}

func (s *Service) reloadProjects(ctx context.Context) error {
	all, err := s.Q.ListAllProjects(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.projects = all
	s.mu.Unlock()
	return nil
}

func (s *Service) projectByID(id uuid.UUID) (db.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := slices.IndexFunc(s.projects, func(p db.Project) bool { return p.ID == id })
	if i < 0 {
		return db.Project{}, false
	}
	return s.projects[i], true
}

func (s *Service) projectBySlug(slug string) (db.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := slices.IndexFunc(s.projects, func(p db.Project) bool { return p.Slug == slug && !p.Archived })
	if i < 0 {
		return db.Project{}, false
	}
	return s.projects[i], true
}

func (s *Service) activeProjects() []db.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]db.Project, 0, len(s.projects))
	for _, p := range s.projects {
		if !p.Archived {
			out = append(out, p)
		}
	}
	return out
}

// Subject builds the authorization subject of a person: their org role and their
// effective role per project, direct or through groups.
func (s *Service) Subject(ctx context.Context, u db.AppUser) (authz.Subject, error) {
	rows, err := s.Q.EffectiveRoles(ctx, u.ID)
	if err != nil {
		return authz.Subject{}, err
	}
	roles := make(map[uuid.UUID]authz.Role, len(rows))
	for _, r := range rows {
		roles[r.ProjectID] = authz.ParseRole(string(r.Role))
	}
	return authz.Subject{UserID: u.ID, Org: authz.OrgRole(u.OrgRole), Roles: roles}, nil
}

func authzProject(p db.Project) authz.Project {
	return authz.Project{ID: p.ID, ReportersSeeAll: p.ReportersSeeAll}
}

func repoOf(p db.Project) forge.Repo { return forge.Repo{Owner: p.RepoOwner, Name: p.RepoName} }

// ProjectView is a project as the person sees it.
type ProjectView struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Hint      string `json:"hint"`
	Color     string `json:"color"`
	Vehicle   string `json:"vehicle"`
	Role      string `json:"role"`
	CanCreate bool   `json:"canCreate"`
	Inbox     bool   `json:"inbox"`
}

// VisibleProjects lists the projects the subject may at least view. Deny by default.
func (s *Service) VisibleProjects(sub authz.Subject) []ProjectView {
	out := []ProjectView{}
	for _, p := range s.activeProjects() {
		ap := authzProject(p)
		if !authz.Can(sub, authz.ViewProject, ap) {
			continue
		}
		role := sub.Roles[p.ID].String()
		if sub.IsAdmin() && sub.Roles[p.ID] == authz.RoleNone {
			role = string(sub.Org)
		}
		out = append(out, ProjectView{
			Slug: p.Slug, Name: p.Name, Hint: p.Hint, Color: p.Color, Vehicle: p.Vehicle,
			Role: role, CanCreate: authz.Can(sub, authz.CreateRequest, ap), Inbox: p.Inbox,
		})
	}
	return out
}

// FormView is an issue form mapped to a request kind.
type FormView struct {
	Kind        string      `json:"kind"`
	Template    string      `json:"template"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Fields      []FieldView `json:"fields"`
}

type FieldView struct {
	ID          string   `json:"id,omitempty"`
	Type        string   `json:"type"`
	Label       string   `json:"label,omitempty"`
	Description string   `json:"description,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Value       string   `json:"value,omitempty"`
	Options     []string `json:"options,omitempty"`
	Multiple    bool     `json:"multiple,omitempty"`
	Required    bool     `json:"required"`
}

// Forms returns the request forms of a project, one per kind that has a form.
func (s *Service) Forms(ctx context.Context, sub authz.Subject, slug string) ([]FormView, error) {
	p, err := s.creatableProject(sub, slug)
	if err != nil {
		return nil, err
	}
	forms, err := s.Forge.IssueForms(ctx, repoOf(p))
	if err != nil {
		return nil, err
	}
	cp, _ := s.Cfg.Project(p.Slug)
	out := []FormView{}
	for _, kind := range config.Kinds {
		file := s.Cfg.FormFor(cp, kind)
		form, ok := forms[file]
		if !ok {
			continue
		}
		fv := FormView{Kind: kind, Template: file, Name: form.Name, Description: form.Description, Fields: []FieldView{}}
		for _, f := range form.Fields {
			fv.Fields = append(fv.Fields, FieldView{
				ID: f.ID, Type: f.Type, Label: f.Label, Description: f.Description, Placeholder: f.Placeholder,
				Value: f.Value, Options: f.Options, Multiple: f.Multiple, Required: f.Required,
			})
		}
		out = append(out, fv)
	}
	return out, nil
}

// creatableProject resolves a project the subject may file requests in. A project they
// cannot even see answers ErrNotFound; one they see but cannot file in, ErrForbidden.
func (s *Service) creatableProject(sub authz.Subject, slug string) (db.Project, error) {
	p, ok := s.projectBySlug(slug)
	if !ok || !authz.Can(sub, authz.ViewProject, authzProject(p)) {
		return db.Project{}, ErrNotFound
	}
	if !authz.Can(sub, authz.CreateRequest, authzProject(p)) {
		return db.Project{}, ErrForbidden
	}
	return p, nil
}

func (s *Service) audit(ctx context.Context, actor uuid.UUID, action, targetType, targetID string, details []byte) error {
	if details == nil {
		details = []byte("{}")
	}
	return s.Q.InsertAudit(ctx, db.InsertAuditParams{
		ActorID: uuid.NullUUID{UUID: actor, Valid: actor != uuid.Nil}, Via: "portal",
		Action: action, TargetType: targetType, TargetID: targetID, Details: details,
	})
}
