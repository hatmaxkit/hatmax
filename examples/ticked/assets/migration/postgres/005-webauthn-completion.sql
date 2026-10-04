-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
ALTER TABLE authenticators ADD COLUMN replay_revision BIGINT NOT NULL DEFAULT 1 CHECK (replay_revision > 0);
ALTER TABLE sessions DROP CONSTRAINT sessions_proof_method_check;
ALTER TABLE sessions ADD COLUMN proof_factor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sessions ADD COLUMN proof_factor_revision BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD CONSTRAINT sessions_proof_shape CHECK (
 (proof_method = 1 AND proof_factor_id = '' AND proof_factor_revision = 0)
 OR (proof_method = 2 AND octet_length(proof_factor_id) BETWEEN 1 AND 128 AND proof_factor_id !~ '[^ -~]' AND proof_factor_revision > 0));
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_check;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_check;
ALTER TABLE auth_pending ALTER COLUMN password_at DROP NOT NULL;
ALTER TABLE auth_pending ADD COLUMN factor_bindings BYTEA NOT NULL DEFAULT '' CHECK (octet_length(factor_bindings) <= 4096);
ALTER TABLE auth_pending ADD COLUMN actor_id TEXT NOT NULL DEFAULT '';
ALTER TABLE auth_pending ADD COLUMN actor_digest BYTEA NOT NULL DEFAULT '';
ALTER TABLE auth_pending ADD COLUMN actor_generation BIGINT NOT NULL DEFAULT 0;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_lifetime CHECK (expires_at - created_at BETWEEN INTERVAL '1 minute' AND INTERVAL '10 minutes');
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_shape CHECK (
 (purpose = 1 AND password_at IS NOT NULL AND password_at <= created_at AND factor_bindings = '' AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 2 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 3 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0));

-- +migrate Down
DELETE FROM sessions WHERE proof_method = 2;
DELETE FROM auth_pending WHERE purpose <> 1;
ALTER TABLE sessions DROP CONSTRAINT sessions_proof_shape;
ALTER TABLE sessions DROP COLUMN proof_factor_id, DROP COLUMN proof_factor_revision;
ALTER TABLE sessions ADD CONSTRAINT sessions_proof_method_check CHECK (proof_method = 1);
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_shape;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_lifetime;
ALTER TABLE auth_pending DROP COLUMN factor_bindings, DROP COLUMN actor_id, DROP COLUMN actor_digest, DROP COLUMN actor_generation;
ALTER TABLE auth_pending ALTER COLUMN password_at SET NOT NULL;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_check CHECK (purpose = 1);
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_check CHECK (password_at <= created_at AND expires_at - created_at BETWEEN INTERVAL '1 minute' AND INTERVAL '10 minutes');
ALTER TABLE authenticators DROP COLUMN replay_revision;
