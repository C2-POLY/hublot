// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/C2-POLY/hublot/internal/config"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
)

// The cache follows GitHub: webhooks bring each change, periodic resyncs repair what a
// lost delivery missed (PRD §8). Only issues linked to a request are cached. Every
// change of a request runs in one transaction that locks its row, so two events about
// the same request never interleave.

const (
	// resyncOverlap re-reads a little before the last resync: clocks and GitHub's
	// search index are not exact.
	resyncOverlap = 5 * time.Minute
	// deliveryRetention bounds the webhook_delivery table; GitHub redeliveries are
	// requested within days.
	deliveryRetention = 30 * 24 * time.Hour
)

// RecordDelivery stores a webhook delivery and queues its processing in the same
// transaction: a delivery is recorded with its job or not at all. It reports false for
// a delivery already recorded (a redelivery), which is not queued again.
func (s *Service) RecordDelivery(ctx context.Context, deliveryID, event string, ev forge.Event) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	n, err := s.Q.WithTx(tx).RecordWebhookDelivery(ctx, db.RecordWebhookDeliveryParams{DeliveryID: deliveryID, Event: event})
	if err != nil || n == 0 {
		return false, err
	}
	if _, err := s.Jobs.InsertTx(ctx, tx, eventJob(ev), nil); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// Enqueue queues an event's processing the way a webhook delivery would.
func (s *Service) Enqueue(ctx context.Context, ev forge.Event) error {
	_, err := s.Jobs.Insert(ctx, eventJob(ev), nil)
	return err
}

func eventJob(ev forge.Event) river.JobArgs {
	if ev.Kind == forge.EventProjectItem {
		return ProjectItemArgs{ContentNodeID: ev.ContentNodeID}
	}
	return IssueEventArgs{Event: ev}
}

// ProcessIssueEvent applies an issues event to the request linked to the issue, if any.
func (s *Service) ProcessIssueEvent(ctx context.Context, ev forge.Event) error {
	switch ev.Action {
	case "transferred":
		if ev.NewRepo == nil || ev.NewIssue == nil {
			// GitHub did not say where it went: the resyncs find it by its body marker.
			slog.WarnContext(ctx, "transfer without destination, resyncing", "repo", ev.Repo.String(), "number", ev.Issue.Number)
			return s.enqueueResyncAll(ctx)
		}
		return s.transferred(ctx, ev.Repo, ev.Issue, *ev.NewRepo, *ev.NewIssue)
	case "deleted":
		return s.deleted(ctx, ev.Repo, ev.Issue)
	}
	return s.applyIssue(ctx, ev.Repo, ev.Issue, nil)
}

// ProcessProjectItem re-reads the project status of the issue behind a Projects item.
func (s *Service) ProcessProjectItem(ctx context.Context, contentNodeID string) error {
	req, err := s.Q.FindRequestByNodeID(ctx, contentNodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	st := s.statusConfig(req.ProjectID)
	if st == nil {
		return nil
	}
	repo := forge.Repo{Owner: req.RepoOwner, Name: req.RepoName}
	ps, err := s.Forge.ProjectStatus(ctx, repo, int(req.IssueNumber), st.Project, st.Field)
	if errors.Is(err, forge.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *db.Queries, tx pgx.Tx) error {
		locked, err := q.LockRequest(ctx, req.ID)
		if err != nil {
			return err
		}
		if locked.RepoOwner != req.RepoOwner || locked.RepoName != req.RepoName || locked.IssueNumber != req.IssueNumber {
			return nil // moved meanwhile: the event about the move brings the rest
		}
		if err := q.SetIssueProjectStatus(ctx, db.SetIssueProjectStatusParams{
			ProjectStatus: ps, RepoOwner: req.RepoOwner, RepoName: req.RepoName, IssueNumber: req.IssueNumber,
		}); err != nil {
			return err
		}
		return s.recompute(ctx, q, tx, req.ID)
	})
}

// ResyncProject reads the portal issues of a project's repository and applies them:
// every one when full (or on the first run), otherwise those updated since the last run.
// It returns how many issues GitHub returned.
func (s *Service) ResyncProject(ctx context.Context, slug string, full bool) (int, error) {
	p, err := s.Q.GetProjectBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	var since time.Time
	if !full && p.SyncedAt != nil {
		since = p.SyncedAt.Add(-resyncOverlap)
	}
	started := time.Now()
	repo := repoOf(p)
	issues, err := s.Forge.ListPortalIssues(ctx, repo, since)
	if err != nil {
		return 0, err
	}
	st := s.statusConfig(p.ID)
	for _, issue := range issues {
		var ps *string
		if st != nil {
			// A Project field change does not touch the issue: only a full resync is sure to see it.
			v, err := s.Forge.ProjectStatus(ctx, repo, issue.Number, st.Project, st.Field)
			if err != nil && !errors.Is(err, forge.ErrNotFound) {
				return 0, err
			}
			ps = &v
		}
		if err := s.applyIssue(ctx, repo, issue, ps); err != nil {
			return 0, fmt.Errorf("resync %s#%d: %w", repo, issue.Number, err)
		}
	}
	if err := s.Q.SetProjectSynced(ctx, db.SetProjectSyncedParams{ID: p.ID, SyncedAt: &started}); err != nil {
		return 0, err
	}
	if err := s.Q.PruneWebhookDeliveries(ctx, started.Add(-deliveryRetention)); err != nil {
		return 0, err
	}
	return len(issues), nil
}

