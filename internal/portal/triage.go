// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/C2-POLY/hublot/internal/authz"
	"github.com/C2-POLY/hublot/internal/config"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
)

// Triage (PRD §5 V1): the requests nobody classified yet, across every project the person
// may triage, and the actions that classify them. The triage state lives in GitHub (the
// label portal:new); each action is written to GitHub first, then applied to the cache
// the way a webhook would be. Once GitHub accepted the write, the action is audited even
// when the local step fails (details.local_error), since it happened in GitHub.

// Triage actions, as in POST /api/triage/{id}/{action}.
const (
	TriageAccept   = "accept"
	TriageKind     = "kind"
	TriageTransfer = "transfer"
	TriagePrivate  = "private"
	TriageClose    = "close"
)

// Close reasons a triager may give; "completed" belongs to the team's work in GitHub.
var closeReasons = []string{"not_planned", "duplicate"}

type TriageInput struct {
	Kind    string `json:"kind"`    // kind
	Project string `json:"project"` // transfer: the destination project slug
	Private *bool  `json:"private"` // private
	Reason  string `json:"reason"`  // close: not_planned or duplicate
}

type Inbox struct {
	Requests []RequestView `json:"requests"`
	// Projects are the ones the person may triage: the filters and the transfer destinations.
	Projects []ProjectRef `json:"projects"`
}

// triageProjects lists the active projects the subject may triage.
func (s *Service) triageProjects(sub authz.Subject) []db.Project {
	out := []db.Project{}
	for _, p := range s.activeProjects() {
		if authz.Can(sub, authz.TriageProject, authzProject(p)) {
			out = append(out, p)
		}
	}
	return out
}

// CanTriage reports whether the subject may triage at least one project.
func (s *Service) CanTriage(sub authz.Subject) bool { return len(s.triageProjects(sub)) > 0 }

