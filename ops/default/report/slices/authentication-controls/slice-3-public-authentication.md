<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Finite Public Authentication

Status: reviewing
Delivery set: authentication-controls
Plan: `ops/default/plan/authentication-controls.md`
Tracker: `ops/default/tracker/authentication-controls.md`
Branch: `feat/authentication-ingress`
PR: pending

## Purpose

Bound public authentication HTTP admission and remove account disclosure through
registration and password failures. This is Slice 3 of 5 in the
[plan](../../../plan/authentication-controls.md) and
[tracker](../../../tracker/authentication-controls.md). Typed terminal security
observation and actual browser acceptance remain in Slices 4 and 5.

## Delivered Behavior

- One worker-free fixed counter/window per canonical peer replaces timestamp
  slices and hidden cleanup. Peer capacity refuses unknown peers without live
  eviction; admission/explicit cleanup inspects at most one finite batch.
- Ticked installs one shared boundary for authentication POST routes after
  explicit proxy resolution. Finite active capacity rejects immediately and
  remains occupied through response acknowledgment. Existing narrower recovery
  guards remain effective.
- Valid new/duplicate/throttled registration acknowledges identically and directs
  to normal sign-in. Signup never calls sign-in or issues a session/cookie.
  Candidate-only policy feedback is safe; operating registration failure uses
  a uniform unavailable response without account or backend detail.
- Missing/inactive/wrong-password/throttled and operating public password failures
  share one status/message and configured acknowledgment target. Initial
  enrollment and fallback password starts use the same denial policy; successful
  restricted challenge contracts remain intact.
- Body ingress and server reads/writes have finite deadlines. Caller/server
  cancellation stops acknowledgment and releases active capacity; explicit close
  refuses new admissions before infrastructure cancellation and shutdown.
- Authentication URI logging is omitted for GET and POST because query strings
  may contain credential material. Ordinary route logging remains enabled. Public
  handlers render no raw identity, cookie, SQL/driver error or request body.

## Implementation Notes

T3.1: `24b86323236e`; T3.2: `fd0c0a41e80e`.

A finite map indexes list nodes containing only canonical peer, window start and
count. A rotating cleanup cursor inspects at most the configured batch; an
expired target counter renews directly even outside that cursor. Exact expiry
retirement and denied-window stability do not require a ticker or background
worker. Actual canonical peer selection reuses the trusted `ClientIP` contract.

Ingress captures work/ack deadlines at acceptance. Public credential handlers
require that context before service calls. Work derives from the largest actual
credential/factor/recovery timeout, with a five-second floor for existing
recovery initiation. Default acknowledgment is six seconds and validation
requires work plus 100ms observation reserve and 100ms margin. Waiting stays
synchronous and uses the original caller context; no detached timer is created.

The actual HTTP read deadline is retained until net/http finishes body handling.
An initial development test caught premature deadline clearing, which allowed a
stalled body drain; the corrected actual connection test passes. The owning
server resets connection deadlines between requests and bounds writes separately.
Ticked propagates application cancellation through `BaseContext` and closes
admission before canceling/stopping dependencies.

Forms admit at most 16 KiB, exact single fields, supported URL-encoded media and
no query input. Same-origin protection precedes service work. Public registration
maps classified uniqueness/admission outcomes to neutral navigation; operating
failure is non-authorizing. `ErrUserInactive` replaces free-form inactive state.
Fallback joins missing/inactive/stale classifications with its expected operation
sentinel while keeping operating lookup failure distinct internally.

HTTP tests use actual PostgreSQL adapters, Argon2 proof, production handlers and
actual HTTP connections. Templates used for candidate feedback are transport
fixture assets. Observation wrappers count actual calls/conflicts; they do not
manufacture proof, storage or sessions. Isolated schemas are removed by fixtures.
No mail/provider request, new dependency or migration was needed.

## Contracts Added or Changed

