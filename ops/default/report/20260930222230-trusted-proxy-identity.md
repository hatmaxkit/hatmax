<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Trusted Proxy Identity

Status: delivered
Ticket: [TKT-20260930211800](../ticket/solved/20260930211800-trust-proxy-headers-only-from-approved-peers.md)
Branch: `fix/ticket-20260930211800-trusted-proxies`
PR: [#68](https://forge.adrianpk.com/hatmax/hatmax/pulls/68)
Integrated into `dev`: `8fcaedd5e3e32f758b5203baec2fc7aa52c318e9`

## Delivered Behavior

- Default stacks ignore forwarded IP headers unless proxy networks are
  explicitly approved. No implicit loopback or private-network trust exists.
- `ProxyHeaders` keeps separate connection-peer and client identities in
  request context without rewriting `RemoteAddr`.
- `InternalOnly` checks the original peer. Forged client headers cannot grant
  internal access, including through an approved public proxy.
- `RateLimit` uses the same resolved client identity, or the peer when no
  policy is installed. Untrusted header changes cannot reset a rate bucket.
- Proxy chains stop at the nearest unapproved hop. Malformed reachable hops
  fall back to the peer; multiple header lines retain their order.
- IPv4, IPv6, and IPv4-mapped addresses use canonical parsing. Internal IPv6
  detection covers the complete unique-local range.

## Public Contract

`DefaultStack` and `DefaultInternal` accept variadic `netip.Prefix` networks.
Existing zero-argument calls compile and trust no proxies. Callers using these
functions as typed values must account for the new variadic signatures.
Custom stacks install `ProxyHeaders` before rate limiting and before any
middleware that rewrites the peer. Handlers use `ClientIP(r)` for client IPs.
Chi logging continues to show the connection peer.

The [middleware reference](../../../docs/reference/middleware/README.md#client-ip-and-trusted-proxies)
defines header parsing, proxy configuration, and migration behavior. Its
network parsing uses the standard [Go netip API](https://pkg.go.dev/net/netip).
Chain traversal follows the trusted-proxy boundary described by
[MDN's X-Forwarded-For documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/X-Forwarded-For).

## Validation

All successful Go commands used `GOTOOLCHAIN=go1.26.7`.

- Before the fix, `go test ./middleware -run 'Test(InternalSpoof|RateLimitSpoof)$' -count=1`
  failed for both forwarded headers: internal access returned 200 instead of
  403, and a changed rate-limit header returned 200 instead of 429.
- `make check`: passed, including source licensing, formatting, vet, all
  tests, coverage threshold, and configured strict lint. Total coverage: 80.1%.
  Database tests used an isolated native PostgreSQL cluster on loopback,
  stopped after validation. No application database or Docker settings changed.
- `make docs-check`: passed.
- `go test -race ./middleware -count=1`: passed after the final test additions.
- `go test ./middleware -run '^$' -fuzz '^FuzzProxyHeaders$' -fuzztime=10s -parallel=2`:
  passed, 255,007 executions. Arbitrary headers preserved the peer, produced
  canonical client identities, and could not enable implicit proxy trust.
- `make lint-strict`: passed after the final test additions.
- `git diff --check`: passed.

An exploratory `golangci-lint run ./middleware` with its default rules found
six existing diagnostics in locale and telemetry code. These are outside
the repository's configured strict rules and were not changed by this ticket.

## Boundary

Only architecture finding F1 is addressed. Other review tickets remain open.
`InternalOnly` is a connection-network restriction, not end-client
authorization. Approved proxies must maintain trustworthy forwarded headers.
Forwarded scheme handling and unrelated middleware are unchanged.