func (s *Service) enqueueResyncAll(ctx context.Context) error {
	for _, p := range s.activeProjects() {
		if _, err := s.Jobs.Insert(ctx, ResyncProjectArgs{Project: p.Slug}, nil); err != nil {
			return err
		}
	}
	return nil
}

// applyIssue updates the cache of a linked issue (and its project status when given),
// following the issue when it now lives somewhere else, then recomputes the status.
func (s *Service) applyIssue(ctx context.Context, repo forge.Repo, issue forge.Issue, projectStatus *string) error {
	req, err := s.findRequest(ctx, repo, issue)
	if errors.Is(err, pgx.ErrNoRows) {
		req, err = s.findMovedRequest(ctx, repo, issue)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // not a portal request
	}
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *db.Queries, tx pgx.Tx) error {
		locked, err := q.LockRequest(ctx, req.ID)
		if err != nil {
			return err
		}
		if !at(locked, repo, issue.Number) {
			if err := s.move(ctx, q, locked, repo, issue); err != nil {
				return err
			}
		} else if err := upsertCache(ctx, q, repo, issue); err != nil {
			return err
		}
		if projectStatus != nil {
			if err := q.SetIssueProjectStatus(ctx, db.SetIssueProjectStatusParams{
				ProjectStatus: *projectStatus, RepoOwner: repo.Owner, RepoName: repo.Name, IssueNumber: int32(issue.Number),
			}); err != nil {
				return err
			}
		}
		return s.recompute(ctx, q, tx, req.ID)
	})
}

func (s *Service) transferred(ctx context.Context, from forge.Repo, old forge.Issue, to forge.Repo, moved forge.Issue) error {
	req, err := s.findRequest(ctx, from, old)
	if errors.Is(err, pgx.ErrNoRows) {
		// Not ours, or a resync already followed it.
		return s.applyIssue(ctx, to, moved, nil)
	}
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *db.Queries, tx pgx.Tx) error {
		locked, err := q.LockRequest(ctx, req.ID)
		if err != nil {
			return err
		}
		if err := s.move(ctx, q, locked, to, moved); err != nil {
			return err
		}
		slog.InfoContext(ctx, "request followed its issue", "request", req.ID, "from", from.String(), "to", to.String(), "number", moved.Number)
		return s.recompute(ctx, q, tx, req.ID)
	})
}

func (s *Service) deleted(ctx context.Context, repo forge.Repo, issue forge.Issue) error {
	req, err := s.findRequest(ctx, repo, issue)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *db.Queries, tx pgx.Tx) error {
		locked, err := q.LockRequest(ctx, req.ID)
		if err != nil {
			return err
		}
		if err := q.MarkIssueDeleted(ctx, db.MarkIssueDeletedParams{
			RepoOwner: locked.RepoOwner, RepoName: locked.RepoName, IssueNumber: locked.IssueNumber,
		}); err != nil {
			return err
		}
		return s.recompute(ctx, q, tx, req.ID)
	})
}

// findRequest finds the request of an issue by its place, then by its node id.
func (s *Service) findRequest(ctx context.Context, repo forge.Repo, issue forge.Issue) (db.Request, error) {
	req, err := s.Q.FindRequestByIssue(ctx, db.FindRequestByIssueParams{
		RepoOwner: repo.Owner, RepoName: repo.Name, IssueNumber: int32(issue.Number),
	})
	if errors.Is(err, pgx.ErrNoRows) && issue.NodeID != "" {
		return s.Q.FindRequestByNodeID(ctx, issue.NodeID)
	}
	return req, err
}

