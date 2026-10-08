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

This correction does not establish integrated acceptance. The three existing
generator runtime tickets still block their documented workflows. Resolve those
through their own ticket delivery before revalidating the complete documentation
set. Preserve the same controller run, seven integrated slices and conversation.
