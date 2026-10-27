// SPDX-License-Identifier: AGPL-3.0-or-later

package mail

import (
	"strings"
	"testing"
)

func TestStatusChanged(t *testing.T) {
	statuses := []string{"received", "planned", "in_progress", "in_test", "available", "answered", "closed"}
	for _, lang := range []string{"fr", "en"} {
		for _, st := range statuses {
			m, ok := StatusChanged("marie@acme.test", lang, "Marie", "Le PDF", st, "https://p/demandes/1", "https://p/preferences")
			if !ok {
				t.Fatalf("%s %s: no words", lang, st)
			}
			if !strings.Contains(m.Subject, "Le PDF") || !strings.Contains(m.Text, "Le PDF") ||
				!strings.Contains(m.Text, "\nhttps://p/demandes/1\n") || !strings.Contains(m.Text, "https://p/preferences") {
				t.Errorf("%s %s: %q / %q", lang, st, m.Subject, m.Text)
			}
		}
	}

	m, _ := StatusChanged("a@acme.test", "de", "", "Export\r\nBcc: x", "available", "l", "p")
	if m.Subject != "Votre demande est disponible : Export Bcc: x" {
		t.Errorf("french by default, one line: %q", m.Subject)
	}
	if !strings.HasPrefix(m.Text, "Bonjour,\n") {
		t.Errorf("no name: %q", m.Text)
	}
	if _, ok := StatusChanged("a@acme.test", "fr", "", "X", "triaged", "l", "p"); ok {
		t.Error("unknown status")
	}
}
