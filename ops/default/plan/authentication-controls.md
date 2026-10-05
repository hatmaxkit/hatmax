<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Controls Delivery Plan

Date: 2026-10-05
Status: Active
Approved: 2026-10-05
Delivery set: authentication-controls
Concern: [Authentication controls](../spec/authentication-controls.md)
Model: [Controls model](../spec/authentication-controls-model.md)
Parent: [Authentication security](../spec/authentication-security.md)
Base branch: `dev`
Planning base: `9070714609a4dc323d1a1b5991ed7cb806c11677`
Active slice: Slice 2 — Guarded password entry
Execution gate: Open
Tracker: [Delivery tracker](../tracker/authentication-controls.md)
Slice strategy: layered
Reason: the shared storage/admission contract must be valid before mandatory
password-path integration; ingress/transport and observations then close distinct
boundaries before real browser acceptance. Each slice compiles and passes its
focused checks without placeholders or intentionally failing tests.
Go baseline: 1.27.1

## Approved Scope

The concern, model, bounds and implementation scope were approved on 2026-10-05.
The user authorized this internal slicing map and implementation. Delivered
credentials, sessions, authenticators and account recovery are prerequisites.
Preserve their authority and completed gates. Continue the next recorded slice
immediately after canonical maintainer merge verification. No main promotion,
release, external provider send, key rotation or all-factor-loss lifecycle work.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Durable credential admission | `feat/credential-admission` | `feat(slice-1): add shared credential admission` | `ops/default/report/slices/authentication-controls/slice-1-credential-admission.md` |
| Slice 2 | Guarded password entry | `feat/guarded-password-entry` | `feat(slice-2): guard password authentication paths` | `ops/default/report/slices/authentication-controls/slice-2-password-entry.md` |
| Slice 3 | Finite public authentication | `feat/authentication-ingress` | `feat(slice-3): bound public authentication ingress` | `ops/default/report/slices/authentication-controls/slice-3-public-authentication.md` |
| Slice 4 | Security observations | `feat/authentication-events` | `feat(slice-4): observe bounded authentication outcomes` | `ops/default/report/slices/authentication-controls/slice-4-security-events.md` |
| Slice 5 | Controls acceptance | `test/authentication-controls-acceptance` | `test(slice-5): verify authentication controls integration` | `ops/default/report/slices/authentication-controls/slice-5-controls-acceptance.md` |

## Task Map

| Task | Work | Expected commit |
| --- | --- | --- |
| T1.1 | Implement config, typed credential admission and private HMAC PostgreSQL capacity/window transactions; settle matching representations. | `feat(auth): add shared credential admission` |
| T1.2 | Prove real shared admission races, expiry/cooldown, no resets/refunds, capacity/cleanup, deadlines and storage costs; document the usable admission contract. | `test(auth): verify credential admission transactions` |
| T2.1 | Wire mandatory shared admission before all actual password entry work, preserving canonical identity and current credential/session/factor authority. | `feat(auth): guard password authentication paths` |
| T2.2 | Prove cross-path budgets, denied/unknown/inactive/canceled/stale outcomes and current MFA with actual adapters; keep consumers compiling. | `test(auth): verify shared password entry budgets` |
| T3.1 | Replace unbounded peer limiting with finite worker-free counters; wire shared authentication ingress and safe neutral registration/failure responses. | `feat(auth): bound public authentication ingress` |
| T3.2 | Prove real HTTP/body/proxy/capacity/timing/shutdown behavior and one-winner registration without automatic login; update visible guidance and Unreleased. | `test(auth): verify neutral public authentication` |
| T4.1 | Add closed redacted security observations at actual service entrypoints, finite callback admission and safe delivery-failure diagnostics. | `feat(auth): observe bounded security outcomes` |
| T4.2 | Prove transaction/proof/outcome provenance, no leaks, saturation, late failure and re-entry without changing authority. | `test(auth): verify security observation boundaries` |
| T5.1 | Extend the finite real browser/PostgreSQL acceptance harness across registration/password entry/guard/events and retained recovery/MFA journeys. | `test(auth): exercise authentication controls journeys` |
| T5.2 | Complete finite affected regressions and AC-01 through AC-08 evidence, public/config/User Guide obligations and the exact integrated candidate preparation. | `test(auth): close authentication controls evidence` |

## Validation Boundaries

Slice 1 delivers callable admission and actual storage; existing sign-in paths
remain for Slice 2. Before runtime edits settle Go/config/SQL names together in
the companion model. The HMAC namespace/key is application-owned; no secrets are
stored in source, config examples, logs or operational artifacts.

Record and execute finite checks with bounded per-worktree scratch:

- `make source-license-check`, `make vet`, `make lint-strict`, `make docs-check`.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web`.
- Slice 1: `go test -tags=integration -race -v -run '^TestCredentialAdmissionTransactions$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth`.
- Slice 1: `go test -run '^$' -fuzz '^FuzzCredentialIdentity$' -fuzztime=20s -parallel=2 -timeout=60s ./auth`.
- Required database settings fail the integration selector when absent. Use an
  owned PostgreSQL instance with isolated schemas; record actual counter races,
  rollback, lock deadlines and finite row/index cost measurements.
- Record the exact focused selectors before later task validation. Browser
  prerequisites are mandatory and use actual production handlers/adapters and
  retained virtual WebAuthn evidence; do not manufacture authentication.

No aggregate gate per task/slice. After all five canonical merges, finalize one
immutable integrated dev candidate, then run `make check` once. Hatmax has no
nightly; the local exact-candidate gate is authoritative for this set. Required
repository corrections use `fix/authentication-controls-validation` from failing
dev, focused checks and one PR to dev; revalidate the corrected integrated
candidate after its verified merge. No correction branch for a green candidate.

## Exit Gates

Close only with five delivered reports, all ten task/commit/PR mappings, AC-01
through AC-08 exact evidence and the integrated gate. Supported AUTH-07 admission,
errors and observation do not close cumulative authenticator disabling/rebinding,
all-factor-loss identity proofing, external audit or consumer assurance.
