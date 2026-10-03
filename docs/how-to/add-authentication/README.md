<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Add Authentication

Hatmax authentication owns password checks and session behavior. The
application owns persistence through `auth.Queries`.

## Implement the persistence boundary

Provide all methods in `auth.Queries`:

```go
type Queries interface {
	CreateUser(context.Context, string, string, string, time.Time, time.Time) (*User, error)
	GetUserByEmail(context.Context, string) (*User, error)
	GetUserByID(context.Context, string) (*User, error)
	CreateSession(context.Context, CredentialState, SessionRecord) (*Session, error)
	ReplacePassword(context.Context, CredentialState, string, time.Time) (*User, error)
	ValidateSession(context.Context, SessionDigest, SessionActivity, time.Duration) (*ValidatedSession, error)
	DeleteSession(context.Context, SessionDigest) error
	DeleteExpiredSessions(context.Context, int) (int64, error)
}
```

Return owned user/session snapshots and store only the session digest. Validate
active/current account version and exact absolute/inactivity expiry under locks,
using current time evaluated after the wait. Apply relevant activity in that same
transaction; polling must leave activity unchanged. Return `sql.ErrNoRows` for
missing user reads, `ErrSessionNotFound` for missing session lookups and
`ErrCredentialChanged` for stale account state. Preserve user `CreatedAt` for
TOTP grace-period checks.

## Create the service

```go
account, err := auth.NewService(queries, cfg, checker, logger)
if err != nil { return err }
```

Use `Signup`, `Signin`, `Signout`, and `ValidateSession` from handlers. Set the
session cookie after sign-in:

```go
session, err := account.Signin(r.Context(), email, password)
if err != nil {
	http.Error(w, "invalid credentials", http.StatusUnauthorized)
	return
}

auth.SetSessionCookie(w, session.Token, int(time.Until(session.ExpiresAt).Seconds()))
```

## Protect routes

```go
r.Group(func(r chi.Router) {
	r.Use(auth.RequireAuth(account, auth.RelevantActivity))
	r.Get("/account", accountPage)
})
```

`SetSessionCookie` creates a secure cookie. Use HTTPS for browser testing.

## Verify the boundary

Request `/account` without a session cookie. The response is `303` with
`Location: /signin`. Sign in over HTTPS and repeat the request with the cookie;
the protected handler receives the user, user ID and safe session metadata in
its context. Background routes use `auth.NoActivity`. Validate session settings
at startup; invalid lifetimes/cadence fail construction.

See [Authentication Reference](../../reference/authentication/README.md).
