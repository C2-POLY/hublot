// SPDX-License-Identifier: AGPL-3.0-or-later

// Package config loads the portal configuration from a YAML file.
// Nothing about a particular customer lives in the code (PRD §3, principle 3).
// ${VAR} references in the file are expanded from the environment, so secrets stay out of it.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL     string `yaml:"base_url"`
	Listen      string `yaml:"listen"`
	DatabaseURL string `yaml:"database_url"`
	// Secret signs attachment links and must be long and random.
	Secret          string        `yaml:"secret"`
	CookieSecure    *bool         `yaml:"cookie_secure"`
	SessionLifetime time.Duration `yaml:"session_lifetime"`
	// TrustProxyHeaders reads the client IP from X-Forwarded-For / X-Real-IP. Enable it only
	// behind a reverse proxy that sets them, or the per-IP rate limit can be bypassed.
	TrustProxyHeaders bool `yaml:"trust_proxy_headers"`
	// WebDir serves the SPA from disk when the binary was built without the embedded one.
	WebDir string `yaml:"web_dir"`

	Auth        Auth        `yaml:"auth"`
	Mail        Mail        `yaml:"mail"`
	Storage     Storage     `yaml:"storage"`
	Attachments Attachments `yaml:"attachments"`
	Forge       Forge       `yaml:"forge"`
	// Kinds maps each request kind (problem, idea, question) to an issue form file.
	Kinds    map[string]string `yaml:"kinds"`
	Projects []Project         `yaml:"projects"`
}

type Auth struct {
	AllowedDomains  []string `yaml:"allowed_domains"`
	BootstrapOwners []string `yaml:"bootstrap_owners"`
}

type Mail struct {
	Driver   string `yaml:"driver"` // smtp or log
	From     string `yaml:"from"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Storage struct {
	Driver string `yaml:"driver"` // fs or s3
	Dir    string `yaml:"dir"`
	S3     S3     `yaml:"s3"`
}

type S3 struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type Attachments struct {
	MaxBytes int64 `yaml:"max_bytes"`
}

type Forge struct {
	Driver string `yaml:"driver"` // github or fake
	GitHub GitHub `yaml:"github"`
	// AuthorEmail controls how the author's address appears in the issue body: masked, full or hidden.
	AuthorEmail string `yaml:"author_email"`
	// FakeDir holds the issue forms served by the fake forge, one folder per repository.
	FakeDir string `yaml:"fake_dir"`
}

type GitHub struct {
	AppID          int64  `yaml:"app_id"`
	InstallationID int64  `yaml:"installation_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
	APIURL         string `yaml:"api_url"` // GitHub Enterprise only
}

type Project struct {
	Slug            string            `yaml:"slug"`
	Name            string            `yaml:"name"`
	Hint            string            `yaml:"hint"`
	Color           string            `yaml:"color"`
	Vehicle         string            `yaml:"vehicle"`
	Repo            string            `yaml:"repo"` // owner/name
	ReportersSeeAll bool              `yaml:"reporters_see_all"`
	Inbox           bool              `yaml:"inbox"`
	Kinds           map[string]string `yaml:"kinds"`
}

func (p Project) RepoParts() (owner, name string) {
	owner, name, _ = strings.Cut(p.Repo, "/")
	return owner, name
}

// FormFor returns the issue form file used for a kind in this project.
func (c *Config) FormFor(p Project, kind string) string {
	if f, ok := p.Kinds[kind]; ok {
		return f
	}
	return c.Kinds[kind]
}

func (c *Config) Project(slug string) (Project, bool) {
	for _, p := range c.Projects {
		if p.Slug == slug {
			return p, true
		}
	}
	return Project{}, false
}

func (c *Config) Secure() bool {
	if c.CookieSecure != nil {
		return *c.CookieSecure
	}
	return strings.HasPrefix(c.BaseURL, "https://")
}

func (c *Config) DomainAllowed(email string) bool {
	_, domain, ok := strings.Cut(email, "@")
	return ok && slices.Contains(c.Auth.AllowedDomains, strings.ToLower(domain))
}

func (c *Config) IsBootstrapOwner(email string) bool {
	return slices.Contains(c.Auth.BootstrapOwners, strings.ToLower(email))
}

var (
	Kinds    = []string{"problem", "idea", "question"}
	Vehicles = []string{"boat", "submarine", "train", "plane", "rocket", "car", "bus"}
	slugRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
	colorRe  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// Load reads the file at path, expands environment references and checks the result.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(expandEnv(string(raw)))
}

var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// expandEnv replaces only explicit ${VAR} references, so a literal $ in a value survives.
func expandEnv(text string) string {
	return envRef.ReplaceAllStringFunc(text, func(ref string) string {
		return os.Getenv(envRef.FindStringSubmatch(ref)[1])
	})
}

