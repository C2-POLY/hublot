// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"
	"sync"
	"time"
)

// FakeBot and FakeAppID are the identity the fake forge opens its issues with.
const (
	FakeBot   = "hublot-dev[bot]"
	FakeAppID = int64(1)
)

// Fake is an in-memory forge for local development and tests: issue forms are read
// from a folder per repository (<owner>/<name>/*.yml, falling back to _default/*.yml)
// and issues live in memory.
type Fake struct {
	forms fs.FS
	// LastNumber, when set, gives the highest issue number already used in a repository,
	// so numbering survives a restart while the database keeps the earlier requests.
	LastNumber func(context.Context, Repo) (int, error)
	// Types are the organisation issue types IssueTypes returns.
	Types []string

	mu     sync.Mutex
	last   map[string]int
	bodies map[string][]NewIssue
	issues map[string]map[int]*fakeIssue
}

type fakeIssue struct {
	Issue
	body          string
	projectStatus string
}

func NewFake(forms fs.FS) *Fake {
	return &Fake{
		forms: forms, Types: []string{"Bug", "Feature", "Task"},
		last: map[string]int{}, bodies: map[string][]NewIssue{}, issues: map[string]map[int]*fakeIssue{},
	}
}

func (f *Fake) IssueForms(_ context.Context, repo Repo) (map[string]IssueForm, error) {
	dir := repo.Owner + "/" + repo.Name
	entries, err := fs.ReadDir(f.forms, dir)
	if err != nil {
		dir = "_default"
		if entries, err = fs.ReadDir(f.forms, dir); err != nil {
			return map[string]IssueForm{}, nil
		}
	}
	forms := map[string]IssueForm{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		data, err := fs.ReadFile(f.forms, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		form, err := ParseIssueForm(data)
		if err != nil {
			return nil, fmt.Errorf("%s/%s: %w", dir, e.Name(), err)
		}
		forms[e.Name()] = form
	}
	return forms, nil
}

func (f *Fake) CreateIssue(ctx context.Context, repo Repo, in NewIssue) (Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, err := f.next(ctx, repo)
	if err != nil {
		return Issue{}, err
	}
	key := repo.String()
	issue := Issue{
		Number:    n,
		NodeID:    fakeNodeID(repo, n),
		Title:     in.Title,
		State:     "open",
		Labels:    slices.Clone(in.Labels),
		Type:      in.Type,
		UpdatedAt: time.Now(),
		RequestID: RequestMarker(in.Body),
		Author:    FakeBot,
		AuthorBot: true,
		ViaApp:    FakeAppID,
	}
	f.bodies[key] = append(f.bodies[key], in)
	f.put(repo, &fakeIssue{Issue: issue, body: in.Body})
	slog.InfoContext(ctx, "fake forge: issue created", "repo", key, "number", issue.Number, "title", in.Title, "labels", in.Labels)
	return copyIssue(issue), nil
}

// next returns the next issue number of a repository. f.mu must be held.
func (f *Fake) next(ctx context.Context, repo Repo) (int, error) {
	key := repo.String()
	if _, seen := f.last[key]; !seen && f.LastNumber != nil {
		n, err := f.LastNumber(ctx, repo)
		if err != nil {
			return 0, err
		}
		f.last[key] = n
	}
	f.last[key]++
	return f.last[key], nil
}

func (f *Fake) put(repo Repo, i *fakeIssue) {
	key := repo.String()
	if f.issues[key] == nil {
		f.issues[key] = map[int]*fakeIssue{}
	}
	f.issues[key][i.Number] = i
}

func fakeNodeID(repo Repo, n int) string {
	return fmt.Sprintf("FAKE_%s_%d", strings.ReplaceAll(repo.String(), "/", "_"), n)
}

func copyIssue(i Issue) Issue {
	i.Labels = slices.Clone(i.Labels)
	return i
}

func (f *Fake) GetIssue(_ context.Context, repo Repo, number int) (Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i, ok := f.issues[repo.String()][number]
	if !ok {
		return Issue{}, ErrNotFound
	}
	return copyIssue(i.Issue), nil
}

func (f *Fake) ListPortalIssues(_ context.Context, repo Repo, since time.Time) ([]Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []Issue{}
	for _, i := range f.issues[repo.String()] {
		if slices.Contains(i.Labels, LabelPortal) && !i.UpdatedAt.Before(since) {
			out = append(out, copyIssue(i.Issue))
		}
	}
	slices.SortFunc(out, func(a, b Issue) int { return a.Number - b.Number })
	return out, nil
}

// ProjectStatus ignores the Project number and field: the fake keeps one status per issue.
func (f *Fake) ProjectStatus(_ context.Context, repo Repo, number, _ int, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i, ok := f.issues[repo.String()][number]
	if !ok {
		return "", ErrNotFound
	}
	return i.projectStatus, nil
}

func (f *Fake) TransferIssue(ctx context.Context, issueNodeID string, to Repo) (Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, issues := range f.issues {
		for n, i := range issues {
			if i.NodeID != issueNodeID {
				continue
			}
			number, err := f.next(ctx, to)
			if err != nil {
				return Issue{}, err
			}
			delete(issues, n)
			moved := *i
			moved.Number, moved.NodeID, moved.UpdatedAt = number, fakeNodeID(to, number), time.Now()
			f.put(to, &moved)
			slog.InfoContext(ctx, "fake forge: issue transferred", "from", key, "number", n, "to", to.String(), "new_number", number)
			return copyIssue(moved.Issue), nil
		}
	}
	return Issue{}, ErrNotFound
}

func (f *Fake) UpdateIssue(_ context.Context, repo Repo, number int, in IssueUpdate) (Issue, error) {
	return f.Change(repo, number, FakeChange{
		State: in.State, StateReason: in.StateReason, Type: in.Type,
		AddLabels: in.AddLabels, RemoveLabels: in.RemoveLabels,
	})
}

func (f *Fake) IssueTypes(context.Context, string) ([]string, error) {
	return slices.Clone(f.Types), nil
}

func (f *Fake) OwnIssue(i Issue) bool {
	return i.AuthorBot && i.Author == FakeBot && i.ViaApp == FakeAppID
}

// FakeChange is what the development endpoint can change on a fake issue.
// Zero values leave the matching part unchanged; Milestone and ProjectStatus
// are pointers so they can be cleared.
type FakeChange struct {
	State         string
	StateReason   string
	Type          string
	Labels        []string // replaces every label when not nil
	AddLabels     []string
	RemoveLabels  []string
	Milestone     *string
	ProjectStatus *string
}

// Change applies a change to a fake issue, as someone working in GitHub would.
func (f *Fake) Change(repo Repo, number int, c FakeChange) (Issue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i, ok := f.issues[repo.String()][number]
	if !ok {
		return Issue{}, ErrNotFound
	}
	if c.Labels != nil {
		i.Labels = slices.Clone(c.Labels)
	}
	for _, l := range c.AddLabels {
		if !slices.Contains(i.Labels, l) {
			i.Labels = append(i.Labels, l)
		}
	}
	i.Labels = slices.DeleteFunc(i.Labels, func(l string) bool { return slices.Contains(c.RemoveLabels, l) })
	now := time.Now()
	switch c.State {
	case "closed":
		i.State, i.StateReason, i.ClosedAt = "closed", c.StateReason, &now
		if i.StateReason == "" {
			i.StateReason = "completed"
		}
	case "open":
		i.State, i.StateReason, i.ClosedAt = "open", "reopened", nil
	}
	if c.Type != "" {
		i.Type = c.Type
	}
	if c.Milestone != nil {
		i.Milestone = *c.Milestone
	}
	if c.ProjectStatus != nil {
		i.projectStatus = *c.ProjectStatus
	}
	i.UpdatedAt = now
	return copyIssue(i.Issue), nil
}

// Created returns what was filed in a repository, for tests.
func (f *Fake) Created(repo Repo) []NewIssue {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]NewIssue(nil), f.bodies[repo.String()]...)
}
