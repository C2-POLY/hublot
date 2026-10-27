// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi_test

// The sync suite: GitHub webhooks (and resyncs) move a request through its statuses,
// follow a transferred issue, and never weaken who sees what.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/C2-POLY/hublot/internal/forge"
	"github.com/C2-POLY/hublot/internal/httpapi"
)

const secret = "test-secret"

// webhook posts a delivery signed with sign (empty: no signature header).
func (e *env) webhook(event, delivery, sign string, payload any) (int, []byte) {
	e.t.Helper()
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", e.srv.URL+"/webhooks/github", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-GitHub-Delivery", delivery)
	if sign != "" {
		req.Header.Set("X-Hub-Signature-256", forge.Sign(sign, raw))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

// send delivers a signed event with a fresh delivery id and expects it queued.
func (e *env) send(event string, payload any) {
	e.t.Helper()
	if st, body := e.webhook(event, uuid.NewString(), secret, payload); st != http.StatusAccepted {
		e.t.Fatalf("%s delivery: %d %s", event, st, body)
	}
}

// issuePayload is an issues delivery; at orders it against the other snapshots.
func issuePayload(action, repo string, i forge.Issue, at time.Time) map[string]any {
	owner, name, _ := strings.Cut(repo, "/")
	labels := []map[string]string{}
	for _, l := range i.Labels {
		labels = append(labels, map[string]string{"name": l})
	}
	issue := map[string]any{
		"number": i.Number, "node_id": i.NodeID, "title": i.Title, "state": i.State,
		"state_reason": i.StateReason, "labels": labels, "updated_at": at.UTC().Format(time.RFC3339),
	}
	if i.Milestone != "" {
		issue["milestone"] = map[string]string{"title": i.Milestone}
	}
	return map[string]any{
		"action": action, "issue": issue,
		"repository": map[string]any{"name": name, "full_name": repo, "owner": map[string]string{"login": owner}},
	}
}

type requestView struct {
	Status  string
	Number  int
	Project struct{ Slug string }
	History []struct{ Status string }
}

func (c *client) request(id string) (int, requestView) {
	c.e.t.Helper()
	st, body := c.do("GET", "/api/requests/"+id, nil)
	var v requestView
	json.Unmarshal(body, &v)
	return st, v
}

func (v requestView) history() []string {
	out := []string{}
	for _, h := range v.History {
		out = append(out, h.Status)
	}
	return out
}

// eventually waits for the background jobs to bring the request to a status.
func (c *client) eventually(id, want string) requestView {
	c.e.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		_, v := c.request(id)
		if v.Status == want {
			return v
		}
		if time.Now().After(deadline) {
			c.e.t.Fatalf("request %s: status %q, want %q", id, v.Status, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// problem files a problem (the default form of file is a question).
func (c *client) problem(project, title string) string {
	c.e.t.Helper()
	st, body := c.do("POST", "/api/requests", map[string]any{
		"project": project, "kind": "problem", "title": title,
		"fields": map[string]any{"where": "Rapports", "impact": "C'est gênant"},
	})
	if st != http.StatusCreated {
		c.e.t.Fatalf("file %q in %s: %d %s", title, project, st, body)
	}
	var v struct{ ID string }
	json.Unmarshal(body, &v)
	return v.ID
}

func (e *env) fakeIssue(repo forge.Repo, number int) forge.Issue {
	e.t.Helper()
	i, err := e.forge.GetIssue(context.Background(), repo, number)
	if err != nil {
		e.t.Fatal(err)
	}
	return i
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

var shop, app = forge.Repo{Owner: "acme", Name: "shop"}, forge.Repo{Owner: "acme", Name: "app"}

func TestWebhookDelivery(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")
	id := marie.file("shop", "Le PDF est illisible")
	issue := e.fakeIssue(shop, 1)
	issue.Milestone = "v2.1"
	payload := issuePayload("milestoned", "acme/shop", issue, time.Now().Add(time.Second))

	st, _ := e.webhook("issues", "d-1", "", payload)
	eq(t, "missing signature", st, http.StatusUnauthorized)
	st, _ = e.webhook("issues", "d-1", "wrong-secret", payload)
	eq(t, "wrong signature", st, http.StatusUnauthorized)
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", e.srv.URL+"/webhooks/github", bytes.NewReader(append(raw, ' ')))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-GitHub-Delivery", "d-1")
	req.Header.Set("X-Hub-Signature-256", forge.Sign(secret, raw))
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	eq(t, "signature of another body", resp.StatusCode, http.StatusUnauthorized)
	eq(t, "nothing recorded", e.count("SELECT count(*) FROM webhook_delivery"), 0)

	st, _ = e.webhook("issues", "d-1", secret, payload)
	eq(t, "signed", st, http.StatusAccepted)
	marie.eventually(id, "planned")

	st, body := e.webhook("issues", "d-1", secret, payload)
	eq(t, "redelivery", st, http.StatusOK)
	eq(t, "says so", string(bytes.TrimSpace(body)), `{"status":"duplicate"}`)
	eq(t, "one delivery", e.count("SELECT count(*) FROM webhook_delivery WHERE delivery_id = 'd-1'"), 1)
	eq(t, "one job", e.count("SELECT count(*) FROM river_job WHERE kind = 'process_issue_event'"), 1)

	st, _ = e.webhook("ping", "d-2", secret, map[string]any{"zen": "Keep it logically awesome."})
	eq(t, "event not followed", st, http.StatusNoContent)
	st, _ = e.webhook("issues", "d-3", secret, map[string]any{"action": "opened"})
	eq(t, "malformed", st, http.StatusBadRequest)

	// An issue the portal did not file changes nothing.
	other := forge.Issue{Number: 99, NodeID: "I_99", Title: "Not ours", State: "closed", StateReason: "completed"}
	e.send("issues", issuePayload("closed", "acme/shop", other, time.Now()))
	_, v := marie.request(id)
	eq(t, "unrelated issue", v.Status, "planned")
}

func TestStatusFollowsGitHub(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("paul@acme.test", "app", "reporter")
	marie, paul := e.login("marie@acme.test"), e.login("paul@acme.test")
	now := time.Now()

	problem := marie.problem("shop", "Le PDF est illisible")
	st, v := marie.request(problem)
	eq(t, "filed", fmt.Sprintf("%d %s %v", st, v.Status, v.history()), "200 received [received]")
	i := e.fakeIssue(shop, 1)

	i.Milestone = "v2.1"
	e.send("issues", issuePayload("milestoned", "acme/shop", i, now.Add(1*time.Second)))
	marie.eventually(problem, "planned")

	i.State, i.StateReason = "closed", "completed"
	e.send("issues", issuePayload("closed", "acme/shop", i, now.Add(3*time.Second)))
	v = marie.eventually(problem, "available")
	eq(t, "history", v.history(), []string{"received", "planned", "available"})

	// A delivery older than what the cache holds never rolls the request back.
	old := i
	old.State, old.StateReason = "open", "reopened"
	e.send("issues", issuePayload("reopened", "acme/shop", old, now.Add(2*time.Second)))
	time.Sleep(300 * time.Millisecond)
	_, v = marie.request(problem)
	eq(t, "late delivery ignored", v.Status, "available")

	// A question closed as completed is answered; closed as not planned, closed.
	st, body := marie.do("POST", "/api/requests", map[string]any{
		"project": "shop", "kind": "question", "title": "Comment exporter ?", "fields": map[string]any{"question": "?"},
	})
	eq(t, "question", st, http.StatusCreated)
	var q struct{ ID string }
	json.Unmarshal(body, &q)
	qi := e.fakeIssue(shop, 2)
	qi.State, qi.StateReason = "closed", "completed"
	e.send("issues", issuePayload("closed", "acme/shop", qi, time.Now().Add(time.Second)))
	marie.eventually(q.ID, "answered")

	idea := marie.file("shop", "Mode sombre")
	ii := e.fakeIssue(shop, 3)
	ii.State, ii.StateReason = "closed", "not_planned"
	e.send("issues", issuePayload("closed", "acme/shop", ii, time.Now().Add(time.Second)))
	marie.eventually(idea, "closed")

	e.send("issues", issuePayload("deleted", "acme/shop", e.fakeIssue(shop, 1), time.Now()))
	marie.eventually(problem, "closed")
	eq(t, "one notification per change", e.count(
		"SELECT count(*) FROM river_job WHERE kind = 'notify_status' AND args->>'request_id' = $1", problem), 3)

	// The project status field (app maps it) comes from projects_v2_item deliveries.
	mobile := paul.file("app", "Notification manquante")
	status := "In progress"
	e.forge.Change(app, 1, forge.FakeChange{ProjectStatus: &status})
	e.send("projects_v2_item", map[string]any{"action": "edited",
		"projects_v2_item": map[string]string{"content_node_id": e.fakeIssue(app, 1).NodeID, "content_type": "Issue"}})
	paul.eventually(mobile, "in_progress")

	st, _ = paul.do("GET", "/api/requests/"+problem, nil)
	eq(t, "status changes do not open other projects", st, http.StatusNotFound)
	eq(t, "lists still filtered", paul.titles("/api/requests?scope=all&project=shop"), []string{"status 404"})
}

func TestTransferredIssue(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("paul@acme.test", "app", "reporter")
	e.grant("vic@acme.test", "app", "viewer")
	e.grant("tom@acme.test", "shop", "triager")
	marie, paul, vic, tom := e.login("marie@acme.test"), e.login("paul@acme.test"), e.login("vic@acme.test"), e.login("tom@acme.test")

	id := marie.file("shop", "Le PDF est illisible")
	old := e.fakeIssue(shop, 1)
	moved := forge.Issue{Number: 41, NodeID: "I_moved", Title: old.Title, State: "open", Milestone: "v3", Labels: old.Labels}
	payload := issuePayload("transferred", "acme/shop", old, time.Now())
	newIssue := issuePayload("opened", "acme/app", moved, time.Now())
	payload["changes"] = map[string]any{"new_issue": newIssue["issue"], "new_repository": newIssue["repository"]}
	e.send("issues", payload)

	v := marie.eventually(id, "planned")
	eq(t, "moved", fmt.Sprint(v.Project.Slug, " ", v.Number), "app 41")
	eq(t, "old cache row dropped", e.count("SELECT count(*) FROM issue_cache WHERE repo_name = 'shop'"), 0)
	eq(t, "its author still sees it", marie.titles("/api/requests?scope=mine"), []string{"Le PDF est illisible"})
	eq(t, "a viewer of the new project", vic.titles("/api/requests?scope=all&project=app"), []string{"Le PDF est illisible"})
	st, _ := paul.do("GET", "/api/requests/"+id, nil)
	eq(t, "a reporter of the new project does not", st, http.StatusNotFound)
	st, _ = tom.do("GET", "/api/requests/"+id, nil)
	eq(t, "the old project's triager no longer does", st, http.StatusNotFound)
}

func TestResync(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")
	ctx := context.Background()

	id := marie.problem("shop", "Le PDF est illisible")
	e.forge.Change(shop, 1, forge.FakeChange{State: "closed", StateReason: "completed"})
	_, v := marie.request(id)
	eq(t, "no webhook, no change yet", v.Status, "received")
	if _, err := e.portal.ResyncProject(ctx, "shop", false); err != nil {
		t.Fatal(err)
	}
	_, v = marie.request(id)
	eq(t, "resync", v.Status, "available")

	// Transferred without any delivery: the next resync of the new repository finds
	// the request by the marker of the issue body.
	other := marie.file("shop", "Export CSV")
	if _, err := e.forge.TransferIssue(ctx, e.fakeIssue(shop, 2).NodeID, app); err != nil {
		t.Fatal(err)
	}
	if _, err := e.portal.ResyncProject(ctx, "app", true); err != nil {
		t.Fatal(err)
	}
	_, v = marie.request(other)
	eq(t, "followed by its marker", fmt.Sprint(v.Project.Slug, " ", v.Number), "app 1")

	_, err := e.portal.ResyncProject(ctx, "nope", false)
	eq(t, "unknown project", err != nil, true)
}

func TestFakeForgeEndpoint(t *testing.T) {
	e := setup(t)
	e.grant("paul@acme.test", "app", "reporter")
	paul := e.login("paul@acme.test")
	id := paul.file("app", "Notification manquante")

	post := func(url string, body string) int {
		resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	eq(t, "milestone", post(e.srv.URL+"/dev/fake/issues/acme/app/1", `{"milestone": "v2"}`), http.StatusAccepted)
	paul.eventually(id, "planned")
	eq(t, "project status", post(e.srv.URL+"/dev/fake/issues/acme/app/1", `{"project_status": "Done"}`), http.StatusAccepted)
	paul.eventually(id, "available")
	eq(t, "closed", post(e.srv.URL+"/dev/fake/issues/acme/app/1", `{"state": "closed", "state_reason": "not_planned"}`), http.StatusAccepted)
	paul.eventually(id, "closed")
	eq(t, "unknown issue", post(e.srv.URL+"/dev/fake/issues/acme/app/9", `{"state": "closed"}`), http.StatusNotFound)

	// Never mounted with the github driver.
	cfg := *e.cfg
	cfg.Forge.Driver = "github"
	gh := httptest.NewServer((&httpapi.Server{Cfg: &cfg, Auth: e.server.Auth, Portal: e.portal}).Handler())
	defer gh.Close()
	if st := post(gh.URL+"/dev/fake/issues/acme/app/1", `{"state": "open"}`); st == http.StatusAccepted {
		t.Error("dev endpoint mounted with the github driver")
	}
	eq(t, "issue untouched", e.fakeIssue(app, 1).State, "closed")
}
