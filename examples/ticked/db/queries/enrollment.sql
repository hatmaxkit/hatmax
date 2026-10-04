-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: EnsureWebAuthnHandle :one
INSERT INTO webauthn_subjects(user_id,rp_id,handle) VALUES ($1,$2,$3)
ON CONFLICT (user_id,rp_id) DO UPDATE SET rp_id = EXCLUDED.rp_id RETURNING handle;

-- name: CountAuthenticators :one
SELECT COUNT(*) FROM authenticators WHERE user_id = $1;

-- name: CreateEnrollment :exec
INSERT INTO auth_pending(digest,purpose,record_version,user_id,auth_version,rp_id,user_handle,ceremony_data,
required_proof,policy_revision,max_age_us,password_at,created_at,expires_at,rp_binding)
VALUES ($1,1,1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);

-- name: EnrollmentSubject :one
SELECT user_id FROM auth_pending WHERE digest = $1 AND purpose = 1 AND record_version = 1;

-- name: LockEnrollment :one
SELECT * FROM auth_pending WHERE digest = $1 AND purpose = 1 AND record_version = 1 FOR UPDATE;

-- name: CountEnrollments :one
SELECT COUNT(*) FROM auth_pending WHERE user_id = $1;

-- name: ReclaimSubjectEnrollments :execrows
WITH expired AS (
 SELECT p.digest FROM auth_pending p WHERE p.user_id = $1 AND p.expires_at <= clock_timestamp()
 ORDER BY digest LIMIT 10 FOR UPDATE
)
DELETE FROM auth_pending USING expired WHERE auth_pending.digest = expired.digest;

-- name: EnsureFactorBudget :exec
INSERT INTO auth_factor_budgets(user_id,window_start,attempts) VALUES ($1,$2,0) ON CONFLICT DO NOTHING;

-- name: LockFactorBudget :one
SELECT * FROM auth_factor_budgets WHERE user_id = $1 FOR UPDATE;

-- name: ChargeFactorBudget :exec
UPDATE auth_factor_budgets SET window_start = $2, attempts = $3, cooldown_until = $4 WHERE user_id = $1;

-- name: LeaseEnrollment :exec
UPDATE auth_pending SET attempts = attempts + 1, revision = revision + 1, lease_until = $2 WHERE digest = $1;

-- name: ReleaseEnrollment :exec
UPDATE auth_pending SET lease_until = NULL WHERE digest = $1 AND revision = $2;

-- name: ConfirmAuthenticator :exec
INSERT INTO authenticators(id,user_id,rp_id,kind,record_version,credential_id,public_key,credential_data,sign_count,
backup_eligible,backup_state,user_verified,created_at)
VALUES ($1,$2,$3,1,1,$4,$5,$6,$7,$8,$9,true,$10);

-- name: AdvanceAuthenticatorVersion :exec
UPDATE users SET auth_version = auth_version + 1, updated_at = $2 WHERE id = $1;

-- name: DeleteSubjectEnrollments :exec
DELETE FROM auth_pending WHERE user_id = $1;

-- name: DeleteExpiredEnrollments :execrows
WITH expired AS (
 SELECT p.digest FROM auth_pending p WHERE p.expires_at <= clock_timestamp()
 ORDER BY expires_at,digest LIMIT $1::integer FOR UPDATE SKIP LOCKED
)
DELETE FROM auth_pending USING expired WHERE auth_pending.digest = expired.digest;
