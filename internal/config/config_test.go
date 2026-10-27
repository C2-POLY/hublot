// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"strings"
	"testing"
	"time"
)

const minimal = `
base_url: https://portal.example.com/
database_url: postgres://x
auth: {allowed_domains: [Example.com]}
mail: {driver: log}
storage: {dir: /tmp/x}
forge: {github: {webhook_secret: s3cret}}
projects:
  - {slug: shop, name: Shop, color: "#B9AEFF", vehicle: boat, repo: org/shop, kinds: {idea: feature.yml}}
  - slug: app
    name: App
    color: "#FF9A62"
    vehicle: plane
    repo: org/app
    status: {project: 12, options: {Todo: planned, In progress: in_progress}}
`

func TestDefaults(t *testing.T) {
	c, err := Parse(minimal)
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != "https://portal.example.com" || !c.Secure() || c.Listen != ":8080" {
		t.Errorf("base: %q %v %q", c.BaseURL, c.Secure(), c.Listen)
	}
	if !c.DomainAllowed("marie@EXAMPLE.com") || c.DomainAllowed("marie@evil.com") || c.DomainAllowed("nope") {
		t.Error("domains")
	}
	p, _ := c.Project("shop")
	if c.FormFor(p, "idea") != "feature.yml" || c.FormFor(p, "problem") != "bug.yml" {
		t.Error("per-project kinds override the defaults")
	}
	if c.Sync.ResyncInterval != time.Hour || p.Status != nil {
		t.Errorf("sync: %v, status: %+v", c.Sync.ResyncInterval, p.Status)
	}
	app, _ := c.Project("app")
	if app.Status.Field != "Status" || app.Status.Project != 12 || app.Status.Options["In progress"] != "in_progress" {
		t.Errorf("status block: %+v", app.Status)
	}
}

func TestSyncValidation(t *testing.T) {
	_, err := Parse(`
base_url: https://portal.example.com
database_url: postgres://x
auth: {allowed_domains: [example.com]}
mail: {driver: log}
storage: {dir: /tmp/x}
sync: {resync_interval: 10s}
projects:
  - slug: shop
    name: Shop
    color: "#B9AEFF"
    vehicle: boat
    repo: org/shop
    status: {options: {Done: shipped}}
`)
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"webhook_secret is required", "resync_interval", "status.project",
		`unknown status "shipped"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
	if _, err := Parse(strings.Replace(minimal, "forge: {github: {webhook_secret: s3cret}}", "forge: {driver: fake}", 1)); err != nil {
		t.Errorf("the fake forge needs no webhook secret: %v", err)
	}
}

func TestValidation(t *testing.T) {
	_, err := Parse(`
base_url: not-a-url
projects:
  - {slug: Shop!, color: blue, vehicle: horse, repo: shop}
  - {slug: a, name: A, color: "#FFFFFF", vehicle: boat, repo: o/a, inbox: true}
  - {slug: b, name: B, color: "#FFFFFF", vehicle: boat, repo: o/b, inbox: true}
`)
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"base_url", "database_url", "allowed_domains", "slug", "name is required",
		"color", "vehicle", "owner/name", "only one project can be the inbox"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
}

func TestIssueTypes(t *testing.T) {
	c, err := Parse(minimal + "issue_types: {problem: Bug, idea: Feature, question: Question}\n")
	if err != nil {
		t.Fatal(err)
	}
	if c.IssueTypes["idea"] != "Feature" {
		t.Errorf("issue types: %v", c.IssueTypes)
	}
	_, err = Parse(minimal + "issue_types: {rant: Bug, idea: \"\"}\n")
	for _, want := range []string{`issue_types: unknown kind "rant"`, "issue_types: idea needs a type name"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in: %v", want, err)
		}
	}
}

func TestExpandOnlyExplicitReferences(t *testing.T) {
	t.Setenv("HUBLOT_X", "value")
	if got := expandEnv("a: ${HUBLOT_X}\nb: pa$word $HUBLOT_X"); got != "a: value\nb: pa$word $HUBLOT_X" {
		t.Error(got)
	}
}

func TestExampleConfigIsValid(t *testing.T) {
	t.Setenv("HUBLOT_DATABASE_URL", "postgres://x")
	t.Setenv("HUBLOT_GITHUB_APP_ID", "1")
	t.Setenv("HUBLOT_GITHUB_INSTALLATION_ID", "2")
	t.Setenv("HUBLOT_GITHUB_WEBHOOK_SECRET", "s3cret")
	if _, err := Load("../../hublot.example.yaml"); err != nil {
		t.Error(err)
	}
	t.Setenv("HUBLOT_BASE_URL", "http://localhost:5173")
	t.Setenv("HUBLOT_STORAGE_DIR", t.TempDir())
	t.Setenv("HUBLOT_SMTP_HOST", "localhost")
	if _, err := Load("../../deploy/dev/hublot.dev.yaml"); err != nil {
		t.Error(err)
	}
}
