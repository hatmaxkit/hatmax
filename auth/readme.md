# auth

Session-based authentication service with optional 2FA support.

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

## Middleware

```go
// Require authentication
r.Use(auth.RequireAuth(svc))

// Optional authentication (adds user to context if present)
r.Use(auth.OptionalAuth(svc))

// Require 2FA setup (use after RequireAuth)
r.Use(auth.RequireTOTP(auth.TOTPEnforcement{
    Enabled:   func() bool { return settingsSvc.GetBool(ctx, "security.require_2fa") },
    GraceDays: func() int { return settingsSvc.GetInt(ctx, "security.2fa_grace_period_days") },
    SetupURL:  "/settings/2fa",
}))
```

## User Model

```go
type User struct {
    ID             string
    Email          string
    PasswordHash   string
    Roles          []string
    Active         bool
    TOTPSecret     string     // TOTP secret key
    TOTPEnabled    bool       // Whether 2FA is active
    TOTPVerifiedAt *time.Time // When 2FA was verified
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// Helper methods
user.HasRole("admin")
user.HasAnyRole("admin", "moderator")
user.NeedsTOTPSetup()
user.InTOTPGracePeriod(7)
```

## Context Helpers

```go
// In handlers (after middleware)
user, ok := auth.GetUser(r.Context())
userID := auth.GetUserID(r.Context())
```

## Queries Interface

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

Implement with sqlc or manually.

## TOTP Primitives

For TOTP code generation and validation, see `crypto/totp.go`:

```go
// Generate TOTP key
key, _ := crypto.GenerateTOTPKey("MyApp", "user@example.com")

// Generate QR code
png, _ := crypto.GenerateQRCodePNG(key, 200)

// Validate code
valid := crypto.ValidateTOTPCode(secret, code)

// Backup codes
plain, hashed, _ := crypto.GenerateBackupCodes(8)
valid, index := crypto.VerifyBackupCode(code, hashed)
```
