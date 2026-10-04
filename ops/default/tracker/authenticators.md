<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authenticator Tracker

Date: 2026-10-04
Status: Active
Approved: 2026-10-04
Delivery set: authenticators
Plan: [Delivery plan](../plan/authenticators.md)
Concern: [Authenticator proposal](../spec/authenticators.md)
Model: [Authenticator model](../spec/authenticators-model.md)
Parent: [Authentication security foundation](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `247487f273a54a8e65a8e6b5f86191f38bcb0be8`
Active slice: Slice 3
Active tasks: T3.1
Execution gate: Open

## Slice Status

| Slice | Short name | Status | Branch | PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Restricted enrollment | delivered | `feat/authenticator-enrollment` | `feat(slice-1): add restricted authenticator enrollment` | `#98` | `ops/default/report/slices/authenticators/slice-1-enrollment.md` |
| Slice 2 | WebAuthn completion | delivered | `feat/webauthn-completion` | `feat(slice-2): complete WebAuthn authentication and step-up` | `#99` | `ops/default/report/slices/authenticators/slice-2-webauthn-completion.md` |
| Slice 3 | TOTP and backup proof | active | `feat/authenticator-fallback` | `feat(slice-3): add replay-resistant TOTP and backup proof` | pending | `ops/default/report/slices/authenticators/slice-3-fallback-proof.md` |
| Slice 4 | Authorized factor changes | pending | `feat/authenticator-control` | `feat(slice-4): enforce authorized authenticator changes` | pending | `ops/default/report/slices/authenticators/slice-4-factor-control.md` |
| Slice 5 | Browser acceptance | pending | `test/authenticator-acceptance` | `test(slice-5): verify authenticator browser integration` | pending | `ops/default/report/slices/authenticators/slice-5-browser-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit | Required evidence |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `feat(auth): add restricted WebAuthn enrollment` | `776b4aca0ab80318fa11d86035684d00efca2b5a` | Reviewed dependency/profile; typed bounded pending/budget/setup, actual registration and synchronized adapter |
| T1.2 | complete | `test(auth): verify enrollment isolation and bounds` | `f5b62edccb970f2af7e20f80d6f77f1890a3b437` | Protocol rejection, pending isolation, durable admission/budgets and real PostgreSQL setup races |
| T2.1 | complete | `feat(auth): complete WebAuthn proof atomically` | `f7df235f4411be7a2bb7eaa0f211077295edba61` | Actual assertion, closed facts and atomic counter/pending/session insertion or rotation |
| T2.2 | complete | `test(auth): verify WebAuthn completion transactions` | `1b01ea4ba0a64a87705996b37320e745cd20d863` | Real signatures/one winner, current counter/flags, post-lock time and full rollback |
| T3.1 | active | `feat(auth): add replay-resistant fallback proof` | pending | Encrypted TOTP, exact accepted steps and salted domain-bound one-use backup proof |
| T3.2 | pending | `test(auth): verify fallback replay and policy` | pending | Trusted-time/step and code races, salts/KDF, no weak-policy substitution and durable budget exhaustion |
| T4.1 | pending | `feat(auth): enforce authorized factor changes` | pending | Recent trusted authority, bounded own-factor operations, version/revocation and coherent actor rotation |
| T4.2 | pending | `test(auth): verify authenticator change authority` | pending | Subject isolation, post-lock freshness, last-factor/current-proof constraints, mutation races and rollback |
| T5.1 | pending | `test(auth): exercise authenticator browser journeys` | pending | Actual browser/virtual device against production handlers, real PostgreSQL and required stronger proof |
| T5.2 | pending | `test(auth): close authenticator integration evidence` | pending | Cookie/parser/deadline regression, supported guidance and AU/AUTH evidence mapping |

## Dependencies and Execution Gate

Credential security and authentication sessions are delivered on canonical `dev`.
The concern/model, selected dependency/profile/defaults and five-slice plan were
approved on 2026-10-04. The concern/model are promoted together; Slice 1/T1.1
is delivered. Slices 1-2 are delivered. Slice 3/T3.1 is active after verified PR #99 merge. The required branch is `feat/authenticator-fallback`.

Before T1.1 settle matching Go/SQL/wire fields and exact pinned verifier/key
dependencies. Before T1.2 record actual finite integration/fuzz selectors;
before T5.1 record the real browser acceptance invocation/prerequisites.
Use actual protocol/TOTP/KDF verification and real persistence. Pending labels,
setup flags and fake method claims never count as authenticator acceptance.

## Completion Gates

- [x] Concern/model/dependency/profile/defaults and plan reviewed; concern promoted and Slice 1 activated.
- [x] Slice 1 merged; delivered enrollment report and real setup/budget evidence recorded.
- [x] Slice 2 merged; delivered WebAuthn completion/step-up and atomic replay evidence recorded.
- [ ] Slice 3 merged; delivered TOTP/backup verification, replay and policy evidence recorded.
- [ ] Slice 4 merged; delivered current-authority/factor-change evidence recorded.
- [ ] Slice 5 merged; delivered real browser integration and supported documentation recorded.
- [ ] AU-01 through AU-10 mapped to actual deterministic, verifier, race, persistence and browser evidence.
- [ ] Exact immutable integrated `dev` candidate passes `make check`.
- [ ] Supported AUTH-03/AUTH-05 coverage and remaining AUTH-06/AUTH-07/application assurance obligations recorded.

## Planning Validation

- `make docs-check` — passed, including local links and example compilation.
- `make source-license-check` — passed: 764 headers, 104 content-preserving annotations.
- `git diff --check` — passed.
- Spec/model fields, operation purposes, proof authority, slice/task ordering,
  exact branches/titles/report paths and closed execution state — consistent.
- The four proposal artifacts contain no outer-product references.

No runtime, browser or full delivery-set evidence is claimed for this planning
proposal. No dependency has been added and previous completed set gates are not
repeated.

## Slice 1 Focused Validation Bindings

- Protocol/parser fuzz: `go test ./auth -run '^$' -fuzz '^FuzzEnrollmentResponse$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work in the target.
- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`; `DB_HOST` and explicit owned-cluster settings are required.
- T1.1 passed focused auth/config/adapter/HTTP tests, real PostgreSQL registration/replay/revocation, strict lint, licensing, documentation and diff checks. No aggregate delivery-set gate ran.

T1.2 passed protocol rejection/binding tests, finite constructor/HTTP bounds,
actual registration with both supported backup classes, 20-second parser fuzz
(354797 executions), real PostgreSQL races/rollback/budgets/cleanup/uniqueness and
credential/session regression, affected-package race checks, vet, strict lint,
source licensing and docs/example compilation. Pending replay issues no session.
The schema uses a new `004-authenticators.sql` migration for existing databases.
The complete set's integrated gate remains scheduled after Slice 5 merge.

Slice 1 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/98
Report introduction: `3f04dc1c00fce625ead88c7611b97da62176a897`.
Verified PR #98 merge: `d916a27f53e2d0d42d67aed246d018194aacbfcf`.
Next checkpoint: review/merge Slice 2, verify canonical state, then activate the already-approved Slice 3.

## Slice 2 Focused Validation Bindings

- Parser fuzz: `go test ./auth -run '^$' -fuzz '^FuzzAssertionResponse$' -fuzztime=20s -parallel=2 -timeout=60s`; no KDF/database work.
- Real PostgreSQL: `go test -race -tags=integration ./examples/ticked/internal/feat/auth -run '^Test(WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$' -count=1 -timeout=120s`; explicit owned-cluster connection settings required.
- T2.1 passed actual signed authentication/step-up with PostgreSQL, package and regression races, strict default/tagged lint, source licensing and docs/example compilation. T2.2 passed signed negative protocol, replay/backup/counter/ownership conflicts, post-lock/post-write time checks, durable admission and full rollback evidence. Parser fuzz completed 484362 executions without failure; PostgreSQL regression races, package races, vet, strict lint, licensing and documentation checks passed.


Slice 2 PR: https://forge.adrianpk.com/hatmax/hatmax/pulls/99
Report introduction: `379370c2b25b63cc1726ed6a8af88984a179baab`.
Verified PR #99 merge: `3d451cf7a0ba7eaf622ec732cc388c379062b3f2`.
Next: implement approved Slice 3/T3.1/T3.2.
