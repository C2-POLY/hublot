// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/C2-POLY/hublot/internal/forge"
)

// Background jobs run on River, in the portal's Postgres (no other queue to operate).
// A job that fails is retried with backoff, so a GitHub hiccup only delays the cache.

// IssueEventArgs carries an issues webhook event.
type IssueEventArgs struct {
	Event forge.Event `json:"event"`
}

func (IssueEventArgs) Kind() string { return "process_issue_event" }

// ProjectItemArgs carries a projects_v2_item webhook event: the issue behind the item.
type ProjectItemArgs struct {
	ContentNodeID string `json:"content_node_id"`
}

func (ProjectItemArgs) Kind() string { return "process_project_item" }

// ResyncProjectArgs resyncs one project's repository.
type ResyncProjectArgs struct {
	Project string `json:"project"`
	Full    bool   `json:"full,omitempty"`
}

func (ResyncProjectArgs) Kind() string { return "resync_project" }

// NotifyStatusArgs tells a request's author that its status changed.
type NotifyStatusArgs struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
}

func (NotifyStatusArgs) Kind() string { return "notify_status" }

type issueEventWorker struct {
	river.WorkerDefaults[IssueEventArgs]
	s *Service
}

func (w *issueEventWorker) Work(ctx context.Context, job *river.Job[IssueEventArgs]) error {
	return w.s.ProcessIssueEvent(ctx, job.Args.Event)
}

type projectItemWorker struct {
	river.WorkerDefaults[ProjectItemArgs]
	s *Service
}

func (w *projectItemWorker) Work(ctx context.Context, job *river.Job[ProjectItemArgs]) error {
	return w.s.ProcessProjectItem(ctx, job.Args.ContentNodeID)
}

type resyncWorker struct {
	river.WorkerDefaults[ResyncProjectArgs]
	s *Service
}

func (w *resyncWorker) Work(ctx context.Context, job *river.Job[ResyncProjectArgs]) error {
	n, err := w.s.ResyncProject(ctx, job.Args.Project, job.Args.Full)
	if errors.Is(err, ErrNotFound) {
		return nil // project removed from the configuration since
	}
	if err == nil {
		slog.InfoContext(ctx, "resync done", "project", job.Args.Project, "issues", n)
	}
	return err
}

// notifyWorker is where status e-mails are sent from; until then it records the change.
type notifyWorker struct {
	river.WorkerDefaults[NotifyStatusArgs]
}

func (w *notifyWorker) Work(ctx context.Context, job *river.Job[NotifyStatusArgs]) error {
	slog.InfoContext(ctx, "request status changed", "request", job.Args.RequestID, "status", job.Args.Status)
	return nil
}

// NewJobs builds the job queue client for s. With work, it runs the workers and the
// periodic resync of every project once started (serve); without, it only queues jobs
// for a running server to process (command line).
func NewJobs(s *Service, work bool) (*river.Client[pgx.Tx], error) {
	cfg := &river.Config{Logger: slog.Default()}
	if work {
		workers := river.NewWorkers()
		river.AddWorker(workers, &issueEventWorker{s: s})
		river.AddWorker(workers, &projectItemWorker{s: s})
		river.AddWorker(workers, &resyncWorker{s: s})
		river.AddWorker(workers, &notifyWorker{})
		cfg.Workers = workers
		cfg.Queues = map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 4}}
		every := s.Cfg.Sync.ResyncInterval
		for _, p := range s.Cfg.Projects {
			slug := p.Slug
			cfg.PeriodicJobs = append(cfg.PeriodicJobs, river.NewPeriodicJob(
				river.PeriodicInterval(every),
				func() (river.JobArgs, *river.InsertOpts) {
					// One per period even with several replicas.
					return ResyncProjectArgs{Project: slug}, &river.InsertOpts{
						UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: every},
					}
				},
				&river.PeriodicJobOpts{ID: "resync_" + slug, RunOnStart: true},
			))
		}
	}
	return river.NewClient(riverpgxv5.New(s.Pool), cfg)
}
