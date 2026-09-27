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
	CreateSession(context.Context, string, string, string, time.Time, time.Time) (*Session, error)
	GetSessionByToken(context.Context, string) (*Session, error)
	DeleteSession(context.Context, string) error
	DeleteExpiredSessions(context.Context) error
}
```

Return `sql.ErrNoRows` for missing users and sessions. Preserve `CreatedAt` on
loaded users because TOTP grace-period checks use it.

## Create the service

```go
account := auth.NewService(queries, cfg, logger)
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
	r.Use(auth.RequireAuth(account))
	r.Get("/account", accountPage)
})
```

`SetSessionCookie` creates a secure cookie. Use HTTPS for browser testing.

## Verify the boundary

Request `/account` without a session cookie. The response is `303` with
`Location: /signin`. Sign in over HTTPS and repeat the request with the cookie;
the protected handler receives the user in its context.

See [Authentication Reference](../../reference/authentication/index.md).
