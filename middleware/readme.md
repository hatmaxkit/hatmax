# middleware

HTTP middleware for chi router.

## Usage

```go
r := chi.NewRouter()

// Default stack (RequestID, RealIP, Logger, Recoverer)
for _, mw := range middleware.DefaultStack() {
    r.Use(mw)
}

// Locale detection (cookie, Accept-Language header)
r.Use(middleware.Locale(middleware.LocaleConfig{
    Default:   "en",
    Available: []string{"en", "es", "de"},
}))

// Static asset caching (1 year, immutable)
r.Route("/static", func(r chi.Router) {
    r.Use(middleware.StaticCache)
    r.Handle("/*", http.FileServer(http.FS(staticFS)))
})

// Role-based access
r.Use(middleware.RequireRole(model.RoleAdmin))

// Rate limiting
r.Use(middleware.RateLimit(100, time.Minute))
```

## Locale

```go
// In handler, get current locale
locale := middleware.GetLocale(r.Context())

// Set locale preference cookie
middleware.SetLocaleCookie(w, "es")
```

See: `locale.go`, `cache.go`, `requestid.go`, `roles.go`, `ratelimit.go`, `stack.go`.
