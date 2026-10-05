<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Security Observations

Status: delivered
Delivery set: authentication-controls
Plan: [Authentication controls plan](../../../plan/authentication-controls.md)
Tracker: [Authentication controls tracker](../../../tracker/authentication-controls.md)
Branch: `feat/authentication-events`
Implementation base: `09497e836932dbd84c8ad90294f18805172900d8`
Task commits: T4.1 `466894a7ba57`, T4.2 `10d48427296d`
PR: `#110`

## Purpose

Provide the approved AUTH-07 best-effort observation boundary without granting
new proof authority or changing delivered authentication/recovery transactions.

## Delivered Behavior

One required typed observer is shared by credentials, enrollment, WebAuthn,
fallback, factor and recovery services. The 26 public mutation/proof entrypoints
attempt one terminal observation after their private operation and all defers
finish. Internal helpers, lookups, listings, cleanup and GET do not emit another
mutation event. Transport refusals before a core call produce no core event.

Events separate committed mutation, completed session authentication, pending
issuance/enrollment/proof and classified denial. Only actual verifier results,
checked trusted actor proof and confirmed storage results supply facts. Setup
flags remain explanatory, never MFA authority. Unknown storage acknowledgment
remains `operating_unknown`; no rollback, completion or automatic retry is inferred.

The closed event carries an independent UUID, UTC time, operation/outcome and
optional trusted subject/record references and actual method. References are
at most 128 ASCII letters/digits/underscore/hyphen; unsafe values are omitted.
Checked serialized JSON is at most 1024 bytes. No credential, bearer/digest,
raw identity/IP/user-agent, TOTP/backup material, link, arbitrary map or error
text enters the shape. Ticked supplies a checked application logger adapter.

## Implementation Notes

Public wrappers call synchronous private implementations before observation.
Package-private attempt facts capture trusted loaded/reserved state and verifier
results; they never become caller proof. This places callbacks after transaction,
reservation-release and verifier-slot defers, using the original caller context
rather than a canceled internal work context.

Observation uses a finite non-waiting channel (default 2, bounds 1–16), a
synchronous callback deadline (default 100ms, bounds 1–100ms), and a deadline
strictly before the caller's when needed. No detached worker, queue or retry is
created. Fixed atomic counters classify delivered, saturated, rejected, canceled,
deadline and operating delivery. Callback/error-classification panics are
redacted operating diagnostics. Late nil return cannot count as delivered.

The observer is caller-owned, cooperative and concurrency-safe. Its failure
cannot replace an authentication result, roll back a mutation or repeat work.
Explicit discard is supported without audit assurance. Recovery notification
intents retain their independent durable atomic mutation contract.

## Contracts Added or Changed

- `SecurityEvent`, closed `SecurityOperation`/`SecurityOutcome`, typed
  `SecurityObserver` and immutable `SecurityObservationSettings`.
- `NewSecurityObservations(observer, cfg)` and fixed `Diagnostics()` counters;
  `ErrSecurityObservationRejected` classifies consumer rejection.
- `NewService(queries, cfg, checker, admission, observations, logger)` requires
  one initialized observation instance; child services inherit it. Active
  consumers and fixtures supply a shared adapter or explicit discard.
- `security_observation.timeout` and `concurrency` fail invalid construction
  and `Config.Validate`; omitted values choose finite documented defaults.
- Authentication/configuration reference, construction how-to, User Guide,
  Ticked configuration/assembly and the existing Unreleased outcome agree.

## Files of Interest

- `auth/security_event.go` — closed shape, delivery budget and diagnostics.
- `auth/security_operations.go` — actual public operation observation wrappers.
- `config/security_observation.go` — finite validated configuration.
- `examples/ticked/internal/feat/auth/security.go` — application observer.
- `examples/ticked/internal/feat/auth/security_integration_test.go` — real
  transaction/proof/lock/saturation/re-entry and ambiguous-result evidence.
- `examples/ticked/internal/web/security_integration_test.go` — actual
  handler, logger, neutral response and committed cookie evidence.

## Validation

Focused evidence uses Go 1.27.1 and owned PostgreSQL 18.6, isolated/dropped
schemas, per-worktree `TMPDIR`/`GOTMPDIR` and `GOFLAGS=-p=2`. No aggregate
`make check` or actual browser journey ran for this slice. The owned database
was stopped after all checks and a zero-other-client check.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed.
- `make docs-check` — passed.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^(TestSecurityObservationTransactions|TestCredentialTransactions|TestPasswordEntryBudgets|TestSessionTransactions|TestEnrollmentTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorChanges|TestFactorAuthority|TestPasswordChangeTransactions|TestPasswordResetTransactions)$' -count=1 -timeout=240s ./examples/ticked/internal/feat/auth` — passed in 103.026s.
- `go test -tags=integration -race -run '^(TestSecurityObservationTransactions|TestWebAuthnTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 24.170s after malformed-assertion classification refinement.
- `go test -tags=integration -race -run '^(TestAuthenticationObservationTransactions|TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/web` — passed in 80.693s.
- `go test -run '^$' -fuzz '^FuzzSecurityEvent$' -fuzztime=20s -parallel=2 -timeout=60s ./auth` — passed in 20.030s, 300449 executions.
- `go test -tags=browser -run '^$' ./examples/ticked/internal/web` — passed; consumer compilation only.
- `golangci-lint run --build-tags=integration --new-from-rev HEAD --whole-files --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5` — passed on changed integration files against T4.1 `466894a7ba57`; unrelated files preserved.
- `git diff --check` — passed.

Actual callbacks acquired the real subject row with a 25ms lock timeout after
operation return. All 26 observed entrypoints covered committed/pending/proof
semantics without duplicate helpers. Password-only/pending work did not issue
MFA sessions. Signed UV WebAuthn, real TOTP and one-use backup proof matched
actual committed methods; delivered invalidation, replay, rotation, ownership
and recovery notification regressions remained valid.

A callback failure preserved the committed user. Late success preserved the
actual session and counted deadline rather than delivery. A real session deletion
committed while the only callback slot was occupied; saturation skipped delivery
without waiting. A callback re-entered actual signout and emitted its independent
terminal event without locks or capacity deadlock. An adapter deliberately lost
the acknowledgment after real session insertion: no bearer returned, no retry
occurred, one extra stored session remained and the event stayed unknown.

Actual HTTP new/duplicate registration retained equal navigation/no cookie;
denial retained no cookie; successful sign-in retained `200`, `HX-Redirect`
and a secure cookie validated against real storage. Malformed/query input and
GET emitted no core mutation. Captured application logs/events excluded malicious
input, credentials, account identity and cookie contents. Initial fixture checks
were corrected to the existing explanatory pending-enrollment result and htmx
login response; product authentication behavior was not changed to fit a test.

AC-07 has actual service/adapter/HTTP evidence. AC-08 has the recorded focused
security regression; real browser and exact integrated acceptance remain Slice 5.

## Risks and Follow-ups

- Cancellation is cooperative. An observer or logging sink that ignores its
  context violates the adapter contract; core cannot forcibly terminate it.
- Timely callback success is best effort, not durable or exactly-once audit.
  Applications own sink behavior, access, retention, personal-data handling and
  mandatory audit persistence. Explicit discard establishes no audit delivery.
- Slice 5 owns real browser journeys, remaining evidence mapping and the single
  full gate on immutable dev after all five canonical merges. No release,
  external provider send or authenticator-loss lifecycle closure is included.
