-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
CREATE TABLE totp_authenticators (
 id TEXT PRIMARY KEY CHECK (octet_length(id) BETWEEN 1 AND 128 AND id !~ '[^ -~]'),
 user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
 record_version SMALLINT NOT NULL DEFAULT 1 CHECK (record_version = 1),
 key_id TEXT NOT NULL CHECK (octet_length(key_id) BETWEEN 1 AND 64 AND key_id !~ '[^ -~]'),
 envelope BYTEA NOT NULL CHECK (octet_length(envelope) = 61),
 accepted_step BIGINT NOT NULL CHECK (accepted_step >= 1),
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
 replay_revision BIGINT NOT NULL DEFAULT 1 CHECK (replay_revision > 0),
 created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE backup_sets (
 id TEXT PRIMARY KEY CHECK (octet_length(id) BETWEEN 1 AND 128 AND id !~ '[^ -~]'),
 user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE backup_codes (
 id TEXT PRIMARY KEY CHECK (id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
 set_id TEXT NOT NULL REFERENCES backup_sets(id) ON DELETE CASCADE,
 verifier TEXT NOT NULL CHECK (octet_length(verifier) BETWEEN 1 AND 128 AND verifier LIKE '$argon2id$v=19$%')
);
ALTER TABLE sessions DROP CONSTRAINT sessions_proof_shape;
ALTER TABLE sessions ADD COLUMN proof_factor_at TIMESTAMPTZ;
ALTER TABLE sessions ADD CONSTRAINT sessions_proof_shape CHECK (
 (proof_method = 1 AND proof_factor_id = '' AND proof_factor_revision = 0 AND proof_factor_at IS NULL)
 OR (proof_method = 2 AND octet_length(proof_factor_id) BETWEEN 1 AND 128 AND proof_factor_id !~ '[^ -~]' AND proof_factor_revision > 0 AND proof_factor_at IS NULL)
 OR (proof_method IN (3,4) AND octet_length(proof_factor_id) BETWEEN 1 AND 128 AND proof_factor_id !~ '[^ -~]' AND proof_factor_revision > 0 AND proof_factor_at IS NOT NULL AND proof_verified_at <= proof_factor_at AND proof_factor_at = authenticated_at));
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_shape;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_user_handle_check;
ALTER TABLE auth_pending ALTER COLUMN rp_id DROP NOT NULL;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_rp_shape CHECK (
 (purpose IN (1,2,3) AND rp_id IS NOT NULL AND octet_length(user_handle) = 32)
 OR (purpose IN (4,5,6) AND rp_id IS NULL AND user_handle = '' AND rp_binding = decode(repeat('00',32),'hex') AND factor_bindings = ''));
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_shape CHECK (
 (purpose = 1 AND password_at IS NOT NULL AND password_at <= created_at AND factor_bindings = '' AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 2 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 3 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0)
 OR (purpose IN (4,5) AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 6 AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0));

-- +migrate Down
DELETE FROM sessions WHERE proof_method IN (3,4);
DELETE FROM auth_pending WHERE purpose IN (4,5,6);
ALTER TABLE sessions DROP CONSTRAINT sessions_proof_shape;
ALTER TABLE sessions DROP COLUMN proof_factor_at;
ALTER TABLE sessions ADD CONSTRAINT sessions_proof_shape CHECK (
 (proof_method = 1 AND proof_factor_id = '' AND proof_factor_revision = 0)
 OR (proof_method = 2 AND octet_length(proof_factor_id) BETWEEN 1 AND 128 AND proof_factor_id !~ '[^ -~]' AND proof_factor_revision > 0));
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_rp_shape;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_shape;
ALTER TABLE auth_pending ALTER COLUMN rp_id SET NOT NULL;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_user_handle_check CHECK (octet_length(user_handle) = 32);
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_shape CHECK (
 (purpose = 1 AND password_at IS NOT NULL AND password_at <= created_at AND factor_bindings = '' AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 2 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 3 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0));
DROP TABLE backup_codes;
DROP TABLE backup_sets;
DROP TABLE totp_authenticators;
