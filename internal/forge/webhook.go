// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/go-github/v88/github"
)

// VerifySignature checks GitHub's X-Hub-Signature-256 header (sha256=<hex HMAC of the
// raw body>) in constant time. An empty secret accepts nothing.
func VerifySignature(secret, header string, body []byte) bool {
	hexSig, ok := strings.CutPrefix(header, "sha256=")
	if secret == "" || !ok {
		return false
	}
	got, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

// Sign returns the X-Hub-Signature-256 value of a body, for tests and tools.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

const (
	EventIssue       = "issue"
	EventProjectItem = "project_item"
)

// Event is a webhook delivery in portal terms.
type Event struct {
	Kind   string // EventIssue or EventProjectItem
	Action string
	// Repo and Issue: the issue the event is about (for a transfer, where it was).
	Repo  Repo
	Issue Issue
	// NewRepo and NewIssue: where a transferred issue went, when GitHub says so.
	NewRepo  *Repo
	NewIssue *Issue
	// ContentNodeID: the issue behind a Projects item.
	ContentNodeID string
}

var ErrBadPayload = errors.New("forge: malformed webhook payload")

// ParseGitHubEvent reads a delivery of the given X-GitHub-Event. ok is false for events
// the portal does not follow (ping, pull requests, items that are not issues…).
func ParseGitHubEvent(name string, body []byte) (ev Event, ok bool, err error) {
	if name != "issues" && name != "projects_v2_item" {
		return Event{}, false, nil
	}
	var p struct {
		Action     string             `json:"action"`
		Issue      *ghIssue           `json:"issue"`
		Repository *github.Repository `json:"repository"`
		Changes    struct {
			NewIssue      *ghIssue           `json:"new_issue"`
			NewRepository *github.Repository `json:"new_repository"`
		} `json:"changes"`
		Item *struct {
			ContentNodeID string `json:"content_node_id"`
			ContentType   string `json:"content_type"`
		} `json:"projects_v2_item"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return Event{}, false, ErrBadPayload
	}
	if name == "projects_v2_item" {
		if p.Item == nil || p.Item.ContentType != "Issue" || p.Item.ContentNodeID == "" {
			return Event{}, false, nil
		}
		return Event{Kind: EventProjectItem, Action: p.Action, ContentNodeID: p.Item.ContentNodeID}, true, nil
	}
	if p.Issue == nil || p.Repository == nil || p.Issue.GetNumber() == 0 {
		return Event{}, false, ErrBadPayload
	}
	if p.Issue.IsPullRequest() {
		return Event{}, false, nil
	}
	ev = Event{Kind: EventIssue, Action: p.Action, Repo: repoOf(p.Repository), Issue: p.Issue.issue()}
	if p.Action == "transferred" && p.Changes.NewIssue != nil && p.Changes.NewRepository != nil {
		to, moved := repoOf(p.Changes.NewRepository), p.Changes.NewIssue.issue()
		ev.NewRepo, ev.NewIssue = &to, &moved
	}
	return ev, true, nil
}

func repoOf(r *github.Repository) Repo {
	owner, name, _ := strings.Cut(r.GetFullName(), "/")
	if r.GetOwner().GetLogin() != "" {
		owner, name = r.GetOwner().GetLogin(), r.GetName()
	}
	return Repo{Owner: owner, Name: name}
}
