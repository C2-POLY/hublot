// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
	"sync"
)

// Fake is an in-memory forge for local development and tests: issue forms are read
// from a folder per repository (<owner>/<name>/*.yml, falling back to _default/*.yml)
// and issues live in memory.
type Fake struct {
	forms fs.FS

	mu     sync.Mutex
	issues map[string][]Issue
	bodies map[string][]NewIssue
}

func NewFake(forms fs.FS) *Fake {
	return &Fake{forms: forms, issues: map[string][]Issue{}, bodies: map[string][]NewIssue{}}
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
	key := repo.String()
	issue := Issue{
		Number: len(f.issues[key]) + 1,
		NodeID: fmt.Sprintf("FAKE_%s_%d", strings.ReplaceAll(key, "/", "_"), len(f.issues[key])+1),
		Title:  in.Title,
		State:  "open",
		Labels: in.Labels,
	}
	f.issues[key] = append(f.issues[key], issue)
	f.bodies[key] = append(f.bodies[key], in)
	slog.InfoContext(ctx, "fake forge: issue created", "repo", key, "number", issue.Number, "title", in.Title, "labels", in.Labels)
	return issue, nil
}

// Created returns what was filed in a repository, for tests.
func (f *Fake) Created(repo Repo) []NewIssue {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]NewIssue(nil), f.bodies[repo.String()]...)
}
