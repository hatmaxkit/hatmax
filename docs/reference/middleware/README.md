<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Middleware

`middleware` supplies chi middleware for request identity, locale, caching,
roles, rate limits, same-origin checks, and telemetry. The implementation
note is [middleware/readme.md](../../../middleware/readme.md).

## Stacks

`DefaultStack(trustedProxies ...netip.Prefix)` returns `RequestID`,
`ProxyHeaders`, chi `Logger`, and chi `Recoverer`, in that order. Calling
`DefaultStack()` trusts no proxies and ignores forwarded IP headers.

`DefaultInternal(trustedProxies ...netip.Prefix)` returns that stack plus
`InternalOnly`. The zero-argument call also trusts no proxies.

`InternalOnly` responds `403` with `Forbidden` unless the connection peer is
loopback, IPv4 private (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`),
or IPv6 unique-local (`fc00::/7`). Unparseable addresses are rejected.
IPv4-mapped IPv6 addresses are normalized to IPv4 before classification.
When `ProxyHeaders` is installed, the original connection peer is retained
even if later middleware changes `RemoteAddr`.

This is a connection-network restriction, not end-client authorization.
A private proxy can pass it even when forwarding a public client's request.
An approved public proxy does not pass it by forwarding a private client IP.

## Client IP and trusted proxies

`ProxyHeaders(trustedProxies ...netip.Prefix)` snapshots the connection peer
and records a resolved client IP in the request context. `ClientIP(r)` reads
that value. Without this middleware, `ClientIP` reads only the connection
peer and ignores headers. Valid IPs are canonicalized; an unparseable
`RemoteAddr` is returned unchanged.

Only a connection peer inside an explicitly supplied network can provide a
forwarded identity. Private and loopback peers are not implicitly trusted.
Supply IPv4 networks as IPv4 CIDRs, including for IPv4-mapped IPv6 peers.
The middleware copies its policy at construction and never rewrites
`RemoteAddr`. Chi's logger therefore reports the connection peer.

For an approved peer:

- All `X-Forwarded-For` header lines form one ordered comma-separated list.
  The list is traversed from right to left through approved hops. The first
  untrusted address is the client IP; entries to its left are ignored.
- If every hop is approved, the leftmost address is the client IP.
- A malformed hop reached during traversal falls back to the connection
  peer, without trying `X-Real-IP`.
- Only when `X-Forwarded-For` is absent, one `X-Real-IP` value may supply
  the client IP. Multiple values or a malformed value fall back to the peer.
- Header addresses must be bare IPv4 or IPv6 literals. Surrounding spaces
  are removed. Ports, CIDRs, hostnames, brackets, and IPv6 zones are rejected.

Install `ProxyHeaders` before `RateLimit` and before any middleware that
rewrites `RemoteAddr`. Do not install chi `RealIP` before it. Use narrow
proxy networks and configure those proxies to overwrite or correctly append
forwarded headers; trusting a proxy does not sanitize its configuration.

```go
trustedProxy := netip.MustParsePrefix("192.0.2.10/32")
router := app.NewRouter(logger,
	app.WithMiddleware(middleware.DefaultStack(trustedProxy)...),
)
```

For a custom stack, install `middleware.ProxyHeaders(trustedProxy)` directly.
Existing applications that relied on automatic header trust must configure
approved proxies and use `ClientIP(r)` instead of reading a rewritten
`RemoteAddr`. Zero-argument stack calls still work; their function signatures
now accept variadic network prefixes.

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

`RequireRoles(validator, requirement, activity, roles...)` reads the cookie named
`auth.SessionCookieName`. A missing cookie, or a `ValidateSession` error,
clears the session cookie on the validation failure and redirects to
`/signin` with `303`. A user with none of the roles responds `403`. Success
stores the user, user ID and safe session metadata, then calls the next handler.
The requirement is current trusted server policy. Roles and setup flags cannot
substitute for its proof properties. Activity is a trusted server choice; background polling uses `auth.NoActivity`.

## Rate limit

`NewRateLimiter(RateLimitConfig)` returns a limiter and configuration error.
Each canonical peer owns one counter and fixed window. Zero fields default to
12 requests per minute, 1024 peers and a cleanup batch of 128. Valid bounds are
1–1000 requests, a 1s–1h window, 1–10000 peers and a 1–1000 cleanup batch.
Construction starts no goroutine or ticker.

`Allow(ip)` charges one allowed operation. Denial does not extend a live window.
A full table rejects an unknown peer without evicting any live counter. Each
admission and explicit `Cleanup()` inspect at most one configured batch;
expired target counters also renew directly. Cleanup needs no shutdown hook.

`RateLimit` uses `ClientIP(r)`, so it shares the installed proxy policy.
Without `ProxyHeaders`, rate buckets use only the connection peer. Ports
are removed with IPv4/IPv6 address parsing, and equivalent IPv4-mapped or
IPv6 representations share a bucket. A refused call responds `429` with
`Too many requests`.

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

`RequireSameOrigin` reads `X-Forwarded-Proto` independently of `ProxyHeaders`.
The latter's trusted-client-IP policy does not validate that scheme header.
Configure an edge proxy to remove untrusted forwarded scheme values; direct TLS
takes precedence. This local source check does not establish an external proxy
configuration or a browser authentication workflow.
