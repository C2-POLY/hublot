# Changelog

Notable changes to Hublot. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and versions follow [Semantic Versioning](https://semver.org/): while the version starts with 0,
each 0.X.0 release delivers a milestone and nothing is promised stable yet.

Planned: 0.2.0 webhooks, full status lifecycle, triage, notifications · 0.3.0 guided creation,
MCP server · 0.4.0 production install and setup wizard · 1.0.0 open source release.

## [Unreleased]

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

[Unreleased]: https://github.com/et-technology-consulting/hublot/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/et-technology-consulting/hublot/releases/tag/v0.1.0
