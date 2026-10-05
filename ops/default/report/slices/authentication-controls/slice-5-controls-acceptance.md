<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 5: Controls Acceptance

Status: reviewing
Delivery set: authentication-controls
Plan: [Authentication controls plan](../../../plan/authentication-controls.md)
Tracker: [Authentication controls tracker](../../../tracker/authentication-controls.md)
Branch: `test/authentication-controls-acceptance`
Implementation base: `947fb0c4c113`
Task commit: T5.1 `85e88174f0c8`; T5.2 introduces this report.
PR: pending

## Purpose

Verify the four delivered runtime controls together at actual browser, PostgreSQL
and HTTP boundaries, preserve existing MFA/recovery authority and prepare the
single immutable integrated gate after the final maintainer merge.

## Delivered Behavior

Both existing browser journeys now start with two concurrent registration requests
for one address: one committed user, one classified uniqueness refusal, equal
neutral sign-in navigation and no automatic cookie/session. Candidate-only policy
feedback and malformed/ambiguous/oversized forms retain redaction. Missing,
incorrect and exhausted password requests share the six-second fixture response
and issue no cookie. Actual failed password calls prepare a nearly spent durable
window; the final browser failure charges it and the following correct password
is denied before verification.

One common ingress with two active slots retains both admitted requests through
acknowledgment. A third browser request receives `429` without a core event; both
completed replies release their slots. Getter routes expose only active count or
redacted events and never approve authentication. The fixture applies migration
009 with existing schema-local credentials/authenticators/recovery migrations.

The actual application `SecurityLogger` remains shared through WebAuthn, TOTP,
backup, factor and recovery operations. Browser acceptance validates event shape,
unique IDs, committed mutation/proof provenance, all four actual proof methods,
redacted application logs and complete timely delivery. The authenticator and
recovery journeys captured 44 and 80 terminal observations respectively; setup
observations are counted separately.

Actual navigator registration/assertion, UP/UV/signature failures, pending-token
isolation, actor rotation, last-factor protection, one-use backup and TOTP replay
continue to pass. The recovery journey retains neutral initiation, fragment/GET
safety, same-origin protection, recent real MFA for password change, Unicode
replacement and all-session invalidation, token/replay/backup consumption, required
operator role/MFA, failed dispatch/notice retry and a lost committed reset response.

## Implementation Notes

Extend the retained Node/CDP harness and production handler/service/adapter graph;
no alternate authentication or successful proof substitute is introduced. Browser
executables and actual database settings remain mandatory and absent prerequisites
fail. Each test owns a random schema and browser profile. Finite fixture settings
are 20 proof attempts, ten registrations, two active requests and 1000 peer
requests; default production configuration is unchanged. Recovery retains its
explicit 40-attempt authenticator subject window and narrower ingress limits.

Children have 210/300-second deadlines; the combined selector has 600 seconds.
CDP operations and owned-profile cleanup are bounded. Header/read/write/idle
server deadlines and a concurrency-safe 512-line/2048-byte capture bound the
fixture. Admitted core observations are awaited before measuring capacity refusal,
so another request's terminal event cannot be attributed to the refused one.
Initial development assertion failures concerned fixture classification, a static
form placeholder, observation ordering and setup-counter accounting; the corrected
final selector passed without product runtime changes.

The tracker inventories all four password-verifier call sites and maps AC-01
through AC-08 to real evidence and remaining obligations. Earlier parser-fuzz and
finite SQL row/index measurements remain in delivered reports; they were not
repeated for this acceptance-only slice. No additional dependency is introduced.

## Contracts Added or Changed

No product API, persistence, migration, proof authority or configuration default
changes. Browser/reference commands now include migration 009, current child
bounds and the combined controls selector. Core/configuration/example/User Guide
agree on required shared admission/observation, neutral transport, finite ingress
and actual proof. Existing Unreleased notes already describe the delivered product
outcome; acceptance adds no separate user-facing capability.

The concern/model and parent record four delivered runtime slices and reviewing
acceptance. Implementation remains Partial until final merge and the exact full
gate; broader foundation and consumer obligations remain explicit.

## Files of Interest

- `examples/ticked/internal/web/authenticator_browser_test.go` — actual assembly,
  shared observer, admission setup, finite server and delivery accounting.
- `examples/ticked/internal/web/controls_browser_test.go` — finite actual logger
  capture, checked event/proof provenance and redaction.
- `examples/ticked/internal/web/testdata/controls.mjs` — production browser
  registration/password/ingress journeys, consumed admission and equal replies.
- `examples/ticked/internal/web/account_recovery_browser_test.go` — shared logger
  through retained production recovery consumer construction.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#authentication-controls-browser-acceptance),
  [configuration](../../../../../docs/reference/configuration/README.md#authentication-http-ingress)
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md).
- [Requirement evidence](../../../tracker/authentication-controls.md#requirement-evidence)
  and [candidate preparation](../../../tracker/authentication-controls.md#integrated-candidate-preparation).

## Validation

Local Go 1.27.1, PostgreSQL 18.6, Node v26.8.1 and Chromium 151.0.7922.173.
Owned socket `/tmp/hatmax-credential-pg-56439`, port 56439, no TCP listener;
fixtures create/drop only their schemas. `NODE_BIN=/usr/sbin/node` and
`CHROMIUM_BIN=/usr/sbin/chromium`; required `DB_*` settings select that instance.
Use per-worktree `TMPDIR=$PWD/.tmp/build`, `GOTMPDIR=$PWD/.tmp/build`, `GOFLAGS=-p=2`.
Logs stay in ignored scratch. These are focused local integration checks, not
production, external-provider or full-repository gate results.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues before each task commit.
- `make docs-check` — passed, including examples and local links.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=browser -race -v -run '^(TestAuthenticatorBrowser|TestAccountRecoveryBrowser)$' -count=1 -timeout=600s ./examples/ticked/internal/web` — passed in 152.503s; recovery 107.07s and authenticators 44.39s.
- `go test -tags=integration -race -run '^(TestCredentialAdmissionTransactions|TestPasswordEntryBudgets|TestCredentialTransactions|TestSessionTransactions|TestEnrollmentTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorChanges|TestFactorAuthority|TestPasswordChangeTransactions|TestPasswordResetTransactions|TestSecurityObservationTransactions)$' -count=1 -timeout=300s ./examples/ticked/internal/feat/auth` — passed in 122.724s.
- `go test -tags=integration -race -run '^(TestAuthenticationIngressTransactions|TestPublicAuthenticationTransactions|TestAuthenticationObservationTransactions|TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=300s ./examples/ticked/internal/web` — passed in 117.721s.
- `golangci-lint run --build-tags=browser --new-from-rev HEAD --whole-files --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 --fix` — passed against implementation base before T5.1 commit.
- `node --check examples/ticked/internal/web/testdata/controls.mjs` — passed.
- `git diff --check` — passed.

## Risks and Follow-ups

- Maintainer merge and the single exact immutable integrated dev `make check`
  remain pending. Close the set only after that gate is green. Required repository
  corrections use the approved validation-fix branch and dev PR, followed by the
  corrected integrated candidate gate; no correction branch for a green result.
- Applications own canonical identity/alias convergence, stable namespace/key,
  replica configuration, trusted proxies, cleanup/shutdown and deployment admission.
  This fixture timing is not a production side-channel or throughput audit.
- AUTH-07 supported resource admission/errors/observation do not close cumulative
  authenticator disabling/rebinding, all-factor-loss identity proofing, durable
  audit retention or consumer/deployment assurance. The parent remains Partial.
