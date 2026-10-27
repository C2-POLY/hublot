# Contributing

Thank you for helping. A few rules keep the project coherent.

## Contributor License Agreement

Every external contribution requires signing the Contributor License Agreement (CLA)
of ET Technology Consulting before it can be merged. The CLA assistant asks for it on
your first pull request.

Why: the project is AGPL-3.0, and the CLA keeps it possible for ET Technology Consulting
to offer a commercial license or change the license later. A DCO sign-off is not enough
for that.

> The CLA text is not final yet (see `docs/HUBLOT-reprise.md`, §4). Until it is published,
> external pull requests cannot be merged.

## Principles to respect

1. GitHub is the single source of truth. A feature that stores work outside GitHub is refused.
2. Configuration, never hard-coded values: no trace of a particular customer in the code.
3. Every read and write goes through `authz.Can`, on the server. Never hide data only in the UI.
4. No GitHub jargon in the team-facing screens: *demande*, *logiciel*, *disponible*.

## Practicalities

- Code, comments and commit messages in English; commits follow
  [Conventional Commits](https://www.conventionalcommits.org/) (`feat(auth): …`, `fix(web): …`).
- Every source file starts with `SPDX-License-Identifier: AGPL-3.0-or-later`.
- Dependencies must be AGPL-compatible (MIT, BSD, Apache-2.0, MPL-2.0).
- `make lint` and `make test` must pass. Access rules come with isolation tests in
  `internal/httpapi/isolation_test.go`.
- Changing queries: edit `internal/db/queries/*.sql`, then `make generate`.
- Do not copy code or designs from other feedback or tracker tools.
