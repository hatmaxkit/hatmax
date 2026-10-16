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
Source ticket: [Bounded object storage and signed uploads](../ticket/in_progress/20261016081501-support-bounded-object-storage-and-signed-uploads.md)
Plan: [Delivery plan](../plan/object-storage.md)
Base branch: `dev`
Planning base: `60fcca7f8f88059608b353420dfa7513c0b78279`
Execution authorized: 2026-10-16; five-slice Looper delivery, exact-candidate full gate and restricted closure
Integration authorized: Controller-owned Rebase + Fast-forward into dev
Active slice: Slice 4 - Object storage guides
Next slice: Slice 5 - Integrated acceptance
Active tasks: None; Slice 4 awaiting controller integration
Execution gate: Open
Worker budget: 300 total turns; 10 consecutive stalled turns

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Bounded local objects | delivered | `feat/bounded-local-storage` | `feat(slice-1): store bounded private local objects` | [#123](https://forge.adrianpk.com/hatmax/hatmax/pulls/123) | `ops/default/report/slices/object-storage/slice-1-local-objects.md` |
| Slice 2 | Private S3 objects | delivered | `feat/private-s3-storage` | `feat(slice-2): inspect private S3 objects` | [#124](https://forge.adrianpk.com/hatmax/hatmax/pulls/124) | `ops/default/report/slices/object-storage/slice-2-s3-objects.md` |
| Slice 3 | Signed POST uploads | delivered | `feat/signed-storage-uploads` | `feat(slice-3): sign constrained remote uploads` | [#125](https://forge.adrianpk.com/hatmax/hatmax/pulls/125) | `ops/default/report/slices/object-storage/slice-3-signed-uploads.md` |
| Slice 4 | Object storage guides | reviewing | `docs/object-storage-workflows` | `docs(slice-4): guide private object storage workflows` | [#126](https://forge.adrianpk.com/hatmax/hatmax/pulls/126) | `ops/default/report/slices/object-storage/slice-4-storage-guides.md` |
| Slice 5 | Integrated acceptance | pending | `test/object-storage-acceptance` | `test(slice-5): verify object storage acceptance` | pending | `ops/default/report/slices/object-storage/slice-5-integrated-acceptance.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(storage): implement bounded local object operations` | `b6ba9e4509918ec2fe6fca7a491a63a552ee10fd` |
| T1.2 | completed | `test(storage): verify local ownership and resource bounds` | `94681613bb9ef8075979dd9b0ede1d99500b8259` |
| T2.1 | completed | `feat(storage): implement private S3 object operations` | `9de7c8aef2d8a487cf192ad3b5899e64157d2fb5` |
| T2.2 | completed | `test(storage): verify private S3 transport outcomes` | `b85c9496d412f0a759e4640264ffa0689c2fdd0c` |
| T3.1 | completed | `feat(storage): sign exact-target bounded POST uploads` | `2b0802dc245271aadab6b41ce6d6968dcb004db8` |
| T3.2 | completed | `test(storage): verify signed upload enforcement` | `25e7d3ca3f7a78bf0a5b765043e0ae1a148cc38a` |
| T4.1 | completed | `docs(storage): explain private object storage contracts` | `7161cc892c5f0e56ddd5eb11c9c96d647c8d43ee` |
| T4.2 | completed | `test(storage): execute documented storage workflows` | `270d1a89d8d562dd84f8ec008d014c35fb1592d9` |
| T5.1 | pending | `test(storage): verify integrated object lifecycles` | pending |
| T5.2 | pending | `test(storage): bind exact-candidate acceptance evidence` | pending |

The plan owns task scopes, named future selectors and finite bounds. Each PR
requires `GOWORK=off scripts/check-object-storage.sh slice N` first and
`git diff --check` last. T1.2-T4.2 deliver runner modes 1-4 and their required local/remote/signing/guide roots. Later modes remain
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
- [x] Pinned native fixture built with Go 1.27.1; source/module/build information and binary SHA-256 bound before Slice 2 integration tests.
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

Slices 1-3 of 5 are delivered through canonical PRs #123, #124 and #125.
PR #125 is verified merged into dev at
`33b825eb1bfa77d64423eea8a2e0c1e7ca616cbc`. Its three exact-HEAD
controller checks passed, including actual signed multipart race roots and
confirmed owned fixture cleanup. Slice 4 T4.1/T4.2 deliver linked executable guides, exact source/output controls
and historical image compatibility evidence. Focused ordinary and actual-server
walkthroughs passed with confirmed owned cleanup. Its report is reviewing;
Canonical PR #126 is open against dev. Its references are committed before
final clean-HEAD controller validation and integration.

Slice 1 report introduction: `4719f3678de9ab4b761e1faf8281aa206c4f4d98`.

Slice 2 report introduction: `73436bade00dc1d42025cfb511bf04aff3719d15`.

Slice 3 report introduction: `c75d41100e9c9856f378a5ca221c54bc8718ec96`.

Slice 4 report introduction: `eda2f220f1dff8c187e1428d4d0b202bad912eb2`.
