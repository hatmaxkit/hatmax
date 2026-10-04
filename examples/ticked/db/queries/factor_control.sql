-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: SafeFactors :many
SELECT id, 1::smallint AS kind, revision, created_at, backup_eligible, backup_state FROM authenticators a WHERE a.user_id = $1
UNION ALL
SELECT id, 2::smallint AS kind, revision, created_at, false, false FROM totp_authenticators t WHERE t.user_id = $1
ORDER BY id LIMIT 21;

-- name: RemoveWebAuthnFactor :execrows
DELETE FROM authenticators a WHERE a.user_id = $1 AND id = $2 AND revision = $3;

-- name: RemoveTOTPFactor :execrows
DELETE FROM totp_authenticators t WHERE t.user_id = $1 AND id = $2 AND revision = $3;

-- name: CountFactorSetups :one
SELECT COUNT(*) FROM auth_pending WHERE user_id = $1 AND purpose IN (1,4,7);

-- name: LockManagedAuthenticator :one
SELECT * FROM authenticators WHERE user_id = $1 AND id = $2 FOR UPDATE;
