<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Locale-Aware Formatting Delivery Tracker

Date: 2026-10-18
Status: Completed
Completed: 2026-10-18
Approved: 2026-10-18
Specification approved: 2026-10-18
Planning authorized: 2026-10-18
Delivery set: locale-aware-formatting
Concern: [Locale-aware exact number formatting](../spec/locale-aware-formatting.md)
Plan: [Delivery plan](../plan/locale-aware-formatting.md)
Base branch: `dev`
Planning base: `3e1b31b63db934d7d8b1c1287db68b6f500a7c27`
Execution authorized: 2026-10-18
Integration authority: Controller-owned Rebase + Fast-forward into `dev`
Active slice: None
Next slice: None
Active tasks: None
Execution gate: Passed at `ff45f00bf867d1763376c21b941946c74878116a`; documentation closure only

## Slice Status

| Slice | Short name | Status | Branch | Expected PR title | PR | Report |
| --- | --- | --- | --- | --- | --- | --- |
| Slice 1 | Exact localized display | delivered | `feat/locale-aware-formatting` | `feat(slice-1): format exact values by locale` | [#128](https://forge.adrianpk.com/hatmax/hatmax/pulls/128) | `ops/default/report/slices/locale-aware-formatting/slice-1-exact-formatting.md` |

## Tasks

| Task | Status | Expected commit | Commit |
| --- | --- | --- | --- |
| T1.1 | completed | `feat(format): add exact locale-aware number display` | `0078533f8a40999651927592e50edfa68422777a` |
| T1.2 | completed | `docs(format): explain exact localized presentation` | `ca0e8d5aeb17b151ee48a001c1e92353d419b186` |

The plan owns task scope and the ordered package/documentation gate. The
dedicated controller owns integration and the final integrated gate. The
[Slice 1 report](../report/slices/locale-aware-formatting/slice-1-exact-formatting.md)
records implemented LF01-LF06 behavior and focused checks.

## Prerequisites and Acceptance

- [x] Specification approved; independent core ownership established.
- [x] One-slice plan and matching tracker prepared.
- [x] Planning artifacts committed/pushed; execution and integration activated.
- [x] Slice 1 implemented and integrated into `dev`.
- [x] LF01-LF06 verified at the exact integrated candidate.
- [x] Report and delivery metadata closed; published module identity available.

| Criterion | Owning slice | Evidence status |
| --- | --- | --- |
| LF01 | Slice 1 | Locale, digit, grouping and fallback cases passed at the validated candidate. |
| LF02 | Slice 1 | Exact precision and scale cases passed at the validated candidate. |
| LF03 | Slice 1 | Minor-unit boundary cases passed at the validated candidate. |
| LF04 | Slice 1 | Admission and rejection cases passed at the validated candidate. |
| LF05 | Slice 1 | Parallel shared-instance cases passed the candidate's race check. |
| LF06 | Slice 1 | Real escaped template example, legacy helpers and documentation checks passed at the validated candidate. |

## Validation Record

PR #128 merged through controller-owned Rebase + Fast-forward on 2026-10-18 at
`e7ea97bc28f2810acf018bbd7f6eb82dc8105198`. All five ordered PR-head checks
passed through `looper validate` at that exact clean commit in approximately
11.14 seconds. Focused checks and criterion evidence are recorded in the slice
report. These results do not replace the final integrated gate.

Gate prerequisites are available: Go with race support, a C compiler,
golangci-lint, Make and Bash. The approved format/render/ui and documentation
gate requires no database, provider, browser or service fixture. The primary
checkout remains on `dev`; its lint cache is local and ignored. The controller
executed the unchanged selected gate against
`ff45f00bf867d1763376c21b941946c74878116a` on 2026-10-18. Its exact-command
receipt `I42QOJY5OSZ4G6X5OSEH2DLKYE` records success in approximately 35.12
seconds, covering all five commands in the plan's canonical order. The report
records the full command. No runtime checks were repeated for documentation
closure; only configured closure paths differ from the validated candidate.

## Published Module Identity

`GOWORK=off go list -m -json hatmax.adrianpk.com@ff45f00bf867d1763376c21b941946c74878116a`
resolved `hatmax.adrianpk.com@v0.6.1-0.20261018112343-ff45f00bf867` on
2026-10-18. Go reports the canonical Forgejo repository and full origin hash
`ff45f00bf867d1763376c21b941946c74878116a`. This is the validated runtime
candidate; subsequent closure changes are documentation only.
