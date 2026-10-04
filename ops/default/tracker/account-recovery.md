<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Account Recovery Tracker

Date: 2026-10-04
Status: Active
Approved: 2026-10-04
Delivery set: account-recovery
Plan: [Delivery plan](../plan/account-recovery.md)
Concern: [Account lifecycle and password recovery](../spec/account-recovery.md)
Model: [Account recovery model](../spec/account-recovery-model.md)
Parent: [Authentication security](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `cd9647584a004d2dd0563fe915654ffd316abb8e`
Active slice: Slice 2 — Protected password change
Active tasks: None
Execution gate: Open

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Mailbox verification | delivered | `feat/account-mailbox-verification` | `feat(slice-1): add bounded mailbox verification` | `#103` | `ops/default/report/slices/account-recovery/slice-1-mailbox-verification.md` |
| Slice 2 | Protected password change | reviewing | `feat/account-password-change` | `feat(slice-2): enforce recent-proof password changes` | pending | `ops/default/report/slices/account-recovery/slice-2-password-change.md` |
| Slice 3 | Mailbox password reset | pending | `feat/account-password-reset` | `feat(slice-3): add one-use mailbox password reset` | pending | `ops/default/report/slices/account-recovery/slice-3-password-reset.md` |
| Slice 4 | Recovery acceptance | pending | `test/account-recovery-acceptance` | `test(slice-4): verify account recovery integration` | pending | `ops/default/report/slices/account-recovery/slice-4-recovery-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit | Required evidence |
| --- | --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): add bounded mailbox verification` | `334046be21b4` | Actual core token flow and synchronized PostgreSQL/HTTP/mail/notice contracts |
| T1.2 | completed | `test(auth): verify mailbox lifecycle and transactions` | `77395af0a3c1` | Purpose/state isolation, time/budget/cleanup bounds, real races/rollback and transport safety |
| T2.1 | completed | `feat(auth): enforce recent-proof password changes` | `c0abf13f36e5` | Actual actor-derived proof, shared policy/KDF and atomic revocation/notification |
| T2.2 | completed | `test(auth): verify password change authority` | `ecf129548459` | Current actor/factor/policy/time rejection, concurrency and full rollback |
| T3.1 | pending | `feat(auth): add one-use mailbox password reset` | pending | Previously verified target, one-use token, credential/revocation/notice and no MFA bypass |
| T3.2 | pending | `test(auth): verify password reset isolation and MFA` | pending | Token and reset/change races, rollback, invalidation and real retained MFA/replay state |
| T4.1 | pending | `test(auth): exercise account recovery browser journeys` | pending | Actual production browser/PostgreSQL lifecycle and current-proof re-entry |
| T4.2 | pending | `test(auth): close account recovery integration evidence` | pending | Complete supported documentation, finite failure regression and acceptance mapping |

## Dependencies and Execution Gate

The concern/model and implementation scope were approved on 2026-10-04. This
internal four-slice map implements that scope. Credential/session/authenticator
prerequisites are delivered. Spec/model/plan/tracker agree; commit this planning
state before creating Slice 1's recorded branch and canonical worktree.

Before runtime edits settle matching Go/SQL/wire representations. Before T1.2
record finite real PostgreSQL and parser fuzz commands. Before T4.1 record the
production browser invocation and mandatory prerequisites. No fake store can
establish transaction semantics or production non-enumeration.

## Completion Gates

- [x] Concern/model behavior and implementation scope approved; four-slice map recorded.
- [x] Planning state committed and Slice 1 branch/worktree created from verified dev.
- [x] Slice 1 merged and report delivered; mailbox/token transaction evidence recorded.
- [ ] Slice 2 merged and report delivered; recent-proof credential change evidence recorded.
- [ ] Slice 3 merged and report delivered; one-use reset/MFA preservation evidence recorded.
- [ ] Slice 4 merged and report delivered; actual browser acceptance and supported guidance recorded.
- [ ] AR-01 through AR-10 mapped to exact evidence.
- [ ] Exact immutable integrated dev candidate passes `make check`.
- [ ] Supported AUTH-06 coverage and remaining lifecycle/AUTH-07/consumer obligations recorded.

## Planning Validation

- `make docs-check` — passed, including local links and example compilation.
- `make source-license-check` — passed: 816 headers, 109 annotations.
- `make lint-strict` — passed: zero issues.
- Spec/model approval, four exact branches/titles/report paths, eight tasks and
  active Slice 1 state agree across the delivery documents.

No runtime or final integrated evidence is claimed for the planning change.
Completed prerequisite gates are not repeated.

## Slice 1 Finite Validation Selectors

- `go test -tags=integration -race -run '^(TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestEnrollmentTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorChanges|TestFactorAuthority|TestSessionTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`
- `go test -tags=integration -race -run '^TestMailboxTransportTransactions$' -count=1 -timeout=90s ./examples/ticked/internal/web`
- `go test -run '^$' -fuzz '^FuzzRecoveryToken$' -fuzztime=20s -parallel=2 -timeout=60s ./auth`
- Explicit owned PostgreSQL connection settings are required. Mail is captured;
  no active external provider is used. Each database test isolates its schema.
- Transport tests cover production form handlers, trusted link origin,
  pre-lookup capacity, fixed acknowledgment timing and token-free GET handling.

## Slice 1 Focused Evidence

- Core/config/model/middleware and Ticked unit tests passed with race detection.
- Real PostgreSQL 18.6 lifecycle, denial budgets, concurrency, rollback, retention
  and actual WebAuthn preservation passed; affected session, credential,
  enrollment, WebAuthn, fallback and factor regression passed in 68.026s.
- Production mailbox transport with actual PostgreSQL and captured mail passed
  in 19.259s, including equal six-second initiation targets and notification retry.
- Final mailbox-specific transaction regression passed in 9.033s, including
  completion budget persistence, notification capacity rollback and cancellation.
- Canonical token parser fuzz passed for 20s with two workers: 439917 executions.
- `make source-license-check`, `make vet`, `make lint-strict`, `make docs-check`
  and whitespace checks passed. The full integrated gate remains after Slice 4.

## Slice 1 Merge and Slice 2 Activation

PR `#103` is canonically merged; its head and `origin/dev` agree at
`89376d9bbb4974a8873529586e4605e1dd6d4871`. Slice 1 is delivered. Slice 2
continues the approved delivery set; no password reset or final integration
gate is included in this unit.

## Slice 2 Finite Validation Selectors

- `go test -tags=integration -race -run '^(TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`
- `go test -tags=integration -race -run '^TestPasswordChangeTransportTransactions$' -count=1 -timeout=90s ./examples/ticked/internal/web`
- Actual isolated PostgreSQL schemas, the shared password verifier and real
  authenticator proof are required. Mail is captured without external sending.

## Slice 2 Focused Evidence

- Actual core password checker/verifier and protected transport unit tests passed
  with race detection, including shared KDF admission and complete Unicode input.
- Real PostgreSQL 18.6 transaction regression passed in 78.652s using the recorded
  selector. Password-change authority cases include no-factor permission, actual
  UV WebAuthn, explicit TOTP/backup MFA, foreign/stale actor snapshots, post-lock
  factor/generation/time checks and expiry after a slow final write.
- Concurrent changes produce one winner. Eight admitted requests share one actual
  KDF slot; rejected hashes retain their committed kind-3 attempts. Checker error,
  cancellation, candidate rejection, notification capacity and fault injection
  at every final mutation boundary preserve atomic rollback without refunds.
- Mailbox verification and established factor/replay/backup state remain; consumed
  backups stay consumed and normal required-MFA authentication works afterward.
- Production PostgreSQL/form/captured-mail transport passed in 1.396s. GET and
  forbidden requests cannot mutate. Failed notification delivery preserves the
  completed change and intent; success clears the cookie without issuing access.
- The full integrated gate and complete browser journeys remain after the
  corresponding approved slices; no provider delivery result is claimed.
