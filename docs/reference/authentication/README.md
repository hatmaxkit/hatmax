<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Authentication

`auth` creates users and sessions through a caller-supplied `Queries`
implementation. The implementation note is
[auth/readme.md](../../../auth/readme.md).

Password hashes for signup and sign-in use `model.HashPassword` and
`model.ComparePassword`. Session identifiers and tokens use `model.NewID`.

## User and session

`User` fields are `ID`, `Email`, `PasswordHash`, `Roles`, `Active`,
`TOTPSecret`, `TOTPEnabled`, `TOTPVerifiedAt`, `CreatedAt`, and `UpdatedAt`.

`HasRole` reports an exact role match. `HasAnyRole` reports whether any
supplied role matches. `NeedsTOTPSetup` is true when `TOTPSecret` is empty.
`InTOTPGracePeriod(days)` is false when `days` is not positive. Otherwise it
is true while the current time is before `CreatedAt` plus that many days.

`Session` fields are `ID`, `UserID`, `Token`, `ExpiresAt`, and `CreatedAt`.

## Queries

`Queries` is the persistence boundary:

| Method | Arguments |
| --- | --- |
| `CreateUser` | id, email, password hash, created at, updated at |
| `GetUserByEmail` | email |
| `GetUserByID` | id |
| `CreateSession` | id, user id, token, expires at, created at |
| `GetSessionByToken` | token |
| `DeleteSession` | session id |
| `DeleteExpiredSessions` | none |

A missing row is `sql.ErrNoRows`. The service maps that error to its own
sentinels.

## Service

`NewService(queries, cfg, log)` stores those values.

| Error | When |
| --- | --- |
| `ErrInvalidEmail` | `Signup` receives an empty email. |
| `ErrPasswordTooShort` | The password is shorter than `cfg.Auth.PasswordMinLen`. |
| `ErrEmailTaken` | `GetUserByEmail` returns a user. |
| `ErrUserNotFound` | Sign-in, session validation, or `GetUserByID` finds no row. |
| `ErrInvalidPassword` | `model.ComparePassword` rejects the password. |
| `ErrSessionNotFound` | Sign-out finds no session. |
| `ErrSessionExpired` | `ExpiresAt` is before `model.Now`. |

An inactive user on sign-in or session validation returns the error text
`user is not active`. That value is not one of the sentinels above.

`Signup` returns the created user. `Signin` parses `cfg.Auth.SessionTTL`. An
invalid duration uses 24 hours. The session token is a new model ID.
`Signout` deletes the session found by token. `ValidateSession` returns the
user when the session exists, has not expired, and the user is active.
`GetUserByID` returns that user, including an inactive user.
`CleanupExpiredSessions` calls `DeleteExpiredSessions`.

Other query failures are wrapped with `cannot check email`, `cannot hash
password`, `cannot create user`, `cannot get user`, `cannot create session`,
`cannot get session`, `cannot delete session`, or `cannot cleanup expired
sessions`.

## Context

A nil context makes `GetUserID`, `GetUser`, and `GetSession` return the zero
value and false. `WithUserID`, `WithUser`, and `WithSession` store those
values. `RequireAuth` and `OptionalAuth` store the user and the user ID. They
do not store the session.

## Middleware

`SessionCookieName` is `session`.

`RequireAuth` redirects to `/signin` with status `303` when the cookie is
missing or `ValidateSession` returns an error. Success stores the user and
calls the next handler. It does not clear the cookie.

`OptionalAuth` calls the next handler without a user when the cookie is
missing or validation fails.

`SetSessionCookie` sets `session` on path `/`, with the supplied `MaxAge`,
`HttpOnly`, `Secure`, and `SameSite=Lax`. `ClearSessionCookie` sets the same
cookie with an empty value and `MaxAge` of `-1`.

`RequireTOTP` calls the next handler when `Enabled` is nil or returns false,
when no user is in the context, when `TOTPEnabled` is true, or when
`InTOTPGracePeriod` accepts `GraceDays`. A nil `GraceDays` uses zero days.
Otherwise it redirects to `SetupURL`, or `/totp-setup` when that field is
empty, with status `303`.