// Inbox returns the open requests labelled portal:new of the projects the subject may
// triage, the inbox project first, then the newest. Nothing to triage is an empty list.
func (s *Service) Inbox(ctx context.Context, sub authz.Subject) (Inbox, error) {
	projects := s.triageProjects(sub)
	out := Inbox{Requests: []RequestView{}, Projects: []ProjectRef{}}
	ids := make([]uuid.UUID, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
		out.Projects = append(out.Projects, projectRef(p))
		if authz.ViaOrgRoleOnly(sub, p.ID) {
			if err := s.audit(ctx, sub.UserID, "triage.list", "project", p.Slug, nil); err != nil {
				return Inbox{}, err
			}
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Q.ListTriageInbox(ctx, db.ListTriageInboxParams{ProjectIds: ids, Label: forge.LabelNew})
	if err != nil {
		return Inbox{}, err
	}
	for _, r := range rows {
		row := requestRow(r)
		ar, _, ok := s.authzRequest(row)
		if !ok || !authz.Can(sub, authz.TriageRequest, ar) {
			continue
		}
		if v, ok := s.visible(sub, row); ok {
			out.Requests = append(out.Requests, v)
		}
	}
	return out, nil
}

// Triage applies one triage action to a request. A request the subject cannot see answers
// ErrNotFound; one they see but may not triage, ErrForbidden. The answer is the request
// as the subject now sees it, nil when they no longer may.
func (s *Service) Triage(ctx context.Context, sub authz.Subject, id uuid.UUID, action string, in TriageInput) (*RequestView, error) {
	r, err := s.Q.GetRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	ar, p, ok := s.authzRequest(requestRow(r))
	if !ok || !authz.Can(sub, authz.ViewRequest, ar) {
		return nil, ErrNotFound
	}
	if !authz.Can(sub, authz.TriageRequest, ar) {
		return nil, ErrForbidden
	}
	repo := forge.Repo{Owner: r.RepoOwner, Name: r.RepoName}
	details := map[string]any{}
	if authz.ViaOrgRoleOnly(sub, p.ID) {
		details["via_org_role"] = true
	}

	// Each action writes to GitHub and returns the local step that follows.
	var local func() error
	switch action {
	case TriageAccept:
		local, err = s.updateIssue(ctx, repo, int(r.IssueNumber), forge.IssueUpdate{RemoveLabels: []string{forge.LabelNew}})
	case TriagePrivate:
		if in.Private == nil {
			return nil, &ValidationError{Fields: map[string]string{"private": "required"}}
		}
		details["private"] = *in.Private
		up := forge.IssueUpdate{RemoveLabels: []string{forge.LabelPrivate}}
		if *in.Private {
			up = forge.IssueUpdate{AddLabels: []string{forge.LabelPrivate}}
		}
		local, err = s.updateIssue(ctx, repo, int(r.IssueNumber), up)
	case TriageClose:
		if !slices.Contains(closeReasons, in.Reason) {
			return nil, &ValidationError{Fields: map[string]string{"reason": "invalid"}}
		}
		details["reason"] = in.Reason
		local, err = s.updateIssue(ctx, repo, int(r.IssueNumber), forge.IssueUpdate{
			State: "closed", StateReason: in.Reason, RemoveLabels: []string{forge.LabelNew},
		})
	case TriageKind:
		details["from"], details["to"] = r.Kind, in.Kind
		local, err = s.changeKind(ctx, p, r, in.Kind)
	case TriageTransfer:
		var to db.Project
		if to, err = s.transferTarget(sub, ar, in.Project); err != nil {
			return nil, err
		}
		details["from"], details["to"] = p.Slug, to.Slug
		local, err = s.transfer(ctx, r, to)
	default:
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	localErr := local()
	if localErr != nil {
		details["local_error"] = true
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return nil, err
	}
	// The write happened in GitHub: audit it even when the request was cancelled meanwhile.
	if err := s.audit(context.WithoutCancel(ctx), sub.UserID, "triage."+action, "request", id.String(), raw); err != nil {
		return nil, errors.Join(localErr, err)
	}
	if localErr != nil {
		return nil, localErr
	}
	updated, err := s.Q.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	if v, ok := s.visible(sub, requestRow(updated)); ok {
		return &v, nil
	}
	return nil, nil
}

// updateIssue writes a change to GitHub; the local step applies the issue GitHub answered with.
func (s *Service) updateIssue(ctx context.Context, repo forge.Repo, number int, up forge.IssueUpdate) (func() error, error) {
	issue, err := s.Forge.UpdateIssue(ctx, repo, number, up)
	if err != nil {
		return nil, err
	}
	return func() error { return s.applyIssue(ctx, repo, issue, nil) }, nil
}

// changeKind swaps the labels of the old kind's form for the new one's, sets the issue
// type (issue_types, else the new form's type, else unchanged); the local step sets the
// request's kind.
func (s *Service) changeKind(ctx context.Context, p db.Project, r db.GetRequestRow, kind string) (func() error, error) {
	if !slices.Contains(config.Kinds, kind) {
		return nil, &ValidationError{Fields: map[string]string{"kind": "invalid"}}
	}
	repo := forge.Repo{Owner: r.RepoOwner, Name: r.RepoName}
	forms, err := s.Forge.IssueForms(ctx, repo)
	if err != nil {
		return nil, err
	}
	cp, _ := s.Cfg.Project(p.Slug)
	next, ok := forms[s.Cfg.FormFor(cp, kind)]
	if !ok {
		return nil, &ValidationError{Fields: map[string]string{"kind": "invalid"}}
	}
	if kind == r.Kind {
		return func() error { return nil }, nil
	}
	up := forge.IssueUpdate{AddLabels: next.Labels, Type: next.Type}
	if t := s.Cfg.IssueTypes[kind]; t != "" {
		up.Type = t
	}
	for _, l := range forms[s.Cfg.FormFor(cp, r.Kind)].Labels {
		if !slices.Contains(next.Labels, l) {
			up.RemoveLabels = append(up.RemoveLabels, l)
		}
	}
	issue, err := s.Forge.UpdateIssue(ctx, repo, int(r.IssueNumber), up)
	if err != nil {
		return nil, err
	}
	return func() error {
		// The kind is the portal's own data; set it before the cache path recomputes the
		// status, which depends on it (a closed question is answered, not available).
		if err := s.Q.SetRequestKind(ctx, db.SetRequestKindParams{ID: r.ID, Kind: kind}); err != nil {
			return err
		}
		return s.applyIssue(ctx, repo, issue, nil)
	}, nil
}

// transferTarget resolves the destination of a transfer: a project the subject cannot
// see answers ErrNotFound, one they see but may not triage, ErrForbidden.
func (s *Service) transferTarget(sub authz.Subject, r authz.Request, slug string) (db.Project, error) {
	to, ok := s.projectBySlug(slug)
	if !ok || !authz.Can(sub, authz.ViewProject, authzProject(to)) {
		return db.Project{}, &ValidationError{Fields: map[string]string{"project": "invalid"}}
	}
	if to.ID == r.Project.ID {
		return db.Project{}, &ValidationError{Fields: map[string]string{"project": "same"}}
	}
	if !authz.Can(sub, authz.TransferRequest, authz.Transfer{Request: r, To: authzProject(to)}) {
		return db.Project{}, ErrForbidden
	}
	return to, nil
}

// transfer moves the issue to the destination's repository, then follows it as the
// transferred webhook would: the request points at the new issue, origin_project_id stays.
func (s *Service) transfer(ctx context.Context, r db.GetRequestRow, to db.Project) (func() error, error) {
	moved, err := s.Forge.TransferIssue(ctx, r.IssueNodeID, repoOf(to))
	if err != nil {
		return nil, err
	}
	from := forge.Repo{Owner: r.RepoOwner, Name: r.RepoName}
	return func() error {
		return s.transferred(ctx, from, forge.Issue{Number: int(r.IssueNumber), NodeID: r.IssueNodeID}, repoOf(to), moved)
	}, nil
}

func projectRef(p db.Project) ProjectRef {
	return ProjectRef{Slug: p.Slug, Name: p.Name, Color: p.Color, Vehicle: p.Vehicle}
}
