<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions Tracker

Date: 2026-10-03
Status: Active
Approved: 2026-10-03
Delivery set: authentication-sessions
Plan: [Delivery plan](../plan/authentication-sessions.md)
Concern: [Authentication and sessions](../spec/authentication-sessions.md)
Model: [Authentication and sessions model](../spec/authentication-sessions-model.md)
Parent: [Authentication security foundation](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `56b1cb05ea4243fbb87fe41ef3c7bb83b72e4fb5`
Active slice: Slice 2
Active tasks: T2.1
Execution gate: Open

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Secure session lifecycle | delivered | `feat/auth-session-lifecycle` | `feat(slice-1): enforce secure session lifecycle` | `#95` | `ops/default/report/slices/authentication-sessions/slice-1-session-lifecycle.md` |
| Slice 2 | Required proof and outcomes | active | `feat/auth-required-proof` | `feat(slice-2): enforce authentication proof requirements` | pending | `ops/default/report/slices/authentication-sessions/slice-2-required-proof.md` |
| Slice 3 | Reauthentication and control | pending | `feat/auth-session-control` | `feat(slice-3): add atomic session reauthentication and control` | pending | `ops/default/report/slices/authentication-sessions/slice-3-session-control.md` |

## Tasks

| Task | Status | Expected commit | Commit | Validation evidence required |
| --- | --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): enforce secure session lifecycle` | `7b8d950beed02aaf2dcdd25c8da785173b93c13d` | Stored/issued types, bounded secrets/lifetimes, coherent current-state validation and updated active consumers |
| T1.2 | completed | `test(auth): verify session lifecycle boundaries` | `61b65a17cc5a693d85a328b0d78e48862c83ceff` | Canonical parser fuzz, exact expiry/time/coalescing and real Postgres stale-state/rollback; AS-01/AS-02/AS-06/AS-09 |
| T2.1 | pending | `feat(auth): enforce required proof and authentication outcomes` | pending | Trusted policy revision, verified password facts, safe metadata and distinct non-authorizing outcomes |
| T2.2 | pending | `test(auth): verify proof outcome isolation` | pending | Password success, strong-policy denial, freshness/equality and pending/full isolation; AS-03/AS-04 |
| T3.1 | pending | `feat(auth): add atomic session reauthentication and control` | pending | Rotation/revocation/management/admission/cleanup with finite required storage contracts |
| T3.2 | pending | `test(auth): verify session control integration` | pending | Real concurrent rotation/revocation/state/capacity/rollback and consumer integration; AS-05 through AS-09 |

## Dependencies and Execution Gate

The credential-security prerequisite is delivered and verified in its
[completed tracker](credential-security.md). Parent AUTH-03/AUTH-04 behavior is
approved. The concern/model/three-slice plan and tracker were approved on
2026-10-03; Slice 1 is delivered and Slice 2/T2.1 is active.

The concern/model are promoted together and direct links updated. Before the first implementation change,
settle exact Go/SQL bindings against the logical model. Before T1.2, record the
exact repository integration and token-fuzz test commands. Do not substitute
missing database evidence or future-method test fixtures for a passing result.

AUTH-05 is not a prerequisite for the supported password session path or for
rejecting unmet proof. It is a prerequisite for actual stronger-proof completion,
protocol challenge/factor consumption and an executable pending continuation.
No public proof-assertion callback or client assurance flag substitutes for it.

## Completion Gates

- [x] Concern/model/plan/tracker reviewed and approved; Slice 1 activated.
- [x] Slice 1 merged and delivered report/current-state evidence recorded.
- [ ] Slice 2 merged and delivered report/proof-outcome evidence recorded.
- [ ] Slice 3 merged and delivered report/session-control evidence recorded.
- [ ] AS-01 through AS-09 covered by appropriate deterministic/race/persistence evidence.
- [ ] Exact integrated `dev` candidate passes `make check`.
- [ ] AUTH-04 coverage, partial AUTH-03 and remaining authenticator/consumer obligations recorded.

## Planning Validation

- `make docs-check` — passed, including example compilation.
- `make source-license-check` — passed.
- `git diff --check` — passed.
- Direct links, spec/model boundaries, slice branches/titles/report paths, task
  commits and closed execution gates — consistent.
- New concern/model/plan/tracker contain no outer-product references.

No runtime implementation tests or full delivery-set gate are claimed for this
planning proposal. The completed credential-security gate is not rerun.

## Slice 1 Validation Bindings

T1.1 focused compilation/tests, source licensing, vet and strict lint passed.
Before T1.2, the selected commands are:

- `go test -tags=integration -race -count=1 -run '^Test(SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth`
- `go test ./auth -run '^$' -fuzz '^FuzzSessionToken$' -fuzztime=20s -parallel=2`

Integration requires an externally supplied real PostgreSQL connection. Tests
cover stored digests, current-state invalidation, post-lock time, activity rollback
and bounded cleanup. Fuzzing exercises only the bounded canonical bearer parser.

T1.2 passed the selected integration/race command against PostgreSQL 18.6 and
643311 token-parser fuzz executions in 20 seconds. Focused package/race tests,
Ticked tests, example compilation, licensing, vet, strict lint and documentation
checks passed. Tagged integration lint also passed with zero issues. No generator
runtime caller uses the changed session API; no generator files were changed.
The delivery-set `make check` gate remains after all three slices merge.

Slice 1 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/95. Merge on
2026-10-03 was verified at `b8012ff89551b5027a9686d42ed3ddf5ba89495b` on canonical dev.
Slice 2 is activated under the same approved set; full validation remains after
all three merges.
