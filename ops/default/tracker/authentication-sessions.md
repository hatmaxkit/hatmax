<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions Tracker

Date: 2026-10-03
Status: Proposed
Delivery set: authentication-sessions
Plan: [Delivery plan](../plan/authentication-sessions.md)
Concern: [Authentication and sessions](../spec/drafts/authentication-sessions.md)
Model: [Authentication and sessions model](../spec/drafts/authentication-sessions-model.md)
Parent: [Authentication security foundation](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `56b1cb05ea4243fbb87fe41ef3c7bb83b72e4fb5`
Active slice: None
Active tasks: None
Execution gate: Closed

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Secure session lifecycle | pending | `feat/auth-session-lifecycle` | `feat(slice-1): enforce secure session lifecycle` | pending | `ops/default/report/slices/authentication-sessions/slice-1-session-lifecycle.md` |
| Slice 2 | Required proof and outcomes | pending | `feat/auth-required-proof` | `feat(slice-2): enforce authentication proof requirements` | pending | `ops/default/report/slices/authentication-sessions/slice-2-required-proof.md` |
| Slice 3 | Reauthentication and control | pending | `feat/auth-session-control` | `feat(slice-3): add atomic session reauthentication and control` | pending | `ops/default/report/slices/authentication-sessions/slice-3-session-control.md` |

## Tasks

| Task | Status | Expected commit | Commit | Validation evidence required |
| --- | --- | --- | --- | --- |
| T1.1 | pending | `feat(auth): enforce secure session lifecycle` | pending | Stored/issued types, bounded secrets/lifetimes, coherent current-state validation and updated active consumers |
| T1.2 | pending | `test(auth): verify session lifecycle boundaries` | pending | Canonical parser fuzz, exact expiry/time/coalescing and real Postgres stale-state/rollback; AS-01/AS-02/AS-06/AS-09 |
| T2.1 | pending | `feat(auth): enforce required proof and authentication outcomes` | pending | Trusted policy revision, verified password facts, safe metadata and distinct non-authorizing outcomes |
| T2.2 | pending | `test(auth): verify proof outcome isolation` | pending | Password success, strong-policy denial, freshness/equality and pending/full isolation; AS-03/AS-04 |
| T3.1 | pending | `feat(auth): add atomic session reauthentication and control` | pending | Rotation/revocation/management/admission/cleanup with finite required storage contracts |
| T3.2 | pending | `test(auth): verify session control integration` | pending | Real concurrent rotation/revocation/state/capacity/rollback and consumer integration; AS-05 through AS-09 |

## Dependencies and Execution Gate

The credential-security prerequisite is delivered and verified in its
[completed tracker](credential-security.md). Parent AUTH-03/AUTH-04 behavior is
approved. This proposed concern/model and three-slice plan still require review
before activation; no runtime implementation or feature branch is authorized by
this tracker alone.

At approval, promote the concern/model together, fix direct links, record the
approved status and activate Slice 1/T1.1. Before the first implementation change,
settle exact Go/SQL bindings against the logical model. Before T1.2, record the
exact repository integration and token-fuzz test commands. Do not substitute
missing database evidence or future-method test fixtures for a passing result.

AUTH-05 is not a prerequisite for the supported password session path or for
rejecting unmet proof. It is a prerequisite for actual stronger-proof completion,
protocol challenge/factor consumption and an executable pending continuation.
No public proof-assertion callback or client assurance flag substitutes for it.

## Completion Gates

- [ ] Concern/model/plan/tracker reviewed and approved; Slice 1 activated.
- [ ] Slice 1 merged and delivered report/current-state evidence recorded.
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
