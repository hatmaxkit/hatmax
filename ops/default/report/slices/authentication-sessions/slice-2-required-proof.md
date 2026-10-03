<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Required Proof and Outcomes

Status: reviewing
Delivery set: authentication-sessions
Plan: [Delivery plan](../../../plan/authentication-sessions.md)
Tracker: [Delivery tracker](../../../tracker/authentication-sessions.md)
Branch: `feat/auth-required-proof`
PR: `#96`

## Purpose

Implement AS-03/AS-04 and affected AS-01/AS-09 obligations: explicit trusted
access requirements, actual verified password facts and non-authorizing
outcomes when stronger proof is unavailable. Enrollment and account roles must
never substitute for authentication proof.

## Delivered Behavior

- Sign-in completes only the supported password requirement. MFA requirements
  return pending enrollment or pending proof according to enrollment state;
  phishing-resistant MFA returns denied. Unmet requirements produce no bearer,
  entropy request, stored session or executable continuation.
- Safe session metadata includes the policy revision and actual password
  verification time. Protected operations require a matching current revision,
  a supported proof method and the selected freshness bound. Equality at the
  maximum age rejects; activity never refreshes authentication proof.
- PostgreSQL evaluates proof and policy after account/session locks and before
  activity writes. Proof that expires while issuance or validation waits for a
  lock rejects without inserting a session or updating activity.
- Service and middleware callers select explicit trusted requirements. Ticked
  owns its password-only policy and sets cookies only for completed results.
  Pending, denied and inconsistent result values render an unavailable-method
  failure without a cookie, redirect or continuation endpoint.
- API references, affected how-to/User Guide, example guidance and Unreleased
  describe the current supported behavior. The crypto reference now matches
  the existing padded URL Base64 token encoding.

## Implementation Notes

The password verifier produces proof facts only after successful verification.
The required storage adapter rechecks account state and the same requirement
before committing issuance. Existing classified credential errors return no
result; verifier/storage failures remain operating errors.

Database constraints permit only the supported password method and ordered
creation/proof/authentication times. The adapter rejects unknown integer methods
before narrowing them to the public enum, including values that would otherwise
wrap to the password method. Revision and proof failures precede activity writes.

Real PostgreSQL tests hold the account or session lock until actual password
proof becomes stale, then verify rejection and unchanged persistence. Transport
tests also supply contradictory result values to establish that only completed,
satisfied results can issue a cookie. These fixtures do not establish MFA proof.
SQL source and relevant SQLC output changed together; unrelated generated changes
and dependencies were excluded. No active generator caller uses the changed API.

## Contracts Added or Changed

- `AccessRequirement` contains a finite required proof profile, an explicit
  policy revision of 1 through 128 printable ASCII bytes and `MaxAge`. Zero age
  disables freshness; otherwise the bound is 1s through absolute lifetime in
  whole microseconds. Invalid requirements fail before credentials or storage.
- `VerifiedProof` contains a supported method and verification time. Password
  is the only supported verified method. Safe `Session` adds proof and revision;
  enrollment flags cannot create stronger facts.
- `Signin` receives a requirement and returns `AuthenticationResult`. Outcomes
  are completed, pending enrollment, pending proof and denied; reasons are
  satisfied or method unavailable. `CompletedSession` guards transport issuance.
- Required `CreateSession` and `ValidateSession` storage operations receive the
  requirement; service and auth/role middleware validation require it explicitly.
  There is no public proof-assertion completion API, pending secret or table.

## Files of Interest

- [Proof policies and outcomes](../../../../../auth/proof.go),
  [service](../../../../../auth/service.go) and
  [proof tests](../../../../../auth/proof_test.go).
- [PostgreSQL adapter](../../../../../examples/ticked/internal/feat/auth/queries.go)
  and [proof transaction tests](../../../../../examples/ticked/internal/feat/auth/proof_integration_test.go).
- [Ticked transport tests](../../../../../examples/ticked/internal/web/handler_test.go).
- [Authentication reference](../../../../../docs/reference/authentication/README.md).

## Validation

Focused checks passed with Go 1.27.1 and golangci-lint 2.12.2. Build temporary
storage used local `.tmp/build` with `GOFLAGS=-p=2`. Integration used an isolated
PostgreSQL 18.6 schema with externally supplied connection settings. The owned
local database was stopped after validation.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test ./auth ./config ./crypto ./model ./middleware` — passed.
- `go test -race ./auth ./config ./crypto ./model ./middleware` — passed.
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -run '^$' ./examples/...` — passed through `make docs-check`.
- `go test -tags=integration -race -count=1 -run '^Test(ProofTransactions|SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth` — passed; password facts, stronger-policy no-issuance, policy/freshness before activity, post-lock issuance/validation expiry and lifecycle/credential regression.
- `golangci-lint run --build-tags integration --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./examples/ticked/internal/feat/auth/...` — passed, zero issues.
- `git diff --check` — passed.

Deterministic tests establish exact freshness equality, revision mismatch,
invalid requirements, unsupported/corrupt methods, future times and outcome
isolation. No parser change required another fuzz campaign. The full `make check`
gate remains after all three slices merge; no full-set result is claimed here.

## Risks and Follow-ups

Custom stores/callers must adopt the required policy/proof APIs and schema; no
historical-data migration is supplied. Applications own current policy selection.
Middleware captures its trusted requirement at wiring time; policy changes must
update that wiring or use current per-request service validation.

AUTH-03 remains partial. AUTH-05 owns actual stronger-proof verification and
atomic factor consumption before any executable pending continuation. Slice 3
supplies atomic reauthentication/rotation, management, admission and revocation.
