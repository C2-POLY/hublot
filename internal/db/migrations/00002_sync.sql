-- SPDX-License-Identifier: AGPL-3.0-or-later
-- Milestone 2: the cache follows GitHub through webhooks and periodic resyncs,
-- and each request keeps the history of its portal status.

-- +goose Up
ALTER TABLE issue_cache
    ADD COLUMN node_id          text NOT NULL DEFAULT '',
    ADD COLUMN milestone        text NOT NULL DEFAULT '',
    ADD COLUMN issue_type       text NOT NULL DEFAULT '',
    ADD COLUMN project_status   text NOT NULL DEFAULT '',
    ADD COLUMN closed_at        timestamptz,
    -- GitHub's own updated_at: a webhook delivered late never overwrites a newer state.
    ADD COLUMN issue_updated_at timestamptz,
    -- Set when the issue was deleted in GitHub; the request then reads as closed.
    ADD COLUMN deleted_at       timestamptz;

-- The last computed status; request_status keeps how it got there.
ALTER TABLE request ADD COLUMN status text NOT NULL DEFAULT 'received';
CREATE INDEX request_issue_node ON request (issue_node_id);

-- When the last resync of the project's repository started; NULL means never (full resync).
ALTER TABLE project ADD COLUMN synced_at timestamptz;

-- GitHub delivery ids already accepted, so a redelivery is processed once.
CREATE TABLE webhook_delivery (
    delivery_id text PRIMARY KEY,
    event       text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX webhook_delivery_received ON webhook_delivery (received_at);

-- Status history: the request timeline, and one notification per (request, status).
CREATE TABLE request_status (
    id         bigserial PRIMARY KEY,
    request_id uuid NOT NULL REFERENCES request (id) ON DELETE CASCADE,
    status     text NOT NULL,
    at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX request_status_request ON request_status (request_id, at);

-- Requests filed before this migration start their history on their filing date;
-- the first resync computes their real status.
INSERT INTO request_status (request_id, status, at) SELECT id, 'received', created_at FROM request;

-- +goose Down
DROP TABLE request_status;
DROP TABLE webhook_delivery;
ALTER TABLE project DROP COLUMN synced_at;
DROP INDEX request_issue_node;
ALTER TABLE request DROP COLUMN status;
ALTER TABLE issue_cache
    DROP COLUMN node_id,
    DROP COLUMN milestone,
    DROP COLUMN issue_type,
    DROP COLUMN project_status,
    DROP COLUMN closed_at,
    DROP COLUMN issue_updated_at,
    DROP COLUMN deleted_at;