`middleware.NewRateLimiter(RateLimitConfig)` now returns a limiter and error.
Callers must handle validated finite configuration. Defaults are 12 requests per
minute, 1024 peers and a cleanup batch of 128; explicit finite bounds are in the
[middleware reference](../../../../../docs/reference/middleware/README.md#rate-limit).

`AuthenticationIngressConfig` and `Config.AuthenticationIngressSettings()` add
validated process-local peer/active/body/acknowledgment bounds. Ticked requires one
initialized common boundary before public credential work; standalone assembly
must also own finite server deadlines and close/cancellation. This local peer
budget complements mandatory durable identity admission across replicas.

Registration responds `303`, `Location: /signin` and `HX-Redirect: /signin`, with
one neutral body and no cookie for new/duplicate/throttled identity. Operating
registration failure responds `503` at the same target. Public password denial
responds `403` with `Authentication unavailable`, no cookie or identity retry
header. Canceled waits return unavailable without browser authority. Successful
sign-in retains its committed secure cookie; pending challenges grant no session.

## Files of Interest

- `middleware/ratelimit.go` and `middleware/ratelimit_test.go`: finite canonical counters, bounded cleanup, races and parser fuzz.
- `config/authentication_ingress.go`: immutable settings and work/ack bounds.
- `examples/ticked/internal/web/authentication_ingress.go`: common admission, active ownership, request context, form bounds and URI log isolation.
- `examples/ticked/internal/web/handler.go`, `enrollment.go` and `fallback.go`: neutral credential transports with unchanged successful authority.
- `examples/ticked/main.go`: explicit composition, finite server deadlines and close/cancel/shutdown order.
- `examples/ticked/internal/web/authentication_ingress_integration_test.go`: actual HTTP/PostgreSQL boundary evidence.
- [Configuration](../../../../../docs/reference/configuration/README.md#authentication-http-ingress), [authentication reference](../../../../../docs/reference/authentication/README.md#neutral-public-password-transport) and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#use-neutral-registration-and-password-responses).

## Validation

Focused checks passed with Go 1.27.1 and owned PostgreSQL 18.6. Scratch/build work
used bounded per-worktree `TMPDIR`, `GOTMPDIR` and `GOFLAGS=-p=2`. Actual fixtures
isolated/dropped schemas. The owned database was stopped after all checks and a
zero-other-client check. No full integrated gate or actual browser journey ran.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed: zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^(TestAuthenticationIngressTransactions|TestPublicAuthenticationTransactions|TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=240s ./examples/ticked/internal/web` — passed in 98.956s.
- `go test -tags=integration -race -run '^(TestCredentialTransactions|TestPasswordEntryBudgets)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 3.937s.
- `go test -run '^$' -fuzz '^FuzzRateLimitPeer$' -fuzztime=20s -parallel=2 -timeout=60s ./middleware` — passed in 20.082s, 434566 executions.
- `go test -tags=browser -run '^$' ./examples/ticked/internal/web` — passed; construction compilation only.
- `git diff --check` — passed.

One active real request remained admitted after its account work completed and
through acknowledgment; the next request was rejected before body/account work.
Caller cancellation and application `BaseContext` cancellation released capacity;
close refused admissions while existing work was pending. A real stalled body
ended at the five-second read deadline. Actual route/proxy tests shared one peer
allowance, rejected spoofed/equivalent identities and unknown-peer overflow, and
retained live counters. Unit/race tests verify finite counters, one-batch cleanup,
expiry renewal and exactly five of 50 concurrent allowed requests.

Two actual concurrent registration commands produced one user and one classified
uniqueness conflict, identical six-second acknowledgments and zero sessions.
Missing/inactive/wrong/throttled and operating password failures matched their
six-second response target with an eight-second fixture upper bound. Actual
successful sign-in validated its secure cookie against a committed session.
Enrollment/TOTP setup retained restricted challenge payloads without a cookie.
Safe candidate feedback matched across existing/new identity; injected database
faults and captured authentication/service/request logs disclosed no raw error
or credential material. Existing affected recovery HTTP regressions passed.

AC-04 has focused ingress evidence. AC-05 has actual registration/HTTP evidence;
its browser boundary remains Slice 5. AC-06 has actual neutral password, accepted
proof/challenge, cancellation and capacity evidence. This is a finite fixture
result, not deployment throughput or a production side-channel audit.

## Risks and Follow-ups

- Slice 4 owns shared typed terminal observations and bounded failure/re-entry
  evidence. Ordinary authentication URI logging is omitted; this slice does not
  deliver a complete security event stream or external audit delivery.
- Slice 5 owns actual browser journeys, final AC-01 through AC-08 mapping and
  affected regression. Full `make check` remains after five canonical merges;
  AUTH-07 remains partial until delivery-set closure.
- Applications own canonical identity, private durable admission material,
  explicit proxy trust, finite server deadlines and cleanup scheduling. These
  process-local bounds do not provide an internet-wide abuse detector.
- Acknowledgment is a configured response target. Existing Argon2 work remains
  synchronous and non-preemptible; bounded fixture timing does not establish
  constant-time responses under arbitrary production load or external assurance.
