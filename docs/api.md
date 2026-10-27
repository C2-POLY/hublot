# Portal HTTP API (milestone 1)

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
  "attachments": [Attachment]
}
```

`status` is one of `received`, `planned`, `in_progress`, `in_test`, `available`, `answered`, `closed`.
Milestone 1 only produces `received` (open) and `closed`; the rest arrive with webhooks in milestone 2.
`number` is the GitHub issue number, shown small, only to talk to a developer.
