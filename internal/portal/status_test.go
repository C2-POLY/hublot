// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import "testing"

func TestStatus(t *testing.T) {
	options := map[string]string{"Todo": StatusPlanned, "In progress": StatusInProgress, "In review": StatusInTest, "Done": StatusAvailable}
	for _, c := range []struct {
		name    string
		kind    string
		issue   IssueState
		options map[string]string
		want    string
	}{
		{"open", "problem", IssueState{State: "open"}, nil, StatusReceived},
		{"reopened", "problem", IssueState{State: "open", StateReason: "reopened"}, nil, StatusReceived},
		{"milestone", "idea", IssueState{State: "open", Milestone: "v2.1"}, nil, StatusPlanned},
		{"project status", "problem", IssueState{State: "open", ProjectStatus: "In progress"}, options, StatusInProgress},
		{"project status beats milestone", "problem", IssueState{State: "open", Milestone: "v2", ProjectStatus: "In review"}, options, StatusInTest},
		{"unmapped option falls back to milestone", "problem", IssueState{State: "open", Milestone: "v2", ProjectStatus: "Backlog"}, options, StatusPlanned},
		{"project status without mapping", "problem", IssueState{State: "open", ProjectStatus: "Done"}, nil, StatusReceived},
		{"completed problem", "problem", IssueState{State: "closed", StateReason: "completed"}, nil, StatusAvailable},
		{"completed idea", "idea", IssueState{State: "closed", StateReason: "completed", ProjectStatus: "In progress"}, options, StatusAvailable},
		{"completed question", "question", IssueState{State: "closed", StateReason: "completed"}, nil, StatusAnswered},
		{"not planned", "problem", IssueState{State: "closed", StateReason: "not_planned"}, nil, StatusClosed},
		{"duplicate", "question", IssueState{State: "closed", StateReason: "duplicate"}, nil, StatusClosed},
		{"closed without reason", "problem", IssueState{State: "closed"}, nil, StatusClosed},
		{"deleted", "problem", IssueState{State: "open", Milestone: "v2", Deleted: true}, nil, StatusClosed},
	} {
		if got := Status(c.kind, c.issue, c.options); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
