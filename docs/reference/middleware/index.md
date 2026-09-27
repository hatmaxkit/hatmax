# Middleware

`middleware` supplies chi middleware for request identity, locale, caching,
roles, rate limits, same-origin checks, and telemetry. The implementation
note is [middleware/readme.md](../../../middleware/readme.md).

The readme calls `RateLimit(100, time.Minute)`. The function accepts a
`*RateLimiter`. Construct one with `NewRateLimiter(limit, window)` and pass
it to `RateLimit`.

## Stacks

`DefaultStack` returns `RequestID`, chi `RealIP`, chi `Logger`, and chi
`Recoverer`, in that order.

`DefaultInternal` returns that stack plus `InternalOnly`.

`InternalOnly` responds `403` with `Forbidden` when `RemoteAddr` is not a
loopback address, an address in `10.0.0.0/8`, `172.16.0.0/12`, or
`192.168.0.0/16`, or a host text that starts with `fc00:` or `fd00:`. An
unparseable address is rejected. IPv6 unique-local detection uses that text
prefix, not `net.IP.IsPrivate`.

## Request ID

`RequestID` reads `X-Request-ID`. An empty header becomes a new UUID. The
value is stored in the request context and echoed in the response header
`X-Request-ID`. `GetRequestID` returns that value, or `""` when the context
is nil or has no ID.

## Locale

`Locale` stores the chosen locale under the context key `locale`. Detection
order:

1. The `locale` cookie, when `Available` is empty or contains the cookie value.
2. `Accept-Language` tags, in header order, after the quality suffix and any
   `-region` suffix are removed and the tag is lowercased.
3. `LocaleConfig.Default`.

`GetLocale` returns the context value. A nil context or a missing value
returns `en`.

`SetLocaleCookie` sets cookie `locale` on path `/`, with `MaxAge` of 365
days, `HttpOnly`, and `SameSite=Lax`. It does not set `Secure`.

## Cache

`StaticCache` sets `Cache-Control: public, max-age=31536000, immutable` and
calls the next handler.

## Roles

`RequireRole` reads `auth.GetUser`. No user responds `401` with
`Unauthorized`. A user without `HasRole(role)` responds `403` with
`Forbidden`.

`RequireAnyRole` uses the same status codes and calls `HasAnyRole`.

`RequireRoles(validator, roles...)` reads the cookie named
`auth.SessionCookieName`. A missing cookie, or a `ValidateSession` error,
clears the session cookie on the validation failure and redirects to
`/signin` with `303`. A user with none of the roles responds `403`. Success
stores the user with `auth.WithUser` and calls the next handler.

## Rate limit

`NewRateLimiter(limit, window)` starts a cleanup goroutine that runs every
five minutes. `Allow(ip)` records the current time and returns true while
that IP has fewer than `limit` timestamps inside the window. The next call
returns false and does not record another timestamp.

`RateLimit` takes the client IP from the first `X-Forwarded-For` entry, then
`X-Real-IP`, then `RemoteAddr` with the last `:port` removed. A refused call
responds `429` with `Too many requests`.

## Same origin

`RequireSameOrigin` allows `GET`, `HEAD`, `OPTIONS`, and `TRACE` without
checking a source. Every other method responds `403` with `Forbidden` unless
the request passes the source check.

`Sec-Fetch-Site: cross-site` fails the check, case-insensitively. Otherwise
the check parses `Origin`, or `Referer` when `Origin` is empty. The URL must
have a scheme, a host, and no user info. Its scheme must match the request
scheme and its host must match `Request.Host` after default ports are added:
`80` for `http` and `443` for `https`.

The request scheme is `https` when `Request.TLS` is set. Otherwise it is the
first comma-separated `X-Forwarded-Proto` value, lowercased. Otherwise it is
`http`.

## Telemetry

`RequestCounter` is `IncrementRequests()`. `CrashRecorder` is
`RecordPanic(message, endpoint, method)`.

`TelemetryCounter` increments the counter when it is non-nil, then calls the
next handler. `TelemetryRecovery` records a recovered panic when the recorder
is non-nil and responds `500` with `Internal Server Error`.
