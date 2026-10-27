# Portal HTTP API (milestone 2)

The SPA talks to the Go server under `/api`. All bodies are JSON (`application/json`)
except attachment upload (`multipart/form-data`).

- Authentication: an `HttpOnly` session cookie (`hublot_session`) set by `POST /api/auth/verify`.
- Every non-GET request must carry the header `X-Hublot: 1` (CSRF guard, together with `SameSite=Lax`).
- Errors: `{"error": {"code": "not_found", "message": "..."}}` with a matching status.
  Codes: `bad_request`, `unauthenticated` (401), `forbidden` (403), `not_found` (404),
  `rate_limited` (429), `too_large` (413), `unsupported_type` (415), `validation` (422, with `fields`), `internal`.
- Authorization is enforced server side on every call. A resource the user may not see answers `404`, never `403`,
  so its existence does not leak.

## Auth

| Method | Path | Body | Answer |
|---|---|---|---|
| `POST` | `/api/auth/magic-link` | `{"email": "marie@acme.fr", "lang": "fr"}` | `202 {}` always (no account enumeration); `429` when rate limited; `422` with `fields.email` when the address is malformed |
| `POST` | `/api/auth/verify` | `{"token": "..."}` | `200 {"user": Me}` and sets the cookie; `400 invalid_token` when expired, used or unknown |
| `POST` | `/api/auth/logout` | — | `204` |

The e-mail link points to the SPA route `/connexion/verifier?token=...`, which calls `POST /api/auth/verify`.
The token is never consumed by a GET, so mail scanners that prefetch links cannot burn it.

## Me

`GET /api/me` → `200 Me` or `401`.

```json
{
  "id": "0f8c…",
  "email": "marie@acme.fr",
  "name": "Marie",
  "lang": "fr",
  "orgRole": "member",
  "canTriage": false,
  "projects": [
    {
      "slug": "online-shop",
      "name": "Boutique en ligne",
      "hint": "Le site marchand",
      "color": "#B9AEFF",
      "vehicle": "boat",
      "role": "reporter",
      "canCreate": true,
      "inbox": false
    }
  ]
}
```

`projects` lists only the projects the user may at least view. `orgRole` is `owner`, `admin` or `member`.
`vehicle` is one of `boat`, `submarine`, `train`, `plane`, `rocket`, `car`, `bus`.
`inbox: true` marks the shared intake project shown as "Je ne sais pas" in the request flow.
`canTriage` is true when the user may triage at least one project (shows the triage entry).

## Preferences

| Method | Path | Body | Answer |
|---|---|---|---|
| `GET` | `/api/me/preferences` | — | `200 {"notifyStatus": true}` or `401` |
| `PATCH` | `/api/me/preferences` | `{"notifyStatus": false}` | `200` with the stored preferences; `422` with `fields.notifyStatus` when missing; `401` |

`notifyStatus` (default `true`): e-mail the user when one of their requests changes status. The e-mail is sent
once per request and status, in the user's language, only while the request still has that status and the user
may still see it (same rule as `GET /api/requests/{id}`). It carries the request title and a link to
`/demandes/{id}`, never its content, and a link to the SPA route `/preferences`.

## Request forms

`GET /api/projects/{slug}/forms` → `200 {"forms": [Form]}` (only when the user may create requests there).

The forms come from the repository's `.github/ISSUE_TEMPLATE/*.yml` files, mapped to the three request kinds.

```json
{
  "kind": "problem",
  "template": "bug.yml",
  "name": "Signaler un problème",
  "description": "…",
  "fields": [
    {"id": "what", "type": "textarea", "label": "Ce qui s'est passé", "description": "…", "placeholder": "…", "required": false},
    {"id": "where", "type": "input", "label": "Où ?", "required": true},
    {"id": "severity", "type": "dropdown", "label": "Gravité", "options": ["Bloquant", "Gênant"], "multiple": false, "required": true},
    {"id": "consent", "type": "checkboxes", "label": "…", "options": ["J'ai vérifié"], "required": false},
    {"type": "markdown", "value": "Texte d'aide en Markdown"}
  ]
}
```

`kind` is `problem`, `idea` or `question`. A kind without a form is absent from the list.

## Attachments

| Method | Path | Body | Answer |
|---|---|---|---|
| `POST` | `/api/attachments` | multipart, field `file` | `201 Attachment` |
| `GET` | `/api/attachments/{id}` | — | the file (only to its author, or to anyone who may see the linked request) |
| `DELETE` | `/api/attachments/{id}` | — | `204`, only for the author and only while not yet linked to a request |

```json
{"id": "6b1e…", "name": "capture.png", "contentType": "image/png", "size": 48213, "url": "/api/attachments/6b1e…"}
```

Accepted types (checked on the content, not the extension): PNG, JPEG, GIF, WebP, PDF, plain text. Maximum 10 MB by default.

Issue bodies link each attachment as `{base_url}/files/{id}`, outside `/api` so it opens in a
browser tab. It follows the same rules as `GET /api/attachments/{id}`. Signed out, it answers
`302` to `/connexion?next=/files/{id}`, and the person comes back to the file after signing in.
Attachments are plain links in the issue, never inline images: GitHub fetches images without
the reader's Hublot session.

## Requests

### Create

`POST /api/requests`

```json
{
  "project": "online-shop",
  "kind": "problem",
  "title": "Le rapport PDF est illisible",
  "fields": {"what": "…", "severity": "Gênant", "consent": ["J'ai vérifié"]},
  "attachments": ["6b1e…"],
  "private": false
}
```

→ `201 Request`. `422 validation` with `fields: {"title": "required", "where": "required"}` when something is missing.

### List and read

