// SPDX-License-Identifier: AGPL-3.0-or-later

// Package forge is the only way the portal reaches GitHub (PRD §8, forge adapter).
// It speaks in portal terms so another forge could be added later without touching callers.
package forge

import (
	"context"
	"errors"
	"time"
)

type Repo struct {
	Owner string
	Name  string
}

func (r Repo) String() string { return r.Owner + "/" + r.Name }

// NewIssue is what the portal files on behalf of a person.
type NewIssue struct {
	Title  string
	Body   string
	Labels []string
	Type   string // organisation issue type, empty when the form sets none
}

type Issue struct {
	Number      int
	NodeID      string
	Title       string
	State       string // open or closed
	StateReason string // completed, not_planned, duplicate, reopened or empty
	Labels      []string
	Milestone   string // title of the milestone, empty when none
	Type        string // organisation issue type, empty when none
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	// RequestID is read from the body marker: the portal request the issue was filed for,
	// empty for an issue the portal did not write (or when the body was not available).
	RequestID string
}

// IssueUpdate changes an issue. Zero values leave the matching part unchanged.
type IssueUpdate struct {
	AddLabels    []string
	RemoveLabels []string
	State        string // open or closed
	StateReason  string // with State closed: completed, not_planned or duplicate
	Type         string // organisation issue type
}

type Forge interface {
	// IssueForms returns the parsed .github/ISSUE_TEMPLATE/*.yml forms of a repository, keyed by file name.
	IssueForms(ctx context.Context, repo Repo) (map[string]IssueForm, error)
	CreateIssue(ctx context.Context, repo Repo, in NewIssue) (Issue, error)
	// GetIssue answers ErrNotFound for a missing, deleted or transferred-away issue, and for pull requests.
	GetIssue(ctx context.Context, repo Repo, number int) (Issue, error)
	// ListPortalIssues returns the issues labelled portal, open or closed, updated since the
	// given time (every one when since is zero), with RequestID filled.
	ListPortalIssues(ctx context.Context, repo Repo, since time.Time) ([]Issue, error)
	// ProjectStatus returns the option of a single-select field of an organisation Project
	// for an issue, empty when the issue is not in that Project or the field is not set.
	ProjectStatus(ctx context.Context, repo Repo, number, project int, field string) (string, error)
	// TransferIssue moves an issue to another repository; the issue gets a new number there.
	TransferIssue(ctx context.Context, issueNodeID string, to Repo) (Issue, error)
	UpdateIssue(ctx context.Context, repo Repo, number int, in IssueUpdate) (Issue, error)
	// IssueTypes lists the names of the organisation's issue types.
	IssueTypes(ctx context.Context, org string) ([]string, error)
}

var ErrNotFound = errors.New("forge: not found")

// Label names the portal relies on.
const (
	LabelPortal  = "portal"
	LabelPrivate = "portal:private"
	// LabelNew marks a request nobody has triaged yet; a triage action removes it.
	LabelNew = "portal:new"
)
