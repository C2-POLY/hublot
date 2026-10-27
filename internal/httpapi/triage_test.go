// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
)

// also gives one more role to someone grant already created.
func (e *env) also(email, project, role string) {
	e.t.Helper()
	ctx := context.Background()
	u, err := e.q.GetUserByEmail(ctx, email)
	if err != nil {
		e.t.Fatal(err)
	}
	p, err := e.q.GetProjectBySlug(ctx, project)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.q.UpsertUserGrant(ctx, db.UpsertUserGrantParams{
		ProjectID: p.ID, UserID: uuid.NullUUID{UUID: u.ID, Valid: true}, Role: db.ProjectRole(role),
	}); err != nil {
		e.t.Fatal(err)
	}
}

type triageItem struct {
	ID      string
	Title   string
	Kind    string
	Status  string
	Private bool
	Number  int
	Project struct{ Slug string }
	Author  struct{ Name string }
}

func (c *client) inbox() (int, []triageItem, []string) {
	c.e.t.Helper()
	st, body := c.do("GET", "/api/triage", nil)
	var out struct {
		Requests []triageItem
		Projects []struct{ Slug string }
	}
	json.Unmarshal(body, &out)
	projects := []string{}
	for _, p := range out.Projects {
		projects = append(projects, p.Slug)
	}
	return st, out.Requests, projects
}

func (c *client) inboxTitles() []string {
	c.e.t.Helper()
	_, items, _ := c.inbox()
	out := []string{}
	for _, i := range items {
		out = append(out, i.Title)
	}
	return out
}

func (c *client) triage(id, action string, body any) (int, triageItem) {
	c.e.t.Helper()
	st, raw := c.do("POST", "/api/triage/"+id+"/"+action, body)
	var v triageItem
	json.Unmarshal(raw, &v)
	return st, v
}

func (c *client) canTriage() bool {
	c.e.t.Helper()
	_, body := c.do("GET", "/api/me", nil)
	var me struct{ CanTriage bool }
	json.Unmarshal(body, &me)
	return me.CanTriage
}

func (e *env) audits(action, target string) int {
	e.t.Helper()
	return e.count("SELECT count(*) FROM audit_log WHERE action = $1 AND target_id = $2", action, target)
}

func TestTriageInbox(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("carl@acme.test", "shop", "contributor")
	e.grant("vic@acme.test", "shop", "viewer")
	e.grant("tina@acme.test", "shop", "triager")
	e.also("tina@acme.test", "open", "manager")
	e.grant("otto@acme.test", "app", "triager")
	marie, carl, vic, tina, otto := e.login("marie@acme.test"), e.login("carl@acme.test"),
		e.login("vic@acme.test"), e.login("tina@acme.test"), e.login("otto@acme.test")
	boss := e.login("boss@acme.test")

	e.grant("ana@acme.test", "open", "reporter")
	ana := e.login("ana@acme.test")
	ana.file("open", "Dans open")
	marie.file("shop", "Le PDF est illisible")
	marie.file("shop", "Export plus rapide")

	st, items, projects := tina.inbox()
	eq(t, "triager", st, http.StatusOK)
	eq(t, "newest first, across projects", tina.inboxTitles(), []string{"Export plus rapide", "Le PDF est illisible", "Dans open"})
	eq(t, "projects she may triage", projects, []string{"shop", "open"})
	eq(t, "item", items[0].Project.Slug+" "+items[0].Kind+" "+items[0].Status+" "+items[0].Author.Name, "shop question received marie")
	eq(t, "triager of another project", otto.inboxTitles(), []string{})
	for name, c := range map[string]*client{"reporter": marie, "contributor": carl, "viewer": vic} {
		st, items, projects := c.inbox()
		eq(t, name+" status", st, http.StatusOK)
		eq(t, name+" sees nothing", len(items)+len(projects), 0)
		eq(t, name+" has no triage entry", c.canTriage(), false)
	}
	eq(t, "triager has the entry", tina.canTriage(), true)

	eq(t, "owner sees every inbox", len(boss.inboxTitles()), 3)
	eq(t, "owner without a grant is audited", e.audits("triage.list", "shop"), 1)
	tina.inbox()
	eq(t, "a triager is not", e.audits("triage.list", "shop"), 1)

	t.Run("inbox project first", func(t *testing.T) {
		e.cfg.Projects[2].Inbox = true
		if err := e.portal.SyncProjects(context.Background()); err != nil {
			t.Fatal(err)
		}
		eq(t, "open is the inbox", tina.inboxTitles(), []string{"Dans open", "Export plus rapide", "Le PDF est illisible"})
	})
}

