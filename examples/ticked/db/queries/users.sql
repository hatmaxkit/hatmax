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
INSERT INTO sessions (id, user_id, token, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token, expires_at, created_at;

-- name: GetSessionByToken :one
SELECT id, user_id, token, expires_at, created_at
FROM sessions
WHERE token = $1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW();

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
