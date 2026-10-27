// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"strings"
	"testing"
)

const minimal = `
base_url: https://portal.example.com/
database_url: postgres://x
secret: 0123456789abcdef0123456789abcdef
auth: {allowed_domains: [Example.com]}
mail: {driver: log}
storage: {dir: /tmp/x}
projects:
  - {slug: shop, name: Shop, color: "#B9AEFF", vehicle: boat, repo: org/shop, kinds: {idea: feature.yml}}
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
}

func TestValidation(t *testing.T) {
	_, err := Parse(`
base_url: not-a-url
secret: short
projects:
  - {slug: Shop!, color: blue, vehicle: horse, repo: shop}
  - {slug: a, name: A, color: "#FFFFFF", vehicle: boat, repo: o/a, inbox: true}
  - {slug: b, name: B, color: "#FFFFFF", vehicle: boat, repo: o/b, inbox: true}
`)
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"base_url", "database_url", "secret", "allowed_domains", "slug", "name is required",
		"color", "vehicle", "owner/name", "only one project can be the inbox"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
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
	t.Setenv("HUBLOT_SECRET", strings.Repeat("s", 40))
	t.Setenv("HUBLOT_GITHUB_APP_ID", "1")
	t.Setenv("HUBLOT_GITHUB_INSTALLATION_ID", "2")
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
