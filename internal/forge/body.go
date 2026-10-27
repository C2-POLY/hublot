// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// BodyMarker opens every issue body the portal writes. The version lets the
// format evolve without breaking older issues, and the id lets a resync
// find the request an issue belongs to.
func BodyMarker(requestID string) string {
	return fmt.Sprintf("<!-- hublot:request v1 id=%s -->", requestID)
}

type BodyInput struct {
	RequestID   string
	Form        IssueForm
	Values      map[string][]string
	Attachments []BodyAttachment
	AuthorName  string
	AuthorEmail string // already masked or emptied according to the configuration
	Lang        string
}

type BodyAttachment struct {
	Name string
	URL  string
}

// RenderBody writes the issue body the way GitHub renders a submitted issue form
// (### label, then the answer), so developers read portal issues like any other.
func RenderBody(in BodyInput) string {
	t := bodyText[in.Lang]
	if t == nil {
		t = bodyText["en"]
	}
	var b strings.Builder
	b.WriteString(BodyMarker(in.RequestID))
	b.WriteString("\n\n")
	for _, f := range in.Form.Fields {
		if f.Type == "markdown" {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", oneLine(f.Label))
		values := in.Values[f.ID]
		switch {
		case f.Type == "checkboxes":
			for _, opt := range f.Options {
				mark := " "
				if slices.Contains(values, opt) {
					mark = "x"
				}
				fmt.Fprintf(&b, "- [%s] %s\n", mark, oneLine(opt))
			}
		case len(values) == 0 || strings.TrimSpace(strings.Join(values, "")) == "":
			b.WriteString("_No response_\n")
		default:
			b.WriteString(neutralise(strings.TrimSpace(strings.Join(values, ", "))))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(in.Attachments) > 0 {
		fmt.Fprintf(&b, "### %s\n\n", t["attachments"])
		for _, a := range in.Attachments {
			// A link, never an inline image: GitHub fetches images without the reader's
			// Hublot session, so they would show broken.
			fmt.Fprintf(&b, "- [%s](%s)\n", escapeLinkText(a.Name), a.URL)
		}
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	author := oneLine(in.AuthorName)
	if in.AuthorEmail != "" {
		author = fmt.Sprintf("%s (%s)", author, in.AuthorEmail)
	}
	fmt.Fprintf(&b, t["footer"]+"\n", author, in.RequestID)
	return b.String()
}

var bodyText = map[string]map[string]string{
	"fr": {"attachments": "Captures", "footer": "Déposée depuis le portail par %s · demande `%s`"},
	"en": {"attachments": "Attachments", "footer": "Filed from the portal by %s · request `%s`"},
}

// MaskEmail keeps the first letter and the domain: marie@acme.fr -> m***@acme.fr.
func MaskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return ""
	}
	return local[:1] + "***@" + domain
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func escapeLinkText(s string) string {
	return strings.NewReplacer("[", "\\[", "]", "\\]", "\n", " ").Replace(s)
}

var mention = regexp.MustCompile(`@([A-Za-z0-9])`)

// neutralise keeps a person's text from forging the portal's markers or sections,
// and from notifying GitHub users or teams with @mentions.
func neutralise(s string) string {
	s = strings.ReplaceAll(s, "<!--", "&lt;!--")
	s = mention.ReplaceAllString(s, "@\u200b$1")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "---") || strings.HasPrefix(t, "===") {
			lines[i] = "\\" + t
		}
	}
	return strings.Join(lines, "\n")
}
