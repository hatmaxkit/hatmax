<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Reauthentication and Control

Status: delivered
Delivery set: authentication-sessions
Plan: [Delivery plan](../../../plan/authentication-sessions.md)
Tracker: [Delivery tracker](../../../tracker/authentication-sessions.md)
Branch: `feat/auth-session-control`
PR: `#97`
Merged: 2026-10-04
Integration: `93a992e2b258aabfafe3465be4846f64ec164333`

## Purpose

Implement AS-05 through AS-09 with regression of AS-01 through AS-04:
actual password reauthentication, atomic secret rotation, recent-proof
self-service management and bounded retained-session admission. Complete the
approved password/session contracts without claiming a stronger authenticator.

## Delivered Behavior

- Reauthentication repeats actual password verification for a live current
  session. It can refresh old password proof, while policy revision, supported
  method, account state and both expiries still apply. Stronger current policy
  rejects password-only state; expired sessions require a new sign-in.
- Rotation preserves record ID and creation time, replaces the digest and
  increments generation once. It refreshes password authentication/activity and
  applicable lifetime clocks atomically. Failed proof, entropy or storage work
  returns no new secret and preserves the old valid session. After commit the
  old bearer cannot validate, rotate or revoke its replacement.
- Session listing and selected/current/other/all revocation derive ownership
  from the revalidated actor bearer. Storage checks current recent proof after
  locks; foreign selected IDs cannot terminate another subject's session.
  Pages contain safe metadata, current ID and an opaque bounded next cursor.
- Admission reclaims at most 100 expired subject rows and counts all retained
  rows under the insertion subject lock. Capacity rejects without eviction;
  concurrent inserts cannot exceed the selected limit. Revocation removes rows
  and explicit cleanup remains cancellable and batch-bounded.
- Ticked provides a Sessions screen, password reauthentication and self-service
  revocation. Only completed committed rotation replaces the secure cookie.
  Failed revocation/sign-out does not report success or clear the cookie.
  API/configuration references, how-to/User Guide and Unreleased match the APIs.

## Implementation Notes

Rotation checks captured account version, old digest/generation, policy and old
expiry against one trusted post-lock clock, then checks replacement proof against
that same clock. Proof time is captured before entropy work. The conditional
SQL update replaces all mutable fields together; commit failure returns no
successful result. Management locks subject before sessions, with stable record
ID ordering for multi-record deletion, and evaluates freshness after waits.

The keyset query stays scoped to the actor subject, orders by record ID and fetches
at most page size plus one. Pages reflect current transaction state rather than
one historical snapshot across requests. Expired/stale retained rows can be
listed for termination but cannot authorize an operation. Public page metadata
contains neither digest nor bearer. Cursors confer no permission.

Real PostgreSQL tests use actual core password-verifier results for rotations.
A deferred constraint trigger blocks a real rotation at commit; a relevant touch
waits behind it and then rejects the old digest without changing the replacement.
Other triggers force rotation/deletion/reclamation failures and prove full rollback.
Admission tests isolate the storage cap with records based on actual password
session facts; they do not establish any unsupported method's acceptance.

SQL source, relevant SQLC output and the initial subject index changed together.
Unrelated generated output/dependencies were excluded. No active generator caller
uses the changed API and no generator source changed. Bearer-only sign-out
remains current-secret withdrawal; recent-proof management independently governs
selecting/listing sessions. Applications retain cross-subject administrative
policy ownership; the self-service APIs accept no target subject or claimed proof.

## Contracts Added or Changed

- `Reauthenticate(ctx, token, password, requirement)` returns the existing typed
  authentication result only after supported proof and committed rotation.
  Required `RotateSession` storage receives expected account state, current
  digest/generation, replacement record and trusted requirement.
- `CreateSession` also receives the configured retained-row admission cap.
  `ErrSessionCapacity` classifies exhaustion; `ErrSessionGeneration` classifies
  stale/exhausted generations. Unknown selection/cursor values fail before IO.
