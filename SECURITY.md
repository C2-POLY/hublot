# Security policy

Hublot holds wide GitHub permissions on behalf of people who have only the access they were
given. A flaw that lets someone see a project, a request or an attachment they were not
granted is treated as critical.

## Reporting

Please do not open a public issue. E-mail **security@et-technology-consulting.com** with
the steps to reproduce and the affected version. You will get an answer within 3 business
days. Fixes are released before the details are published.

## What the portal does

- Authorization is checked on the server at every request (`internal/authz`), deny by default;
  an isolation test suite guards it. Hidden resources answer 404, not 403.
- Sign-in links are single-use, valid 15 minutes, stored hashed, rate limited per address
  and per IP. Sessions are random tokens stored hashed, in `HttpOnly`, `SameSite=Lax` cookies.
  State-changing API calls require an `X-Hublot` header (CSRF guard).
- Attachments: the type is detected from the content (PNG, JPEG, GIF, WebP, PDF, plain text;
  never SVG or HTML), size is capped, files are served with a sandboxing CSP and `nosniff`.
- Attachment links written in GitHub issues are HMAC-signed per file, so developers can
  open them without a portal account. Anyone holding such a link can open that one file:
  treat issue bodies accordingly. With S3 storage the link redirects to a 5-minute URL.
- The GitHub App private key and every secret come from the environment or mounted files,
  never from the repository.
- Role, group and grant changes, and consultations by admins without a project role, are
  written to the audit log.
