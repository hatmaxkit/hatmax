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

// Same-origin protection for state-changing browser requests
r.Use(middleware.RequireSameOrigin)
```

## Same-origin request protection

`RequireSameOrigin` allows safe HTTP methods without source headers. Other
methods require an `Origin` or `Referer` whose scheme and host match the
request. `Sec-Fetch-Site: cross-site` is always rejected. HTTPS is resolved
from direct TLS or the first `X-Forwarded-Proto` value.

## Locale

```go
// In handler, get current locale
locale := middleware.GetLocale(r.Context())

// Set locale preference cookie
middleware.SetLocaleCookie(w, "es")
```

See: `csrf.go`, `locale.go`, `cache.go`, `requestid.go`, `roles.go`,
`ratelimit.go`, `stack.go`.
