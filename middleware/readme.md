<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# middleware

HTTP middleware for chi router.

## Usage

This router fragment belongs in the composition root before route registration.
The complete lifecycle is shown in the [bootstrap procedure](../docs/how-to/bootstrap-application/README.md).
`staticFS` is the application's embedded static filesystem. Role checks belong
on authenticated route groups rather than the public health/static routes.

```go
limiter, err := middleware.NewRateLimiter(middleware.RateLimitConfig{
    Limit: 100, Window: time.Minute, MaxPeers: 1024, CleanupBatch: 128,
})
if err != nil {
    return err
}

r := chi.NewRouter()
r.Use(middleware.DefaultStack()...)
r.Use(middleware.Locale(middleware.LocaleConfig{
    Default: "en", Available: []string{"en", "es", "de"},
}))
r.Use(middleware.RateLimit(limiter))
r.Use(middleware.RequireSameOrigin)

r.Route("/static", func(r chi.Router) {
    r.Use(middleware.StaticCache)
    r.Handle("/*", http.FileServer(http.FS(staticFS)))
})
```

## Same-origin request protection

`RequireSameOrigin` allows safe HTTP methods without source headers. Other
methods require an `Origin` or `Referer` whose scheme and host match the
request. `Sec-Fetch-Site: cross-site` is always rejected. HTTPS is resolved
from direct TLS or the first `X-Forwarded-Proto` value.

## Trusted proxy identity

The default stack ignores forwarded IP headers. To resolve clients behind
an approved proxy, pass its network explicitly:

```go
trustedProxy := netip.MustParsePrefix("192.0.2.10/32")
for _, mw := range middleware.DefaultStack(trustedProxy) {
    r.Use(mw)
}
```

`middleware.ClientIP(r)` returns the resolved client; `RemoteAddr` remains
the connection peer. `RateLimit` uses the same identity. `InternalOnly`
always restricts the connection network, not the forwarded client's network.
For custom stacks, install `ProxyHeaders` before rate limiting and before
any middleware that changes `RemoteAddr`.

See [Middleware reference](../docs/reference/middleware/README.md#client-ip-and-trusted-proxies)
for chain traversal, header requirements, and migration details.

## Locale

```go
// In handler, get current locale
locale := middleware.GetLocale(r.Context())

// Set locale preference cookie
middleware.SetLocaleCookie(w, "es")
```

See: `csrf.go`, `locale.go`, `cache.go`, `requestid.go`, `roles.go`,
`ratelimit.go`, `stack.go`, `proxy.go`.
