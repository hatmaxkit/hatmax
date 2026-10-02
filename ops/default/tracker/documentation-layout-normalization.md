<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Documentation Layout Normalization Tracker

Status: Delivered
Delivery set: documentation-layout-normalization
Plan: `ops/default/plan/documentation-layout-normalization.md`
Ticket: `ops/default/ticket/solved/20260929091830-normalize-documentation-entrypoints-and-media.md`
Spec: `ops/default/spec/boxed-diataxis-documentation.md`
Base branch: `dev`
Planning base: `73fcc8c076b30259b522da8fbb056bf6ae123da1`
Active slice: none
Active tasks: none
Execution gate: delivered on `dev` at
`e168d282a632accb9a45ba7a3f2701cc4236f00a`

## Slice Status

| Slice | Short Name | Status | Branch | PR Title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | canonical README layout | delivered | `feat/documentation-readme-layout` | `feat(slice-1): adopt forge-native documentation layout` | #47 | `ops/default/report/slices/documentation-layout-normalization/slice-1-canonical-readme-layout.md` |

## Slice 1 Tasks

| Task | Status | Expected Commit | Commit | Validation |
| --- | --- | --- | --- | --- |
| T1.1 | complete | `docs: adopt forge-native documentation entrypoints` | `a854c27` | `make docs-check` passed; link and layout audit passed |
| T1.2 | complete | `docs(generator): adopt the README documentation contract` | `92320c0` | accepted-spec and Book-rule audit passed |
| T1.3 | complete | `feat(generator): render README documentation entrypoints` | `6ba9c6b` | `go test ./generator/... ./internal/hatmaxcli/...` passed; `make lint-strict` passed |
| T1.4 | complete | `docs(ops): report documentation layout normalization` | `b6225d7` | `git diff --check` passed |

## Completion Gates

- [x] The maintainer approved the layout decision and delivery order.
- [x] The plan and tracker define one atomic slice.
- [x] The plan and tracker are committed on `dev`.
- [x] Slice 1 is delivered through its recorded branch, report, pull request,
  review, and merge.
- [x] The exact integrated `dev` commit passes `make check`.
- [x] Forgejo renders directory `README.md` files as documentation landing
  pages.
- [x] The ticket is solved with its implementation commits recorded.

## Current Gate

Slice 1 merged through pull request #47. `make check` passed on integrated
commit `e168d282a632accb9a45ba7a3f2701cc4236f00a`, and Forgejo rendered the root
and reference directory README files as landing pages. The delivery set is
closed.
