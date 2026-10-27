// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

// Portal statuses, in plain words for people outside the team (docs/api.md).
const (
	StatusReceived   = "received"
	StatusPlanned    = "planned"
	StatusInProgress = "in_progress"
	StatusInTest     = "in_test"
	StatusAvailable  = "available"
	StatusAnswered   = "answered"
	StatusClosed     = "closed"
)

// IssueState is what the status is computed from: the cached issue.
type IssueState struct {
	State         string
	StateReason   string
	Milestone     string
	ProjectStatus string
	Deleted       bool
}

// Status computes the portal status of a request. options maps the options of the
// project's status field (config `status.options`) and may be nil. First match wins:
//  1. closed as completed: answered for a question, available otherwise;
//     closed for any other reason (not planned, duplicate…), or deleted: closed;
//  2. open with a mapped project status: that status;
//  3. open with a milestone: planned;
//  4. open: received.
func Status(kind string, issue IssueState, options map[string]string) string {
	if issue.Deleted {
		return StatusClosed
	}
	if issue.State == "closed" {
		if issue.StateReason != "completed" {
			return StatusClosed
		}
		if kind == "question" {
			return StatusAnswered
		}
		return StatusAvailable
	}
	if s, ok := options[issue.ProjectStatus]; ok && issue.ProjectStatus != "" {
		return s
	}
	if issue.Milestone != "" {
		return StatusPlanned
	}
	return StatusReceived
}
