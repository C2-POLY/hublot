# Changelog

Notable changes to Hublot. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and versions follow [Semantic Versioning](https://semver.org/): while the version starts with 0,
each 0.X.0 release delivers a milestone and nothing is promised stable yet.

Planned: 0.3.0 guided creation, MCP server · 0.4.0 production install and setup wizard ·
1.0.0 open source release. Roadmap, release notes, comments, votes and deployment-based statuses
are scheduled in between.

## [Unreleased]

## [0.2.0] - 2026-10-27

Milestone 2: requests follow what happens in GitHub, people are told when they move, and
triagers sort everything in one place.

### Added

- GitHub App webhook (`/webhooks/github`): signature checked, deliveries deduplicated, work
  queued in PostgreSQL (River). Issue changes, transfers and Project status changes update the
  portal; a periodic resync, also available as `hublot resync`, repairs lost deliveries.
- Request statuses computed from GitHub: an optional GitHub Project status field per project,
  else the planned version, else open or closed and its reason. Every change is kept in a
  status history.
- E-mail to the author when their request changes status, once per status, in their language,
  never to someone who can no longer see the request. Preferences page to turn it off.
- Triage inbox for triagers, managers and admins across their projects: accept, change kind
  (labels and optional issue type), move to another project, make private or public, close as
  not planned or duplicate. Written to GitHub first, every action audited.
- Filters by project, kind and status on my requests and all requests, kept in the address.
- Parcel timeline on the request page with a plain sentence (the planned version when there is
  one), vertical on phones.
- Home block listing the requests that moved since the previous visit.
- With the fake forge, `/dev/fake/issues/...` plays the part of GitHub for local development.

### Changed

- New requests carry the `portal:new` label until they are triaged.

## [0.1.0] - 2026-10-27

Milestone 1: people outside the development team can sign in and file a request that becomes
a GitHub issue.

### Added

- Sign-in with a magic link: single use, valid 15 minutes, stored hashed, rate limited per
  address and per IP, limited to the allowed domains. No password.
- Per-project roles (viewer, reporter, contributor, triager, manager), groups and organisation
  roles, all checked by one authorization function that denies by default, with an isolation
  test suite. Hidden resources answer 404.
- Request forms read from each repository's `.github/ISSUE_TEMPLATE/*.yml`, for three kinds
  of request: problem, idea, question.
- Filing a request in four guided steps, on desktop and mobile, with screenshots pasted,
  dropped, picked or captured. Attachments are checked on their content and stored on disk
  or in S3-compatible storage.
- Attachment links in issues open only with a Hublot session allowed to see the request.
- My requests and a parcel-style tracking page.
- Command line to manage people, groups and access (`hublot user`, `group`, `grant`),
  written to the audit log.
- Portal in French and English, in the Franc visual identity, with the vehicle journey on
  the sign-in page.
- Landing page in French and English, published to GitHub Pages.
- One Docker image (binary with the web app embedded), PostgreSQL, a fake forge and Mailpit
  for local development.

[Unreleased]: https://github.com/C2-POLY/hublot/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/C2-POLY/hublot/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/C2-POLY/hublot/releases/tag/v0.1.0
