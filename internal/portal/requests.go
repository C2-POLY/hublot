// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/et-technology-consulting/hublot/internal/authz"
	"github.com/et-technology-consulting/hublot/internal/config"
	"github.com/et-technology-consulting/hublot/internal/db"
	"github.com/et-technology-consulting/hublot/internal/forge"
)

const (
	maxTitle       = 200
	maxValue       = 10_000
	maxAttachments = 10
)

// CreateInput is a request as the person filed it. Field values are a string,
// or a list of strings for checkboxes and multiple dropdowns.
type CreateInput struct {
	Project     string         `json:"project"`
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	Fields      map[string]any `json:"fields"`
	Attachments []uuid.UUID    `json:"attachments"`
	Private     bool           `json:"private"`
}

type RequestView struct {
	ID          uuid.UUID        `json:"id"`
	Project     ProjectRef       `json:"project"`
	Kind        string           `json:"kind"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	Private     bool             `json:"private"`
	Number      int32            `json:"number"`
	CreatedAt   time.Time        `json:"createdAt"`
	Author      AuthorRef        `json:"author"`
	Attachments []AttachmentView `json:"attachments"`
}

type ProjectRef struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Vehicle string `json:"vehicle"`
}

type AuthorRef struct {
	Name string `json:"name"`
}

// CreateRequest validates the request against the repository's issue form, files the
// issue in GitHub, then records the person <-> issue link and the cache.
func (s *Service) CreateRequest(ctx context.Context, sub authz.Subject, author db.AppUser, in CreateInput) (RequestView, error) {
	p, err := s.creatableProject(sub, in.Project)
	if err != nil {
		return RequestView{}, err
	}
	invalid := map[string]string{}
	title := strings.Join(strings.Fields(in.Title), " ")
	switch {
	case title == "":
		invalid["title"] = "required"
	case utf8.RuneCountInString(title) > maxTitle:
		invalid["title"] = "too_long"
	}

	var form forge.IssueForm
	if !slices.Contains(config.Kinds, in.Kind) {
		invalid["kind"] = "invalid"
	} else {
		forms, err := s.Forge.IssueForms(ctx, repoOf(p))
		if err != nil {
			return RequestView{}, err
		}
		cp, _ := s.Cfg.Project(p.Slug)
		f, ok := forms[s.Cfg.FormFor(cp, in.Kind)]
		if !ok {
			invalid["kind"] = "invalid"
		}
		form = f
	}
	values := validateFields(form, in.Fields, invalid)

	attachments, err := s.ownUnlinked(ctx, author.ID, in.Attachments, invalid)
	if err != nil {
		return RequestView{}, err
	}
	if len(invalid) > 0 {
		return RequestView{}, &ValidationError{Fields: invalid}
	}

	id := uuid.New()
	labels := slices.Clone(form.Labels)
	labels = append(labels, forge.LabelPortal)
	if in.Private {
		labels = append(labels, forge.LabelPrivate)
	}
	bodyAttachments := make([]forge.BodyAttachment, 0, len(attachments))
	for _, a := range attachments {
		bodyAttachments = append(bodyAttachments, forge.BodyAttachment{
			Name: a.Name, URL: s.SignedURL(a.ID), Image: strings.HasPrefix(a.ContentType, "image/"),
		})
	}
	body := forge.RenderBody(forge.BodyInput{
		RequestID: id.String(), Form: form, Values: values, Attachments: bodyAttachments,
		AuthorName: author.Name, AuthorEmail: s.authorEmail(author.Email), Lang: author.Lang,
	})
	issue, err := s.Forge.CreateIssue(ctx, repoOf(p), forge.NewIssue{Title: title, Body: body, Labels: labels, Type: form.Type})
	if err != nil {
		return RequestView{}, err
	}

	// The issue now exists in GitHub, the source of truth. If recording the link fails,
	// the marker in its body lets a resync attach it back to its author.
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return RequestView{}, err
	}
	defer tx.Rollback(ctx)
	q := s.Q.WithTx(tx)
	row, err := q.CreateRequest(ctx, db.CreateRequestParams{
		ID: id, ProjectID: p.ID, AuthorID: author.ID, Kind: in.Kind,
		RepoOwner: p.RepoOwner, RepoName: p.RepoName,
		IssueNumber: int32(issue.Number), IssueNodeID: issue.NodeID,
	})
	if err != nil {
		return RequestView{}, fmt.Errorf("record request for %s#%d: %w", repoOf(p), issue.Number, err)
	}
	if err := q.UpsertIssueCache(ctx, db.UpsertIssueCacheParams{
		RepoOwner: p.RepoOwner, RepoName: p.RepoName, IssueNumber: int32(issue.Number),
		Title: issue.Title, State: issue.State, StateReason: issue.StateReason, Labels: nonNil(issue.Labels),
	}); err != nil {
		return RequestView{}, err
	}
	for i := range attachments {
		n, err := q.LinkAttachment(ctx, db.LinkAttachmentParams{
			RequestID: uuid.NullUUID{UUID: id, Valid: true}, ID: attachments[i].ID, AuthorID: author.ID,
		})
		if err != nil {
			return RequestView{}, err
		}
		if n == 0 {
			// Filed with another request or deleted meanwhile. The issue already exists in
			// GitHub; its marker lets a resync attach it back to its author.
			return RequestView{}, &ValidationError{Fields: map[string]string{"attachments": "invalid"}}
		}
		attachments[i].RequestID = uuid.NullUUID{UUID: id, Valid: true}
	}
	if err := tx.Commit(ctx); err != nil {
		return RequestView{}, err
	}

	return s.view(p, row.ID, row.Kind, issue.Title, issue.State, issue.Labels, row.IssueNumber, row.CreatedAt, author.Name, attachments), nil
}

// validateFields checks the answers against the form and returns them as lists of strings.
// Unknown field ids are dropped: the form in the repository is the reference.
func validateFields(form forge.IssueForm, raw map[string]any, invalid map[string]string) map[string][]string {
	values := map[string][]string{}
	for _, f := range form.Fields {
		if f.Type == "markdown" {
			continue
		}
		vals, ok := asStrings(raw[f.ID])
		if !ok {
			invalid[f.ID] = "invalid"
			continue
		}
		for i := range vals {
			vals[i] = strings.TrimSpace(vals[i])
		}
		vals = slices.DeleteFunc(vals, func(v string) bool { return v == "" })
		for _, v := range vals {
			if utf8.RuneCountInString(v) > maxValue {
				invalid[f.ID] = "too_long"
			}
		}
		switch f.Type {
		case "input", "textarea":
			if len(vals) > 1 {
				invalid[f.ID] = "invalid"
			}
		case "dropdown":
			if len(vals) > 1 && !f.Multiple {
				invalid[f.ID] = "invalid"
			}
			if !allIn(vals, f.Options) {
				invalid[f.ID] = "invalid"
			}
		case "checkboxes":
			if !allIn(vals, f.Options) {
				invalid[f.ID] = "invalid"
			}
			if !allIn(f.RequiredOptions, vals) {
				invalid[f.ID] = "required"
			}
		}
		if f.Required && len(vals) == 0 {
			invalid[f.ID] = "required"
		}
		values[f.ID] = vals
	}
	return values
}

func asStrings(v any) ([]string, bool) {
	switch t := v.(type) {
	case nil:
		return nil, true
	case string:
		return []string{t}, true
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

func allIn(vals, allowed []string) bool {
	for _, v := range vals {
		if !slices.Contains(allowed, v) {
			return false
		}
	}
	return true
}

func (s *Service) authorEmail(email string) string {
	switch s.Cfg.Forge.AuthorEmail {
	case "full":
		return email
	case "hidden":
		return ""
	}
	return forge.MaskEmail(email)
}

// ListMine returns the subject's own requests that they may still see.
func (s *Service) ListMine(ctx context.Context, sub authz.Subject) ([]RequestView, error) {
	rows, err := s.Q.ListRequestsByAuthor(ctx, sub.UserID)
	if err != nil {
		return nil, err
	}
	out := []RequestView{}
	for _, r := range rows {
		if v, ok := s.visible(sub, requestRow(r)); ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// ListProject returns every request of a project the subject may see.
func (s *Service) ListProject(ctx context.Context, sub authz.Subject, slug string) ([]RequestView, error) {
	p, ok := s.projectBySlug(slug)
	if !ok || !authz.Can(sub, authz.ViewProject, authzProject(p)) {
		return nil, ErrNotFound
	}
	if authz.ViaOrgRoleOnly(sub, p.ID) {
		if err := s.audit(ctx, sub.UserID, "requests.list", "project", p.Slug, nil); err != nil {
			return nil, err
		}
	}
	rows, err := s.Q.ListRequestsByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := []RequestView{}
	for _, r := range rows {
		if v, ok := s.visible(sub, requestRow(r)); ok {
			out = append(out, v)
		}
	}
	return out, nil
}

// GetRequest returns one request with its attachments, or ErrNotFound.
func (s *Service) GetRequest(ctx context.Context, sub authz.Subject, id uuid.UUID) (RequestView, error) {
	r, err := s.Q.GetRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return RequestView{}, ErrNotFound
	}
	if err != nil {
		return RequestView{}, err
	}
	row := requestRow(r)
	v, ok := s.visible(sub, row)
	if !ok {
		return RequestView{}, ErrNotFound
	}
	if authz.ViaOrgRoleOnly(sub, row.ProjectID) {
		if err := s.audit(ctx, sub.UserID, "request.view", "request", id.String(), nil); err != nil {
			return RequestView{}, err
		}
	}
	atts, err := s.Q.ListRequestAttachments(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return RequestView{}, err
	}
	for _, a := range atts {
		v.Attachments = append(v.Attachments, attachmentView(a))
	}
	return v, nil
}

// row is the common shape of the request list queries.
type row struct {
	ID, ProjectID, OriginProjectID, AuthorID uuid.UUID
	Kind, Title, State, AuthorName           string
	Labels                                   []string
	IssueNumber                              int32
	CreatedAt                                time.Time
}

func requestRow[T db.GetRequestRow | db.ListRequestsByAuthorRow | db.ListRequestsByProjectRow](r T) row {
	// The three generated row types share the same fields.
	g := db.GetRequestRow(r)
	return row{
		ID: g.ID, ProjectID: g.ProjectID, OriginProjectID: g.OriginProjectID, AuthorID: g.AuthorID,
		Kind: g.Kind, Title: g.Title, State: g.State, AuthorName: g.AuthorName,
		Labels: g.Labels, IssueNumber: g.IssueNumber, CreatedAt: g.CreatedAt,
	}
}

func (s *Service) authzRequest(r row) (authz.Request, db.Project, bool) {
	p, ok := s.projectByID(r.ProjectID)
	if !ok {
		return authz.Request{}, db.Project{}, false
	}
	return authz.Request{
		Project: authzProject(p), OriginProject: r.OriginProjectID,
		AuthorID: r.AuthorID, Private: slices.Contains(r.Labels, forge.LabelPrivate),
	}, p, true
}

func (s *Service) visible(sub authz.Subject, r row) (RequestView, bool) {
	ar, p, ok := s.authzRequest(r)
	if !ok || !authz.Can(sub, authz.ViewRequest, ar) {
		return RequestView{}, false
	}
	return s.view(p, r.ID, r.Kind, r.Title, r.State, r.Labels, r.IssueNumber, r.CreatedAt, r.AuthorName, nil), true
}

func (s *Service) view(p db.Project, id uuid.UUID, kind, title, state string, labels []string,
	number int32, created time.Time, author string, atts []db.Attachment) RequestView {
	v := RequestView{
		ID:          id,
		Project:     ProjectRef{Slug: p.Slug, Name: p.Name, Color: p.Color, Vehicle: p.Vehicle},
		Kind:        kind,
		Title:       title,
		Status:      status(state),
		Private:     slices.Contains(labels, forge.LabelPrivate),
		Number:      number,
		CreatedAt:   created,
		Author:      AuthorRef{Name: author},
		Attachments: []AttachmentView{},
	}
	for _, a := range atts {
		v.Attachments = append(v.Attachments, attachmentView(a))
	}
	return v
}

// status translates GitHub's state into the portal's plain words. Milestone 1 knows only
// open and closed; planned, in progress, in test and available come with webhooks.
func status(state string) string {
	if state == "closed" {
		return "closed"
	}
	return "received"
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