func Parse(text string) (*Config, error) {
	c := &Config{}
	if err := yaml.Unmarshal([]byte(text), c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.defaults()
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

func (c *Config) defaults() {
	if c.Listen == "" {
		c.Listen = ":8080"
	}
	if c.SessionLifetime == 0 {
		c.SessionLifetime = 30 * 24 * time.Hour
	}
	if c.Mail.Port == 0 {
		c.Mail.Port = 587
	}
	if c.Storage.Driver == "" {
		c.Storage.Driver = "fs"
	}
	if c.Attachments.MaxBytes == 0 {
		c.Attachments.MaxBytes = 10 << 20
	}
	if c.Forge.Driver == "" {
		c.Forge.Driver = "github"
	}
	if c.Forge.AuthorEmail == "" {
		c.Forge.AuthorEmail = "masked"
	}
	if c.Kinds == nil {
		c.Kinds = map[string]string{"problem": "bug.yml", "idea": "idea.yml", "question": "question.yml"}
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	for i, d := range c.Auth.AllowedDomains {
		c.Auth.AllowedDomains[i] = strings.ToLower(strings.TrimSpace(d))
	}
	for i, e := range c.Auth.BootstrapOwners {
		c.Auth.BootstrapOwners[i] = strings.ToLower(strings.TrimSpace(e))
	}
}

func (c *Config) validate() error {
	var errs []error
	if u, err := url.Parse(c.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, errors.New("base_url must be an absolute URL"))
	}
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("database_url is required"))
	}
	if len(c.Secret) < 32 {
		errs = append(errs, errors.New("secret must be at least 32 characters"))
	}
	if len(c.Auth.AllowedDomains) == 0 {
		errs = append(errs, errors.New("auth.allowed_domains needs at least one domain"))
	}
	if !slices.Contains([]string{"smtp", "log"}, c.Mail.Driver) {
		// No default: the log driver prints sign-in links, it must be chosen on purpose.
		errs = append(errs, fmt.Errorf("mail.driver %q: use smtp or log", c.Mail.Driver))
	}
	if c.Mail.Driver == "smtp" && (c.Mail.Host == "" || c.Mail.From == "") {
		errs = append(errs, errors.New("mail.host and mail.from are required with the smtp driver"))
	}
	if c.Storage.Driver == "s3" && (c.Storage.S3.Endpoint == "" || c.Storage.S3.Bucket == "") {
		errs = append(errs, errors.New("storage.s3.endpoint and storage.s3.bucket are required with the s3 driver"))
	}
	if !slices.Contains([]string{"fs", "s3"}, c.Storage.Driver) {
		errs = append(errs, fmt.Errorf("storage.driver %q: use fs or s3", c.Storage.Driver))
	}
	if c.Storage.Driver == "fs" && c.Storage.Dir == "" {
		errs = append(errs, errors.New("storage.dir is required with the fs driver"))
	}
	if !slices.Contains([]string{"github", "fake"}, c.Forge.Driver) {
		errs = append(errs, fmt.Errorf("forge.driver %q: use github or fake", c.Forge.Driver))
	}
	if !slices.Contains([]string{"masked", "full", "hidden"}, c.Forge.AuthorEmail) {
		errs = append(errs, fmt.Errorf("forge.author_email %q: use masked, full or hidden", c.Forge.AuthorEmail))
	}
	for k := range c.Kinds {
		if !slices.Contains(Kinds, k) {
			errs = append(errs, fmt.Errorf("kinds: unknown kind %q", k))
		}
	}
	seen := map[string]bool{}
	inboxes := 0
	for _, p := range c.Projects {
		where := "project " + p.Slug
		if !slugRe.MatchString(p.Slug) {
			errs = append(errs, fmt.Errorf("%s: slug must be lowercase letters, digits and dashes", where))
		}
		if seen[p.Slug] {
			errs = append(errs, fmt.Errorf("%s: duplicate slug", where))
		}
		seen[p.Slug] = true
		if p.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name is required", where))
		}
		if !colorRe.MatchString(p.Color) {
			errs = append(errs, fmt.Errorf("%s: color must look like #B9AEFF", where))
		}
		if !slices.Contains(Vehicles, p.Vehicle) {
			errs = append(errs, fmt.Errorf("%s: vehicle must be one of %s", where, strings.Join(Vehicles, ", ")))
		}
		if owner, name := p.RepoParts(); owner == "" || name == "" || strings.Contains(name, "/") {
			errs = append(errs, fmt.Errorf("%s: repo must be owner/name", where))
		}
		for k := range p.Kinds {
			if !slices.Contains(Kinds, k) {
				errs = append(errs, fmt.Errorf("%s: unknown kind %q", where, k))
			}
		}
		if p.Inbox {
			inboxes++
		}
	}
	if inboxes > 1 {
		errs = append(errs, errors.New("only one project can be the inbox"))
	}
	return errors.Join(errs...)
}
