-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: IssueMailboxToken :exec
INSERT INTO mailbox_tokens(id,user_id,purpose,target,auth_version,policy_revision,secret_digest,created_at,expires_at,attempt_limit)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (user_id,purpose) DO UPDATE SET id=EXCLUDED.id,target=EXCLUDED.target,auth_version=EXCLUDED.auth_version,
policy_revision=EXCLUDED.policy_revision,secret_digest=EXCLUDED.secret_digest,created_at=EXCLUDED.created_at,
expires_at=EXCLUDED.expires_at,attempt_limit=EXCLUDED.attempt_limit,attempts=0,revision=1,lease_until=NULL,consumed_at=NULL,revoked_at=NULL;

-- name: MailboxSubject :one
SELECT user_id FROM mailbox_tokens WHERE id=$1;

-- name: LockMailboxToken :one
SELECT * FROM mailbox_tokens WHERE id=$1 FOR UPDATE;

-- name: EnsureRecoveryBudget :exec
INSERT INTO recovery_budgets(user_id,kind,window_start,attempts) VALUES ($1,$2,$3,0) ON CONFLICT DO NOTHING;

-- name: LockRecoveryBudget :one
SELECT * FROM recovery_budgets WHERE user_id=$1 AND kind=$2 FOR UPDATE;

-- name: ChargeRecoveryBudget :exec
UPDATE recovery_budgets SET window_start=$3,attempts=$4 WHERE user_id=$1 AND kind=$2;

-- name: ChargeMailboxToken :exec
UPDATE mailbox_tokens SET attempts=attempts+1,revision=revision+1 WHERE id=$1;

-- name: LeaseMailboxToken :exec
UPDATE mailbox_tokens SET lease_until=$2 WHERE id=$1 AND revision=$3;

-- name: ReleaseMailboxToken :exec
UPDATE mailbox_tokens SET lease_until=NULL WHERE id=$1 AND revision=$2 AND consumed_at IS NULL AND revoked_at IS NULL;

-- name: VerifyCurrentMailbox :exec
UPDATE users SET mailbox_verified_at=$2,auth_version=auth_version+1,updated_at=$2 WHERE id=$1;

-- name: ConsumeMailboxToken :exec
UPDATE mailbox_tokens SET consumed_at=$2,lease_until=NULL,revision=revision+1 WHERE id=$1;

-- name: RevokeSubjectMailboxTokens :exec
UPDATE mailbox_tokens SET revoked_at=$2,lease_until=NULL,revision=revision+1 WHERE user_id=$1 AND consumed_at IS NULL AND revoked_at IS NULL;

-- name: CountRecoveryNotices :one
SELECT count(*) FROM recovery_notices WHERE user_id=$1;

-- name: CreateRecoveryNotice :exec
INSERT INTO recovery_notices(id,user_id,kind,destination,created_at) VALUES ($1,$2,$3,$4,$5);

-- name: ReclaimRecoveryNotices :execrows
WITH retired AS (
 SELECT n.id FROM recovery_notices n WHERE n.user_id=$1 AND (n.delivered_at IS NOT NULL OR n.created_at+INTERVAL '7 days'<=clock_timestamp())
 ORDER BY n.created_at,n.id LIMIT 20 FOR UPDATE
)
DELETE FROM recovery_notices USING retired WHERE recovery_notices.id=retired.id;

-- name: DeleteExpiredMailboxTokens :execrows
WITH retired AS (
 SELECT id FROM mailbox_tokens WHERE COALESCE(consumed_at,revoked_at,expires_at)+INTERVAL '24 hours'<=clock_timestamp()
 AND (lease_until IS NULL OR lease_until<=clock_timestamp())
 ORDER BY COALESCE(consumed_at,revoked_at,expires_at),id LIMIT $1::integer FOR UPDATE SKIP LOCKED
)
DELETE FROM mailbox_tokens USING retired WHERE mailbox_tokens.id=retired.id;

-- name: LockRecoveryNotices :many
SELECT * FROM recovery_notices WHERE user_id=$1 AND delivered_at IS NULL AND attempts<5 AND created_at+INTERVAL '7 days'>clock_timestamp()
ORDER BY created_at,id LIMIT sqlc.arg(page_limit)::integer FOR UPDATE SKIP LOCKED;

-- name: AttemptRecoveryNotice :exec
UPDATE recovery_notices SET attempts=attempts+1 WHERE id=$1;

-- name: DeliverRecoveryNotice :exec
UPDATE recovery_notices SET delivered_at=clock_timestamp() WHERE id=$1;

-- name: RecoveryPasswordFactors :many
SELECT id, 1::smallint AS kind, revision FROM authenticators a WHERE a.user_id=$1
UNION ALL
SELECT id, 2::smallint AS kind, revision FROM totp_authenticators t WHERE t.user_id=$1
ORDER BY id LIMIT 21;
