<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Documentation Conformance Validation Correction

Date: 2026-10-08
Status: delivered
Delivery set: documentation-conformance
Plan: [Delivery plan](../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../tracker/documentation-conformance.md)
Branch: `fix/documentation-conformance-validation`
PR: [#119](https://forge.adrianpk.com/hatmax/hatmax/pulls/119)

## Purpose

Correct the controller's final-gate timeout on candidate
`ba95fab8bc52a77daf894b269c11cb3a5517bb6f`. PostgreSQL's password prompt
preferred its inherited controlling terminal over the supplied fixture input.

## Delivered Behavior

Commands with scripted input run in a private session without a controlling
terminal. The documented `createuser --pwprompt` command consumes its owned
fixture input rather than waiting for an operator in the controller terminal.
Other command invocations retain their existing process-group behavior.
Finite deadlines, group cancellation and command receipts remain in force.
Correction introduction: `547ce501fd7317b9521569ed3e4022de21e801d7`.
PR #119 merged through SHA-bound Rebase + Fast-forward at
`ee16d2fe9bfe3158853a2920893be1bb61e0b76c`; Acta event
`01M4E2A3PASA63KJJZX1FQF2T0` records the canonical integration.

## Implementation Notes

`TestCommandInput` failed under a real terminal before the correction and passed
after it. The actual owned PostgreSQL/Ticked procedure also passed under a
terminal, including native role/database creation and coordinated shutdown.

## Files of Interest

- [Command execution](../../../scripts/documentation-conformance/runtime.go).
- [Input regression](../../../scripts/documentation-conformance/runtime_test.go).

## Validation

- `GOWORK=off go test -run '^TestCommandInput$' -count=1 ./scripts/documentation-conformance` passed with a controlling terminal.
- `GOWORK=off go test -count=1 -timeout=2m ./scripts/documentation-conformance` passed.
- `make lint-strict` passed.
- `GOWORK=off go vet ./scripts/documentation-conformance` passed.
- `GOWORK=off go run ./scripts/documentation-conformance check` passed with 451 identities.
- `scripts/documentation-conformance/data-fixture.sh "$fixture" go run ./scripts/documentation-conformance identity-ticked "$fixture"` passed with `GOWORK=off`, real required tools and a fresh owned fixture under a terminal. Its PostgreSQL cluster stopped successfully.
- `git diff --check` passed.

## Risks and Follow-ups

This correction alone does not establish integrated acceptance. The separate
runtime deliveries below remove the remaining generator failures. Preserve the
same controller run, seven integrated slices and conversation for revalidation.

## Separately Approved Runtime Corrections

The maintainer approved the three independent tickets on 2026-10-08. They were
delivered individually to `dev`:

- TKT-20261008135600: `e88a34ff4c89` preserves SQLC table row types after field
  evolution. A regression reproduced the original compiler failure; real
  PostgreSQL create/get/list/update preserves the required timestamp.
- TKT-20261008135846: `bfe44acd2d08` separates generated test assignments from
  assertions. The original project lint failure reproduces before the fix;
  generated validation, SQLC, PostgreSQL tests and lint pass afterward.
- TKT-20261008134713: `6cdb5fb35a07` recognizes delegated composition, preserves
  thin main and supplies native SQLC tooling for first-feature evolution. Bare
  and initial-feature scaffolds create another feature and add a timestamp with
  real project validation. Ambiguous roots and stale plans remain rejected.

All three deliveries passed applicable generator tests, `make lint-strict` and
`make docs-check`. The documentation workbench now requires completed resumed
conversation, timestamp and validation workflows instead of reproduced failures.
Its native run passed all three named workflow tests with owned PostgreSQL;
shutdown succeeded. This remains focused evidence, not the integrated gate.
