-- SPDX-License-Identifier: AGPL-3.0-or-later
-- 0.2.0 follow-up screens: the home shows what moved since the person's last visit.

-- +goose Up
-- last_seen_at: the last time the person opened the home.
-- previous_visit_at: where the current visit looks back to, the last_seen_at of the
-- previous visit; it moves on once the person has been away long enough (portal.VisitGap).
ALTER TABLE app_user
    ADD COLUMN last_seen_at      timestamptz,
    ADD COLUMN previous_visit_at timestamptz;

-- +goose Down
ALTER TABLE app_user DROP COLUMN previous_visit_at, DROP COLUMN last_seen_at;
