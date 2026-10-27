// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestRequestFilters(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.also("marie@acme.test", "app", "reporter")
	e.grant("paul@acme.test", "app", "reporter")
	marie, paul, boss := e.login("marie@acme.test"), e.login("paul@acme.test"), e.login("boss@acme.test")

	marie.file("shop", "Question boutique")       // shop #1
	planned := marie.problem("shop", "PDF coupé") // shop #2
	marie.problem("app", "Notification manquante")
	paul.file("app", "Comment exporter ?")
	e.milestone(2)
	marie.eventually(planned, "planned")

	t.Run("mine", func(t *testing.T) {
		eq(t, "no filter", marie.titles("/api/requests?scope=mine"),
			[]string{"Notification manquante", "PDF coupé", "Question boutique"})
		eq(t, "kind", marie.titles("/api/requests?scope=mine&kind=problem"), []string{"Notification manquante", "PDF coupé"})
		eq(t, "status", marie.titles("/api/requests?status=planned"), []string{"PDF coupé"})
		eq(t, "project", marie.titles("/api/requests?scope=mine&project=app"), []string{"Notification manquante"})
		eq(t, "combined", marie.titles("/api/requests?scope=mine&project=shop&kind=problem&status=planned"), []string{"PDF coupé"})
		eq(t, "combined, nothing", marie.titles("/api/requests?scope=mine&project=shop&kind=idea"), []string{})
	})

	t.Run("all", func(t *testing.T) {
		eq(t, "boss, status", boss.titles("/api/requests?scope=all&project=shop&status=planned"), []string{"PDF coupé"})
		eq(t, "boss, kind", boss.titles("/api/requests?scope=all&project=app&kind=question"), []string{"Comment exporter ?"})
		eq(t, "boss, combined", boss.titles("/api/requests?scope=all&project=shop&kind=question&status=received"),
			[]string{"Question boutique"})
	})

	t.Run("invalid", func(t *testing.T) {
		for _, path := range []string{
			"/api/requests?kind=bug",
			"/api/requests?status=done",
			"/api/requests?scope=mine&project=nope",
			"/api/requests?scope=all&project=shop&kind=bug",
			"/api/requests?scope=all&project=shop&status=Planned",
		} {
			st, body := marie.do("GET", path, nil)
			eq(t, path, st, http.StatusUnprocessableEntity)
			var v struct {
				Error struct{ Fields map[string]string }
			}
			json.Unmarshal(body, &v)
			eq(t, path+" names the field", len(v.Error.Fields), 1)
		}
	})

	t.Run("isolation", func(t *testing.T) {
		// A project the person cannot see answers like one that does not exist.
		_, hidden := paul.do("GET", "/api/requests?scope=mine&project=shop", nil)
		_, unknown := paul.do("GET", "/api/requests?scope=mine&project=nope", nil)
		eq(t, "hidden project = unknown project", string(hidden), string(unknown))
		eq(t, "all, hidden project", paul.titles("/api/requests?scope=all&project=shop&kind=problem"), []string{"status 404"})
		eq(t, "all, hidden project, status", paul.titles("/api/requests?scope=all&project=shop&status=planned"), []string{"status 404"})
		// In a project he sees, a filter never reveals what the role hides.
		eq(t, "reporter, kind", paul.titles("/api/requests?scope=all&project=app&kind=problem"), []string{})
		eq(t, "reporter, status", paul.titles("/api/requests?scope=all&project=app&status=received"), []string{"Comment exporter ?"})
		eq(t, "mine, status", paul.titles("/api/requests?status=planned"), []string{})
	})
}

func TestRequestDetailMilestone(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")
	id := marie.problem("shop", "PDF coupé")

	detail := func() map[string]any {
		st, body := marie.do("GET", "/api/requests/"+id, nil)
		eq(t, "detail", st, http.StatusOK)
		var v map[string]any
		json.Unmarshal(body, &v)
		return v
	}
	_, ok := detail()["milestone"]
	eq(t, "no milestone yet", ok, false)

	e.milestone(1)
	v := marie.eventually(id, "planned")
	eq(t, "history", v.history(), []string{"received", "planned"})
	d := detail()
	eq(t, "milestone", d["milestone"], "v2.1")
	for _, h := range d["history"].([]any) {
		at, _ := time.Parse(time.RFC3339Nano, h.(map[string]any)["at"].(string))
		eq(t, "dated step", time.Since(at) < time.Minute, true)
	}
}

func TestMovedSinceLastVisit(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.also("marie@acme.test", "app", "reporter")
	e.grant("paul@acme.test", "shop", "reporter")
	marie, paul := e.login("marie@acme.test"), e.login("paul@acme.test")
	ctx := context.Background()
	exec := func(sql string) {
		t.Helper()
		if _, err := e.pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	// away moves every recorded moment an hour back, as if an hour had passed.
	away := func() {
		exec("UPDATE app_user SET last_seen_at = last_seen_at - interval '1 hour'")
		exec("UPDATE request_status SET at = at - interval '1 hour'")
	}
	type moved struct {
		Title, Status string
		MovedAt       *time.Time
	}
	home := func() []moved {
		t.Helper()
		st, body := marie.do("GET", "/api/home", nil)
		if st != http.StatusOK {
			t.Fatalf("home: %d %s", st, body)
		}
		var v struct{ Moved []moved }
		json.Unmarshal(body, &v)
		return v.Moved
	}
	titles := func(list []moved) []string {
		out := []string{}
		for _, m := range list {
			out = append(out, m.Title+":"+m.Status)
		}
		return out
	}

	a := marie.problem("shop", "PDF coupé")  // shop #1
	b := marie.problem("shop", "Export CSV") // shop #2
	d := marie.problem("app", "Notification manquante")
	p := paul.problem("shop", "Mode sombre") // shop #3

	eq(t, "first visit: nothing to compare with", titles(home()), []string{})

	away()
	e.milestone(1)
	e.milestone(3)
	marie.eventually(a, "planned")
	paul.eventually(p, "planned")
	got := home()
	eq(t, "only her request that moved", titles(got), []string{"PDF coupé:planned"})
	if len(got) == 1 && (got[0].MovedAt == nil || time.Since(*got[0].MovedAt) > time.Minute) {
		t.Errorf("movedAt: %v", got[0].MovedAt)
	}
	eq(t, "a reload within the visit keeps it", titles(home()), []string{"PDF coupé:planned"})

	away()
	eq(t, "the next visit starts from the last one", titles(home()), []string{})

	// Marie loses access to shop: its moves no longer show, the app one does.
	exec(`DELETE FROM project_grant WHERE user_id = (SELECT id FROM app_user WHERE email = 'marie@acme.test')
	      AND project_id = (SELECT id FROM project WHERE slug = 'shop')`)
	away()
	e.milestone(2)
	i := e.fakeIssue(app, 1)
	i.Milestone = "v1"
	e.send("issues", issuePayload("milestoned", "acme/app", i, time.Now().Add(time.Second)))
	marie.eventually(d, "planned")
	for deadline := time.Now().Add(10 * time.Second); e.count("SELECT count(*) FROM request WHERE id = $1 AND status = 'planned'", b) == 0; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("shop #2 not planned")
		}
	}
	eq(t, "hidden requests never show", titles(home()), []string{"Notification manquante:planned"})
}