// findMovedRequest finds the request of an issue through the marker of its body, for an
// issue transferred without the portal hearing of it. A marker copied into another
// issue must not steal the request: the request follows only if its recorded issue is
// gone, or is this very issue (GitHub redirects a transferred issue).
func (s *Service) findMovedRequest(ctx context.Context, repo forge.Repo, issue forge.Issue) (db.Request, error) {
	id, err := uuid.Parse(issue.RequestID)
	if err != nil {
		return db.Request{}, pgx.ErrNoRows
	}
	req, err := s.Q.FindRequestByID(ctx, id)
	if err != nil {
		return db.Request{}, err
	}
	current, err := s.Forge.GetIssue(ctx, forge.Repo{Owner: req.RepoOwner, Name: req.RepoName}, int(req.IssueNumber))
	switch {
	case errors.Is(err, forge.ErrNotFound):
		return req, nil
	case err != nil:
		return db.Request{}, err
	case current.NodeID == issue.NodeID:
		return req, nil
	}
	slog.WarnContext(ctx, "issue carries the marker of another request, ignored", "repo", repo.String(), "number", issue.Number, "request", id)
	return db.Request{}, pgx.ErrNoRows
}

// move points a request at the issue's new place: the project whose repository it is
// now in (unchanged when no project uses that repository), and the cache follows.
func (s *Service) move(ctx context.Context, q *db.Queries, req db.Request, to forge.Repo, issue forge.Issue) error {
	projectID := req.ProjectID
	if p, ok := s.projectByRepo(to); ok {
		projectID = p.ID
	}
	if err := upsertCache(ctx, q, to, issue); err != nil {
		return err
	}
	if err := q.MoveRequest(ctx, db.MoveRequestParams{
		ID: req.ID, ProjectID: projectID, RepoOwner: to.Owner, RepoName: to.Name,
		IssueNumber: int32(issue.Number), IssueNodeID: issue.NodeID,
	}); err != nil {
		return err
	}
	return q.DeleteIssueCache(ctx, db.DeleteIssueCacheParams{
		RepoOwner: req.RepoOwner, RepoName: req.RepoName, IssueNumber: req.IssueNumber,
	})
}

// recompute derives the request's status from its cache. When it changed, the history
// gets a row and the notification job is queued, in the caller's transaction.
func (s *Service) recompute(ctx context.Context, q *db.Queries, tx pgx.Tx, id uuid.UUID) error {
	req, err := q.FindRequestByID(ctx, id)
	if err != nil {
		return err
	}
	c, err := q.GetIssueCache(ctx, db.GetIssueCacheParams{RepoOwner: req.RepoOwner, RepoName: req.RepoName, IssueNumber: req.IssueNumber})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	status := Status(req.Kind, IssueState{
		State: c.State, StateReason: c.StateReason, Milestone: c.Milestone,
		ProjectStatus: c.ProjectStatus, Deleted: c.DeletedAt != nil,
	}, s.statusOptions(req.ProjectID))
	if status == req.Status {
		return nil
	}
	if err := q.SetRequestStatus(ctx, db.SetRequestStatusParams{ID: id, Status: status}); err != nil {
		return err
	}
	if err := q.InsertRequestStatus(ctx, db.InsertRequestStatusParams{RequestID: id, Status: status}); err != nil {
		return err
	}
	_, err = s.Jobs.InsertTx(ctx, tx, NotifyStatusArgs{RequestID: id, Status: status},
		&river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true}})
	return err
}

func (s *Service) inTx(ctx context.Context, fn func(*db.Queries, pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(s.Q.WithTx(tx), tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func upsertCache(ctx context.Context, q *db.Queries, repo forge.Repo, i forge.Issue) error {
	var updated *time.Time
	if !i.UpdatedAt.IsZero() {
		updated = &i.UpdatedAt
	}
	return q.UpsertIssueCache(ctx, db.UpsertIssueCacheParams{
		RepoOwner: repo.Owner, RepoName: repo.Name, IssueNumber: int32(i.Number), NodeID: i.NodeID,
		Title: i.Title, State: i.State, StateReason: i.StateReason, Labels: nonNil(i.Labels),
		Milestone: i.Milestone, IssueType: i.Type, ClosedAt: i.ClosedAt, IssueUpdatedAt: updated,
	})
}

func at(r db.Request, repo forge.Repo, number int) bool {
	return r.RepoOwner == repo.Owner && r.RepoName == repo.Name && int(r.IssueNumber) == number
}

// projectByRepo returns the active project of a repository. GitHub names ignore case.
func (s *Service) projectByRepo(repo forge.Repo) (db.Project, bool) {
	for _, p := range s.activeProjects() {
		if strings.EqualFold(p.RepoOwner, repo.Owner) && strings.EqualFold(p.RepoName, repo.Name) {
			return p, true
		}
	}
	return db.Project{}, false
}

func (s *Service) statusConfig(projectID uuid.UUID) *config.ProjectStatus {
	p, ok := s.projectByID(projectID)
	if !ok {
		return nil
	}
	cp, _ := s.Cfg.Project(p.Slug)
	return cp.Status
}

func (s *Service) statusOptions(projectID uuid.UUID) map[string]string {
	if st := s.statusConfig(projectID); st != nil {
		return st.Options
	}
	return nil
}
