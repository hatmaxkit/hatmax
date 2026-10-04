-- SPDX-FileCopyrightText: 2026 Adrian PK
-- SPDX-License-Identifier: Apache-2.0
--
-- This file is part of Hatmax. See LICENSE for license terms.

-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, roles, active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, password_hash, auth_version, roles, active, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, auth_version, roles, active, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, auth_version, roles, active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, password_hash, auth_version, roles, active, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUserRoles :exec
UPDATE users
SET roles = $2, updated_at = $3, auth_version = auth_version + 1
WHERE id = $1;

-- name: UpdateUserActive :exec
UPDATE users
SET active = $2, updated_at = $3, auth_version = auth_version + 1
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (id, user_id, token_digest, auth_version, policy_revision, generation, proof_method, proof_verified_at, proof_factor_id, proof_factor_revision, authenticated_at, created_at, last_activity_at, expires_at, inactivity_us)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING *;

-- name: GetSessionByDigest :one
SELECT *
FROM sessions WHERE token_digest = $1;

-- name: GetSessionForUpdate :one
SELECT *
FROM sessions WHERE token_digest = $1 FOR UPDATE;

-- name: UpdateSessionActivity :one
UPDATE sessions SET last_activity_at = $2 WHERE id = $1
RETURNING *;

-- name: SessionClock :one
SELECT clock_timestamp()::timestamptz;

-- name: DeleteSession :execrows
DELETE FROM sessions WHERE token_digest = $1;

-- name: DeleteExpiredSessions :execrows
WITH expired AS (
    SELECT id FROM sessions
    WHERE expires_at <= clock_timestamp()
       OR last_activity_at + inactivity_us * INTERVAL '1 microsecond' <= clock_timestamp()
    ORDER BY expires_at, id LIMIT $1::integer FOR UPDATE SKIP LOCKED
)
DELETE FROM sessions USING expired WHERE sessions.id = expired.id;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetUserForAuth :one
SELECT id, email, password_hash, auth_version, roles, active, created_at, updated_at
FROM users WHERE id = $1 FOR UPDATE;

-- name: ReplacePassword :one
UPDATE users SET password_hash = $2, auth_version = auth_version + 1, updated_at = $3
WHERE id = $1
RETURNING id, email, password_hash, auth_version, roles, active, created_at, updated_at;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: RotateSession :one
UPDATE sessions SET token_digest = sqlc.arg(new_digest), generation = generation + 1,
    policy_revision = sqlc.arg(policy_revision), proof_method = sqlc.arg(proof_method), proof_verified_at = sqlc.arg(proof_verified_at), proof_factor_id = sqlc.arg(proof_factor_id), proof_factor_revision = sqlc.arg(proof_factor_revision),
    authenticated_at = sqlc.arg(authenticated_at), last_activity_at = sqlc.arg(last_activity_at), expires_at = sqlc.arg(expires_at), inactivity_us = sqlc.arg(inactivity_us)
WHERE token_digest = sqlc.arg(old_digest) AND generation = sqlc.arg(expected_generation)
RETURNING *;

-- name: ReclaimSubjectSessions :execrows
WITH expired AS (
 SELECT s.id FROM sessions s WHERE s.user_id = $1 AND
 (s.expires_at <= clock_timestamp() OR s.last_activity_at + s.inactivity_us * INTERVAL '1 microsecond' <= clock_timestamp())
 ORDER BY s.id LIMIT 100 FOR UPDATE
)
DELETE FROM sessions USING expired WHERE sessions.id = expired.id;

-- name: CountSubjectSessions :one
SELECT COUNT(*) FROM sessions WHERE user_id = $1;

-- name: LockSubjectSessions :many
SELECT id FROM sessions WHERE user_id = $1 ORDER BY id LIMIT 101 FOR UPDATE;

-- name: ListSubjectSessions :many
SELECT *
FROM sessions WHERE user_id = sqlc.arg(subject) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(page_limit)::integer;

-- name: RevokeSubjectSessions :execrows
DELETE FROM sessions WHERE user_id = sqlc.arg(subject) AND
 (sqlc.arg(scope)::integer = 4
  OR (sqlc.arg(scope)::integer = 3 AND id <> sqlc.arg(actor_id))
  OR (sqlc.arg(scope)::integer = 1 AND id = sqlc.arg(actor_id))
  OR (sqlc.arg(scope)::integer = 2 AND id = sqlc.arg(selected_id)));
