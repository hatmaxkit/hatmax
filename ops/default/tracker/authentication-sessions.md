<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions Tracker

Date: 2026-10-03
Status: Completed
Approved: 2026-10-03
Completed: 2026-10-04
Delivery set: authentication-sessions
Plan: [Delivery plan](../plan/authentication-sessions.md)
Concern: [Authentication and sessions](../spec/authentication-sessions.md)
Model: [Authentication and sessions model](../spec/authentication-sessions-model.md)
Parent: [Authentication security foundation](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `56b1cb05ea4243fbb87fe41ef3c7bb83b72e4fb5`
Active slice: None
Active tasks: None
Execution gate: Closed

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Secure session lifecycle | delivered | `feat/auth-session-lifecycle` | `feat(slice-1): enforce secure session lifecycle` | `#95` | `ops/default/report/slices/authentication-sessions/slice-1-session-lifecycle.md` |
| Slice 2 | Required proof and outcomes | delivered | `feat/auth-required-proof` | `feat(slice-2): enforce authentication proof requirements` | `#96` | `ops/default/report/slices/authentication-sessions/slice-2-required-proof.md` |
| Slice 3 | Reauthentication and control | delivered | `feat/auth-session-control` | `feat(slice-3): add atomic session reauthentication and control` | `#97` | `ops/default/report/slices/authentication-sessions/slice-3-session-control.md` |

## Tasks

| Task | Status | Expected commit | Commit | Validation evidence required |
| --- | --- | --- | --- | --- |
| T1.1 | completed | `feat(auth): enforce secure session lifecycle` | `7b8d950beed02aaf2dcdd25c8da785173b93c13d` | Stored/issued types, bounded secrets/lifetimes, coherent current-state validation and updated active consumers |
| T1.2 | completed | `test(auth): verify session lifecycle boundaries` | `61b65a17cc5a693d85a328b0d78e48862c83ceff` | Canonical parser fuzz, exact expiry/time/coalescing and real Postgres stale-state/rollback; AS-01/AS-02/AS-06/AS-09 |
| T2.1 | completed | `feat(auth): enforce required proof and authentication outcomes` | `05046a930635cd39cce8bd5249af14f6906bf3b8` | Trusted policy revision, verified password facts, safe metadata and distinct non-authorizing outcomes |
| T2.2 | completed | `test(auth): verify proof outcome isolation` | `9be7080cbe6be8d23d11ba6571573d0347ca9bcb` | Password success, strong-policy denial, freshness/equality and pending/full isolation; AS-03/AS-04 |
| T3.1 | completed | `feat(auth): add atomic session reauthentication and control` | `5933e5440f973defd813c41757137c79150b9f0b` | Rotation/revocation/management/admission/cleanup with finite required storage contracts |
| T3.2 | completed | `test(auth): verify session control integration` | `69d95585859d4a8faf235fa6d55cb176838c9766` | Real concurrent rotation/revocation/state/capacity/rollback and consumer integration; AS-05 through AS-09 |

## Dependencies and Execution Gate

The credential-security prerequisite is delivered and verified in its
[completed tracker](credential-security.md). Parent AUTH-03/AUTH-04 behavior is
approved. The concern/model/three-slice plan and tracker were approved on
2026-10-03; all three slices are delivered and the integrated full gate passed
on 2026-10-04. There is no active slice or task.

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
- [x] Slice 2 merged and delivered report/proof-outcome evidence recorded.
- [x] Slice 3 merged and delivered report/session-control evidence recorded.
- [x] AS-01 through AS-09 covered by appropriate deterministic/race/persistence evidence.
- [x] Exact integrated `dev` candidate passes `make check`.
- [x] AUTH-04 coverage, partial AUTH-03 and remaining authenticator/consumer obligations recorded.

## Planning Validation

- `make docs-check` — passed, including example compilation.
- `make source-license-check` — passed.
- `git diff --check` — passed.
- Direct links, spec/model boundaries, slice branches/titles/report paths, task
  commits and closed execution gates — consistent.
- New concern/model/plan/tracker contain no outer-product references.

No runtime implementation tests or full delivery-set gate were claimed for this
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
At Slice 1 completion, the delivery-set `make check` gate was deferred until all
three slices merged; its final result is recorded below.

Slice 1 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/95. Merge on
2026-10-03 was verified at `b8012ff89551b5027a9686d42ed3ddf5ba89495b` on canonical dev.
Slice 2 is activated under the same approved set; full validation remains after
all three merges.

## Slice 2 Validation Bindings

Before T2.2, the exact real PostgreSQL selector is:

- `go test -tags=integration -race -count=1 -run '^Test(ProofTransactions|SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth`

This establishes actual password completion, stronger-policy no-issuance,
revision/proof checks before activity and freshness evaluated after lock waits.
Deterministic unit tests cover exact age equality, unsupported/corrupt methods,
future times and malformed requirements; transport tests reject every
non-completed result before setting cookies.

T2.2 focused package/race tests, Ticked tests, example compilation, PostgreSQL
proof/session/credential transaction tests, tagged/default strict lint, source
licensing, vet and documentation checks passed. The post-lock tests reject both
stale issuance proof and stale protected-operation proof without a row/activity
change. Pending/denied transport fixtures create no cookie or continuation.
At Slice 2 completion, the full delivery-set gate was deferred until Slice 3
merged; its final result is recorded below.

Slice 2 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/96. Report and tracker
are delivered. Merge on 2026-10-03 was verified at
`77b9188b4601e3e2faf2fd2a2ec42ee16c9a1664` on canonical dev; Slice 3 is activated.

## Slice 3 Validation Bindings

Before T3.2, the exact real PostgreSQL selector is:

- `go test -tags=integration -race -count=1 -timeout=60s -run '^Test(ControlTransactions|ProofTransactions|SessionTransactions|CredentialTransactions)$' ./examples/ticked/internal/feat/auth`

Control evidence must establish actual password rotation, competing generations,
revocation/account mutation serialization, post-lock expiry, forced partial-write
rollback, bounded subject pages/recent proof and concurrent retained-row admission.
Focused checks do not substitute for the integrated `make check` after Slice 3 merge.

T3.1 focused package tests, tagged integration compilation, licensing, vet, strict
lint and documentation checks passed. T3.2 owns the recorded real transaction
selector and remaining behavioral evidence.

T3.2 passed focused tests/race, the recorded real PostgreSQL transaction selector,
tagged/default strict lint, licensing, vet and documentation/example compilation.
Control tests establish one winner for competing actual password rotations,
post-lock old-session/replacement-proof expiry, validation waiting on committed
rotation, revocation/password/disable serialization, forced rotation/deletion/
reclamation rollback, owned bounded pages, recent-proof checks after locks and
concurrent retained-row admission. HTTP tests establish completed-only replacement
cookies and truthful revocation/sign-out failure. No real MFA completion is claimed.
The full `make check` passed after verified Slice 3 merge, as recorded below.

Slice 3 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/97. Report and tracker
are delivered. Merge on 2026-10-04 at 00:15:23 Europe/Warsaw was verified at
`93a992e2b258aabfafe3465be4846f64ec164333` on canonical dev.

## Integrated Validation

Candidate: `93a992e2b258aabfafe3465be4846f64ec164333`
Gate: `make check` — passed once on 2026-10-04 with Go 1.27.1,
golangci-lint 2.12.2 and an isolated real PostgreSQL 18.6 test database.
Connection settings were supplied through the repository's `DB_HOST`, `DB_PORT`,
`DB_USER`, `DB_PASSWORD` and `DB_NAME` variables. Build settings were
`TMPDIR=$PWD/.tmp/build`, `GOTMPDIR=$PWD/.tmp/build` and `GOFLAGS=-p=2`.

The gate ran source licensing (760 headers, 104 content-preserving annotations),
formatting, vet, the complete default test suite, coverage and strict lint.
Coverage was 84.8% (required: 80%); lint reported zero issues. Tracked files
remained unchanged after formatting. The owned database was stopped afterward.
No repository correction or validation-correction branch was required.

This is local integrated evidence. Optional tagged/live acceptance targets and
production evidence are separate. The real PostgreSQL transaction/race selectors
above and the three delivered reports establish the session concurrency and
rollback contracts; they are not included implicitly in the default full suite.

## Acceptance Evidence

| Criterion | Delivered evidence |
| --- | --- |
| AS-01 | Slice 1's independent 32-byte secrets, canonical parser, 643311 fuzz executions, digest-only persistence and safe context; Slice 2's no-secret/no-row pending outcomes |
| AS-02 | Slice 1's exact absolute/inactivity equality, validated settings, coalesced relevant activity and immutable proof/absolute clocks; Slice 3's post-lock old-session expiry |
| AS-03 | Slice 2's actual password completion and rejection of unmet stronger requirements/enrollment claims; completed-only HTTP cookies |
| AS-04 | Slice 2's revision, closed method and exact freshness checks; Slice 3's post-lock replacement/management proof rejection |
| AS-05 | Slice 3's actual password rotation/replay, one competing-generation winner, entropy failure and forced commit rollback |
| AS-06 | Credential and session transaction tests for password replacement, disable/re-enable and revocation races; Slice 3's blocked stale touch cannot change the replacement |
| AS-07 | Slice 3's concurrent retained-row cap, bounded expired-row reclamation and owned keyset pages; Slice 1's explicit cancellable bounded cleanup |
| AS-08 | Slice 3's actor-derived subject, foreign-target rejection, finite scopes and recent-proof management after locks |
| AS-09 | Required Ticked PostgreSQL operations and HTTP/template tests, safe middleware metadata, example compilation and complete default regression suite |

## Requirement Evidence and Handoff

| Requirement | Delivered evidence | Remaining consumer or later-work obligation |
| --- | --- | --- |
| AUTH-03 | Trusted per-operation requirements/revisions, actual password facts, safe completed-session metadata, isolated pending/denied outcomes and fail-closed stronger requirements | Partial: AUTH-05 must deliver actual stronger verification, current authenticator checks and atomic factor consumption/session completion before executable pending flows |
| AUTH-04 | Independent CSPRNG bearers, digest storage, validated absolute/inactivity policy, atomic rotation, current-account/version checks, own-session revocation/control and bounded admission/activity/cleanup | Consumers implement required atomic storage contracts and trusted policy; browser presentation, administrative authorization, protected domain rechecks and complete assurance assessment remain application evidence |

The authentication-sessions set is complete for its approved password/session
boundary. AUTH-05, broader AUTH-06/AUTH-07 behavior and consuming-application
account/workspace delivery remain separate. Select a published dependency before
independent consumer validation. This closure authorizes no next concern,
main alignment, release, tagging or deployment.
