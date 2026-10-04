-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
CREATE TABLE IF NOT EXISTS webauthn_subjects (
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 rp_id TEXT NOT NULL CHECK (octet_length(rp_id) BETWEEN 1 AND 253),
 handle BYTEA NOT NULL CHECK (octet_length(handle) = 32),
 PRIMARY KEY (user_id, rp_id), UNIQUE (rp_id, handle)
);

CREATE TABLE IF NOT EXISTS authenticators (
 id TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 rp_id TEXT NOT NULL,
 kind SMALLINT NOT NULL CHECK (kind = 1),
 record_version SMALLINT NOT NULL CHECK (record_version = 1),
 credential_id BYTEA NOT NULL CHECK (octet_length(credential_id) BETWEEN 1 AND 1024),
 public_key BYTEA NOT NULL CHECK (octet_length(public_key) BETWEEN 1 AND 4096),
 credential_data BYTEA NOT NULL CHECK (octet_length(credential_data) BETWEEN 1 AND 65536),
 sign_count BIGINT NOT NULL CHECK (sign_count BETWEEN 0 AND 4294967295),
 backup_eligible BOOLEAN NOT NULL,
 backup_state BOOLEAN NOT NULL,
 user_verified BOOLEAN NOT NULL CHECK (user_verified),
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at TIMESTAMPTZ NOT NULL,
 CHECK (NOT backup_state OR backup_eligible),
 FOREIGN KEY (user_id,rp_id) REFERENCES webauthn_subjects(user_id,rp_id),
 UNIQUE (rp_id, credential_id)
);
CREATE INDEX IF NOT EXISTS idx_authenticators_subject ON authenticators(user_id,id);

CREATE TABLE IF NOT EXISTS auth_pending (
 digest BYTEA PRIMARY KEY CHECK (octet_length(digest) = 32),
 purpose SMALLINT NOT NULL CHECK (purpose = 1),
 record_version SMALLINT NOT NULL CHECK (record_version = 1),
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 auth_version BIGINT NOT NULL CHECK (auth_version > 0),
 rp_id TEXT NOT NULL,
 user_handle BYTEA NOT NULL CHECK (octet_length(user_handle) = 32),
 rp_binding BYTEA NOT NULL CHECK (octet_length(rp_binding) = 32),
 ceremony_data BYTEA NOT NULL CHECK (octet_length(ceremony_data) BETWEEN 1 AND 16384),
 required_proof SMALLINT NOT NULL CHECK (required_proof BETWEEN 1 AND 3),
 policy_revision TEXT NOT NULL CHECK (octet_length(policy_revision) BETWEEN 1 AND 128 AND policy_revision !~ '[^ -~]'),
 max_age_us BIGINT NOT NULL CHECK (max_age_us = 0 OR max_age_us BETWEEN 1000000 AND 600000000),
 password_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 10),
 revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0),
 lease_until TIMESTAMPTZ,
 CHECK (password_at <= created_at AND expires_at - created_at BETWEEN INTERVAL '1 minute' AND INTERVAL '10 minutes'),
 FOREIGN KEY (user_id,rp_id) REFERENCES webauthn_subjects(user_id,rp_id)
);
CREATE INDEX IF NOT EXISTS idx_auth_pending_subject ON auth_pending(user_id,digest);
CREATE INDEX IF NOT EXISTS idx_auth_pending_expiry ON auth_pending(expires_at,digest);

CREATE TABLE IF NOT EXISTS auth_factor_budgets (
 user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 window_start TIMESTAMPTZ NOT NULL,
 attempts INTEGER NOT NULL CHECK (attempts BETWEEN 0 AND 100),
 cooldown_until TIMESTAMPTZ
);

-- +migrate Down
DROP TABLE IF EXISTS auth_factor_budgets;
DROP TABLE IF EXISTS auth_pending;
DROP TABLE IF EXISTS authenticators;
DROP TABLE IF EXISTS webauthn_subjects;
