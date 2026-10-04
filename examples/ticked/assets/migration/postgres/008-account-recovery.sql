-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
ALTER TABLE users ADD COLUMN mailbox_verified_at TIMESTAMPTZ;
CREATE TABLE mailbox_tokens (
 id TEXT PRIMARY KEY CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 purpose SMALLINT NOT NULL CHECK (purpose IN (1,2)),
 target TEXT NOT NULL CHECK (octet_length(target) BETWEEN 1 AND 254 AND target !~ '[[:cntrl:]]'),
 auth_version BIGINT NOT NULL CHECK (auth_version > 0),
 policy_revision TEXT NOT NULL CHECK (octet_length(policy_revision) BETWEEN 1 AND 128 AND policy_revision !~ '[^!-~]'),
 secret_digest BYTEA NOT NULL CHECK (octet_length(secret_digest)=32),
 created_at TIMESTAMPTZ NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0,
 attempt_limit INTEGER NOT NULL CHECK (attempt_limit BETWEEN 1 AND 10),
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
 lease_until TIMESTAMPTZ,
 consumed_at TIMESTAMPTZ,
 revoked_at TIMESTAMPTZ,
 UNIQUE(user_id,purpose),
 CHECK (attempts BETWEEN 0 AND attempt_limit),
 CHECK (expires_at-created_at BETWEEN INTERVAL '1 minute' AND INTERVAL '24 hours'),
 CHECK (purpose<>2 OR expires_at-created_at<=INTERVAL '1 hour'),
 CHECK (consumed_at IS NULL OR revoked_at IS NULL),
 CHECK (lease_until IS NULL OR lease_until>created_at),
 CHECK (consumed_at IS NULL OR consumed_at>=created_at),
 CHECK (revoked_at IS NULL OR revoked_at>=created_at)
);
CREATE INDEX mailbox_tokens_retirement ON mailbox_tokens ((COALESCE(consumed_at,revoked_at,expires_at)),id);
CREATE TABLE recovery_budgets (
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind SMALLINT NOT NULL CHECK (kind BETWEEN 1 AND 3),
 window_start TIMESTAMPTZ NOT NULL,
 attempts INTEGER NOT NULL CHECK (attempts BETWEEN 0 AND 20),
 PRIMARY KEY(user_id,kind)
);
CREATE TABLE recovery_notices (
 id TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 kind SMALLINT NOT NULL CHECK (kind BETWEEN 1 AND 3),
 destination TEXT NOT NULL CHECK (octet_length(destination) BETWEEN 1 AND 254 AND destination !~ '[[:cntrl:]]'),
 created_at TIMESTAMPTZ NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
 delivered_at TIMESTAMPTZ,
 UNIQUE(user_id,kind,created_at)
);
CREATE INDEX recovery_notices_subject ON recovery_notices(user_id,created_at,id);

-- +migrate Down
DROP TABLE recovery_notices;
DROP TABLE recovery_budgets;
DROP TABLE mailbox_tokens;
ALTER TABLE users DROP COLUMN mailbox_verified_at;
