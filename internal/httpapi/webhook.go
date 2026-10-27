// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/C2-POLY/hublot/internal/forge"
)

// maxWebhookBody is far above an issues or projects_v2_item delivery.
const maxWebhookBody = 5 << 20

// githubWebhook accepts a GitHub App delivery: signature checked against the raw body,
// delivery id recorded and job queued in one transaction, then 202 at once; the work
// happens in the background. A redelivery answers 200 and is not processed again.
func (s *Server) githubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", nil)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if !forge.VerifySignature(s.Cfg.Forge.GitHub.WebhookSecret, r.Header.Get("X-Hub-Signature-256"), body) {
		writeError(w, http.StatusUnauthorized, "unauthenticated", nil)
		return
	}
	delivery, event := r.Header.Get("X-GitHub-Delivery"), r.Header.Get("X-GitHub-Event")
	if delivery == "" || event == "" {
		writeError(w, http.StatusBadRequest, "bad_request", nil)
		return
	}
	ev, ok, err := forge.ParseGitHubEvent(event, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", nil)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	queued, err := s.Portal.RecordDelivery(r.Context(), delivery, event, ev)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	if !queued {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

// fakeIssueChange lets a developer act as someone working in GitHub, with the fake
// forge only: it changes the issue in memory and queues the job a webhook would.
func (s *Server) fakeIssueChange(fake *forge.Fake) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		number, err := strconv.Atoi(chi.URLParam(r, "number"))
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", nil)
			return
		}
		var in struct {
			State         string   `json:"state"`
			StateReason   string   `json:"state_reason"`
			Labels        []string `json:"labels"`
			Milestone     *string  `json:"milestone"`
			ProjectStatus *string  `json:"project_status"`
		}
		if !decode(w, r, &in) {
			return
		}
		if in.State != "" && in.State != "open" && in.State != "closed" {
			writeError(w, http.StatusUnprocessableEntity, "validation", map[string]string{"state": "invalid"})
			return
		}
		repo := forge.Repo{Owner: chi.URLParam(r, "owner"), Name: chi.URLParam(r, "repo")}
		issue, err := fake.Change(repo, number, forge.FakeChange{
			State: in.State, StateReason: in.StateReason, Labels: in.Labels,
			Milestone: in.Milestone, ProjectStatus: in.ProjectStatus,
		})
		if errors.Is(err, forge.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", nil)
			return
		}
		if err != nil {
			s.internal(w, r, err)
			return
		}
		ctx := r.Context()
		if err := s.Portal.Enqueue(ctx, forge.Event{Kind: forge.EventIssue, Action: "edited", Repo: repo, Issue: issue}); err != nil {
			s.internal(w, r, err)
			return
		}
		if in.ProjectStatus != nil {
			if err := s.Portal.Enqueue(ctx, forge.Event{Kind: forge.EventProjectItem, Action: "edited", ContentNodeID: issue.NodeID}); err != nil {
				s.internal(w, r, err)
				return
			}
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"number": issue.Number, "state": issue.State,
			"stateReason": issue.StateReason, "labels": issue.Labels, "milestone": issue.Milestone})
	}
}
