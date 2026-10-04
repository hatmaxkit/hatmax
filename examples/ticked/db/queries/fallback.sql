-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: LockTOTP :one
SELECT * FROM totp_authenticators WHERE user_id = $1 FOR UPDATE;

-- name: CreateTOTP :exec
INSERT INTO totp_authenticators(id,user_id,key_id,envelope,accepted_step,created_at)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: AcceptTOTPStep :execrows
UPDATE totp_authenticators SET accepted_step = $2, replay_revision = replay_revision + 1
WHERE id = $1 AND accepted_step < $2 AND replay_revision < 9223372036854775807;

-- name: LockBackupSet :one
SELECT * FROM backup_sets WHERE user_id = $1 FOR UPDATE;

-- name: LockBackupCode :one
SELECT backup_codes.id, backup_codes.set_id, backup_codes.verifier, backup_sets.revision
FROM backup_codes JOIN backup_sets ON backup_sets.id = backup_codes.set_id
WHERE backup_sets.user_id = $1 AND backup_codes.id = $2 FOR UPDATE OF backup_codes;

-- name: DeleteBackupSet :exec
DELETE FROM backup_sets WHERE user_id = $1;

-- name: CreateBackupSet :exec
INSERT INTO backup_sets(id,user_id,created_at) VALUES ($1,$2,$3);

-- name: CreateBackupCode :exec
INSERT INTO backup_codes(id,set_id,verifier) VALUES ($1,$2,$3);

-- name: ConsumeBackupCode :execrows
DELETE FROM backup_codes WHERE id = $1 AND set_id = $2;

-- name: CreateFallback :exec
INSERT INTO auth_pending(digest,purpose,user_id,record_version,auth_version,rp_id,user_handle,rp_binding,ceremony_data,
required_proof,policy_revision,max_age_us,password_at,created_at,expires_at,actor_id,actor_digest,actor_generation)
VALUES ($1,$2,$3,1,$4,NULL,'',decode(repeat('00',32),'hex'),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14);
