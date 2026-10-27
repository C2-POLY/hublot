-- SPDX-License-Identifier: AGPL-3.0-or-later
-- 0.2.0 review: a daily full resync, and indexes behind the status filters.

-- +goose Up
-- When the last full resync of the project's repository started; NULL means never.
-- Only a full resync sees a Project field change whose webhook delivery was lost.
ALTER TABLE project ADD COLUMN full_synced_at timestamptz;

-- The request lists filter by status within an author's or a project's requests.
CREATE INDEX request_author_status ON request (author_id, status, created_at DESC);
CREATE INDEX request_project_status ON request (project_id, status, created_at DESC);

-- +goose Down
DROP INDEX request_project_status;
DROP INDEX request_author_status;
ALTER TABLE project DROP COLUMN full_synced_at;
