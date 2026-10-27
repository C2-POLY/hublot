-- SPDX-License-Identifier: AGPL-3.0-or-later
-- The portal stores only what GitHub cannot carry: its users, their access,
-- attachments, the person <-> issue link and a disposable cache (PRD §7).

-- +goose Up
CREATE TYPE org_role AS ENUM ('member', 'admin', 'owner');
-- Declaration order matters: max(role) gives the effective role (PRD §5 bis).
CREATE TYPE project_role AS ENUM ('viewer', 'reporter', 'contributor', 'triager', 'manager');

CREATE TABLE app_user (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL UNIQUE CHECK (email = lower(email)),
    name          text NOT NULL DEFAULT '',
    org_role      org_role NOT NULL DEFAULT 'member',
    lang          text NOT NULL DEFAULT 'fr',
    suspended     boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);

CREATE TABLE user_group (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE group_member (
    group_id uuid NOT NULL REFERENCES user_group (id) ON DELETE CASCADE,
    user_id  uuid NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE project (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              text NOT NULL UNIQUE,
    name              text NOT NULL,
    hint              text NOT NULL DEFAULT '',
    color             text NOT NULL,
    vehicle           text NOT NULL,
    repo_owner        text NOT NULL,
    repo_name         text NOT NULL,
    reporters_see_all boolean NOT NULL DEFAULT false,
    inbox             boolean NOT NULL DEFAULT false,
    position          integer NOT NULL DEFAULT 0,
    archived          boolean NOT NULL DEFAULT false
);

CREATE TABLE project_grant (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES project (id) ON DELETE CASCADE,
    user_id    uuid REFERENCES app_user (id) ON DELETE CASCADE,
    group_id   uuid REFERENCES user_group (id) ON DELETE CASCADE,
    role       project_role NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((user_id IS NULL) <> (group_id IS NULL))
);
CREATE UNIQUE INDEX project_grant_user ON project_grant (project_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX project_grant_group ON project_grant (project_id, group_id) WHERE group_id IS NOT NULL;

CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    actor_id    uuid REFERENCES app_user (id) ON DELETE SET NULL,
    via         text NOT NULL DEFAULT 'portal',
    action      text NOT NULL,
    target_type text NOT NULL,
    target_id   text NOT NULL,
    details     jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX audit_log_at ON audit_log (at DESC);

CREATE TABLE login_token (
    token_hash bytea PRIMARY KEY,
    email      text NOT NULL,
    lang       text NOT NULL DEFAULT 'fr',
    ip         text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz
);
CREATE INDEX login_token_email ON login_token (email, created_at DESC);
CREATE INDEX login_token_ip ON login_token (ip, created_at DESC);

CREATE TABLE user_session (
    token_hash   bytea PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_session_user ON user_session (user_id);

CREATE TABLE request (
    id                uuid PRIMARY KEY,
    project_id        uuid NOT NULL REFERENCES project (id),
    -- The project it was filed in: its author keeps seeing it after a transfer.
    origin_project_id uuid NOT NULL REFERENCES project (id),
    author_id         uuid NOT NULL REFERENCES app_user (id),
    kind              text NOT NULL,
    repo_owner        text NOT NULL,
    repo_name         text NOT NULL,
    issue_number      integer NOT NULL,
    issue_node_id     text NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (repo_owner, repo_name, issue_number)
);
CREATE INDEX request_author ON request (author_id, created_at DESC);
CREATE INDEX request_project ON request (project_id, created_at DESC);

CREATE TABLE attachment (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id    uuid NOT NULL REFERENCES app_user (id),
    request_id   uuid REFERENCES request (id) ON DELETE SET NULL,
    object_key   text NOT NULL UNIQUE,
    name         text NOT NULL,
    content_type text NOT NULL,
    size         bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX attachment_request ON attachment (request_id);

-- Disposable projection of GitHub issues; rebuilt from GitHub at will.
CREATE TABLE issue_cache (
    repo_owner   text NOT NULL,
    repo_name    text NOT NULL,
    issue_number integer NOT NULL,
    title        text NOT NULL,
    state        text NOT NULL,
    state_reason text NOT NULL DEFAULT '',
    labels       text[] NOT NULL DEFAULT '{}',
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (repo_owner, repo_name, issue_number)
);

-- +goose Down
DROP TABLE issue_cache;
DROP TABLE attachment;
DROP TABLE request;
DROP TABLE user_session;
DROP TABLE login_token;
DROP TABLE audit_log;
DROP TABLE project_grant;
DROP TABLE project;
DROP TABLE group_member;
DROP TABLE user_group;
DROP TABLE app_user;
DROP TYPE project_role;
DROP TYPE org_role;
