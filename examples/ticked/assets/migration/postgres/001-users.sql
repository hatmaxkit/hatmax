-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    auth_version BIGINT NOT NULL DEFAULT 1 CHECK (auth_version > 0),
    roles TEXT[] DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_digest BYTEA UNIQUE NOT NULL CHECK (octet_length(token_digest) = 32),
    auth_version BIGINT NOT NULL CHECK (auth_version > 0),
    policy_revision TEXT NOT NULL CHECK (octet_length(policy_revision) BETWEEN 1 AND 128 AND policy_revision !~ '[^ -~]'),
    generation BIGINT NOT NULL CHECK (generation > 0),
    proof_method SMALLINT NOT NULL CHECK (proof_method = 1),
    proof_verified_at TIMESTAMPTZ NOT NULL,
    authenticated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    last_activity_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    inactivity_us BIGINT NOT NULL CHECK (inactivity_us BETWEEN 60000000 AND 2592000000000),
    CHECK (created_at <= proof_verified_at AND proof_verified_at <= authenticated_at),
    CHECK (created_at <= authenticated_at AND authenticated_at <= last_activity_at AND last_activity_at < expires_at),
    CHECK (expires_at - authenticated_at BETWEEN INTERVAL '1 minute' AND INTERVAL '30 days'),
    CHECK (inactivity_us <= EXTRACT(EPOCH FROM (expires_at - authenticated_at)) * 1000000)
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id, id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at, id);

-- +migrate Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
