// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const markedBody = "<!-- hublot:request v1 id=0f8c2a4e-1b2c-4d5e-8f90-123456789abc -->\n\n### Où ?\n\nIci"

func TestRequestMarker(t *testing.T) {
	if got := RequestMarker(markedBody); got != "0f8c2a4e-1b2c-4d5e-8f90-123456789abc" {
		t.Errorf("marker: %q", got)
	}
	for _, body := range []string{"", "Hello", "Texte\n" + markedBody, "<!-- hublot:request v1 id=evil -->"} {
		if got := RequestMarker(body); got != "" {
			t.Errorf("%q: only a marker opening the body counts, got %q", body, got)
		}
	}
}

func TestSignature(t *testing.T) {
	body := []byte(`{"action":"closed"}`)
	sig := Sign("s3cret", body)
	if !VerifySignature("s3cret", sig, body) {
		t.Error("valid signature refused")
	}
	for name, c := range map[string]struct{ secret, header string }{
		"wrong secret": {"other", sig},
		"missing":      {"s3cret", ""},
		"no prefix":    {"s3cret", strings.TrimPrefix(sig, "sha256=")},
		"not hex":      {"s3cret", "sha256=zz"},
		"no secret":    {"", Sign("", body)},
	} {
		if VerifySignature(c.secret, c.header, body) {
			t.Errorf("%s: accepted", name)
		}
	}
	if VerifySignature("s3cret", sig, []byte(`{"action":"opened"}`)) {
		t.Error("tampered body accepted")
	}
}

func TestParseGitHubEvent(t *testing.T) {
	ev, ok, err := ParseGitHubEvent("issues", []byte(`{
		"action": "milestoned",
		"issue": {"number": 7, "node_id": "I_7", "title": "PDF", "state": "open", "body": "`+strings.ReplaceAll(markedBody, "\n", `\n`)+`",
			"labels": [{"name": "portal"}], "milestone": {"title": "v2.1"}, "type": {"name": "Bug"},
			"updated_at": "2026-10-27T10:00:00Z", "user": {"login": "hublot[bot]", "type": "Bot"},
			"performed_via_github_app": {"id": 42}},
		"repository": {"name": "shop", "full_name": "acme/shop", "owner": {"login": "acme"}}
	}`))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ev.Kind != EventIssue || ev.Action != "milestoned" || ev.Repo.String() != "acme/shop" {
		t.Errorf("event: %+v", ev)
	}
	i := ev.Issue
	if i.Number != 7 || i.Milestone != "v2.1" || i.Type != "Bug" || i.RequestID == "" || i.Labels[0] != "portal" ||
		i.Author != "hublot[bot]" || !i.AuthorBot || i.ViaApp != 42 ||
		!i.UpdatedAt.Equal(time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("issue: %+v", i)
	}

	ev, _, _ = ParseGitHubEvent("issues", []byte(`{
		"action": "transferred",
		"issue": {"number": 7, "node_id": "I_7"},
		"repository": {"name": "shop", "owner": {"login": "acme"}},
		"changes": {"new_issue": {"number": 3, "node_id": "I_3", "state": "open"},
		            "new_repository": {"name": "app", "owner": {"login": "acme"}}}
	}`))
	if ev.NewRepo == nil || ev.NewRepo.String() != "acme/app" || ev.NewIssue.Number != 3 {
		t.Errorf("transfer: %+v", ev)
	}

	ev, ok, _ = ParseGitHubEvent("projects_v2_item", []byte(`{"action": "edited",
		"projects_v2_item": {"content_node_id": "I_7", "content_type": "Issue"}}`))
	if !ok || ev.Kind != EventProjectItem || ev.ContentNodeID != "I_7" {
		t.Errorf("project item: %v %+v", ok, ev)
	}
	for name, body := range map[string]string{
		"draft item":   `{"action": "edited", "projects_v2_item": {"content_node_id": "D_1", "content_type": "DraftIssue"}}`,
		"pull request": `{"action": "opened", "issue": {"number": 1, "pull_request": {"url": "x"}}, "repository": {"name": "a"}}`,
	} {
		event := "issues"
		if strings.Contains(name, "item") {
			event = "projects_v2_item"
		}
		if _, ok, err := ParseGitHubEvent(event, []byte(body)); ok || err != nil {
			t.Errorf("%s: ignored without error, got %v %v", name, ok, err)
		}
	}
	if _, ok, _ := ParseGitHubEvent("ping", []byte(`{}`)); ok {
		t.Error("ping is not followed")
	}
	if _, _, err := ParseGitHubEvent("issues", []byte(`{"action":`)); err == nil {
		t.Error("malformed payload")
	}
}

