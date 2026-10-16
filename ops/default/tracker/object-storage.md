<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Object Storage Delivery Tracker

Date: 2026-10-16
Status: Active
Approved: 2026-10-16
Specification approved: 2026-10-16
Planning authorized: 2026-10-16
Delivery set: object-storage
Concern: [Object storage capabilities](../spec/object-storage.md)
Source ticket: [Bounded object storage and signed uploads](../ticket/open/20261016081501-support-bounded-object-storage-and-signed-uploads.md)
Plan: [Delivery plan](../plan/object-storage.md)
Base branch: `dev`
Planning base: `60fcca7f8f88059608b353420dfa7513c0b78279`
Execution authorized: 2026-10-16; five-slice Looper delivery, exact-candidate full gate and restricted closure
Integration authorized: Controller-owned Rebase + Fast-forward into dev
Active slice: Slice 1 - Bounded local objects
Next slice: Slice 2 - Private S3 objects
Active tasks: None; Slice 1 PR preparation
Execution gate: Open
Worker budget: 300 total turns; 10 consecutive stalled turns

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Bounded local objects | reviewing | `feat/bounded-local-storage` | `feat(slice-1): store bounded private local objects` | pending | `ops/default/report/slices/object-storage/slice-1-local-objects.md` |
| Slice 2 | Private S3 objects | pending | `feat/private-s3-storage` | `feat(slice-2): inspect private S3 objects` | pending | `ops/default/report/slices/object-storage/slice-2-s3-objects.md` |
| Slice 3 | Signed POST uploads | pending | `feat/signed-storage-uploads` | `feat(slice-3): sign constrained remote uploads` | pending | `ops/default/report/slices/object-storage/slice-3-signed-uploads.md` |
| Slice 4 | Object storage guides | pending | `docs/object-storage-workflows` | `docs(slice-4): guide private object storage workflows` | pending | `ops/default/report/slices/object-storage/slice-4-storage-guides.md` |
| Slice 5 | Integrated acceptance | pending | `test/object-storage-acceptance` | `test(slice-5): verify object storage acceptance` | pending | `ops/default/report/slices/object-storage/slice-5-integrated-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(storage): implement bounded local object operations` | `b6ba9e4509918ec2fe6fca7a491a63a552ee10fd` |
| T1.2 | completed | `test(storage): verify local ownership and resource bounds` | `94681613bb9ef8075979dd9b0ede1d99500b8259` |
| T2.1 | pending | `feat(storage): implement private S3 object operations` | pending |
| T2.2 | pending | `test(storage): verify private S3 transport outcomes` | pending |
| T3.1 | pending | `feat(storage): sign exact-target bounded POST uploads` | pending |
| T3.2 | pending | `test(storage): verify signed upload enforcement` | pending |
| T4.1 | pending | `docs(storage): explain private object storage contracts` | pending |
| T4.2 | pending | `test(storage): execute documented storage workflows` | pending |
| T5.1 | pending | `test(storage): verify integrated object lifecycles` | pending |
| T5.2 | pending | `test(storage): bind exact-candidate acceptance evidence` | pending |

The plan owns task scopes, named future selectors and finite bounds. Each PR
requires `GOWORK=off scripts/check-object-storage.sh slice N` first and
`git diff --check` last. T1.2 delivers runner mode 1 and its required local roots. Later modes remain
unavailable until their owning tasks deliver them. Final PR outcomes bind to
the exact clean-HEAD controller receipt; task checks do not replace that receipt.

## Prerequisites and Gates

- [x] Core specification explicitly approved on 2026-10-16.
- [x] Five behavior-first slices and ten task identities defined independently.
- [x] Branch/PR/report and expected task-commit identities agree with plan.
- [x] OS01-OS10 mapped to owners and future required checks.
- [x] Reference server release/source/module identities resolved from authoritative sources.
- [x] Eleven documentation-only closure path names specified, including both ticket move paths.
- [x] Plan accepted and publication authorized on 2026-10-16; activation commit/push verified by controller startup.
- [x] Runtime/controller, Rebase + Fast-forward merges, complete local gate and closure authorized on 2026-10-16.
- [ ] Exact native fixture binary build information/hash bound before integration tests.
- [x] Canonical origin/dev, empty open PR inventory, installed tools and existing controller ownership verified at Slice 1 entry.
- [ ] Five canonical PRs, ten tasks and original reports delivered.
- [ ] Exact complete gate passed; all required storage roots and OS01-OS10 accepted.
- [ ] Restricted closure, source-ticket resolution and Acta completed.

## Acceptance Tracking

| Criterion | Owning slices | Integrated evidence status |
| --- | --- | --- |
| OS01 | 1, 2, 4, 5 | pending |
| OS02 | 1, 5 | pending |
| OS03 | 1, 2, 5 | pending |
| OS04 | 1, 2, 5 | pending |
| OS05 | 3, 4, 5 | pending |
| OS06 | 3, 5 | pending |
| OS07 | 2, 3, 5 | pending |
| OS08 | 1-5 | pending |
| OS09 | 4, 5 | pending |
| OS10 | 5 | pending |

## Current Checkpoint

Slice 1 T1.1 and T1.2 are implemented in the dedicated approved branch. Focused
ordinary storage checks passed; the canonical report is reviewing. Canonical
PR preparation and final clean-HEAD controller validation precede integration.
The primary checkout stays on dev. Slice 2 remains pending.

The remote fixture source remains pinned to
`9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a`; its build identity and acceptance
belong to the later remote-storage tasks.
