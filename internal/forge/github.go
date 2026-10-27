// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v88/github"
)

// GitHub talks to GitHub as a GitHub App installation (never with a personal token).
type GitHub struct {
	client *github.Client

	mu    sync.Mutex
	forms map[string]cachedForms
}

type cachedForms struct {
	at    time.Time
	forms map[string]IssueForm
}

// formsTTL bounds how long a form change in the repository takes to reach the portal.
const formsTTL = 5 * time.Minute

func NewGitHub(appID, installationID int64, privateKeyPath, apiURL string) (*GitHub, error) {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, installationID, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("github app: %w", err)
	}
	opts := []github.ClientOptionsFunc{
		github.WithHTTPClient(&http.Client{Transport: tr, Timeout: 20 * time.Second}),
		github.WithUserAgent("hublot"),
	}
	if apiURL != "" {
		tr.BaseURL = strings.TrimRight(apiURL, "/")
		opts = append(opts, github.WithEnterpriseURLs(apiURL, apiURL))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &GitHub{client: client, forms: map[string]cachedForms{}}, nil
}

func (g *GitHub) IssueForms(ctx context.Context, repo Repo) (map[string]IssueForm, error) {
	g.mu.Lock()
	cached, ok := g.forms[repo.String()]
	g.mu.Unlock()
	if ok && time.Since(cached.at) < formsTTL {
		return cached.forms, nil
	}

	_, entries, _, err := g.client.Repositories.GetContents(ctx, repo.Owner, repo.Name, ".github/ISSUE_TEMPLATE", nil)
	if err != nil {
		if isNotFound(err) {
			return map[string]IssueForm{}, nil
		}
		return nil, fmt.Errorf("github: list issue forms of %s: %w", repo, err)
	}
	forms := map[string]IssueForm{}
	for _, e := range entries {
		name := e.GetName()
		if e.GetType() != "file" || (path.Ext(name) != ".yml" && path.Ext(name) != ".yaml") || name == "config.yml" {
			continue
		}
		file, _, _, err := g.client.Repositories.GetContents(ctx, repo.Owner, repo.Name, e.GetPath(), nil)
		if err != nil {
			return nil, fmt.Errorf("github: read %s of %s: %w", name, repo, err)
		}
		content, err := file.GetContent()
		if err != nil {
			return nil, fmt.Errorf("github: decode %s of %s: %w", name, repo, err)
		}
		form, err := ParseIssueForm([]byte(content))
		if err != nil {
			// One broken form must not take the others down.
			slog.WarnContext(ctx, "skipping invalid issue form", "repo", repo.String(), "file", name, "err", err)
			continue
		}
		forms[name] = form
	}

	g.mu.Lock()
	g.forms[repo.String()] = cachedForms{at: time.Now(), forms: forms}
	g.mu.Unlock()
	return forms, nil
}

func (g *GitHub) CreateIssue(ctx context.Context, repo Repo, in NewIssue) (Issue, error) {
	req := &github.IssueRequest{Title: &in.Title, Body: &in.Body, Labels: &in.Labels}
	if in.Type != "" {
		req.Type = &in.Type
	}
	created, _, err := g.client.Issues.Create(ctx, repo.Owner, repo.Name, req)
	if err != nil {
		return Issue{}, fmt.Errorf("github: create issue in %s: %w", repo, err)
	}
	return fromGitHub(created), nil
}

func fromGitHub(i *github.Issue) Issue {
	out := Issue{
		Number:      i.GetNumber(),
		NodeID:      i.GetNodeID(),
		Title:       i.GetTitle(),
		State:       i.GetState(),
		StateReason: i.GetStateReason(),
	}
	for _, l := range i.Labels {
		out.Labels = append(out.Labels, l.GetName())
	}
	return out
}

func isNotFound(err error) bool {
	var ghErr *github.ErrorResponse
	return errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound
}
