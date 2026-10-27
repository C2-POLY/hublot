// SPDX-License-Identifier: AGPL-3.0-or-later

// Package forge is the only way the portal reaches GitHub (PRD §8, forge adapter).
// It speaks in portal terms so another forge could be added later without touching callers.
package forge

import (
	"context"
	"errors"
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
	State       string
	StateReason string
	Labels      []string
}

type Forge interface {
	// IssueForms returns the parsed .github/ISSUE_TEMPLATE/*.yml forms of a repository, keyed by file name.
	IssueForms(ctx context.Context, repo Repo) (map[string]IssueForm, error)
	CreateIssue(ctx context.Context, repo Repo, in NewIssue) (Issue, error)
}

var ErrNotFound = errors.New("forge: not found")

// Label names the portal relies on.
const (
	LabelPortal  = "portal"
	LabelPrivate = "portal:private"
)
