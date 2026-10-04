<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Restricted Enrollment

Status: delivered
Delivery set: authenticators
Plan: [Delivery plan](../../../plan/authenticators.md)
Tracker: [Delivery tracker](../../../tracker/authenticators.md)
Branch: `feat/authenticator-enrollment`
PR: `#98`
Merged: 2026-10-04
Merge commit: `d916a27f53e2d0d42d67aed246d018194aacbfcf`
T1.1: `776b4aca0ab80318fa11d86035684d00efca2b5a`
T1.2: `f5b62edccb970f2af7e20f80d6f77f1890a3b437`

## Purpose

Deliver initial WebAuthn enrollment from fresh actual password verification,
with restricted continuation and finite durable admission. Registration alone
cannot issue an ordinary or strong session.

## Delivered Behavior

- Active accounts without established factors can begin ES256 registration with
  a stable opaque random RP handle, required resident key/UV and attestation none.
- An independent `enroll1.` bearer, separate digest/collection and exact stored
  ceremony bind subject version, password time, policy and RP configuration.
- Real protocol verification rejects wrong challenge/RP/origin/subject/ID/algorithm,
  missing UP/UV, invalid backup flags and embedded ceremonies. The library COSE
  parser additionally validates curve and point before storing the key.
- Subject-first PostgreSQL transactions serialize admission and confirmation.
  Pending/subject attempts and one finite revision-bound lease commit before
  protocol work; errors and reissue cannot refund or reset those attempts.
- Confirmation inserts one unique RP credential, advances account version and
  deletes prior sessions/pending together. Concurrent/replayed/stale completion
  and failed final commit return no success or session.
- Ticked provides same-origin JSON begin/finish endpoints with finite bodies,
  generic failures, no-store responses and no issued authentication cookie.

## Implementation Notes

`AuthenticatorService` requires typed enrollment storage and shares the existing
credential service's bounded password verifier. No optional storage upgrade,
caller-approved proof callback, second KDF engine or unused future method exists.
Constructor settings are validated owned snapshots; RP identity/origins are never
inferred from headers. Cleanup is explicit and finite, with no owned loop.

The pinned go-webauthn v0.18.2 registration/session/COSE APIs and BSD-3-Clause
license were inspected. Exact module/transitive versions are recorded in
`go.mod`/`go.sum`; fxamacker/cbor v2.9.4 bounds structural parsing before library
parsing. The profile rejects other attestation formats. Model documentation
settles the later encrypted-material binding without introducing a seed interface.

Migration `004-authenticators.sql` adds handles, constrained credential/pending
records and one budget row per subject to existing databases. Required SQLC
queries and generated mappings are synchronized using sqlc v1.31.1.

## Contracts Added or Changed

- Required `NewAuthenticatorService`, begin/finish/cleanup operations and
  `AuthenticatorQueries`; safe output carries no bearer/digest/proof authority.
- `Config.Authenticator` validates trusted RP identity, finite capacities,
  durations/admission and an immutable configuration fingerprint.
- Exact expiry/freshness equality rejects; current account/policy/lease checks
  occur after locks. Failed completion rolls back activation/revocation while
  already-committed admission remains spent. A retry verifies again.
- Product reference, User Guide, package guidance and Unreleased describe only
  the initial registration capability and current session boundary.

## Files of Interest

- [Enrollment service](../../../../../auth/enrollment.go) and
  [bounded parser](../../../../../auth/enrollment_parse.go).
- [PostgreSQL adapter](../../../../../examples/ticked/internal/feat/auth/enrollment.go)
  and [migration](../../../../../examples/ticked/assets/migration/postgres/004-authenticators.sql).
- [HTTP handler](../../../../../examples/ticked/internal/web/enrollment.go).
- [Authentication reference](../../../../../docs/reference/authentication/README.md#restricted-webauthn-enrollment).

## Validation

Focused local checks passed on Go 1.27.1; PostgreSQL evidence uses the owned
18.6 cluster with explicit `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` and
`DB_NAME`. Missing `DB_HOST` fails rather than skipping. Build/lint scratch storage
was local and bounded; the owned database was stopped after validation.

- `go test ./auth ./config ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -race ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web -count=1 -timeout=120s`
- `go test -race ./auth -run '^TestEnrollment' -count=1 -timeout=30s`
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^TestEnrollmentTransactions$' -count=1 -timeout=60s`
- `go test ./auth -run '^$' -fuzz '^FuzzEnrollmentResponse$' -fuzztime=20s -parallel=2 -timeout=60s` — 354797 executions, no failure; no KDF/DB work in the target.
- `make source-license-check` — 775 headers, 105 annotations.
- `make vet`
- `make lint-strict` — zero issues.
- `golangci-lint run --build-tags=integration --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5` — zero issues.
- `make docs-check` — includes example compilation.
- `git diff --check`

Actual PostgreSQL evidence includes one confirmation winner, setup replay,
password/account mutation after protocol verification, deferred commit failure
and successful re-verification, pending/subject budgets and reissue denial,
stale lease release, retained-row concurrency, expired-only cleanup/reclamation,
RP credential uniqueness, post-lock expiry and cancellation during lock waits.
The registration portions of AU-01/AU-03/AU-07/AU-08 and affected AU-10 are covered.

## Risks and Follow-ups

Assertion sign-in/step-up, replay-resistant TOTP/backup proof, established-factor
management and the production browser journey remain Slices 2–5. Existing
password-policy routes do not enforce MFA merely because a key is enrolled.
The complete set's aggregate gate remains after Slice 5 merge; no `make check`,
main alignment, release, tag or deployment ran for this slice. Browser/hardware
certification and complete assurance claims are not established by these fixtures.
