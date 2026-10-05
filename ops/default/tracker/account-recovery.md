<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Account Recovery Tracker

Date: 2026-10-05
Status: Active
Approved: 2026-10-04
Delivery set: account-recovery
Plan: [Delivery plan](../plan/account-recovery.md)
Concern: [Account lifecycle and password recovery](../spec/account-recovery.md)
Model: [Account recovery model](../spec/account-recovery-model.md)
Parent: [Authentication security](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `cd9647584a004d2dd0563fe915654ffd316abb8e`
Active slice: Slice 4 — Recovery acceptance
Active tasks: None
Execution gate: Open

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Mailbox verification | delivered | `feat/account-mailbox-verification` | `feat(slice-1): add bounded mailbox verification` | `#103` | `ops/default/report/slices/account-recovery/slice-1-mailbox-verification.md` |
| Slice 2 | Protected password change | delivered | `feat/account-password-change` | `feat(slice-2): enforce recent-proof password changes` | `#104` | `ops/default/report/slices/account-recovery/slice-2-password-change.md` |
| Slice 3 | Mailbox password reset | delivered | `feat/account-password-reset` | `feat(slice-3): add one-use mailbox password reset` | `#105` | `ops/default/report/slices/account-recovery/slice-3-password-reset.md` |
| Slice 4 | Recovery acceptance | active | `test/account-recovery-acceptance` | `test(slice-4): verify account recovery integration` | pending | `ops/default/report/slices/account-recovery/slice-4-recovery-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit | Required evidence |
| --- | --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): add bounded mailbox verification` | `334046be21b4` | Actual core token flow and synchronized PostgreSQL/HTTP/mail/notice contracts |
| T1.2 | completed | `test(auth): verify mailbox lifecycle and transactions` | `77395af0a3c1` | Purpose/state isolation, time/budget/cleanup bounds, real races/rollback and transport safety |
| T2.1 | completed | `feat(auth): enforce recent-proof password changes` | `c0abf13f36e5` | Actual actor-derived proof, shared policy/KDF and atomic revocation/notification |
| T2.2 | completed | `test(auth): verify password change authority` | `ecf129548459` | Current actor/factor/policy/time rejection, concurrency and full rollback |
| T3.1 | completed | `feat(auth): add one-use mailbox password reset` | `cd3fbcb6f893` | Previously verified target, one-use token, credential/revocation/notice and no MFA bypass |
| T3.2 | completed | `test(auth): verify password reset isolation and MFA` | `8c3f9cb54565` | Token and reset/change races, rollback, invalidation and real retained MFA/replay state |
| T4.1 | completed | `test(auth): exercise account recovery browser journeys` | `b599c7722b43` | Actual production browser/PostgreSQL lifecycle and current-proof re-entry |
| T4.2 | completed | `test(auth): close account recovery integration evidence` | `52096c68eef6` | Complete supported documentation, finite failure regression and acceptance mapping |

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
- [x] Slice 2 merged and report delivered; recent-proof credential change evidence recorded.
- [x] Slice 3 merged and report delivered; one-use reset/MFA preservation evidence recorded.
- [ ] Slice 4 merged and report delivered; actual browser acceptance and supported guidance recorded.
- [x] AR-01 through AR-10 mapped to exact evidence, with integrated AR-10 gate explicitly pending.
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

## Slice 2 Merge and Slice 3 Activation

PR `#104` is canonically merged at `f742484acb7984bb3167a2971fbd269cd1f81fad`.
Slice 2 is delivered. Slice 3 continues the approved set with one-use reset,
previously verified current mailbox and complete MFA/replay preservation.

## Slice 3 Finite Validation Selectors

- `go test -tags=integration -race -run '^(TestPasswordResetTransactions|TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`
- `go test -tags=integration -race -run '^TestPasswordResetTransportTransactions$' -count=1 -timeout=120s ./examples/ticked/internal/web`
- Actual isolated PostgreSQL, shared password verification, captured mail and
  production protected transports are required. Full browser acceptance and
  the one integrated gate remain the final approved slice/set boundaries.


## Slice 3 Focused Evidence

T3.1 and T3.2 are implemented. Actual PostgreSQL races and failures establish
one-use purpose isolation, durable admission, one-winner reset/change/reissue,
post-lock eligibility and expiry checks, complete transaction rollback and all
session/continuation invalidation. Actual WebAuthn, TOTP and backup authentication
preserves counters, accepted steps and consumed codes, with current-policy sign-in
required after reset. Production HTTP/captured-mail tests establish neutral
six-second initiation, protected explicit completion, no session grant, retained
notification intent and actual operator role/proof/version checks.

The finite affected authentication regression passed in 94.370s and production
HTTP evidence passed in 43.693s. Supported core/reference/example/User Guide
contracts now include reset. The full integrated gate and browser acceptance
remain the final approved boundaries; external provider delivery is unproven.


## Slice 3 Merge and Slice 4 Activation

PR `#105` is canonically merged at `15d867fa56c4b8c84bbac7f4cd7399c1ae7eceb1`.
Slice 3 is delivered. Slice 4 continues the approved set with actual browser
recovery journeys, final finite regressions and AR-01 through AR-10 mapping.

## Slice 4 Finite Validation Selectors

- `go test -tags=browser -race -run '^TestAccountRecoveryBrowser$' -count=1 -timeout=300s ./examples/ticked/internal/web`
- Browser prerequisites are mandatory: Node with native WebSocket support,
  Chromium with CDP/virtual WebAuthn support, explicit `NODE_BIN`, `CHROMIUM_BIN`,
  `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` (may be empty), and `DB_NAME`.
  Use an owned isolated PostgreSQL fixture and captured mail; no external send.
  The browser child deadline is 240 seconds; CDP calls and owned-profile cleanup
  remain finite. Missing prerequisites fail the test rather than skip it.
- `go test -tags=integration -race -run '^(TestPasswordResetTransactions|TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`
- `go test -tags=integration -race -run '^(TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/web`
- `go test -run '^$' -fuzz '^FuzzRecoveryToken$' -fuzztime=20s -parallel=2 -timeout=60s ./auth`
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- The single full integrated `make check` remains after all four verified merges.


### Slice 4 Boundary Regression Selector

Before T4.2, extend the finite adapter selector with `TestRecoveryBoundaries`:
`go test -tags=integration -race -run '^(TestRecoveryBoundaries|TestPasswordResetTransactions|TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`.
This covers an earlier caller deadline while the subject lock is held, exact
owned-revision lease release and finite cross-purpose cleanup after retention,
without fabricated authentication or refunded admission.


## Slice 4 Browser Evidence

T4.1 passed the recorded actual Chromium/PostgreSQL browser selector in 75.007s.
The composite fixture uses explicit finite authenticator subject admission of 40;
recovery budgets and ingress defaults remain intact. The journey waits the real
one-minute ingress window before final lost-response/replay checks. Actual forms,
foreign-origin submission, namespace isolation, required-MFA re-entry, retained
backup consumption, operator proof/role and failed dispatch/notice retry passed.

The browser exposed native no-referrer navigation carrying `Origin: null`.
Recovery forms now submit an explicit same-origin fetch with the supported media
type, a ten-second client deadline and no automatic replay. Existing middleware
still rejects opaque/foreign sources. A dropped committed response retains an
invalid old cookie; a consumed token cannot repeat the mutation.


## Acceptance Evidence Map

| Criterion | Exact focused evidence | Remaining boundary |
| --- | --- | --- |
| AR-01 | `TestRecoveryToken`, `TestMailboxRecord`, `TestResetTokenPurpose`, `FuzzRecoveryToken`; `TestPasswordResetTransactions/admission`; purpose/domain/subject snapshot cases | No public bearer; transient mail fixture remains private |
| AR-02 | `TestMailboxTransactions/complete_invalidation` and `preserves_actual_MFA`; `TestMailboxTransportTransactions`; `TestAccountRecoveryBrowser` verification journey | No sign-in/activation authority from mailbox proof |
| AR-03 | `TestPasswordChangeTransactions` current actor/proof/factor/generation/post-lock cases; `TestAccountRecoveryBrowser` actual MFA change/weak-proof denial | Trusted application policy determines acceptable management proof |
| AR-04 | `TestPasswordChangeTransactions` policy/checker/cancellation/shared KDF cases; `TestPasswordResetTransactions/admission`, `durable_budgets`, `failures`; shared verifier unit/race tests | Finite KDF admission is instance-local |
| AR-05 | `TestPasswordResetTransactions/preserved_MFA` actual WebAuthn/TOTP/backup authentication; `TestAccountRecoveryBrowser` post-reset password/MFA and spent/unused backup journeys | All-factor-loss identity proofing remains separate |
| AR-06 | `TestMailboxTransactions` races/rollback; `TestPasswordChangeTransactions`; `TestPasswordResetTransactions/races`, `failures`, `final_checks` | Ambiguous commit errors require operating review |
| AR-07 | `TestMailboxRecord` expiry equality; tagged mailbox/reset budget/expiry cases; `TestRecoveryBoundaries` deadline/owned release/retained cleanup; ingress unit/HTTP/browser cases | Applications invoke cleanup and deployment-wide admission |
| AR-08 | `TestMailboxTransportTransactions`, `TestPasswordChangeTransportTransactions`, `TestPasswordResetTransportTransactions`; `TestAccountRecoveryBrowser` actual forms, fragment/GET, foreign origin and namespace isolation | Timing policy is not a production side-channel audit |
| AR-09 | `TestMailboxDeliveryTransactions`; change/reset failure/notice cases; actual reset operator HTTP role/proof/version tests; browser dispatch/retry/operator/lost-response journey | Captured mail is not external delivery; retry may duplicate notices |
| AR-10 | `make docs-check`, source licensing, vet, strict lint, example compilation, public/core/config/User Guide contracts; all focused selectors above | The exact immutable integrated dev `make check` remains after Slice 4 merge |

The map records all ten criteria while distinguishing focused evidence from the
pending integrated gate. Final delivery requires the fourth verified merge and
that one exact-candidate full check; this table does not close the set early.


## Slice 4 Focused Closure

Both Slice 4 tasks are implemented and the recorded finite selectors passed.
Actual browser and HTTP evidence include the production form correction;
parser fuzzing, caller-deadline/owned-lease/retained-cleanup and affected
credential/session/authenticator regressions passed. Supported API/configuration,
example and User Guide contracts identify the same guarantees and application
obligations. The acceptance map above covers AR-01 through AR-10 and explicitly
retains the integrated gate as pending.

After the fourth verified merge, mark its report delivered and finalize one
immutable integrated dev candidate before `make check`. Do not run that full gate
on this unmerged slice or close AUTH-06/the delivery set early. If it fails and a
repository correction is required, use the already recorded correction branch
and merge procedure; a green candidate needs no correction branch.
