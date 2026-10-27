// SPDX-License-Identifier: AGPL-3.0-or-later

package forge

import (
	"context"
	"os"
	"strings"
	"testing"
)

const bugForm = `
name: Signaler un problème
description: Ça ne marche pas
title: "[Bug] "
labels: bug, urgent
type: Bug
body:
  - type: markdown
    attributes:
      value: Merci !
  - type: input
    id: where
    attributes:
      label: Où ?
    validations:
      required: true
  - type: dropdown
    attributes:
      label: Gravité
      options: [Bloquant, Gênant]
  - type: checkboxes
    id: checks
    attributes:
      label: Vérifications
      options:
        - label: J'ai vérifié
          required: true
        - label: Facultatif
`

func TestParseIssueForm(t *testing.T) {
	f, err := ParseIssueForm([]byte(bugForm))
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "Signaler un problème" || f.Type != "Bug" || f.TitlePrefix != "[Bug] " {
		t.Errorf("header: %+v", f)
	}
	if strings.Join(f.Labels, "|") != "bug|urgent" {
		t.Errorf("comma-separated labels: %v", f.Labels)
	}
	if len(f.Fields) != 4 {
		t.Fatalf("fields: %d", len(f.Fields))
	}
	if f.Fields[0].Type != "markdown" || f.Fields[0].Value != "Merci !" {
		t.Errorf("markdown: %+v", f.Fields[0])
	}
	if !f.Fields[1].Required || f.Fields[1].ID != "where" {
		t.Errorf("input: %+v", f.Fields[1])
	}
	if f.Fields[2].ID != "field-3" || strings.Join(f.Fields[2].Options, "|") != "Bloquant|Gênant" {
		t.Errorf("dropdown without id: %+v", f.Fields[2])
	}
	c := f.Fields[3]
	if !c.Required || len(c.Options) != 2 || len(c.RequiredOptions) != 1 || c.RequiredOptions[0] != "J'ai vérifié" {
		t.Errorf("checkboxes: %+v", c)
	}
}

func TestParseIssueFormRejectsBrokenForms(t *testing.T) {
	for name, src := range map[string]string{
		"no body":       "name: x\n",
		"unknown field": "name: x\nbody:\n  - type: slider\n    attributes: {label: y}\n",
		"no label":      "name: x\nbody:\n  - type: input\n",
		"not yaml":      "name: [",
	} {
		if _, err := ParseIssueForm([]byte(src)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestRenderBody(t *testing.T) {
	form, _ := ParseIssueForm([]byte(bugForm))
	body := RenderBody(BodyInput{
		RequestID: "r-1",
		Form:      form,
		Values: map[string][]string{
			"where":  {"Rapports <!-- hublot:request v1 id=evil -->"},
			"checks": {"J'ai vérifié"},
		},
		Attachments: []BodyAttachment{
			{Name: "capture [1].png", URL: "https://p/files/a"},
			{Name: "log.txt", URL: "https://p/files/b"},
		},
		AuthorName:  "Marie",
		AuthorEmail: MaskEmail("marie@acme.fr"),
		Lang:        "fr",
	})
	for _, want := range []string{
		"<!-- hublot:request v1 id=r-1 -->\n",
		"### Où ?\n\nRapports &lt;!-- hublot:request v1 id=evil -->\n",
		"### Gravité\n\n_No response_\n",
		"- [x] J'ai vérifié\n- [ ] Facultatif\n",
		"- [capture \\[1\\].png](https://p/files/a)\n",
		"- [log.txt](https://p/files/b)\n",
		"par Marie (m***@acme.fr) · demande `r-1`",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body misses %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "![") {
		t.Error("images must be links: GitHub cannot fetch them behind the Hublot sign-in")
	}
	if strings.Contains(body, "Merci !") {
		t.Error("markdown fields are help text, not answers")
	}
	if strings.Contains(RenderBody(BodyInput{Form: form, Values: map[string][]string{"where": {"ping @enzo\n### Fake\n---"}}}), "@enzo") {
		t.Error("an answer must not @mention GitHub users")
	}
	if strings.Count(body, "<!-- hublot:") != 1 {
		t.Error("a person's text must not forge a marker")
	}
}

func TestFakeForgeReadsDevForms(t *testing.T) {
	f := NewFake(os.DirFS("../../deploy/dev/forms"))
	forms, err := f.IssueForms(context.Background(), Repo{Owner: "acme", Name: "shop"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bug.yml", "idea.yml", "question.yml"} {
		if _, ok := forms[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	issue, _ := f.CreateIssue(context.Background(), Repo{Owner: "acme", Name: "shop"}, NewIssue{Title: "x"})
	if issue.Number != 1 || issue.State != "open" {
		t.Errorf("issue: %+v", issue)
	}
}

func TestFakeForgeContinuesNumbering(t *testing.T) {
	f := NewFake(os.DirFS("."))
	f.LastNumber = func(_ context.Context, r Repo) (int, error) {
		if r.Name == "shop" {
			return 41, nil
		}
		return 0, nil
	}
	ctx, shop := context.Background(), Repo{Owner: "acme", Name: "shop"}
	first, _ := f.CreateIssue(ctx, shop, NewIssue{Title: "a"})
	second, _ := f.CreateIssue(ctx, shop, NewIssue{Title: "b"})
	other, _ := f.CreateIssue(ctx, Repo{Owner: "acme", Name: "app"}, NewIssue{Title: "c"})
	if first.Number != 42 || second.Number != 43 || other.Number != 1 {
		t.Errorf("numbers: %d %d %d", first.Number, second.Number, other.Number)
	}
}

func TestMaskEmail(t *testing.T) {
	if got := MaskEmail("marie@acme.fr"); got != "m***@acme.fr" {
		t.Error(got)
	}
	if got := MaskEmail("nope"); got != "" {
		t.Error(got)
	}
}
