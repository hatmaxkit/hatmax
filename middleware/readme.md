# middleware

HTTP middleware for chi router.

## Usage

```go
// Apply default stack (RequestID, RealIP, Logger, Recoverer)
r := chi.NewRouter()
for _, mw := range middleware.DefaultStack() {
    r.Use(mw)
}

// Internal services only (adds IP restriction)
for _, mw := range middleware.DefaultInternal() {
    r.Use(mw)
}

// Individual middleware
r.Use(middleware.RequestID)
r.Use(middleware.RequireRole(model.RoleAdmin))
r.Use(middleware.RateLimit(100, time.Minute))
```

See: `requestid.go`, `roles.go`, `ratelimit.go`, `stack.go`.
