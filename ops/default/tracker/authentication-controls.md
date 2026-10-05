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
Active slice: Slice 3 — Finite public authentication
Execution gate: Open
Plan: [Delivery plan](../plan/authentication-controls.md)
Active tasks: T3.1, T3.2

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Durable credential admission | delivered | `feat/credential-admission` | `feat(slice-1): add shared credential admission` | `#107` | `ops/default/report/slices/authentication-controls/slice-1-credential-admission.md` |
| Slice 2 | Guarded password entry | delivered | `feat/guarded-password-entry` | `feat(slice-2): guard password authentication paths` | `#108` | `ops/default/report/slices/authentication-controls/slice-2-password-entry.md` |
| Slice 3 | Finite public authentication | active | `feat/authentication-ingress` | `feat(slice-3): bound public authentication ingress` | pending | `ops/default/report/slices/authentication-controls/slice-3-public-authentication.md` |
| Slice 4 | Security observations | pending | `feat/authentication-events` | `feat(slice-4): observe bounded authentication outcomes` | pending | `ops/default/report/slices/authentication-controls/slice-4-security-events.md` |
| Slice 5 | Controls acceptance | pending | `test/authentication-controls-acceptance` | `test(slice-5): verify authentication controls integration` | pending | `ops/default/report/slices/authentication-controls/slice-5-controls-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): add shared credential admission` | `f32729cd06e0` |
| T1.2 | completed | `test(auth): verify credential admission transactions` | `c1ed78985aba` |
| T2.1 | completed | `feat(auth): guard password authentication paths` | `dd425e7a244d` |
| T2.2 | completed | `test(auth): verify shared password entry budgets` | `59ddb46fc792` |
| T3.1 | active | `feat(auth): bound public authentication ingress` | pending |
| T3.2 | pending | `test(auth): verify neutral public authentication` | pending |
| T4.1 | pending | `feat(auth): observe bounded security outcomes` | pending |
| T4.2 | pending | `test(auth): verify security observation boundaries` | pending |
| T5.1 | pending | `test(auth): exercise authentication controls journeys` | pending |
| T5.2 | pending | `test(auth): close authentication controls evidence` | pending |

## Completion Gates

- [x] Scope/model and internal slice map approved.
- [x] Planning state committed and Slice 1 canonical worktree activated.
- [x] Slice 1 merged and report delivered; actual durable admission evidence.
- [x] Slice 2 merged and report delivered; actual shared password-entry budgets.
- [ ] Slice 3 merged and report delivered; finite ingress/neutral transport.
- [ ] Slice 4 merged and report delivered; redacted bounded observations.
- [ ] Slice 5 merged and report delivered; actual browser/affected regression.
- [ ] All ten task/commit/PR mappings and AC-01 through AC-08 recorded.
- [ ] Exact immutable integrated candidate passes `make check`.
- [ ] Supported AUTH-07 and remaining lifecycle/consumer boundaries recorded.

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
