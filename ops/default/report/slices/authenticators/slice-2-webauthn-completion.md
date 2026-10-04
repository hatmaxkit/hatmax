<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: WebAuthn Completion

Status: reviewing
Delivery set: authenticators
Plan: [Delivery plan](../../../plan/authenticators.md)
Tracker: [Delivery tracker](../../../tracker/authenticators.md)
Branch: `feat/webauthn-completion`
PR: `#99`
T2.1: `f7df235f4411be7a2bb7eaa0f211077295edba61`
T2.2: `1b01ea4ba0a64a87705996b37320e745cd20d863`

## Purpose

Complete actual subject-first UV WebAuthn authentication and bound session
step-up. Factor replay state, pending consumption and session insertion/rotation
share one PostgreSQL commit. Enrollment metadata supplies no authentication proof.

## Delivered Behavior

- Actual ES256 assertions verify the server challenge, RP, exact origin, signature,
  credential/subject binding and presence/UV through the pinned v0.18.2 verifier.
- Independent `assert1.` and `stepup1.` tokens bind distinct pending purposes,
  captured account/policy/RP state and finite allowed factor security revisions.
  Step-up additionally binds the current actor ID, digest and generation.
- Pending and subject attempts commit before bounded protocol work. Reissue cannot
  reset the shared budget; busy requests do not queue or spend an attempt.
- Strict counter policy accepts zero/zero and otherwise requires an increase.
  Backup eligibility is immutable; current backup state and replay revision update
  with completion. Routine assertions preserve existing valid factor-bound sessions.
- Final subject-first transactions reject stale factor/replay/account/policy/lease
  or actor state, recheck trusted time after locks/writes, enforce session admission
  and atomically insert one session or rotate one current generation.
- Safe proof metadata binds actual verification time and confirmed factor/security
  revision. UV WebAuthn meets password-or-better, MFA and phishing-resistant policy
  in the supported profile; current factor validation applies to session/control.
- Ticked JSON begin/finish endpoints issue a cookie only after successful commit.
  The strong proof endpoint denies ordinary password proof. Failure preserves the
  existing cookie. Password reauthentication produces only password proof.

## Implementation Notes

`NewWebAuthnService` requires typed `WebAuthnQueries`, which includes enrollment
storage; the enrollment-only constructor remains valid. Both protocols share the
same finite verifier slots in the composed service. No optional adapter upgrade,
untyped transaction or caller-submitted proof API is introduced.

Security revision and replay revision are distinct. The final transaction matches
both revisions, counter, current flags and stored verifier data against the
cryptographic snapshot. Actual library counter warnings do not substitute for
core's strict counter policy. Account version, policy and actor lifecycle are
rechecked at the locked time. Any completion error returns no reusable bearer;
durable admission is separate and remains charged.

The additive `005-webauthn-completion.sql` migration extends confirmed factors,
pending purpose constraints and session proof shape. Enrollment password time is
nullable only for assertion purposes. SQL sources and SQLC mappings agree with
required Go contracts and active Ticked wiring. Dependencies remain unchanged.

## Contracts Added or Changed

- `WebAuthnService` supplies `BeginWebAuthnAuthentication`, `BeginWebAuthnStepUp`
  and actual browser-response `FinishWebAuthn`.
- `AssertionPending` captures exact owned ceremony/factor/policy/account bindings;
  `AssertionReservation` and `AssertionCompletion` are typed trusted storage
  commands, never accepted by service methods as client proof receipts.
- `VerifiedProof` supports closed password/WebAuthn methods. WebAuthn requires
  `FactorID` and positive `FactorRevision`; password requires neither.
- Required completion locks subject, pending/actor and factors, then commits
  replay state, pending consumption and access together. Cleanup/retention and
  session admission remain bounded.
- Assertion bounds: 64 KiB body, 8 KiB client data, 1 KiB credential ID, exact
  37-byte authenticator data, 80-byte signature, optional exact 32-byte handle,
  JSON depth 16. This profile rejects assertion attested data/extensions and
  cross/top origins before unbounded protocol parsing.
- Reference, package guidance, User Guide and Unreleased describe actual sign-in
  and step-up with explicit current application policy.

## Files of Interest

- [Assertion service](../../../../../auth/webauthn.go) and
  [bounded parser](../../../../../auth/assertion_parse.go).
- [Proof contract](../../../../../auth/proof.go).
- [PostgreSQL completion](../../../../../examples/ticked/internal/feat/auth/webauthn.go)
  and [additive migration](../../../../../examples/ticked/assets/migration/postgres/005-webauthn-completion.sql).
- [Real verifier/transaction tests](../../../../../examples/ticked/internal/feat/auth/webauthn_integration_test.go).
- [Ticked HTTP endpoints](../../../../../examples/ticked/internal/web/webauthn.go).
- [Authentication reference](../../../../../docs/reference/authentication/README.md#webauthn-authentication-and-step-up).

## Validation

Focused local checks passed on Go 1.27.1. Real PostgreSQL evidence uses the owned
18.6 cluster with explicit connection settings; missing `DB_HOST` fails. Local
scratch storage and Go parallelism were bounded. The owned cluster was stopped
after all tests completed; unrelated services/worktrees were preserved.

- `go test ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -race ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web -count=1 -timeout=120s`
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`
- `go test ./auth -run '^$' -fuzz '^FuzzAssertionResponse$' -fuzztime=20s -parallel=2 -timeout=60s` — 484362 executions, no failure; no KDF/DB work.
- `make source-license-check`
- `make vet`
- `make lint-strict` — zero issues.
- `golangci-lint run --build-tags=integration --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5` — zero issues.
- `make docs-check` — includes example compilation.
- `git diff --check`

Actual signed responses establish invalid challenge/RP/origin/ID/handle/signature,
missing UP/UV, cross/top origin, ceremony/extension and backup-flag rejection;
zero/nonzero counter policy; legitimate backup-state changes; and password/strong
policy separation. PostgreSQL covers one pending/generation winner, concurrent
counterless snapshot conflict, changed/foreign/removed factors, actor/account/
policy/lease conflicts, current-factor session/control rejection, shared budget/
retention/session-cap bounds, purpose isolation and no-queue admission.

Post-lock expiry tests first prove each captured actual verifier command valid,
then cross pending/lease/proof/actor expiry while waiting on a real row lock.
A blocked INSERT additionally expires proof after counter/pending writes and
proves full rollback. Deferred commit failures preserve counter/replay/pending and
old actor state; a retry performs actual verification again. HTTP-only fakes
independently establish cookie/policy/body/origin transport boundaries. AU-01
through AU-04 and affected AU-07/AU-08/AU-10 have focused evidence; AU-09 browser
acceptance remains Slice 5.

## Risks and Follow-ups

TOTP/backup proof, established-factor management and real production browser
journeys remain Slices 3–5. Ordinary routes keep their explicit password-or-better
minimum; enrollment alone does not change policy. Backup flags do not establish
hardware/non-exportability or an assurance certification. No full delivery-set
`make check` ran; the approved aggregate gate remains after Slice 5 merge.
Main alignment, release, tags and deployment remain outside this delivery.