func TestFakeLifecycle(t *testing.T) {
	ctx := context.Background()
	f := NewFake(fstest.MapFS{})
	shop, app := Repo{Owner: "acme", Name: "shop"}, Repo{Owner: "acme", Name: "app"}
	created, _ := f.CreateIssue(ctx, shop, NewIssue{Title: "PDF", Body: markedBody, Labels: []string{"bug", LabelPortal, LabelNew}})
	f.CreateIssue(ctx, shop, NewIssue{Title: "Not ours", Labels: []string{"bug"}})
	if created.RequestID == "" {
		t.Error("request id read from the body")
	}
	if !f.OwnIssue(created) || f.OwnIssue(Issue{Author: "mallory"}) {
		t.Error("the fake opens its issues as its own bot")
	}

	list, _ := f.ListPortalIssues(ctx, shop, time.Time{})
	if len(list) != 1 || list[0].Number != 1 {
		t.Errorf("portal issues: %+v", list)
	}
	if list, _ := f.ListPortalIssues(ctx, shop, time.Now().Add(time.Minute)); len(list) != 0 {
		t.Errorf("since: %+v", list)
	}

	updated, err := f.UpdateIssue(ctx, shop, 1, IssueUpdate{RemoveLabels: []string{LabelNew}, AddLabels: []string{"urgent"},
		State: "closed", StateReason: "not_planned", Type: "Feature"})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(updated.Labels) != "[bug portal urgent]" || updated.State != "closed" ||
		updated.StateReason != "not_planned" || updated.Type != "Feature" || updated.ClosedAt == nil {
		t.Errorf("update: %+v", updated)
	}

	status := "In progress"
	f.Change(shop, 1, FakeChange{ProjectStatus: &status})
	if got, _ := f.ProjectStatus(ctx, shop, 1, 12, "Status"); got != status {
		t.Errorf("project status: %q", got)
	}

	moved, err := f.TransferIssue(ctx, created.NodeID, app)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Number != 1 || moved.NodeID == created.NodeID || moved.Title != "PDF" {
		t.Errorf("transferred: %+v", moved)
	}
	if _, err := f.GetIssue(ctx, shop, 1); err != ErrNotFound {
		t.Errorf("the old place is empty: %v", err)
	}
	if got, _ := f.GetIssue(ctx, app, 1); got.Title != "PDF" {
		t.Errorf("new place: %+v", got)
	}
	if _, err := f.TransferIssue(ctx, "nope", app); err != ErrNotFound {
		t.Errorf("unknown node: %v", err)
	}
	if types, _ := f.IssueTypes(ctx, "acme"); len(types) != 3 {
		t.Errorf("types: %v", types)
	}
}

