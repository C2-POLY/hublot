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
INSERT INTO request (id, project_id, origin_project_id, author_id, kind, repo_owner, repo_name, issue_number, issue_node_id, status)
VALUES (@id, @project_id, @project_id, @author_id, @kind, @repo_owner, @repo_name, @issue_number, @issue_node_id, @status)
RETURNING *;

-- name: UpsertIssueCache :exec
-- An older snapshot (a webhook delivered late) never overwrites a newer one.
-- The project status is kept: it has its own query.
INSERT INTO issue_cache (repo_owner, repo_name, issue_number, node_id, title, state, state_reason, labels,
                         milestone, issue_type, closed_at, issue_updated_at, updated_at)
VALUES (@repo_owner, @repo_name, @issue_number, @node_id, @title, @state, @state_reason, @labels,
        @milestone, @issue_type, @closed_at, @issue_updated_at, now())
ON CONFLICT (repo_owner, repo_name, issue_number) DO UPDATE SET
    node_id = EXCLUDED.node_id, title = EXCLUDED.title, state = EXCLUDED.state,
    state_reason = EXCLUDED.state_reason, labels = EXCLUDED.labels, milestone = EXCLUDED.milestone,
    issue_type = EXCLUDED.issue_type, closed_at = EXCLUDED.closed_at,
    issue_updated_at = EXCLUDED.issue_updated_at, deleted_at = NULL, updated_at = now()
WHERE issue_cache.issue_updated_at IS NULL OR EXCLUDED.issue_updated_at IS NULL
   OR EXCLUDED.issue_updated_at >= issue_cache.issue_updated_at;

-- name: SetIssueProjectStatus :exec
UPDATE issue_cache SET project_status = @project_status, updated_at = now()
WHERE repo_owner = @repo_owner AND repo_name = @repo_name AND issue_number = @issue_number;

-- name: MarkIssueDeleted :exec
UPDATE issue_cache SET deleted_at = now(), updated_at = now()
WHERE repo_owner = $1 AND repo_name = $2 AND issue_number = $3;

-- name: DeleteIssueCache :exec
DELETE FROM issue_cache WHERE repo_owner = $1 AND repo_name = $2 AND issue_number = $3;

-- name: GetIssueCache :one
SELECT * FROM issue_cache WHERE repo_owner = $1 AND repo_name = $2 AND issue_number = $3;

-- name: FindRequestByIssue :one
SELECT * FROM request WHERE repo_owner = $1 AND repo_name = $2 AND issue_number = $3;

-- name: FindRequestByNodeID :one
SELECT * FROM request WHERE issue_node_id = $1 AND issue_node_id <> '' LIMIT 1;

-- name: FindRequestByID :one
SELECT * FROM request WHERE id = $1;

-- name: LockRequest :one
-- Serialises the updates of one request: two events about it never interleave.
SELECT * FROM request WHERE id = $1 FOR UPDATE;

-- name: MoveRequest :exec
-- After a transfer. origin_project_id never changes: the author keeps seeing it.
UPDATE request SET project_id = @project_id, repo_owner = @repo_owner, repo_name = @repo_name,
    issue_number = @issue_number, issue_node_id = @issue_node_id
WHERE id = @id;

-- name: SetRequestStatus :exec
UPDATE request SET status = $2 WHERE id = $1;

-- name: InsertRequestStatus :exec
INSERT INTO request_status (request_id, status) VALUES ($1, $2);

-- name: ListRequestStatus :many
SELECT status, at FROM request_status WHERE request_id = $1 ORDER BY at, id;

-- name: RecordWebhookDelivery :execrows
INSERT INTO webhook_delivery (delivery_id, event) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: PruneWebhookDeliveries :exec
DELETE FROM webhook_delivery WHERE received_at < $1;

-- name: SetProjectSynced :exec
UPDATE project SET synced_at = $2 WHERE id = $1;

-- name: GetRequest :one
SELECT r.*, c.title, c.state, c.state_reason, c.labels, c.milestone, u.name AS author_name
FROM request r
JOIN issue_cache c ON c.repo_owner = r.repo_owner AND c.repo_name = r.repo_name AND c.issue_number = r.issue_number
JOIN app_user u ON u.id = r.author_id
WHERE r.id = $1;

-- name: ListRequestsByAuthor :many
SELECT r.*, c.title, c.state, c.state_reason, c.labels, c.milestone, u.name AS author_name
FROM request r
JOIN issue_cache c ON c.repo_owner = r.repo_owner AND c.repo_name = r.repo_name AND c.issue_number = r.issue_number
JOIN app_user u ON u.id = r.author_id
WHERE r.author_id = $1
ORDER BY r.created_at DESC
LIMIT 200;

-- name: ListRequestsByProject :many
SELECT r.*, c.title, c.state, c.state_reason, c.labels, c.milestone, u.name AS author_name
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

-- name: SetUserNotifyStatus :one
UPDATE app_user SET notify_status = $2 WHERE id = $1 RETURNING notify_status;

-- name: NotificationSent :one
SELECT EXISTS (SELECT 1 FROM request_notification WHERE request_id = $1 AND status = $2);

-- name: RecordNotification :exec
INSERT INTO request_notification (request_id, status) VALUES ($1, $2) ON CONFLICT DO NOTHING;
