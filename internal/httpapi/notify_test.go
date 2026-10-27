// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi_test

// The notification suite: a status change e-mails the request's author once, in their
// language, and never someone who turned it off or can no longer see the request.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/C2-POLY/hublot/internal/mail"
)

// statusMails returns the status e-mails sent to an address (not the sign-in links).
func (e *env) statusMails(to string) []mail.Message {
	out := []mail.Message{}
	for _, m := range e.mailer.Sent() {
		if m.To == to && strings.Contains(m.Text, "/demandes/") {
			out = append(out, m)
		}
	}
	return out
}

// notified waits until n notify_status jobs of the request have completed.
func (e *env) notified(id string, n int) {
	e.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for e.count("SELECT count(*) FROM river_job WHERE kind = 'notify_status' AND args->>'request_id' = $1 AND state = 'completed'", id) < n {
		if time.Now().After(deadline) {
			e.t.Fatalf("request %s: %d notify_status jobs not completed", id, n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// milestone plans issue number of shop through a webhook delivery.
func (e *env) milestone(number int) {
	e.t.Helper()
	i := e.fakeIssue(shop, number)
	i.Milestone = "v2.1"
	e.send("issues", issuePayload("milestoned", "acme/shop", i, time.Now().Add(time.Second)))
}

func TestStatusEmail(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("anna@acme.test", "shop", "reporter")
	if _, err := e.pool.Exec(context.Background(), "UPDATE app_user SET lang = 'en' WHERE email = 'anna@acme.test'"); err != nil {
		t.Fatal(err)
	}
	marie, anna := e.login("marie@acme.test"), e.login("anna@acme.test")

	id := marie.problem("shop", "Le PDF est illisible")
	e.milestone(1)
	marie.eventually(id, "planned")
	e.notified(id, 1)
	sent := e.statusMails("marie@acme.test")
	if len(sent) != 1 {
		t.Fatalf("status e-mails to marie: %d, want 1", len(sent))
	}
	m := sent[0]
	eq(t, "subject", m.Subject, "Votre demande est planifiée : Le PDF est illisible")
	eq(t, "sentence", strings.Contains(m.Text, "Votre demande « Le PDF est illisible » est planifiée."), true)
	eq(t, "button", strings.Contains(m.Text, "\nhttp://portal.test/demandes/"+id+"\n"), true)
	eq(t, "preferences", strings.Contains(m.Text, "http://portal.test/preferences"), true)
	eq(t, "never the content", strings.Contains(m.Text, "gênant") || strings.Contains(m.Text, "Rapports"), false)
	eq(t, "nobody else", len(e.statusMails("anna@acme.test")), 0)

	// A retried job, or one for a status the request already left, sends nothing more.
	ctx := context.Background()
	rid := uuid.MustParse(id)
	if err := e.portal.NotifyStatus(ctx, rid, "planned"); err != nil {
		t.Fatal(err)
	}
	if err := e.portal.NotifyStatus(ctx, rid, "available"); err != nil {
		t.Fatal(err)
	}
	eq(t, "retry does not resend", len(e.statusMails("marie@acme.test")), 1)
	eq(t, "bookkeeping", e.count("SELECT count(*) FROM request_notification WHERE request_id = $1", id), 1)

	// In the author's language.
	other := anna.problem("shop", "Export CSV")
	i := e.fakeIssue(shop, 2)
	i.State, i.StateReason = "closed", "completed"
	e.send("issues", issuePayload("closed", "acme/shop", i, time.Now().Add(time.Second)))
	anna.eventually(other, "available")
	e.notified(other, 1)
	sent = e.statusMails("anna@acme.test")
	if len(sent) != 1 {
		t.Fatalf("status e-mails to anna: %d, want 1", len(sent))
	}
	eq(t, "english subject", sent[0].Subject, "Your request is available: Export CSV")
	eq(t, "english sentence", strings.Contains(sent[0].Text, "Your request “Export CSV” is now available."), true)
	eq(t, "english link", strings.Contains(sent[0].Text, "http://portal.test/demandes/"+other), true)
}

func TestStatusEmailWithheld(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	e.grant("paul@acme.test", "shop", "reporter")
	e.grant("zoe@acme.test", "shop", "reporter")
	marie, paul, zoe := e.login("marie@acme.test"), e.login("paul@acme.test"), e.login("zoe@acme.test")
	ctx := context.Background()

	// Marie loses her access to the project before the change.
	gone := marie.problem("shop", "Le PDF est illisible")
	if _, err := e.pool.Exec(ctx, "DELETE FROM project_grant WHERE user_id = (SELECT id FROM app_user WHERE email = 'marie@acme.test')"); err != nil {
		t.Fatal(err)
	}
	e.milestone(1)
	e.notified(gone, 1)
	eq(t, "no access, no e-mail", len(e.statusMails("marie@acme.test")), 0)

	// Paul turned status e-mails off.
	st, body := paul.do("PATCH", "/api/me/preferences", map[string]any{"notifyStatus": false})
	eq(t, "turned off", st, http.StatusOK)
	eq(t, "answer", strings.TrimSpace(string(body)), `{"notifyStatus":false}`)
	off := paul.problem("shop", "Export CSV")
	e.milestone(2)
	paul.eventually(off, "planned")
	e.notified(off, 1)
	eq(t, "preference off, no e-mail", len(e.statusMails("paul@acme.test")), 0)

	// Zoé was suspended.
	susp := zoe.problem("shop", "Mode sombre")
	if _, err := e.pool.Exec(ctx, "UPDATE app_user SET suspended = true WHERE email = 'zoe@acme.test'"); err != nil {
		t.Fatal(err)
	}
	e.milestone(3)
	e.notified(susp, 1)
	eq(t, "suspended, no e-mail", len(e.statusMails("zoe@acme.test")), 0)
	eq(t, "nothing recorded as sent", e.count("SELECT count(*) FROM request_notification"), 0)
}

func TestPreferences(t *testing.T) {
	e := setup(t)
	e.grant("marie@acme.test", "shop", "reporter")
	marie := e.login("marie@acme.test")

	prefs := func(c *client) (int, map[string]any) {
		st, body := c.do("GET", "/api/me/preferences", nil)
		var v map[string]any
		json.Unmarshal(body, &v)
		return st, v
	}
	st, v := prefs(marie)
	eq(t, "default", st, http.StatusOK)
	eq(t, "on by default", v["notifyStatus"], true)

	st, _ = marie.do("PATCH", "/api/me/preferences", map[string]any{"notifyStatus": false})
	eq(t, "patch", st, http.StatusOK)
	_, v = prefs(marie)
	eq(t, "stored", v["notifyStatus"], false)

	st, _ = marie.do("PATCH", "/api/me/preferences", map[string]any{})
	eq(t, "missing field", st, http.StatusUnprocessableEntity)
	st, _ = marie.do("PATCH", "/api/me/preferences", map[string]any{"notifyStatus": "yes"})
	eq(t, "not a boolean", st, http.StatusBadRequest)
	_, v = prefs(marie)
	eq(t, "unchanged", v["notifyStatus"], false)

	req, _ := http.NewRequest("PATCH", e.srv.URL+"/api/me/preferences", strings.NewReader(`{"notifyStatus": true}`))
	resp, err := marie.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	eq(t, "no CSRF header", resp.StatusCode, http.StatusForbidden)

	anon := &client{e: e, http: &http.Client{}}
	st, _ = anon.do("GET", "/api/me/preferences", nil)
	eq(t, "signed out get", st, http.StatusUnauthorized)
	st, _ = anon.do("PATCH", "/api/me/preferences", map[string]any{"notifyStatus": true})
	eq(t, "signed out patch", st, http.StatusUnauthorized)
	_, v = prefs(marie)
	eq(t, "still off", v["notifyStatus"], false)
}
