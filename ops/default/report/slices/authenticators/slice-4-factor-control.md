<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Authorized Factor Changes

Status: reviewing
Delivery set: authenticators
Plan: [Delivery plan](../../../plan/authenticators.md)
Tracker: [Delivery tracker](../../../tracker/authenticators.md)
Branch: `feat/authenticator-control`
PR: pending
T4.1: `1f7b73b55d3511d5df9cba62a7a4472c5962842a`
T4.2: `b7a2d6ed1fde75ee861084f0e102109d05289308`

## Purpose

Manage subject-owned authenticators under actual recent trusted proof. Additional
and replacement enrollment, removal, version advancement, revocation and optional
actor rotation preserve current policy and proof constituents in one transaction.

## Delivered Behavior

- Safe listing returns at most twenty primary factors in the configured RP scope,
  with kind/ID/security revision/time and verified backup flags. No seed, credential
  key/data, digest, verifier, pending state or bearer enters listing metadata.
- Initial password-authorized enrollment remains restricted to accounts without
  established factors. Additional/replacement operations require recent MFA or
  phishing-resistant MFA under explicit server policy; password-only management
  is denied. A permitted TOTP-only profile can enroll a passkey after actual MFA.
- Replacement captures exact subject-owned target kind/ID/security revision;
  addition has no target. WebAuthn uses the existing real registration verifier.
  TOTP confirms a real code against application-key AEAD-encrypted setup and
  consumes its first accepted step. A second active TOTP record is rejected.
- Established pending uses independent `change1.` secret/digest, purpose 7,
  captured actor/digest/generation/proof times, policy and target/protocol state.
  Finish requires the same actor bearer. Initial/fallback/session and cross-kind
  lookups cannot turn this continuation into another operation or application access.
- The shared finite pending/setup collection and subject budget apply to initial
  and established operations. Reservations commit attempts/lease before real
  verification; failure never refunds admission. Retained capacity includes active
  factors and unconfirmed setups, with bounded expired-row reclamation.
- Activation/replacement/removal advances account version, revokes every old
  session/pending and rotates an eligible actor without changing any proof time,
  activity snapshot or expiry. Removing/replacing its proof constituent revokes
  the actor and returns no bearer. New registration alone supplies no strong proof.
- Last-primary checks use current trusted access policy, configured RP and actual
  available TOTP key identities. Other RP credentials, unavailable-key TOTP and
  backup codes alone cannot count as a usable remaining primary factor.
- Ticked adds a minimal management page and bounded same-origin JSON handlers.
  The application fixes phishing-resistant management; request fields cannot
  weaken it. TOTP controls are explicitly disabled without configured fallback.
  Existing authorized backup regeneration rotates the actor and shows codes once.

## Implementation Notes

`FactorService` requires `FactorQueries` and the existing protocol service. An
explicitly supplied fallback service enables TOTP changes; it is not an optional
storage capability upgrade. The service replaces caller key-identity fields with
a sorted fixed-capacity snapshot of its actual owned key ring.

Migration 007 admits only the constrained established purpose in `auth_pending`.
The bounded versioned payload captures policy, target, actor, protocol/AEAD state;
scalar ownership, version, purpose, policy, time and actor columns are matched on
load. Generated SQLC v1.31.1 bindings and current callers compile together.

Transactions acquire subject before pending/actor and factor locks. Initial
factor acquisitions use stable IDs and owner/RP-constrained lookups. Completion
matches the exact reserved snapshot/revision/lease, current account, actor and
factor state, then checks trusted post-lock and post-write time. Cryptographic
verification occurs outside that transaction after durable reservation. Any
activation/version/deletion/rotation failure rolls back all completion writes;
previous reservation remains spent. No successful count or transient bearer is
returned before commit.

The authority tests inject state changes only after actual production verification,
use real blocking PostgreSQL locks and SQL write failures, and check full durable
rollback. One stale-proof fixture initially violated the database's proof/time
constraint; its mutation was corrected to preserve the stored shape before the
final successful regression. Focused review also corrected usable-factor checks
for other RP credentials and unavailable TOTP keys.

## Contracts Added or Changed

- `FactorPolicy`, `FactorKind`, `FactorSelection` and `Factor` define explicit
  trusted authority and bounded safe target metadata.
- `FactorChangePending`, `FactorChangeRecord` and mandatory `FactorQueries` bind
  actual verification to conditional atomic management storage.
- `NewFactorService`, `List`, `BeginWebAuthnChange`/`FinishWebAuthnChange`,
  `BeginTOTPChange`/`FinishTOTPChange` and `Remove` expose actual established changes.
- `FactorChangeResult` excludes the issued bearer from JSON; transports set or
  clear the cookie according to the committed retained/revoked actor outcome.
- Ticked uses `X-Factor-Change-Token` for finish and fixes management/access policy
  in application assembly. Active initial, assertion, fallback and session APIs
  retain their actual verification boundaries.

## Files of Interest

- `auth/factor_control.go` and `auth/enrollment.go`: core management and shared
  actual registration verification.
- `examples/ticked/internal/feat/auth/factor_control.go`: locked ownership/RP,
  recent-proof/current-state, usable-factor and atomic mutation contracts.
- `examples/ticked/assets/migration/postgres/007-factor-control.sql` and
  `examples/ticked/db/queries/factor_control.sql`: constrained pending purpose,
  bounded safe selection and required generated storage mapping.
- `examples/ticked/internal/feat/auth/factor_control_integration_test.go` and
  `factor_authority_integration_test.go`: actual lifecycle, conflicts, races,
  trusted lock/write time and rollback evidence.
- `examples/ticked/internal/web/factor_control.go`: production handlers and
  minimal safe management presentation.
- `docs/reference/authentication/README.md` and
  `docs/tutorials/user-guide/identity-and-sessions.md`: supported consumer contract
  and management workflow. `CHANGELOG.md` updates the usable authentication outcome.

## Validation

Go 1.27.1 and the owned local PostgreSQL 18.6 cluster; bounded `.tmp/build` scratch,
`TMPDIR`/`GOTMPDIR` and `GOFLAGS=-p=2`. Explicit `DB_HOST`, port, user and database
settings are mandatory for tagged tests; missing prerequisites cannot pass.

- `go test -race ./auth ./config ./crypto ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(FactorAuthority|FactorChanges|FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=180s` — passed, 71.093 seconds.
- `go test ./auth -run '^$' -fuzz '^FuzzFactorSelection$' -fuzztime=20s -parallel=2 -timeout=60s` — passed, 387509 executions; no KDF/database work.
- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `golangci-lint run --build-tags=integration --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./auth/... ./config/... ./crypto/... ./examples/ticked/...` — passed, zero issues.
- `make docs-check` — passed, including local links and example compilation.
- `git diff --check` — passed.

AU-07 and affected AU-02/AU-03/AU-04/AU-08/AU-10 are established by actual
management/lifecycle verification, subject/RP/state conflicts, one-winner races,
last-factor/current-proof protection, reserved attempts, oldest-proof expiry and
forced rollback. HTTP fakes establish transport/policy/cookie/presentation only;
they supply no cryptographic or browser-acceptance evidence.

## Risks and Follow-ups

Real browser acceptance remains Slice 5. Signed registration/assertion fixtures,
real OTP and actual PostgreSQL prove the named verifier/storage contracts; they
are not browser journeys, hardware certification or a complete assurance level.
The exact integrated `make check` runs once after Slice 5 merges. No full gate,
main alignment, release, tag, mirror publication or deployment ran in this slice.
Account recovery/all-factor loss and broader anti-automation/application-domain
requirements retain their separate owners.
