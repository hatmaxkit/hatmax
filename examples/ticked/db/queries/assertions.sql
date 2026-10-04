-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: WebAuthnHandle :one
SELECT handle FROM webauthn_subjects WHERE user_id = $1 AND rp_id = $2;

-- name: SubjectAuthenticators :many
SELECT * FROM authenticators WHERE user_id = $1 AND rp_id = $2 ORDER BY id LIMIT $3::integer;

-- name: LockAuthenticator :one
SELECT * FROM authenticators WHERE id = $1 FOR UPDATE;

-- name: CreateAssertion :exec
INSERT INTO auth_pending(digest,purpose,record_version,user_id,auth_version,rp_id,user_handle,rp_binding,ceremony_data,
required_proof,policy_revision,max_age_us,created_at,expires_at,factor_bindings,actor_id,actor_digest,actor_generation)
VALUES ($1,$2,1,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17);

-- name: AssertionSubject :one
SELECT user_id FROM auth_pending WHERE digest = $1 AND purpose = $2 AND record_version = 1;

-- name: LockAssertion :one
SELECT * FROM auth_pending WHERE digest = $1 AND purpose = $2 AND record_version = 1 FOR UPDATE;

-- name: ConsumeAssertion :execrows
DELETE FROM auth_pending WHERE digest = $1 AND purpose = $2 AND revision = $3;

-- name: AcceptAssertionCounter :exec
UPDATE authenticators SET sign_count = $2, backup_state = $3, credential_data = $4, replay_revision = replay_revision + 1 WHERE id = $1;
