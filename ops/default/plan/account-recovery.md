<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Account Recovery Delivery Plan

Date: 2026-10-04
Status: Active
Approved: 2026-10-04
Delivery set: account-recovery
Slice strategy: behavior-first
Reason: mailbox verification, protected password change, mailbox reset and final
browser acceptance are independently reviewable increments with real adapters.
Concern: [Account lifecycle and password recovery](../spec/account-recovery.md)
Model: [Account recovery model](../spec/account-recovery-model.md)
Tracker: [Delivery tracker](../tracker/account-recovery.md)
Parent: [Authentication security](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `cd9647584a004d2dd0563fe915654ffd316abb8e`
Active slice: Slice 1 — Mailbox verification
Execution gate: Open
Go baseline: 1.27.1

## Approved Scope and Prerequisites

The account-recovery concern/model were approved on 2026-10-04. This internal
plan realizes that approved implementation scope. Credential security, sessions
and authenticators are delivered. Do not repeat their completed aggregate gates.
Each slice keeps active consumers compiling and has its own PR targeting dev.
After each verified merge, execute the next slice in this same set immediately.

Preserve established authenticators, replay state and required assurance.
All-factor-loss/operator identity proofing and broader distributed AUTH-07
infrastructure remain separate. No new core mail worker or outbox framework.
The adapter persists notification intent in the security transaction; the
application owns dispatch. No main alignment, release, tagging or deployment.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Expected report |
| --- | --- | --- | --- | --- |
| Slice 1 | Mailbox verification | `feat/account-mailbox-verification` | `feat(slice-1): add bounded mailbox verification` | `ops/default/report/slices/account-recovery/slice-1-mailbox-verification.md` |
| Slice 2 | Protected password change | `feat/account-password-change` | `feat(slice-2): enforce recent-proof password changes` | `ops/default/report/slices/account-recovery/slice-2-password-change.md` |
| Slice 3 | Mailbox password reset | `feat/account-password-reset` | `feat(slice-3): add one-use mailbox password reset` | `ops/default/report/slices/account-recovery/slice-3-password-reset.md` |
| Slice 4 | Recovery acceptance | `test/account-recovery-acceptance` | `test(slice-4): verify account recovery integration` | `ops/default/report/slices/account-recovery/slice-4-recovery-acceptance.md` |

## Slice 1: Mailbox Verification

Deliver actual issue/reissue and protected confirmation of the current address,
with one-use purpose-bound secrets, durable budgets and complete invalidation.
Do not expose reset/change operations before their protected implementation.

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Implement validated finite recovery configuration, token records/parser/digests, typed storage, mailbox issue/reserve/confirm/cleanup, notification intent and subject-first PostgreSQL transactions. Wire minimal production Ticked verification/mail dispatch and bounded ingress with no unmanaged worker; update affected callers/schema together. | `feat(auth): add bounded mailbox verification` |
| T1.2 | Prove token/subject/purpose/target/version/policy isolation, equality expiry, durable failure budgets, reissue/consume races, rollback, complete session/pending invalidation and mail/HTTP safety with real PostgreSQL. Update usable documentation and Unreleased. | `test(auth): verify mailbox lifecycle and transactions` |

AR-01/AR-02, affected AR-06 through AR-09 and consumer compilation in AR-10.
Settle exact Go/SQL/wire names together before runtime edits. Record finite
integration and parser fuzz selectors before validation; absent required database
settings fail rather than skip. Mail is captured in tests; no production sending.

## Slice 2: Protected Password Change

Deliver password change derived from a current session under actual recent proof,
with shared password policy/KDF and atomic revocation of every session/continuation.

| Task | Work | Expected commit |
| --- | --- | --- |
| T2.1 | Implement actor-bound authorization/admission and protected credential commit with current factor/proof/policy/time rechecks, monotonic version, all-session/pending invalidation and notification intent; add Ticked change form/handler and safe cookie clearing. | `feat(auth): enforce recent-proof password changes` |
| T2.2 | Prove ordinary no-factor and explicitly permitted MFA policies, phishing-resistant default, foreign/stale actor denial, post-lock freshness/factor checks, concurrent changes, checker/hash/admission failures and rollback with actual verification and PostgreSQL. Update supported guides. | `test(auth): verify password change authority` |

AR-03/AR-04 and affected AR-06 through AR-10. No actor retention or password-only
management bypass for an account with established factors.

## Slice 3: Mailbox Password Reset

Deliver issue and one-use reset only through a previously verified current mailbox,
with the same credential policy and transaction engine, preserving MFA.

| Task | Work | Expected commit |
| --- | --- | --- |
| T3.1 | Implement verified-mailbox reset issuance, reserved token verification before shared KDF work and atomic consume/credential/version/revocation/notice; add protected production Ticked reset forms and operator-authorized initiation without disclosing bearer/password. | `feat(auth): add one-use mailbox password reset` |
| T3.2 | Prove stale/expired/replaced/replayed/wrong-purpose reset rejection, no unverified/inactive destination, token/subject budget concurrency, change/reset race, transaction faults and preserved factor/replay/backup state with actual sign-in policy after reset. | `test(auth): verify password reset isolation and MFA` |

AR-01/AR-04/AR-05 and affected AR-06 through AR-10. Reset never authenticates,
activates, disables MFA or supplies all-factor-loss recovery.

## Slice 4: Recovery Acceptance

Deliver final production browser journeys and supported contract/evidence mapping,
after each underlying behavior has real storage/failure evidence.

| Task | Work | Expected commit |
| --- | --- | --- |
| T4.1 | Extend the existing bounded browser/PostgreSQL harness for mailbox verification, password change/reset and ordinary re-entry with required real MFA. Establish protected POST/GET, neutral request responses, cookie/token namespace isolation and lost-response/dispatch behavior. | `test(auth): exercise account recovery browser journeys` |
| T4.2 | Complete finite parser/deadline/cancellation/cleanup regression, API/config/User Guide contracts and AR-01 through AR-10 evidence; document application dispatch/anti-automation/assurance obligations in the final report. | `test(auth): close account recovery integration evidence` |

AR-08 through AR-10 and regression across AR-01 through AR-07. Captured mail and
virtual authenticators prove integration, not actual provider delivery, identity
proofing or production distributed protection.

## Validation and Exit Gates

Use focused tests for each task and the smallest relevant slice regression:

- `go test ./auth ./config ./model ./middleware`
- `go test -race ./auth ./config ./model ./middleware`
- `go test ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`
- `go test -run '^$' ./examples/...`
- `make source-license-check`
- `make vet`
- `make lint-strict` before every task/report commit
- `make docs-check`
- `git diff --check`

Record exact new PostgreSQL integration/race selectors with finite timeouts in
this tracker before each validation task. Include affected credential/session/
authenticator regression. Parser fuzz targets run 20 seconds with two workers,
60-second timeout and no password KDF/database work. Settle the actual browser
invocation, prerequisites and finite timeout before T4.1; missing prerequisites
are failures. Start/stop only an owned test cluster required for validation.

After all four verified merges, run `make check` once for the immutable integrated
dev candidate with an isolated owned PostgreSQL database. HatMax has no nightly;
this approved local final gate remains separate from tagged/browser evidence.
Do not run `make ci`. Corrections requiring repository changes use
`fix/account-recovery-validation`, focused checks and one PR to dev, then the exact
corrected integrated gate after merge. No correction branch for a green candidate.

Close only with four delivered reports, all eight task/commit/PR mappings,
AR-01 through AR-10 and the exact integrated gate. Mark only supported AUTH-06
coverage delivered; all-factor-loss and remaining AUTH-07 obligations remain open.
