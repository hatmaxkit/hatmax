<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Locale-Aware Formatting Delivery Plan

Date: 2026-10-18
Status: Approved
Approved: 2026-10-18
Specification approved: 2026-10-18
Planning authorized: 2026-10-18
Delivery set: locale-aware-formatting
Concern: [Locale-aware exact number formatting](../spec/locale-aware-formatting.md)
Tracker: [Delivery tracker](../tracker/locale-aware-formatting.md)
Base branch: `dev`
Planning base: `3e1b31b63db934d7d8b1c1287db68b6f500a7c27`
Execution authorized: 2026-10-18
Integration authority: Controller-owned Rebase + Fast-forward into `dev`
Active slice: None
Next slice: Slice 1
Execution gate: Publish these artifacts, verify exclusive ownership and start the dedicated controller
Slice strategy: behavior-first
Reason: the additive formatter and its complete documentation form one cohesive package capability; no preparatory or testing-only slice is needed.

## Delivery and Prerequisites

Deliver LF01-LF06 in the existing `format` package using the existing text/locale
dependency. Keep instances immutable and the historical functions compatible.
The set has no dependency on a consumer repository, its controller or its plan.

At activation verify clean canonical `dev`, actual open PR/worktree ownership,
published planning artifacts and the explicit execution/integration instruction.
Use a dedicated project-scoped controller if Looper execution is authorized.
Do not reuse another delivery's checkpoint. This plan does not authorize main
alignment, a version tag, release, mirror publication or a new dependency.

## Ordered Slices

| Slice | Short name | Exact branch | Expected PR title | Report |
| --- | --- | --- | --- | --- |
| Slice 1 | Exact localized display | `feat/locale-aware-formatting` | `feat(slice-1): format exact values by locale` | `ops/default/report/slices/locale-aware-formatting/slice-1-exact-formatting.md` |

## Tasks and Expected Commits

| Task | Deliverable and validation responsibility | Expected commit |
| --- | --- | --- |
| T1.1 | Implement caller-owned locale construction and exact integer/decimal/minor-unit display, finite admission and concurrent independence. Extend a compact table-driven test family for real transformations and rejected input; preserve existing formatter APIs. | `feat(format): add exact locale-aware number display` |
| T1.2 | Integrate a real escaped Go-template usage example and update package/reference/User Guide documentation. Verify legacy format/template consumers, LF01-LF06 and the focused integrated gate; close the one slice with its canonical report. | `docs(format): explain exact localized presentation` |

## Validation Selection and Cost

During implementation run the relevant format test selection or compile check.
The slice and final integrated gate use these commands, in this order:

1. `GOWORK=off go vet ./format ./render ./ui`
2. `GOWORK=off go test -race -count=1 -timeout=120s ./format ./render ./ui`
3. `GOCACHE="$PWD/.tmp/lint/gocache" golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5 ./format`
4. `make docs-check`
5. `git diff --check`

These paths cover the formatter, its direct render/template consumers and the
repository's existing documentation gate. No database, provider or browser is
needed. Do not add a new runner, acceptance-receipt parser or test matrix by
locale/process. A table can cover the five supplied examples plus one additional
locale cheaply in the same package invocation.

Expected warm-cache cost is a few minutes or less, not an observed result.
Record actual elapsed time on the first implementation run; investigate an
unexpected cost before expanding timeouts or repeating broader work. Do not run
`make check` or every repository test as this set's routine closure gate.

Reproduce a failure with its focused package/root first. After correction,
revalidate affected contracts; retain exact candidate/command evidence and follow
the controller's required final-gate path. Never present an older commit's result
as a new pass or weaken assertions to reduce execution time.

## Integration, Closure and Consumption

The slice targets `dev`. Use the canonical `Summary`, `Changes`, `Validation`
PR structure, ending Changes with the immutable slice-report link. The eventual
controller performs the authorized Rebase + Fast-forward integration. Correct
failures within this delivery's scope without requesting renewed authorization;
retain the selected checks and use the controller's bounded repair path when
required. This does not authorize unrelated fixes or wider delivery scope.

Record LF01-LF06 evidence and the tested commit in the existing report/tracker;
no additional acceptance ledger or tests of plan/report text are needed. Close
with documentation/status changes only, verifying links and whitespace without
rerunning unchanged runtime tests solely because completion prose changed.

After authorized publication of the delivered `dev` commit, a consumer can resolve
and verify its published Go module pseudo-version. Record the exact commit and
module identity; do not create a tag or promote main just to satisfy consumption.
