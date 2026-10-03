<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Credential Integration

Status: reviewing
Delivery set: credential-security
Plan: [Credential security plan](../../../plan/credential-security.md)
Tracker: [Credential security tracker](../../../tracker/credential-security.md)
Branch: `feat/credential-auth-integration`
PR: `#94`

## Purpose

Integrate AUTH-01/AUTH-02 credential primitives into reusable signup/sign-in
and persistent storage contracts. Concurrent credential/account changes must
invalidate stale proof before a session can be committed.

## Delivered Behavior

- Signup checks the complete NFC-normalized candidate before creating a salted
  Argon2id record. Password-only access requires at least 15 code points.
- Persistent email uniqueness resolves simultaneous signup to one account and
  one classified `ErrEmailTaken` result, without a pre-insert lookup.
- Sign-in verifies the supported encoded format and distinguishes mismatch
  from invalid records, input, resource admission and context failures.
- Final session creation atomically rechecks the active account and expected
  authentication version. An intervening credential or activation change rejects
  stale proof with `ErrCredentialChanged`.
- Conditional password replacement stores one complete record, increments the
  version and revokes all sessions in one transaction. Competing requests using
  the same version cannot both succeed; failures roll back every change.
- Startup validates policy, deadlines, KDF creation/acceptance costs and shared
  concurrency. Superseded bcrypt APIs and configuration are removed.

## Implementation Notes

`auth.Service` owns one immutable policy/verifier configuration and shared KDF
admission budget. The default password-work timeout is five seconds; the checker
has a two-second limit and an earlier caller deadline wins. Context is checked
around synchronous derivation; Argon2 cannot be interrupted and occupies its
admission slot until completion. Default verifier costs remain the measured
64 MiB, three iterations, four lanes and two active operations from Slice 2.

Storage returns owned snapshots with positive, monotonic `AuthVersion`. The
Ticked adapter locks the user row for both conditional session creation and
password replacement. Activation and role updates increment the version. Email
unique-constraint violations alone map to `ErrEmailTaken`; other persistence
failures remain operating errors. Database operations stay outside `model`.

`ReplacePassword` is a required storage primitive. Authorization and candidate
policy belong to its caller; this slice adds no password-change, reset or
recovery service workflow. The example's bounded checker is a documented finite
common/application-password demonstration, not a compromised-password corpus.
Production applications must select and document their own checking source.

The initial example schema and generated query consumers are updated together.
No legacy credential reader, compatibility interface or historical-data migration
is introduced. Existing session tokens/TTL behavior remains outside this set.

## Contracts Added or Changed

- `NewService(queries, cfg, checker, logger)` returns a service or construction
  error and requires a caller-owned checker.
- `User.AuthVersion` and `CredentialState` identify the verified snapshot.
- Required `Queries.CreateSession` receives expected credential state; required
  `Queries.ReplacePassword` implements conditional replacement and revocation.
- `AuthConfig.PasswordSettings()` validates a startup snapshot; new policy,
  work-timeout and Argon2 settings replace `auth.bcrypt_cost`.
- `model.HashPassword`, `HashPasswordWithCost` and `ComparePassword` are removed;
  active service callers use `PasswordVerifier`.

## Files of Interest

- [Service and storage contracts](../../../../../auth/service.go)
- [Validated startup settings](../../../../../config/password.go)
- [Transactional example adapter](../../../../../examples/ticked/internal/feat/auth/queries.go)
- [Real persistence tests](../../../../../examples/ticked/internal/feat/auth/credential_integration_test.go)
- [Authentication reference](../../../../../docs/reference/authentication/README.md)
- [Configuration reference](../../../../../docs/reference/configuration/README.md)

## Validation

Focused checks used Go 1.27.1, golangci-lint 2.12.2 built with that toolchain
and PostgreSQL 18.6. The persistence test creates/drops isolated schemas in an
explicitly configured test database; `DB_HOST` is required and absence fails.
`DB_PORT`, `DB_USER`, `DB_NAME` and `DB_PASSWORD` supply connection settings.
Tests use 19 MiB/two iterations/one lane to bound test cost, while configuration
checks cover the production defaults and rejection of narrowing/aggregate limits.

```sh
export TMPDIR="$PWD/.tmp/build" GOTMPDIR="$PWD/.tmp/build" GOFLAGS=-p=2
```

- `go test ./auth ./model ./crypto ./config` — passed.
- `go test -race ./auth ./model ./crypto ./config` — passed.
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -count=1 -run '^TestCredentialTransactions$' ./examples/ticked/internal/feat/auth` — passed with real PostgreSQL. Exercises simultaneous Unicode signup, competing replacements, mutation after proof, row-lock serialization with disable/re-enable and forced transaction rollback.
- `go test ./generator/...` — passed.
- `go test -run '^$' ./examples/...` — passed.
- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed; zero issues.
- `make docs-check` — passed, including example compilation.
- `git diff --check` — passed.

The full `make check` gate runs once against the exact integrated `dev` candidate
after this slice merges. These focused checks do not substitute for that gate.

## Risks and Follow-ups

- Downstream adapters must implement the required transaction/version contracts
  and increment authentication state on every relevant account mutation.
- Applications must document checker coverage and choose deployment-specific
  request/attempt and KDF budgets; cancellation is not a hard KDF deadline.
- This credential foundation does not deliver AUTH-03 through AUTH-07 or a
  complete authentication assurance level. Sessions/challenges, MFA/WebAuthn,
  recovery and broader attempt controls require later bounded work.
- Merge and integrated delivery-set validation remain pending. No release,
  main alignment, tag or deployment is included.