// fakeGitHub answers the few REST and GraphQL calls the adapter makes.
func fakeGitHub(t *testing.T) (*GitHub, *[]string) {
	t.Helper()
	calls := &[]string{}
	mux := http.NewServeMux()
	record := func(r *http.Request) { *calls = append(*calls, r.Method+" "+r.URL.Path) }
	issue := func(n int, extra string) string {
		return fmt.Sprintf(`{"number": %d, "node_id": "I_%d", "title": "PDF", "state": "open", "labels": [{"name": "portal"}],
			"updated_at": "2026-10-27T10:00:00Z", "body": %q%s}`, n, n, markedBody, extra)
	}
	mux.HandleFunc("GET /api/v3/repos/acme/shop/issues/7", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		io.WriteString(w, issue(7, `, "milestone": {"title": "v2.1"}, "type": {"name": "Bug"},
			"user": {"login": "hublot[bot]", "type": "Bot"}, "performed_via_github_app": {"id": 42, "slug": "hublot"}`))
	})
	mux.HandleFunc("GET /api/v3/repos/acme/shop/issues/8", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		w.WriteHeader(http.StatusGone)
		io.WriteString(w, `{"message": "This issue was deleted"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/acme/shop/issues", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		q := r.URL.Query()
		if q.Get("state") != "all" || q.Get("labels") != "portal" || q.Get("since") == "" {
			t.Errorf("list query: %s", r.URL.RawQuery)
		}
		if q.Get("page") == "" {
			w.Header().Set("Link", `<http://`+r.Host+r.URL.Path+`?page=2>; rel="next"`)
			io.WriteString(w, "["+issue(1, `, "user": {"login": "hublot[bot]", "type": "Bot"}, "performed_via_github_app": {"id": 42}`)+
				`, {"number": 2, "pull_request": {"url": "x"}}]`)
			return
		}
		io.WriteString(w, "["+issue(3, `, "state": "closed", "state_reason": "completed", "closed_at": "2026-10-27T11:00:00Z",
			"user": {"login": "mallory", "type": "User"}`)+"]")
	})
	mux.HandleFunc("POST /api/v3/repos/acme/shop/issues/7/labels", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		io.WriteString(w, `[]`)
	})
	mux.HandleFunc("DELETE /api/v3/repos/acme/shop/issues/7/labels/{label}", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		if r.PathValue("label") == "missing" {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"message": "Label does not exist"}`)
			return
		}
		io.WriteString(w, `[]`)
	})
	mux.HandleFunc("PATCH /api/v3/repos/acme/shop/issues/7", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		var in map[string]any
		json.NewDecoder(r.Body).Decode(&in)
		if in["state"] != "closed" || in["state_reason"] != "not_planned" || in["type"] != "Feature" || in["labels"] != nil {
			t.Errorf("edit body: %v", in)
		}
		io.WriteString(w, issue(7, `, "state": "closed", "state_reason": "not_planned"`))
	})
	mux.HandleFunc("GET /api/v3/repos/acme/app", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		io.WriteString(w, `{"node_id": "R_app", "name": "app"}`)
	})
	mux.HandleFunc("GET /api/v3/repos/acme/app/issues/4", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		io.WriteString(w, issue(4, ""))
	})
	mux.HandleFunc("GET /api/v3/orgs/acme/issue-types", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		io.WriteString(w, `[{"name": "Bug"}, {"name": "Feature"}]`)
	})
	mux.HandleFunc("POST /api/graphql", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		var in struct {
			Query     string
			Variables map[string]any
		}
		json.NewDecoder(r.Body).Decode(&in)
		switch {
		case strings.Contains(in.Query, "transferIssue"):
			if in.Variables["issue"] != "I_7" || in.Variables["repo"] != "R_app" {
				t.Errorf("transfer variables: %v", in.Variables)
			}
			io.WriteString(w, `{"data": {"transferIssue": {"issue": {"number": 4}}}}`)
		case in.Variables["number"] == float64(404):
			io.WriteString(w, `{"data": {"repository": {"issue": null}}, "errors": [{"type": "NOT_FOUND",
				"path": ["repository", "issue"], "message": "Could not resolve to an Issue with the number of 404."}]}`)
		case in.Variables["number"] == float64(405):
			io.WriteString(w, `{"data": null, "errors": [{"message": "Could not resolve to a Repository with the name 'acme/shop'."}]}`)
		case in.Variables["number"] == float64(500):
			io.WriteString(w, `{"data": null, "errors": [{"type": "RATE_LIMITED", "message": "API rate limit exceeded"}]}`)
		default:
			io.WriteString(w, `{"data": {"repository": {"issue": {"projectItems": {"nodes": [
				{"project": {"number": 3}, "fieldValueByName": {"name": "Done"}},
				{"project": {"number": 12}, "fieldValueByName": {"name": "In progress"}}]}}}}}`)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	g, err := newGitHub(srv.Client(), srv.URL+"/api/v3/", 42)
	if err != nil {
		t.Fatal(err)
	}
	return g, calls
}

func TestGitHubSync(t *testing.T) {
	ctx := context.Background()
	g, calls := fakeGitHub(t)
	shop, app := Repo{Owner: "acme", Name: "shop"}, Repo{Owner: "acme", Name: "app"}

	i, err := g.GetIssue(ctx, shop, 7)
	if err != nil {
		t.Fatal(err)
	}
	if i.NodeID != "I_7" || i.Milestone != "v2.1" || i.Type != "Bug" || i.RequestID == "" {
		t.Errorf("issue: %+v", i)
	}
	if _, err := g.GetIssue(ctx, shop, 8); err != ErrNotFound {
		t.Errorf("deleted issue: %v", err)
	}

	list, err := g.ListPortalIssues(ctx, shop, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Number != 1 || list[1].Number != 3 || list[1].ClosedAt == nil {
		t.Errorf("two pages, pull requests skipped: %+v", list)
	}
	if !g.OwnIssue(i) || len(list) != 2 || !g.OwnIssue(list[0]) || g.OwnIssue(list[1]) {
		t.Errorf("own issues: opened through App 42 by its bot only: %+v %+v", i, list)
	}
	other, _ := newGitHub(http.DefaultClient, "", 7)
	if other.OwnIssue(i) || (&GitHub{}).OwnIssue(Issue{AuthorBot: true}) {
		t.Error("another App's issue, or no App configured, is not ours")
	}

	if st, err := g.ProjectStatus(ctx, shop, 7, 12, "Status"); err != nil || st != "In progress" {
		t.Errorf("project status: %q %v", st, err)
	}
	if st, _ := g.ProjectStatus(ctx, shop, 7, 99, "Status"); st != "" {
		t.Errorf("not in that project: %q", st)
	}
	// A deleted or transferred issue is not found, so its job stops retrying.
	for _, n := range []int{404, 405} {
		if _, err := g.ProjectStatus(ctx, shop, n, 12, "Status"); !errors.Is(err, ErrNotFound) {
			t.Errorf("%d: not found: %v", n, err)
		}
	}
	if _, err := g.ProjectStatus(ctx, shop, 500, 12, "Status"); err == nil || errors.Is(err, ErrNotFound) ||
		!strings.Contains(err.Error(), "rate limit") {
		t.Errorf("other graphql errors surface: %v", err)
	}

	moved, err := g.TransferIssue(ctx, "I_7", app)
	if err != nil || moved.Number != 4 {
		t.Errorf("transfer: %+v %v", moved, err)
	}

	*calls = nil
	updated, err := g.UpdateIssue(ctx, shop, 7, IssueUpdate{
		AddLabels: []string{"urgent"}, RemoveLabels: []string{LabelNew, "missing"},
		State: "closed", StateReason: "not_planned", Type: "Feature",
	})
	if err != nil || updated.StateReason != "not_planned" {
		t.Errorf("update: %+v %v", updated, err)
	}
	want := "[POST /api/v3/repos/acme/shop/issues/7/labels DELETE /api/v3/repos/acme/shop/issues/7/labels/portal:new " +
		"DELETE /api/v3/repos/acme/shop/issues/7/labels/missing PATCH /api/v3/repos/acme/shop/issues/7]"
	if fmt.Sprint(*calls) != want {
		t.Errorf("calls:\n%v\nwant\n%v", *calls, want)
	}

	if types, err := g.IssueTypes(ctx, "acme"); err != nil || fmt.Sprint(types) != "[Bug Feature]" {
		t.Errorf("types: %v %v", types, err)
	}
}
