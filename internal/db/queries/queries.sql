-- SPDX-License-Identifier: AGPL-3.0-or-later

-- name: GetUser :one
SELECT * FROM app_user WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM app_user WHERE email = $1;

-- name: UpsertUserOnLogin :one
INSERT INTO app_user (email, name, org_role, lang, last_login_at)
VALUES (@email, @name, @org_role, @lang, now())
ON CONFLICT (email) DO UPDATE
SET last_login_at = now(),
    org_role = CASE WHEN EXCLUDED.org_role = 'owner' THEN 'owner'::org_role ELSE app_user.org_role END
RETURNING *;

-- name: CreateUser :one
INSERT INTO app_user (email, name, org_role, lang)
VALUES (@email, @name, @org_role, @lang)
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name, org_role = EXCLUDED.org_role
RETURNING *;

-- name: CountLoginTokensByEmailSince :one
SELECT count(*) FROM login_token WHERE email = $1 AND created_at > $2;

-- name: CountLoginTokensByIPSince :one
SELECT count(*) FROM login_token WHERE ip = $1 AND created_at > $2;

-- name: CreateLoginToken :exec
INSERT INTO login_token (token_hash, email, lang, ip, expires_at) VALUES ($1, $2, $3, $4, $5);

-- name: ConsumeLoginToken :one
UPDATE login_token SET used_at = now()
WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
RETURNING email, lang;

-- name: CreateSession :exec
INSERT INTO user_session (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUser :one
SELECT u.* FROM user_session s JOIN app_user u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now() AND NOT u.suspended;

-- name: TouchSession :exec
UPDATE user_session SET last_seen_at = now() WHERE token_hash = $1;

-- name: DeleteSession :exec
DELETE FROM user_session WHERE token_hash = $1;

-- name: UpsertProject :one
INSERT INTO project (slug, name, hint, color, vehicle, repo_owner, repo_name, reporters_see_all, inbox, position, archived)
VALUES (@slug, @name, @hint, @color, @vehicle, @repo_owner, @repo_name, @reporters_see_all, @inbox, @position, false)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name, hint = EXCLUDED.hint, color = EXCLUDED.color, vehicle = EXCLUDED.vehicle,
    repo_owner = EXCLUDED.repo_owner, repo_name = EXCLUDED.repo_name,
    reporters_see_all = EXCLUDED.reporters_see_all, inbox = EXCLUDED.inbox,
    position = EXCLUDED.position, archived = false
RETURNING *;

-- name: ArchiveProjectsNotIn :exec
UPDATE project SET archived = true WHERE NOT (slug = ANY(@slugs::text[]));

-- name: ListProjects :many
SELECT * FROM project WHERE NOT archived ORDER BY position, name;

-- name: ListAllProjects :many
SELECT * FROM project ORDER BY position, name;

-- name: GetProjectBySlug :one
SELECT * FROM project WHERE slug = $1 AND NOT archived;

-- name: EffectiveRoles :many
-- Highest role per project, direct or through a group.
SELECT g.project_id, max(g.role)::project_role AS role
FROM project_grant g
WHERE g.user_id = @user_id::uuid
   OR g.group_id IN (SELECT m.group_id FROM group_member m WHERE m.user_id = @user_id::uuid)
GROUP BY g.project_id;

-- name: UpsertUserGrant :one
INSERT INTO project_grant (project_id, user_id, role) VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) WHERE user_id IS NOT NULL DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: UpsertGroupGrant :one
INSERT INTO project_grant (project_id, group_id, role) VALUES ($1, $2, $3)
ON CONFLICT (project_id, group_id) WHERE group_id IS NOT NULL DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: CreateGroup :one
INSERT INTO user_group (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GetGroupByName :one
SELECT * FROM user_group WHERE name = $1;

-- name: AddGroupMember :exec
INSERT INTO group_member (group_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: InsertAudit :exec
INSERT INTO audit_log (actor_id, via, action, target_type, target_id, details)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CreateRequest :one
INSERT INTO request (id, project_id, origin_project_id, author_id, kind, repo_owner, repo_name, issue_number, issue_node_id)
VALUES (@id, @project_id, @project_id, @author_id, @kind, @repo_owner, @repo_name, @issue_number, @issue_node_id)
RETURNING *;

-- name: UpsertIssueCache :exec
INSERT INTO issue_cache (repo_owner, repo_name, issue_number, title, state, state_reason, labels, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (repo_owner, repo_name, issue_number) DO UPDATE SET
    title = EXCLUDED.title, state = EXCLUDED.state, state_reason = EXCLUDED.state_reason,
    labels = EXCLUDED.labels, updated_at = now();

-- name: GetRequest :one
SELECT r.*, c.title, c.state, c.state_reason, c.labels, u.name AS author_name
FROM request r
JOIN issue_cache c ON c.repo_owner = r.repo_owner AND c.repo_name = r.repo_name AND c.issue_number = r.issue_number
JOIN app_user u ON u.id = r.author_id
WHERE r.id = $1;

-- name: ListRequestsByAuthor :many
SELECT r.*, c.title, c.state, c.state_reason, c.labels, u.name AS author_name
FROM request r
JOIN issue_cache c ON c.repo_owner = r.repo_owner AND c.repo_name = r.repo_name AND c.issue_number = r.issue_number
JOIN app_user u ON u.id = r.author_id
WHERE r.author_id = $1
ORDER BY r.created_at DESC
LIMIT 200;

-- name: ListRequestsByProject :many
SELECT r.*, c.title, c.state, c.state_reason, c.labels, u.name AS author_name
FROM request r
JOIN issue_cache c ON c.repo_owner = r.repo_owner AND c.repo_name = r.repo_name AND c.issue_number = r.issue_number
JOIN app_user u ON u.id = r.author_id
WHERE r.project_id = $1
ORDER BY r.created_at DESC
LIMIT 200;

-- name: CreateAttachment :one
INSERT INTO attachment (author_id, object_key, name, content_type, size)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAttachment :one
SELECT * FROM attachment WHERE id = $1;

-- name: ListRequestAttachments :many
SELECT * FROM attachment WHERE request_id = $1 ORDER BY created_at;

-- name: LinkAttachment :execrows
UPDATE attachment SET request_id = @request_id
WHERE id = @id AND author_id = @author_id AND request_id IS NULL;

-- name: DeleteUnlinkedAttachment :one
DELETE FROM attachment WHERE id = $1 AND author_id = $2 AND request_id IS NULL
RETURNING object_key;

-- name: MaxIssueNumber :one
-- The fake forge continues from here after a restart, so numbers never collide.
SELECT COALESCE(max(issue_number), 0)::int AS number FROM request WHERE repo_owner = $1 AND repo_name = $2;
