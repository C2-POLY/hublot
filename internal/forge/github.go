// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"encoding/json"
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
	// graphqlURL serves the calls REST does not offer (Projects fields, transfers).
	graphqlURL string

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
	if apiURL != "" {
		tr.BaseURL = strings.TrimRight(apiURL, "/")
	}
	return newGitHub(&http.Client{Transport: tr, Timeout: 20 * time.Second}, apiURL)
}

// newGitHub builds the adapter on an authenticated client; tests give it a local server.
func newGitHub(hc *http.Client, apiURL string) (*GitHub, error) {
	opts := []github.ClientOptionsFunc{github.WithHTTPClient(hc), github.WithUserAgent("hublot")}
	graphqlURL := "https://api.github.com/graphql"
	if apiURL != "" {
		opts = append(opts, github.WithEnterpriseURLs(apiURL, apiURL))
		// GitHub Enterprise Server serves REST under /api/v3 and GraphQL at /api/graphql.
		base := strings.TrimSuffix(strings.TrimRight(apiURL, "/"), "/api/v3")
		graphqlURL = base + "/api/graphql"
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &GitHub{client: client, graphqlURL: graphqlURL, forms: map[string]cachedForms{}}, nil
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

func (g *GitHub) GetIssue(ctx context.Context, repo Repo, number int) (Issue, error) {
	i, _, err := g.client.Issues.Get(ctx, repo.Owner, repo.Name, number)
	if isNotFound(err) || isGone(err) {
		return Issue{}, ErrNotFound
	}
	if err != nil {
		return Issue{}, fmt.Errorf("github: get %s#%d: %w", repo, number, err)
	}
	if i.IsPullRequest() {
		return Issue{}, ErrNotFound
	}
	return fromGitHub(i), nil
}

func (g *GitHub) ListPortalIssues(ctx context.Context, repo Repo, since time.Time) ([]Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State: "all", Labels: []string{LabelPortal}, Since: since, Sort: "updated", Direction: "asc",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	out := []Issue{}
	for {
		page, resp, err := g.client.Issues.ListByRepo(ctx, repo.Owner, repo.Name, opts)
		if err != nil {
			return nil, fmt.Errorf("github: list portal issues of %s: %w", repo, err)
		}
		for _, i := range page {
			if !i.IsPullRequest() {
				out = append(out, fromGitHub(i))
			}
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.ListOptions.Page = resp.NextPage
	}
}

const projectStatusQuery = `query($owner: String!, $name: String!, $number: Int!, $field: String!) {
  repository(owner: $owner, name: $name) {
    issue(number: $number) {
      projectItems(first: 20) {
        nodes {
          project { number }
          fieldValueByName(name: $field) { ... on ProjectV2ItemFieldSingleSelectValue { name } }
        }
      }
    }
  }
}`

func (g *GitHub) ProjectStatus(ctx context.Context, repo Repo, number, project int, field string) (string, error) {
	var out struct {
		Repository struct {
			Issue *struct {
				ProjectItems struct {
					Nodes []struct {
						Project          struct{ Number int }
						FieldValueByName *struct{ Name string }
					}
				}
			}
		}
	}
	err := g.graphql(ctx, projectStatusQuery, map[string]any{
		"owner": repo.Owner, "name": repo.Name, "number": number, "field": field,
	}, &out)
	if err != nil {
		return "", fmt.Errorf("github: project status of %s#%d: %w", repo, number, err)
	}
	if out.Repository.Issue == nil {
		return "", ErrNotFound
	}
	for _, n := range out.Repository.Issue.ProjectItems.Nodes {
		if n.Project.Number == project && n.FieldValueByName != nil {
			return n.FieldValueByName.Name, nil
		}
	}
	return "", nil
}

const transferMutation = `mutation($issue: ID!, $repo: ID!) {
  transferIssue(input: {issueId: $issue, repositoryId: $repo, createLabelsIfMissing: true}) {
    issue { number }
  }
}`

func (g *GitHub) TransferIssue(ctx context.Context, issueNodeID string, to Repo) (Issue, error) {
	dest, _, err := g.client.Repositories.Get(ctx, to.Owner, to.Name)
	if err != nil {
		return Issue{}, fmt.Errorf("github: read %s: %w", to, err)
	}
	var out struct {
		TransferIssue struct {
			Issue struct{ Number int }
		}
	}
	err = g.graphql(ctx, transferMutation, map[string]any{"issue": issueNodeID, "repo": dest.GetNodeID()}, &out)
	if err != nil {
		return Issue{}, fmt.Errorf("github: transfer %s to %s: %w", issueNodeID, to, err)
	}
	return g.GetIssue(ctx, to, out.TransferIssue.Issue.Number)
}

func (g *GitHub) UpdateIssue(ctx context.Context, repo Repo, number int, in IssueUpdate) (Issue, error) {
	// Labels one by one, so labels set meanwhile in GitHub are kept.
	if len(in.AddLabels) > 0 {
		if _, _, err := g.client.Issues.AddLabelsToIssue(ctx, repo.Owner, repo.Name, number, in.AddLabels); err != nil {
			return Issue{}, fmt.Errorf("github: label %s#%d: %w", repo, number, err)
		}
	}
	for _, l := range in.RemoveLabels {
		_, err := g.client.Issues.RemoveLabelForIssue(ctx, repo.Owner, repo.Name, number, l)
		if err != nil && !isNotFound(err) {
			return Issue{}, fmt.Errorf("github: unlabel %s#%d: %w", repo, number, err)
		}
	}
	if in.State == "" && in.Type == "" {
		return g.GetIssue(ctx, repo, number)
	}
	req := &github.IssueRequest{}
	if in.State != "" {
		req.State = &in.State
	}
	if in.StateReason != "" {
		req.StateReason = &in.StateReason
	}
	if in.Type != "" {
		req.Type = &in.Type
	}
	edited, _, err := g.client.Issues.Edit(ctx, repo.Owner, repo.Name, number, req)
	if err != nil {
		return Issue{}, fmt.Errorf("github: update %s#%d: %w", repo, number, err)
	}
	return fromGitHub(edited), nil
}

func (g *GitHub) IssueTypes(ctx context.Context, org string) ([]string, error) {
	types, _, err := g.client.Organizations.ListIssueTypes(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("github: issue types of %s: %w", org, err)
	}
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, t.GetName())
	}
	return out, nil
}

// graphql runs one GraphQL call with the installation's credentials.
func (g *GitHub) graphql(ctx context.Context, query string, vars map[string]any, out any) error {
	req, err := g.client.NewRequest(ctx, http.MethodPost, g.graphqlURL, map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	var resp struct {
		Data   json.RawMessage
		Errors []struct{ Message string }
	}
	if _, err := g.client.Do(req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		msgs := make([]string, 0, len(resp.Errors))
		for _, e := range resp.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("graphql: %s", strings.Join(msgs, "; "))
	}
	return json.Unmarshal(resp.Data, out)
}

func fromGitHub(i *github.Issue) Issue {
	out := Issue{
		Number:      i.GetNumber(),
		NodeID:      i.GetNodeID(),
		Title:       i.GetTitle(),
		State:       i.GetState(),
		StateReason: i.GetStateReason(),
		Milestone:   i.GetMilestone().GetTitle(),
		Type:        i.GetType().GetName(),
		UpdatedAt:   i.GetUpdatedAt().Time,
		RequestID:   RequestMarker(i.GetBody()),
	}
	if i.ClosedAt != nil {
		t := i.ClosedAt.Time
		out.ClosedAt = &t
	}
	for _, l := range i.Labels {
		out.Labels = append(out.Labels, l.GetName())
	}
	return out
}

func isNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// isGone matches a deleted issue.
func isGone(err error) bool { return hasStatus(err, http.StatusGone) }

func hasStatus(err error, status int) bool {
	var ghErr *github.ErrorResponse
	return errors.As(err, &ghErr) && ghErr.Response != nil && ghErr.Response.StatusCode == status
}
