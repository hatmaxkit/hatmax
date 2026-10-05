<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Guarded Password Entry

Status: reviewing
Delivery set: authentication-controls
Plan: `ops/default/plan/authentication-controls.md`
Tracker: `ops/default/tracker/authentication-controls.md`
Branch: `feat/guarded-password-entry`
PR: `#108`

## Purpose

Require shared durable admission before registration and all actual password
entry work. This is Slice 2 of 5 in the [plan](../../../plan/authentication-controls.md)
and [tracker](../../../tracker/authentication-controls.md). Finite public ingress,
neutral transport and security observations remain in recorded later slices.

## Delivered Behavior

- Service construction requires shared durable admission. Registration charges
  its own purpose before checker/hash; sign-in, initial WebAuthn enrollment and
  password-based fallback charge one common proof budget before account lookup.
- Reauthentication derives the canonical identity from the validated live actor
  before its password check. Internal fallback helpers charge each check once.
- Raw password encoding/byte bounds reject work before admission. Admitted
  policy/checker failure, successful proof, inactive/invalid/stale state and
  cancellation retain their charges. Lost admission results deny further work.
- Admission consumes the existing whole-operation deadline. Fallback password
  starts use the lesser credential/factor timeout, including actor validation.
- Ticked requires stable application-owned namespace/key provisioning at startup;
  missing or malformed material fails without disclosing it. Existing recovery,
  factor, shared KDF and session authority remain intact.

## Implementation Notes

T2.1: `dd425e7a244d`; T2.2: `59ddb46fc792`.

A package-local structural helper commits the selected admission purpose without
lookup, policy callbacks or cryptography. Actual credential work starts only after
that transaction releases its locks. No optional guard, local fallback, refund or
automatic replay was added. Core preserves password text/NFC processing and does
not invent identity normalization; applications must converge accepted aliases
before passing identity to lookup and admission.

The service owns the required admission pointer. Enrollment and fallback reuse
that same credentials service; constructors cannot silently omit admission.
Ticked decodes canonical base64 key material, constructs its private storage
adapter, clears temporary decoded bytes and passes the validated shared service.
No new migration, generated query, dependency or worker was needed.

Existing real SQL fixtures now apply delivered migration 009. Constructors within
one isolated schema share explicit fixture key/namespace. Extended existing
fixtures select finite 20-proof/10-registration bounds; production defaults stay
ten/three. New boundary fixtures use smaller explicit limits. Unit acceptance
fakes cover orchestration only and make no distributed-accounting claim.

## Contracts Added or Changed

`NewService(queries, cfg, checker, admission, logger)` requires a non-nil initialized
`*CredentialAdmission`. Consumers must construct durable admission explicitly.
`Signin`, `Signup`, initial enrollment, fallback password starts and password
reauthentication now return admission errors before unauthorized credential work.
Proof-input errors keep their existing verifier classification; registration
preserves new-password policy classifications.

`TICKED_CREDENTIAL_NAMESPACE` and `TICKED_CREDENTIAL_KEY` are required application
configuration. Replicas share stable private material and consistent limits.
Cleanup remains explicit and application-owned. Recovery new-password checks and
backup-code verification retain their distinct delivered budgets.

## Files of Interest

- `auth/service.go`: mandatory dependency, structural admission, registration and sign-in.
- `auth/session_control.go`: live-subject reauthentication admission.
- `auth/enrollment.go` and `auth/fallback.go`: shared password entry and whole-operation deadlines.
- `examples/ticked/main.go`: private adapter assembly and required provisioning.
- `examples/ticked/internal/feat/auth/password_entry_integration_test.go`: actual shared entrypoint races, retained charges and result-loss fault.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#standalone-credential-admission),
  [configuration](../../../../../docs/reference/configuration/README.md#credential-admission-limits)
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#bound-password-entry-across-replicas).

## Validation

Focused checks passed with Go 1.27.1 and owned PostgreSQL 18.6. Database fixtures
isolate/drop schemas, capture mail and send nothing externally. Scratch/build work
is bounded per worktree. The owned database was stopped after all checks and a
zero-other-client check. No full integrated gate or browser journey was run.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed: zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^(TestPasswordEntryBudgets|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority|TestPasswordChangeTransactions|TestPasswordResetTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 103.857s.
- `go test -tags=integration -race -run '^(TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/web` — passed in 62.668s.
- `go test -tags=browser -run '^$' ./examples/ticked/internal/web` — passed; compilation only.
- `git diff --check` — passed.

Twenty-four actual requests across two sign-in instances, initial enrollment,
TOTP setup and fallback admitted exactly four proof operations; the twenty
rejected operations performed no account lookup. Registration remained separate
and short-policy failures spent charges without creating users. Successful
sign-in/enrollment/setup, unavailable backup entry and session-derived password
reauthentication used exactly five charges for one actual identity.

The selector also proves inactive/invalid record, stale session commit,
cancellation after charge, checker failure and raw structural rejection. An
admission-result fault returns failure after an actual PostgreSQL commit: account
lookup cannot start and the next operation sees the spent charge. This is an
adapter result-loss fault, not a claim of database/network failure emulation.
An actual held capacity lock obeys an 80ms caller deadline without account lookup.

Affected credential/session/WebAuthn/fallback/factor/recovery and HTTP selectors
retain current proof requirements, atomic authority and MFA. Browser construction
compiles with the mandatory dependency; real browser journeys remain Slice 5.
The identity parser is unchanged from Slice 1's recorded fuzz evidence.

## Risks and Follow-ups

- Slice 3 owns finite common ingress and neutral public registration/failure
  responses. Slice 4 owns redacted observations; Slice 5 owns browser acceptance
  and the final evidence map. AUTH-07 remains partial until set closure.
- Applications own canonical identity, stable private namespace/key, consistent
  replica settings and bounded cleanup scheduling. No host key/configuration or
  external deployment was changed during this delivery.
- Temporary operation denial is not cumulative authenticator disabling/rebinding
  or all-factor-loss recovery. Physical-device and production assurance are not
  established by these focused fixture checks.
