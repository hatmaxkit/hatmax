<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication

`auth` creates users and sessions through a caller-supplied `Queries`
implementation. The implementation note is
[auth/readme.md](../../../auth/readme.md).

Signup uses `model.HashPasswordWithCost(password, cfg.Auth.BCryptCost)`;
sign-in uses `model.ComparePassword`. Session identifiers and tokens use
`model.NewID`.

## Password policy

`NewPasswordPolicy(cfg, checker)` constructs an immutable credential policy.
`PasswordChecker.Disallowed(ctx, password)` checks the complete normalized
candidate against caller-owned common, compromised or application-specific
values. A checker is required, must honor cancellation and support concurrent
calls, and must not retain candidates. The caller documents its source coverage
and provenance; supplying a checker alone does not establish breach coverage.

`PasswordPolicyConfig` has these fields:

| Field | Default | Constraint |
| --- | --- | --- |
| `MinLength` | 15; 8 with `MFARequired` | At least the profile minimum and at most `MaxLength` |
| `MaxLength` | 1024 code points | Between 64 and 1024 |
| `MaxBytes` | 4096 UTF-8 bytes | At least four times `MaxLength`; at most 4096 |
| `CheckTimeout` | 2 seconds | Positive; at most 30 seconds |
| `MFARequired` | false | Trusted server policy; only true when password access always requires MFA |

Zero numeric values select defaults. Other invalid settings or a missing
checker return `ErrPasswordPolicy`. The MFA flag records the application's
requirement; authentication must separately enforce the required factors.

`Prepare(ctx, password)` returns the accepted NFC-normalized candidate for
hashing. It bounds raw bytes before normalization, rejects invalid UTF-8, then
checks normalized code-point and byte lengths. It preserves case and whitespace,
and neither truncates nor imposes character-composition rules. Use NFC processing
when verifying the same credential format. Apply candidate policy when creating
or changing credentials; ordinary sign-in verifies the stored credential.

Length and normalization follow
[NIST SP 800-63B-4 password guidance](https://pages.nist.gov/800-63-4/sp800-63b.html#passwordver).
The policy is available independently; `Service.Signup` currently uses its
existing configuration and bcrypt path. Service integration belongs to the
credential-integration slice.

| Error | Condition |
| --- | --- |
| `ErrPasswordPolicy` | Invalid configuration or an uninitialized policy |
| `ErrPasswordEncoding` | Invalid UTF-8 |
| `ErrPasswordTooShort` | Too few normalized code points |
| `ErrPasswordTooLong` | Character or byte limit exceeded |
| `ErrPasswordDisallowed` | Checker rejects the complete candidate |
| `ErrPasswordCheckFailed` | Checker fails; inspect the wrapped cause with `errors.Is` or `errors.As` |
| `context.Canceled` / `context.DeadlineExceeded` | Request canceled or checker deadline reached |

All preparation failures return an empty candidate. Checker error text is
redacted; the wrapped cause remains diagnostic material and must not be exposed
to users or logged without review. The checker receives the earlier of the
configured deadline and the caller's deadline. Cancellation is cooperative:
no detached goroutine is created to abandon an uncooperative checker. A checker
that returns success after cancellation/deadline cannot approve the password.

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

`Signup` returns the created user. Password hashing uses `auth.bcrypt_cost`,
which defaults to 12 and accepts 4 through 31. An invalid cost returns a wrapped
`bcrypt.InvalidCostError`, without falling back to another cost or creating a
user. Hashing failures, including bcrypt's 72-byte password limit, also prevent
user creation. Increasing the configured cost affects future signups only;
existing hashes remain valid and are not rehashed automatically.

`Signin` parses `cfg.Auth.SessionTTL`. An
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
