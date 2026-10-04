-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- +migrate Up
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_rp_shape;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_shape;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_rp_shape CHECK (
 (purpose IN (1,2,3) AND rp_id IS NOT NULL AND octet_length(user_handle) = 32)
 OR (purpose IN (4,5,6,7) AND rp_id IS NULL AND user_handle = '' AND rp_binding = decode(repeat('00',32),'hex') AND factor_bindings = ''));
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_shape CHECK (
 (purpose = 1 AND password_at IS NOT NULL AND password_at <= created_at AND factor_bindings = '' AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 2 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 3 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0)
 OR (purpose IN (4,5) AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 6 AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0)
 OR (purpose = 7 AND password_at IS NULL AND required_proof IN (2,3) AND max_age_us > 0 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0));


-- +migrate Down
DELETE FROM auth_pending WHERE purpose = 7;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_rp_shape;
ALTER TABLE auth_pending DROP CONSTRAINT auth_pending_purpose_shape;
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_rp_shape CHECK (
 (purpose IN (1,2,3) AND rp_id IS NOT NULL AND octet_length(user_handle) = 32)
 OR (purpose IN (4,5,6) AND rp_id IS NULL AND user_handle = '' AND rp_binding = decode(repeat('00',32),'hex') AND factor_bindings = ''));
ALTER TABLE auth_pending ADD CONSTRAINT auth_pending_purpose_shape CHECK (
 (purpose = 1 AND password_at IS NOT NULL AND password_at <= created_at AND factor_bindings = '' AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 2 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 3 AND password_at IS NULL AND octet_length(factor_bindings) BETWEEN 1 AND 4096 AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0)
 OR (purpose IN (4,5) AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND actor_id = '' AND actor_digest = '' AND actor_generation = 0)
 OR (purpose = 6 AND password_at IS NOT NULL AND password_at <= created_at AND required_proof IN (1,2) AND octet_length(actor_id) BETWEEN 1 AND 128 AND actor_id !~ '[^ -~]' AND octet_length(actor_digest) = 32 AND actor_generation > 0));
