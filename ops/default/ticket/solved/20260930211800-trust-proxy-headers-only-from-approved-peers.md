---
id: TKT-20260930211800
title: Bind proxy-derived identity to trusted peers
status: solved
resolution: fixed
kind: bug
severity: high
priority: high
scope: api
tags: architecture-review, api, hardening
source: review
reported_at: 2026-09-30T21:18:00Z
ready_at: 2026-09-30T22:13:06Z
started_at: 2026-09-30T22:13:06Z
reviewed_at: 2026-09-30T22:22:30Z
closed_at: 2026-09-30T22:33:24Z
branch: fix/ticket-20260930211800-trusted-proxies
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/68
commits: 50409d8aa1e6d32af809cb4c66b6add66042d9e1, 8fcaedd5e3e32f758b5203baec2fc7aa52c318e9
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

DefaultStack installs chi RealIP without a trusted-proxy boundary. DefaultInternal then checks the rewritten RemoteAddr. A request from 192.0.2.10 with X-Forwarded-For: 127.0.0.1 receives HTTP 200 instead of 403. RateLimit independently trusts X-Forwarded-For and X-Real-IP, so changing these headers also changes the rate bucket.

Sources: `middleware/stack.go:19, middleware/stack.go:39, middleware/ratelimit.go:115`.

Evidence: Reproduced by TestObservedInternalBypass and TestObservedRateLimitBypass.

Impact: Directly exposed services can accept forged internal identities. Rate limiting and network access restrictions do not share one authoritative peer policy.

## Expected Outcome

Define one trusted-proxy policy. Preserve the socket peer for InternalOnly and accept forwarded client identity only from approved proxies. Use proper IPv4/IPv6 parsing.

## Validation

Test public peers with forged loopback/private headers, approved and unapproved proxy chains, and IPv4/IPv6 buckets. Verify both access restriction and rate limiting.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f1).

## Implementation

`ProxyHeaders` owns an explicit copied proxy-network policy and records peer
and client identities separately. Default stacks trust no proxies. `RateLimit`
uses `ClientIP`, while `InternalOnly` uses the retained socket peer.
IPv4, IPv6, mapped IPv4, malformed headers, and unapproved chain boundaries
are covered by regression tests and fuzzing.

Delivery: [Trusted proxy identity](../../report/20260930222230-trusted-proxy-identity.md).

Merged into `dev` through PR #68 at
`8fcaedd5e3e32f758b5203baec2fc7aa52c318e9`.
