# auth

Session-based authentication service.

## Usage

```go
// Create service with your Queries implementation
svc := auth.NewService(queries, cfg, log)

// Signup
user, err := svc.Signup(ctx, "user@example.com", "password123")

// Signin (returns session with token)
session, err := svc.Signin(ctx, "user@example.com", "password123")

// Validate session (e.g., in middleware)
user, err := svc.ValidateSession(ctx, sessionToken)

// Signout
svc.Signout(ctx, sessionToken)
```

## API

```go
type Queries interface {
    CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    GetUserByID(ctx context.Context, id string) (*User, error)
    CreateSession(ctx context.Context, id, userID, token string, expiresAt, createdAt time.Time) (*Session, error)
    GetSessionByToken(ctx context.Context, token string) (*Session, error)
    DeleteSession(ctx context.Context, sessionID string) error
    DeleteExpiredSessions(ctx context.Context) error
}
```

Implement with sqlc or manually. See `middleware.go` for HTTP middleware and `context.go` for request context helpers.