- `ListSessions(ctx, token, requirement, cursor)` returns `SessionPage`.
  `RevokeSessions(ctx, token, requirement, selection)` returns a deletion count.
  Required storage methods revalidate current actor and recent proof atomically.
  Selections are finite current/selected/others/all; selected IDs are at most
  128 printable ASCII bytes. Canonical raw URL Base64 cursors are at most 128 bytes.
- `session_recent_proof_age` defaults to 5m and is 1s through absolute lifetime,
  in whole microseconds. Management uses the tighter operation/configured age.
  `session_max_per_subject` defaults to 10 and `session_page_size` to 50; each is
  bounded at 1 through 100. Invalid configuration fails construction.

## Files of Interest

- [Core control APIs](../../../../../auth/session_control.go) and
  [service/control tests](../../../../../auth/session_control_test.go).
- [Required PostgreSQL operations](../../../../../examples/ticked/internal/feat/auth/session_control.go)
  and [control transaction tests](../../../../../examples/ticked/internal/feat/auth/control_integration_test.go).
- [Ticked transport](../../../../../examples/ticked/internal/web/sessions.go) and
  [transport/template tests](../../../../../examples/ticked/internal/web/sessions_test.go).
- [Session settings](../../../../../config/session.go) and
  [authentication reference](../../../../../docs/reference/authentication/README.md).

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
- `go test -tags=integration -race -count=1 -timeout=60s -run '^Test(ControlTransactions|ProofTransactions|SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth` — passed; actual rotation/replay, one concurrent winner, revocation/password/disable serialization, old-session/replacement-proof and management freshness after locks, stale touch rejection, forced full rollback, bounded owned pages and concurrent admission, plus prior proof/lifecycle/credential regression.
- `golangci-lint run --build-tags integration --fix --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./auth/... ./config/... ./examples/ticked/internal/feat/auth/... ./examples/ticked/internal/web/...` — passed, zero issues.
- `git diff --check` — passed.

Deterministic service/configuration tests establish closed selection/cursor bounds,
management policy tightening, generation exhaustion, entropy/password failure and
earlier-deadline/late-success rejection. HTTP tests establish completed-only
secure replacement cookies, truthful failure and production-template rendering.
The parser is unchanged; another fuzz campaign was not required.

AS-05 maps to rotation/replay/competing generation and rollback tests; AS-06 to
mutation races and blocked stale touch; AS-07 to admission, page and prior bounded
cleanup tests; AS-08 to subject isolation and post-lock recent-proof management;
AS-09 to the required Ticked storage/transport and example compilation. Prior
proof, lifecycle and credential transaction regressions retain AS-01 through AS-04.

The integrated full `make check` gate passed once on 2026-10-04 for exact `dev`
candidate `93a992e2b258aabfafe3465be4846f64ec164333`, after verified PR #97 merge.
It ran source licensing, formatting, vet, the complete default test suite,
coverage and strict lint with an isolated real PostgreSQL 18.6 test database.
Coverage was 84.8% (required: 80%); lint reported zero issues. Formatting left
tracked files unchanged. The owned database was stopped after the gate.
Optional tagged/live acceptance targets remain separate; the focused transaction
and race evidence above supplies the session persistence guarantees.
The [completed tracker](../../../tracker/authentication-sessions.md#requirement-evidence-and-handoff)
maps all acceptance criteria and the bounded AUTH-04 foundation coverage.

## Risks and Follow-ups

Custom stores/callers must adopt the required rotation/management/admission
contracts; no optional upgrade interface or historical-data migration is supplied.
Applications own trusted current policy selection, clocks and any cross-subject
administrative authorization. Protected domain writes retain their own required
current-state recheck; a validated snapshot is not indefinite authority.

AUTH-03 remains partial until AUTH-05 delivers actual stronger-authenticator
verification and atomic factor consumption. Enrollment flags and password rotation
cannot satisfy MFA or phishing resistance. All three slices are delivered and
the exact integrated full gate passed. Browser presentation, production policy,
administrative authorization and complete assurance evidence remain application
obligations. Main alignment, release, tagging and deployment retain separate authority.
