# Hublot

> Working name. Everyone looks through the same porthole.

An open-source portal on top of GitHub. People without a GitHub account file requests
(bugs, ideas, questions) in a guided way, follow them like a parcel, and see what is coming.
The product team triages every repository in one place. **GitHub stays the single source of
truth**: the portal only keeps its users, their access, attachments, the person ↔ issue link
and a disposable cache.

The full product brief, decisions and mockups are in [`docs/HUBLOT-reprise.md`](docs/HUBLOT-reprise.md)
and [`docs/design/maquettes/`](docs/design/maquettes/). The HTTP API is in [`docs/api.md`](docs/api.md).

## Status: 0.2.0

| Done | Next |
|---|---|
| Magic-link sign-in limited to allowed domains | Roadmap and release notes |
| Per-project RBAC, groups, deny by default, isolation test suite | Two-way comments |
| Request forms read from `.github/ISSUE_TEMPLATE/*.yml`, filing with attachments (fs or S3) | Votes, "me too" |
| GitHub webhooks, issue cache, periodic resync, status history | Guided feature/issue creation, MCP server |
| Status e-mails, with a preference to turn them off | Statuses from deployments, priority field |
| Triage inbox: accept, change kind, transfer, confidential, close (written to GitHub) | Daily digest |
| My/all requests with filters, parcel timeline, "moved since your last visit" | Admin screens (access, settings), OIDC |

## Stack

Go (chi, pgx + sqlc, goose, River, go-github + ghinstallation) and React/Vite/TypeScript
(Tailwind, TanStack Query, react-i18next), shipped as one binary in one Docker image,
with PostgreSQL and S3-compatible storage.

```
cmd/hublot          CLI: serve, migrate, user, group, grant, resync
internal/authz      the single authorization function Can(subject, action, resource)
internal/auth       magic links and sessions
internal/portal     projects, requests, attachments (every call goes through authz), GitHub sync jobs
internal/forge      GitHub adapter (GitHub App), fake forge for development, issue forms
internal/httpapi    JSON API, attachment links, GitHub webhook, SPA serving
internal/db         migrations and sqlc queries
web/                the SPA
deploy/dev          development configuration and sample issue forms
site/               landing page (FR and EN), published to GitHub Pages by .github/workflows/pages.yml
```

## Try it

Everything in Docker, with a fake forge (issues stay in memory) and Mailpit for e-mails:

```bash
docker compose up --build
```

Open http://localhost:8080, sign in as `enzo@acme.test` (owner on first sign-in), and read
the link in Mailpit at http://localhost:8025. A fresh owner sees every project; to try a
regular person, grant roles from the command line (see `make seed`).

## Develop

Requirements: Go 1.26, Node 22, Docker, `sqlc` (only to change queries).

```bash
make deps      # Postgres + Mailpit
make seed      # Enzo (owner), Marie (Support group), Paul
make dev       # API with the fake forge
make web-dev   # Vite dev server, in another terminal
make test      # Go tests, including the database isolation suite
make lint
```

Ports taken on your machine? `make dev API_PORT=8090 WEB_PORT=5180 PG_PORT=5439`
(use the same values for `web-dev` and `seed`).

Without GitHub, move a request along as a developer would. `/dev/fake/...` exists only
with `forge.driver: fake`, is unauthenticated, and must never be used in production
(see [`docs/api.md`](docs/api.md)):

```bash
curl -X POST localhost:8080/dev/fake/issues/acme/boutique/1 -d '{"milestone": "v2.1"}'
curl -X POST localhost:8080/dev/fake/issues/acme/boutique/1 -d '{"state": "closed", "state_reason": "completed"}'
```

## Configure

Copy [`hublot.example.yaml`](hublot.example.yaml). Projects, allowed domains, request kinds
and their issue forms are configuration, never code. `${VAR}` references are read from the
environment so secrets stay out of the file. To connect GitHub, create a GitHub App with:
Issues read/write, Metadata read, Contents read, organisation Projects read, and install it
on the organisation.

The portal follows GitHub through the App webhook. In the App settings, set the webhook URL
to `{base_url}/webhooks/github`, a random secret (the same value as
`forge.github.webhook_secret`, e.g. `openssl rand -hex 32`), and subscribe to the **Issues**
and **Projects v2 item** events. Issue forms must not auto-apply the `portal` label:
the portal applies it to the issues it opens, and follows a moved request only into an
issue its App opened. Deliveries are verified, deduplicated by their id and
processed in the background (River jobs, in Postgres). A periodic resync
(`sync.resync_interval`) repairs what a lost delivery missed, re-reading every portal issue
once per `sync.full_resync_interval` (24 hours by default) so a lost Projects v2 item
delivery is repaired too; to re-read every portal issue by hand, after an outage or a
configuration change:

```bash
hublot resync                    # every project
hublot resync -project online-shop
```

A project can take its detailed status (in progress, in test…) from a single-select field
of an organisation Project: see `status` in the example configuration.

Give access from the command line until the admin screens exist:

```bash
hublot user -email marie@example.com -name Marie
hublot group -name Support -member marie@example.com
hublot grant -group Support -project online-shop -role reporter
```

Roles per project: `viewer`, `reporter`, `contributor`, `triager`, `manager`
(see PRD §5 bis). Every change is written to the audit log.

## License

[AGPL-3.0-or-later](LICENSE), © ET Technology Consulting. Contributions require signing the
CLA, see [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).
