<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Controls Tracker

Date: 2026-10-05
Status: Active
Approved: 2026-10-05
Delivery set: authentication-controls
Concern: [Authentication controls](../spec/authentication-controls.md)
Model: [Controls model](../spec/authentication-controls-model.md)
Parent: [Authentication security](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `9070714609a4dc323d1a1b5991ed7cb806c11677`
Active slice: Slice 5 — Controls acceptance
Execution gate: Open
Plan: [Delivery plan](../plan/authentication-controls.md)
Active tasks: None; both Slice 5 tasks completed, maintainer review pending

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Durable credential admission | delivered | `feat/credential-admission` | `feat(slice-1): add shared credential admission` | `#107` | `ops/default/report/slices/authentication-controls/slice-1-credential-admission.md` |
| Slice 2 | Guarded password entry | delivered | `feat/guarded-password-entry` | `feat(slice-2): guard password authentication paths` | `#108` | `ops/default/report/slices/authentication-controls/slice-2-password-entry.md` |
| Slice 3 | Finite public authentication | delivered | `feat/authentication-ingress` | `feat(slice-3): bound public authentication ingress` | `#109` | `ops/default/report/slices/authentication-controls/slice-3-public-authentication.md` |
| Slice 4 | Security observations | delivered | `feat/authentication-events` | `feat(slice-4): observe bounded authentication outcomes` | `#110` | `ops/default/report/slices/authentication-controls/slice-4-security-events.md` |
| Slice 5 | Controls acceptance | reviewing | `test/authentication-controls-acceptance` | `test(slice-5): verify authentication controls integration` | pending | `ops/default/report/slices/authentication-controls/slice-5-controls-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): add shared credential admission` | `f32729cd06e0` |
| T1.2 | completed | `test(auth): verify credential admission transactions` | `c1ed78985aba` |
| T2.1 | completed | `feat(auth): guard password authentication paths` | `dd425e7a244d` |
| T2.2 | completed | `test(auth): verify shared password entry budgets` | `59ddb46fc792` |
| T3.1 | completed | `feat(auth): bound public authentication ingress` | `24b86323236e` |
| T3.2 | completed | `test(auth): verify neutral public authentication` | `fd0c0a41e80e` |
| T4.1 | completed | `feat(auth): observe bounded security outcomes` | `466894a7ba57` |
| T4.2 | completed | `test(auth): verify security observation boundaries` | `10d48427296d` |
| T5.1 | completed | `test(auth): exercise authentication controls journeys` | `85e88174f0c8` |
| T5.2 | completed | `test(auth): close authentication controls evidence` | pending |

## Completion Gates

- [x] Scope/model and internal slice map approved.
- [x] Planning state committed and Slice 1 canonical worktree activated.
- [x] Slice 1 merged and report delivered; actual durable admission evidence.
- [x] Slice 2 merged and report delivered; actual shared password-entry budgets.
- [x] Slice 3 merged and report delivered; finite ingress/neutral transport.
- [x] Slice 4 merged and report delivered; redacted bounded observations.
- [ ] Slice 5 merged and report delivered; actual browser/affected regression.
- [ ] All ten task/commit/PR mappings and AC-01 through AC-08 recorded.
- [ ] Exact immutable integrated candidate passes `make check`.
- [x] Supported AUTH-07 and remaining lifecycle/consumer boundaries recorded.

## Slice 1 Recorded Validation

Use the exact integration and parser-fuzz commands in the approved plan. Database
prerequisites fail when absent. Compare two actual adapter/service instances with
the same namespace/key; prove independent registration purpose, exact final
admission, durable restart/no-reset, capacity, rollback, bounded cleanup and held
lock caller deadlines. Measure finite row/index costs and admission latency;
results describe this fixture, not deployment throughput. Core tests may use a
bounded fake only for orchestration, never for distributed semantics.

The remaining slices use their recorded task scope. No new slice is inferred
from an implementation finding; maintain plan/model/tracker consistency first.
The full integrated gate remains after five verified canonical merges.

## Slice 1 Activation

Planning commit `989c4bbca7b0` is clean and pushed on dev. The recorded
`feat/credential-admission` branch is active in its canonical worktree. Matching
config/Go/SQL representations were settled in the approved companion model
before runtime edits; private key/namespace have no outer-product justification.

## Slice 1 Focused Evidence

The recorded Go 1.27.1 race, real PostgreSQL 18.6 admission selector and finite
identity-parser fuzz run passed on 2026-10-05. Two actual service instances
admitted exactly ten of 24 concurrent proof requests. Separate registration,
restart/key binding, captured policy, capacity, trusted retirement, bounded
cleanup, renewal races, trigger fault rollback and caller lock deadlines passed.
No user rows were manufactured. Exact commands and fixture storage costs belong
to the Slice 1 report. AC-02/AC-03 have admission-layer evidence; their password
entry and transport boundaries remain for later recorded slices.

Licensing, vet, strict lint and documentation checks passed. The full integrated
`make check` gate remains after five canonical merges.

Slice 1 PR #107 is canonically merged into dev at `abb2ea6afe99`; its report
is delivered. Both task commits and focused evidence are recorded. Slice 2
is active under the approved continuation map.

## Slice 2 Focused Evidence

`TestPasswordEntryBudgets` and the exact affected adapter selector passed with
Go 1.27.1, actual PostgreSQL 18.6 and race detection in 103.857s. Twenty-four
requests across two sign-in instances, initial enrollment, TOTP setup and fallback
shared four allowed proof operations; denial performed no account lookup.
Successful proof and session-derived reauthentication, registration/checker
failure, inactive/invalid/stale state, cancellation, lost admission result and
caller lock deadlines retained their actual charges without manufactured users.

The affected recovery HTTP selector passed in 62.668s; six-package race, licensing,
vet, strict lint, documentation and browser consumer compilation passed. The
selector names/commands are in the plan and Slice 2 report. AC-01/AC-02/AC-03 have
password-entry evidence; neutral transport, observation and actual browser
acceptance remain in Slices 3–5. Full integrated validation remains after all
five canonical merges.

Slice 2 PR #108 is canonically merged into dev at `1ae69f0ddb49`; its report
is delivered. Both task commits and focused evidence are recorded. Slice 3
is active under the approved continuation map.

## Slice 3 Focused Evidence

The recorded actual HTTP/PostgreSQL selector passed with race detection in
98.956s; the affected credential/password-entry adapter selector passed in
3.937s. Actual requests retained active capacity through acknowledgment,
rejected malformed/oversized/ambiguous forms before account work, shared
canonical trusted peer limits across routes, denied unknown peers at capacity,
bounded a stalled body and released waits on caller/server cancellation. Close
rejected admissions before infrastructure cancellation.

Two concurrent registration commands produced one user and one classified
uniqueness conflict, identical neutral navigation and no session or cookie.
Missing/inactive/wrong/throttled and operating password failures shared one
six-second response target. Successful sign-in retained a committed secure
cookie; restricted enrollment/setup retained their challenge contracts. Safe
candidate feedback, database-error redaction and authentication URI log isolation
passed. Six-package race, the 20s two-worker peer parser fuzz run, browser
construction compilation and required repository checks passed.

AC-04/AC-05/AC-06 have focused ingress/transport evidence; AC-05 browser
evidence remains for Slice 5. Slice 4 owns terminal typed
observations; Slice 5 owns real browser journeys, final evidence mapping and
post-merge integrated validation. Timing describes the bounded fixture and
response policy, not production constant-time assurance. Full `make check`
remains after all five canonical merges.

Slice 3 PR #109 is canonically merged into dev at `26613b55a897`; its report
is delivered. Slice 4 is active under the approved continuation map. Both task commits and focused
checks are recorded; its PR links immutable report introduction `4ab28ddd0bc4`.
Slice 3 is closed on dev; recorded Slice 4 is executing. No additional scope, main alignment or release is authorized.

## Slice 4 Implementation Evidence

T4.1 implements one required typed observer shared by all actual authentication
services, closed bounded events, trusted private operation facts and post-defers
synchronous delivery. Finite non-waiting callback slots, caller-shorter deadlines,
fixed atomic diagnostics, late-success rejection and re-entry preserve authority.
The application supplies its observer; discard is explicit and has no audit
assurance. Actual adapter/browser consumer construction compiles.

Go 1.27.1 six-package race, source licensing, vet, strict lint and documentation
checks passed. Configuration bounds, callback saturation/re-entry/cancellation
and finite shape fuzz seeds passed. Actual PostgreSQL provenance and retained
security authority remain the recorded T4.2 task. Full integrated validation
remains after five canonical merges.

## Slice 4 Focused Evidence

Actual PostgreSQL adapter/security regression passed in 103.026s; malformed
assertion provenance refinement passed in 24.170s. HTTP/application observer and
affected recovery regression passed in 80.693s. All 26 public entrypoints have
trusted committed/pending/proof outcome evidence, released subject locks and
no duplicate internal helper event. Real saturation, failed/late delivery,
reentrant signout and lost acknowledgment preserve actual authority.

Six-package race, 20s two-worker event fuzz (300449 executions), browser consumer
compilation and scoped integration lint passed. AC-07 has focused real evidence;
AC-08's browser/integrated gate remains Slice 5 after canonical merge.

Final source licensing (888 headers, 112 annotations), vet, strict lint and
documentation checks passed. The owned PostgreSQL cluster is stopped after
zero other clients; actual browser journeys and `make check` remain the approved
Slice 5/integrated boundary. Both Slice 4 tasks are complete; PR #110 targets dev and links immutable
report introduction `10d48427296d`. Maintainer merge is pending. After verified
merge, close Slice 4 on dev and immediately activate recorded Slice 5 browser/
controls acceptance without another go-ahead.

Slice 4 PR #110 is canonically merged into dev at `e7a9c36e172b`; its
report is delivered. Both task mappings and focused checks are recorded.
Slice 5 is active under the approved continuation map; actual browser journeys
and final affected evidence precede its maintainer merge. The full immutable
integrated gate remains after all five canonical merges.

## Slice 5 Focused Evidence

Both recorded actual Chromium 151.0.7922.173/Node v26.8.1/PostgreSQL 18.6 browser
journeys passed with Go 1.27.1 and race detection in 152.503s. Registration races,
no automatic session, safe candidate feedback, malformed ingress, two-slot active
capacity, final shared password charge, equal neutral replies and retained actual
MFA/recovery passed. The authenticator/recovery journeys checked respectively
44 and 80 unique terminal observations and timely delivery. Fixture setup used
actual failed password operations, never manufactured authentication.

The recorded actual adapter selector passed in 122.724s and actual HTTP selector
in 117.721s. Six-package race, required source-license/vet/lint/docs checks and
tagged browser lint passed. These are finite affected checks, not the full gate.
The final slice report owns the exact selectors, public documentation agreement,
AC mapping and immutable-candidate preparation.

## Requirement Evidence

| ID | Evidence | Remaining boundary |
| --- | --- | --- |
| AC-01 | `Service.Signin`, `Service.Reauthenticate`, `AuthenticatorService.BeginWebAuthnEnrollment`, `FallbackService.BeginTOTPSetup`/`BeginFallbackAuthentication`/`BeginFallbackStepUp` all reach one shared admission before actual password verification; `TestPasswordEntryBudgets`, credential/fallback/authority selectors | Application canonical identity and alias convergence; signup and recovery/factor purposes remain distinct |
| AC-02 | `TestCredentialAdmissionTransactions` and `TestPasswordEntryBudgets`: actual shared instances, last admission, capacity, restart/key binding, no refunds/resets and captured policy; browser final charge | Exact boundary helper evidence is deterministic; distributed transactions use real SQL |
| AC-03 | Admission/password-entry selectors: unknown/inactive identities, malformed input, canceled/ambiguous storage, stale state, policy tightening, capacity, cleanup renewal/rollback and held locks; browser malformed form and exhaustion | Application stable private namespace/key and cleanup scheduling |
| AC-04 | Rate-limit/ingress unit/race plus `TestAuthenticationIngressTransactions`: canonical/trusted peers, bounded maps/active work, cleanup, caller/server cancellation and close ownership; browser two admitted waits and third refusal | Common ingress is process-local; deployment-wide anti-automation is consumer-owned |
| AC-05 | `TestCredentialTransactions`, `TestPublicAuthenticationTransactions`, `TestAuthenticationObservationTransactions` and both browser journeys: one winner/conflict, equal neutral navigation, no automatic session/cookie, candidate-only feedback and redaction | Local fixture timing is not production side-channel assurance |
| AC-06 | Password-entry and actual HTTP selectors: missing/inactive/wrong/throttled/operating neutral policy, accepted cookie/challenges, caller cancellation/capacity; browser missing/wrong/exhausted replies and real MFA | Setup flags grant no proof; consumers select current trusted access policy |
| AC-07 | `TestSecurityObservationTransactions`, actual HTTP logger and both browser journeys: committed/pending/denied/unknown provenance, actual methods, no lock-held callbacks, saturation, late/failing delivery and real re-entry; valid unique redacted capture | Best-effort cooperative observation is not durable audit delivery |
| AC-08 | Recorded adapter/HTTP/race/browser selectors retain invalidation, token/replay/backup consumption and actual proof; core/config/example/User Guide and docs checks agree | Exact immutable integrated dev `make check` remains after final maintainer merge |

## Password Verification Inventory

The four actual password-verifier call sites are `auth/service.go` (sign-in),
`auth/session_control.go` (reauthentication), `auth/enrollment.go` (initial
WebAuthn enrollment) and `auth/fallback.go` (shared password helper for TOTP setup,
fallback authentication and actor step-up). Each commits `CredentialPasswordProof`
before its password verifier; public identity paths do so before account lookup,
while reauthentication resolves the owned actor first. Helper invocations charge
once. The separate fallback backup-record verifier spends its factor budget;
signup hashes under registration admission. New-password checking/hash in
change/reset remains under the delivered recovery authority and budgets.

## Integrated Candidate Preparation

After the verified maintainer merge of Slice 5, close its delivered report/task/PR
mapping on dev and finalize one documentary candidate commit. Record its full
immutable hash and verify clean local dev equals origin/dev before `make check`.
Run the single full gate against that exact commit with mandatory actual
PostgreSQL and bounded scratch. Formatting must leave tracked files unchanged;
record the exact candidate, coverage threshold/result and checks. Documentation
that only records the result does not define another runtime gate candidate.

A required repository correction follows the approved
`fix/authentication-controls-validation` branch and one focused dev PR; run the
full gate on the corrected integrated candidate after its verified merge. A green
candidate needs no correction branch. Supported AUTH-07 resource admission,
neutral errors and observation do not implement cumulative authenticator
disabling/rebinding, all-factor-loss identity proofing or consumer audit/deployment
assurance. The parent remains Partial.
