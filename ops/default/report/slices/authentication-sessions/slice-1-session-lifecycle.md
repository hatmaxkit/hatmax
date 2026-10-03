<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Secure Session Lifecycle

Status: delivered
Delivery set: authentication-sessions
Plan: [Delivery plan](../../../plan/authentication-sessions.md)
Tracker: [Delivery tracker](../../../tracker/authentication-sessions.md)
Branch: `feat/auth-session-lifecycle`
PR: `#95`
Merged: 2026-10-03
Integration: `b8012ff89551b5027a9686d42ed3ddf5ba89495b`

## Purpose

Implement AS-01/AS-02 session secrets and lifecycle validation on the existing
password-authenticated path. Account mutation and lock waits must not authorize
stale or expired sessions. This slice provides the affected AS-06/AS-09 storage,
activity and consumer obligations; proof policies and session management remain
in the next two approved slices.

## Delivered Behavior

- Issuance uses independent 32-byte cryptographic secrets. Strict fixed-size
  padded URL Base64 parsing precedes a purpose-separated SHA-256 lookup.
  Entropy failures issue/persist no session. Storage holds only a 32-byte digest.
- Safe metadata captures positive auth version/generation, authentication,
  creation/activity times, absolute expiry and bounded inactivity duration.
  The raw bearer exists only in the immediate issuance value and cookie.
- Validation serializes user then session locks, checks active/current account
  version and evaluates `clock_timestamp()` after both waits. Equality at
  absolute/inactivity expiry fails; expired sessions cannot be touched live.
- Trusted activity is explicit. Background validation does not update activity;
  concurrent relevant touches coalesce without changing authentication time or
  absolute expiry. Returned user/session snapshots are owned.
- Construction rejects malformed/out-of-range lifetime, cadence and work
  settings. Earlier caller deadlines win; late authorized results are discarded.
  Cleanup deletes one batch of at most 1000 expired rows and returns its count.
- Core middleware, role middleware and Ticked carry safe session/user context.
  API/configuration references, affected how-to/User Guide and Unreleased reflect
  the replacement contracts.

## Implementation Notes

The required adapter validates account state under the credential mutation lock,
then acquires the session lock by digest. The preliminary read only discovers the
subject; it cannot authorize the request. Final state and time are read after
waits. SQL and Go bounds use whole microseconds; conversion bounds stored duration
integers before multiplying into `time.Duration`.

Activity writes commit before returning validated state. A deferred PostgreSQL
constraint trigger forces a commit failure in the test, proving rollback and
absence of an authorized result. Eight concurrent validations produce one
activity write. A separate test holds the session lock until inactivity expires,
then proves that the waiting request rejects without refreshing activity.

The example initial schema and matching SQLC output were replaced together.
Unrelated generated audit/todo changes were excluded; no dependency was added.
No active generator caller uses this session API and no generator source changed.

## Contracts Added or Changed

- `Session` is safe metadata; `SessionRecord` adds `SessionDigest`,
  `IssuedSession` adds the transient token and `ValidatedSession` contains current
  user/session snapshots.
- `Signin` returns `*IssuedSession`. Service/middleware validation requires
  trusted `SessionActivity`; `NoActivity` and `RelevantActivity` are the only
  supported values.
- Required storage receives digests, validates/touches atomically and revokes
  only the presented digest. The plaintext getter and ID-based sign-out are gone.
  Cleanup requires a finite batch and returns a deletion count.
- Session defaults are 24h absolute, 30m inactivity, 1m persistence cadence,
  5s storage timeout and 1000 cleanup rows. Cadence is 1s through 5m and at most
  one quarter of inactivity; lifetimes are 1m through 30 days.

## Files of Interest

- [Session types/parser](../../../../../auth/session.go) and
  [service](../../../../../auth/service.go).
- [Lifecycle configuration](../../../../../config/session.go).
- [PostgreSQL adapter](../../../../../examples/ticked/internal/feat/auth/queries.go)
  and [transaction tests](../../../../../examples/ticked/internal/feat/auth/session_integration_test.go).
- [Authentication reference](../../../../../docs/reference/authentication/README.md).

## Validation

All commands passed with Go 1.27.1 and golangci-lint 2.12.2. Build temporary
storage used local `.tmp/build` with `GOFLAGS=-p=2`. Integration used an isolated
PostgreSQL 18.6 schema; connection settings were supplied externally. The owned
local database was stopped after validation.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test ./auth ./config ./crypto ./model ./middleware` — passed.
- `go test -race ./auth ./config ./crypto ./model ./middleware` — passed.
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -run '^$' ./examples/...` — passed.
- `go test -tags=integration -race -count=1 -run '^Test(SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth` — passed; current-state serialization, post-lock expiry, digest-only storage, owned snapshots, concurrent activity, commit rollback, bounded idle/absolute cleanup and credential regression.
- `golangci-lint run --build-tags integration --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./examples/ticked/internal/feat/auth/...` — passed, zero issues.
- `go test ./auth -run '^$' -fuzz '^FuzzSessionToken$' -fuzztime=20s -parallel=2` — passed, 643311 executions.
- `git diff --check` — passed.

The full `make check` gate remains scheduled for the exact integrated `dev`
candidate after all three slices merge. No full-set result is claimed here.

## Risks and Follow-ups

Custom stores/callers must migrate to the new required APIs and digest schema;
there is no plaintext-token reader or historical schema migration. Conservative
activity coalescing may expire a session up to one persistence interval early.
Current trusted application/database clocks are required; future stored times
fail closed rather than being clamped.

Slice 2 supplies trusted requirement/revision and verified password facts.
Slice 3 supplies atomic reauthentication/rotation, management, admission and
revocation. This password-only lifecycle is not MFA or phishing-resistant proof;
TOTP enrollment/setup flags cannot supply either. AUTH-03 remains partial and
real authenticator completion retains AUTH-05 ownership. Main alignment,
release, tags and deployment are separate authority.
