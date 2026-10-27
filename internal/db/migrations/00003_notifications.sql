-- SPDX-License-Identifier: AGPL-3.0-or-later
-- Milestone 3: authors hear by e-mail when their requests move on.

-- +goose Up
-- Whether the person wants an e-mail when one of their requests changes status.
ALTER TABLE app_user ADD COLUMN notify_status boolean NOT NULL DEFAULT true;

-- Status e-mails already sent: one per (request, status), even when a job is retried.
CREATE TABLE request_notification (
    request_id uuid NOT NULL REFERENCES request (id) ON DELETE CASCADE,
    status     text NOT NULL,
    sent_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (request_id, status)
);

-- +goose Down
DROP TABLE request_notification;
ALTER TABLE app_user DROP COLUMN notify_status;