- `GET /api/requests?scope=mine` → `200 {"requests": [Request]}`: the user's own requests, newest first.
- `GET /api/requests?scope=all&project=online-shop` → every request of that project the user may see.
- `GET /api/requests/{id}` → `200 Request` or `404`.

Both lists take optional filters, combined with AND: `kind` (`problem`, `idea`, `question`), `status`
(one of the statuses below) and, for `scope=mine`, `project` (a slug). A wrong value answers
`422 validation` with `fields: {"kind": "invalid"}`. For `scope=mine`, a project the user may not see
answers exactly like one that does not exist (`422`); for `scope=all` both answer `404`. Filters only
narrow what the user may already see.

```json
{
  "id": "a3c2…",
  "project": {"slug": "online-shop", "name": "Boutique en ligne", "color": "#B9AEFF", "vehicle": "boat"},
  "kind": "problem",
  "title": "Le rapport PDF est illisible",
  "status": "received",
  "private": false,
  "number": 128,
  "createdAt": "2026-10-27T09:12:00Z",
  "author": {"name": "Marie"},
  "attachments": [Attachment],
  "milestone": "Novembre",
  "history": [
    {"status": "received", "at": "2026-10-27T09:12:00Z"},
    {"status": "planned", "at": "2026-10-28T14:03:00Z"}
  ]
}
```

`status` is one of `received`, `planned`, `in_progress`, `in_test`, `available`, `answered`, `closed`,
computed from the GitHub issue each time it changes, first match wins:

1. closed as completed → `answered` for a question, `available` otherwise; closed for any other reason
   (not planned, duplicate) or deleted → `closed`;
2. open, and the project maps the option of its GitHub Project status field (`status` in the project
   configuration) → that status;
3. open with a milestone → `planned`;
4. open → `received`.

`history` (detail only) lists the status changes, oldest first, starting with `received` at filing.
`milestone` (detail only, absent when none) is the version the request is planned for, the GitHub
milestone title.
`number` is the GitHub issue number, shown small, only to talk to a developer. After a transfer in
GitHub, the request follows its issue: `project` and `number` are the new ones.

### Home: what moved since the last visit

`GET /api/home` → `200 {"moved": [Request]}`: the user's own requests whose status changed since their
previous visit, that they may still see, the latest change first, at most 5. Each carries `movedAt`, when
its status last changed. Calling it records the visit: a visit starts when the user opens the home after
30 minutes away and looks back to the last time they opened it before; reloads within a visit answer the
same list. The very first visit answers an empty list.

## Triage

Triage rights: `triager` or `manager` on the request's project, or org `owner`/`admin`. The triage
state lives in GitHub: a new request carries the label `portal:new` until it is accepted or closed.

`GET /api/triage` → `200`:

```json
{
  "requests": [Request],
  "projects": [{"slug": "online-shop", "name": "Boutique en ligne", "color": "#B9AEFF", "vehicle": "boat"}]
}
```

`requests` are the open requests labelled `portal:new` of every project the user may triage, the inbox
project first, then newest first (no `history`, empty `attachments`). `projects` are the projects the
user may triage: the filters and the transfer destinations. Nobody to triage for is an empty answer,
never an error. An org admin reading a project's inbox without a grant on it is audited (`triage.list`).

`POST /api/triage/{id}/{action}` → `200 Request` as the user now sees it (`204` if they no longer may),
`404` when the user cannot see the request, `403` when they see it but may not triage it,
`422 validation` for a bad body. Each action is written to GitHub first, then applied to the cache the
way a webhook would be, and audited (`triage.<action>`, with `via_org_role` for admins without a grant).

| Action | Body | Effect in GitHub | Leaves the inbox |
|---|---|---|---|
| `accept` | none | removes `portal:new` | yes |
| `kind` | `{"kind": "idea"}` | swaps the old kind's form labels for the new one's; sets the issue type from `issue_types` (config), else the new form's `type`, else leaves it; updates the request's kind | no |
| `transfer` | `{"project": "mobile-app"}` | transfers the issue to the destination repository (labels kept); the request follows, its author keeps seeing it. Needs triage rights on both projects (`403`); an unknown or hidden destination, or the same project, is `422` | no |
| `private` | `{"private": true}` | adds or removes `portal:private` | no |
| `close` | `{"reason": "not_planned"}` or `"duplicate"` | closes the issue with that reason, removes `portal:new` | yes |

## GitHub webhook

`POST /webhooks/github`, outside `/api`: no session, no `X-Hublot` header. Configure it on the GitHub App
with the secret `forge.github.webhook_secret` and the events `issues` and `projects_v2_item`.

- `X-Hub-Signature-256` must be the HMAC-SHA256 of the raw body with the secret, or `401`.
- `X-GitHub-Event` and `X-GitHub-Delivery` are required (`400` otherwise, or for a malformed payload).
  Bodies above 5 MB answer `413`.
- `202 {"status": "queued"}`: the delivery id is recorded and its job queued in the same transaction;
  the cache and statuses are updated in the background.
- `200 {"status": "duplicate"}`: a delivery id already received (a redelivery); nothing is queued.
- `204`: an event the portal does not follow (`ping`, pull requests, Projects items that are not issues).

Only issues linked to a request are cached. A periodic resync (`sync.resync_interval`, 1 hour by default)
and `hublot resync` repair what a lost delivery missed.

## Development: fake forge

Only when `forge.driver` is `fake` (never mounted with `github`), act as someone working in GitHub:

`POST /dev/fake/issues/{owner}/{repo}/{number}` with any of

```json
{"state": "closed", "state_reason": "completed", "labels": ["portal", "bug"], "milestone": "v2.1", "project_status": "In progress"}
```

changes the in-memory issue and queues the same job a webhook would → `202` with the issue, `404` for an
issue the fake forge does not hold (it forgets them on restart). `milestone: ""` removes the milestone.