func TestTriageActions(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("carl@acme.test", "shop", "contributor")
	e.grant("vic@acme.test", "shop", "viewer")
	e.grant("paul@acme.test", "app", "reporter")
	e.grant("tina@acme.test", "shop", "triager")
	e.also("tina@acme.test", "app", "triager")
	e.grant("tess@acme.test", "shop", "triager")
	e.also("tess@acme.test", "app", "contributor")
	marie, carl, vic, paul := e.login("marie@acme.test"), e.login("carl@acme.test"), e.login("vic@acme.test"), e.login("paul@acme.test")
	tina, tess, boss := e.login("tina@acme.test"), e.login("tess@acme.test"), e.login("boss@acme.test")

	id := marie.file("shop", "Le PDF est illisible")
	_, v := marie.request(id)
	number := v.Number
	labels := func(repo forge.Repo, n int) []string {
		l := e.fakeIssue(repo, n).Labels
		slices.Sort(l)
		return l
	}
	eq(t, "filed labels", labels(shop, number), []string{"portal", "portal:new", "question"})

	t.Run("who may act", func(t *testing.T) {
		for name, c := range map[string]*client{"author": marie, "contributor": carl, "viewer": vic} {
			st, _ := c.triage(id, "accept", nil)
			eq(t, name, st, http.StatusForbidden)
		}
		st, _ := paul.triage(id, "accept", nil)
		eq(t, "no role on the project", st, http.StatusNotFound)
		st, _ = tina.triage(uuid.NewString(), "accept", nil)
		eq(t, "unknown request", st, http.StatusNotFound)
		st, _ = tina.triage(id, "promote", nil)
		eq(t, "unknown action", st, http.StatusNotFound)
		eq(t, "nothing changed in GitHub", labels(shop, number), []string{"portal", "portal:new", "question"})
	})

	t.Run("private", func(t *testing.T) {
		st, item := tina.triage(id, "private", map[string]any{"private": true})
		eq(t, "status", st, http.StatusOK)
		eq(t, "view", item.Private, true)
		eq(t, "GitHub", labels(shop, number), []string{"portal", "portal:new", "portal:private", "question"})
		st, _ = carl.request(id)
		eq(t, "hidden from a contributor", st, http.StatusNotFound)
		st, _ = marie.request(id)
		eq(t, "its author still sees it", st, http.StatusOK)
		eq(t, "still to triage", tina.inboxTitles(), []string{"Le PDF est illisible"})
		st, _ = tina.triage(id, "private", map[string]any{"private": false})
		eq(t, "public again", st, http.StatusOK)
		st, _ = carl.request(id)
		eq(t, "visible again", st, http.StatusOK)
		st, _ = tina.triage(id, "private", map[string]any{})
		eq(t, "missing value", st, http.StatusUnprocessableEntity)
		eq(t, "audited", e.audits("triage.private", id), 2)
	})

	t.Run("kind", func(t *testing.T) {
		st, item := tina.triage(id, "kind", map[string]any{"kind": "problem"})
		eq(t, "status", st, http.StatusOK)
		eq(t, "request kind", item.Kind, "problem")
		eq(t, "labels swapped", labels(shop, number), []string{"bug", "portal", "portal:new"})
		eq(t, "type from the form", e.fakeIssue(shop, number).Type, "Bug")

		e.cfg.IssueTypes = map[string]string{"idea": "Enhancement"}
		t.Cleanup(func() { e.cfg.IssueTypes = nil })
		_, item = tina.triage(id, "kind", map[string]any{"kind": "idea"})
		eq(t, "request kind", item.Kind, "idea")
		eq(t, "labels swapped again", labels(shop, number), []string{"idee", "portal", "portal:new"})
		eq(t, "type from issue_types", e.fakeIssue(shop, number).Type, "Enhancement")
		st, _ = tina.triage(id, "kind", map[string]any{"kind": "rant"})
		eq(t, "unknown kind", st, http.StatusUnprocessableEntity)
		eq(t, "still to triage", tina.inboxTitles(), []string{"Le PDF est illisible"})
		eq(t, "audited", e.audits("triage.kind", id), 2)
	})

	t.Run("transfer", func(t *testing.T) {
		st, _ := tess.triage(id, "transfer", map[string]any{"project": "app"})
		eq(t, "triager of the source only", st, http.StatusForbidden)
		st, _ = tina.triage(id, "transfer", map[string]any{"project": "nowhere"})
		eq(t, "unknown destination", st, http.StatusUnprocessableEntity)
		st, _ = tina.triage(id, "transfer", map[string]any{"project": "shop"})
		eq(t, "same project", st, http.StatusUnprocessableEntity)
		if _, err := e.forge.GetIssue(context.Background(), shop, number); err != nil {
			t.Fatal("refused transfers must leave the issue in place")
		}

		st, item := tina.triage(id, "transfer", map[string]any{"project": "app"})
		eq(t, "status", st, http.StatusOK)
		eq(t, "new project", item.Project.Slug, "app")
		if _, err := e.forge.GetIssue(context.Background(), shop, number); err == nil {
			t.Error("the issue should have left shop")
		}
		eq(t, "labels kept", labels(app, item.Number), []string{"idee", "portal", "portal:new"})
		r, err := e.q.FindRequestByID(context.Background(), uuid.MustParse(id))
		if err != nil {
			t.Fatal(err)
		}
		shopProject, _ := e.q.GetProjectBySlug(context.Background(), "shop")
		eq(t, "request moved", r.RepoName+" "+r.IssueNodeID, "app "+e.fakeIssue(app, item.Number).NodeID)
		eq(t, "origin kept", r.OriginProjectID, shopProject.ID)
		eq(t, "cache follows", e.count("SELECT count(*) FROM issue_cache WHERE repo_name = 'app' AND issue_number = $1", item.Number), 1)
		st, _ = marie.request(id)
		eq(t, "its author still sees it", st, http.StatusOK)
		st, _ = paul.request(id)
		eq(t, "a reporter of app does not", st, http.StatusNotFound)
		eq(t, "still to triage, now in app", tina.inboxTitles(), []string{"Le PDF est illisible"})
		eq(t, "gone from tess's inbox", tess.inboxTitles(), []string{})
		eq(t, "audited", e.audits("triage.transfer", id), 1)
	})

	t.Run("accept", func(t *testing.T) {
		_, v := marie.request(id)
		st, item := tina.triage(id, "accept", nil)
		eq(t, "status", st, http.StatusOK)
		eq(t, "unchanged status", item.Status, "received")
		eq(t, "GitHub", labels(app, v.Number), []string{"idee", "portal"})
		eq(t, "left the inbox", tina.inboxTitles(), []string{})
		eq(t, "audited", e.audits("triage.accept", id), 1)
	})

	t.Run("close", func(t *testing.T) {
		other := marie.file("shop", "Doublon")
		_, v := marie.request(other)
		st, _ := tina.triage(other, "close", map[string]any{"reason": "completed"})
		eq(t, "a triager does not close as done", st, http.StatusUnprocessableEntity)
		st, item := tina.triage(other, "close", map[string]any{"reason": "duplicate"})
		eq(t, "status", st, http.StatusOK)
		eq(t, "request closed", item.Status, "closed")
		i := e.fakeIssue(shop, v.Number)
		eq(t, "GitHub", i.State+" "+i.StateReason, "closed duplicate")
		eq(t, "no longer new", slices.Contains(i.Labels, forge.LabelNew), false)
		eq(t, "left the inbox", tina.inboxTitles(), []string{})
		marie.eventually(other, "closed")
	})

	t.Run("org admins act, audited as such", func(t *testing.T) {
		third := marie.file("shop", "Pour le patron")
		st, _ := boss.triage(third, "accept", nil)
		eq(t, "status", st, http.StatusOK)
		eq(t, "audited via org role", e.count(
			"SELECT count(*) FROM audit_log WHERE action = 'triage.accept' AND target_id = $1 AND details->>'via_org_role' = 'true'", third), 1)
		eq(t, "a triager's action is not marked", e.count(
			"SELECT count(*) FROM audit_log WHERE target_id = $1 AND details ? 'via_org_role'", id), 0)
	})
}

func TestTriageAuditedWhenLocalStepFails(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("tina@acme.test", "shop", "triager")
	marie, tina := e.login("marie@acme.test"), e.login("tina@acme.test")
	ctx := context.Background()
	id := marie.file("shop", "Le PDF est illisible")

	// The cache refuses every write: GitHub accepts the change, the local step fails.
	for _, sql := range []string{
		`CREATE FUNCTION cache_down() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'cache down'; END $$`,
		`CREATE TRIGGER cache_down BEFORE INSERT OR UPDATE ON issue_cache FOR EACH ROW EXECUTE FUNCTION cache_down()`,
	} {
		if _, err := e.pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	st, _ := tina.triage(id, "accept", nil)
	eq(t, "local step failed", st, http.StatusInternalServerError)
	eq(t, "written to GitHub", slices.Contains(e.fakeIssue(forge.Repo{Owner: "acme", Name: "shop"}, 1).Labels, forge.LabelNew), false)
	eq(t, "audited all the same", e.count(
		"SELECT count(*) FROM audit_log WHERE action = 'triage.accept' AND target_id = $1 AND details->>'local_error' = 'true'", id), 1)
}
