<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 6: Generator and Assisted Workflows

Status: drafting
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-generator-guidance`
PR: pending
Activation revision: `c7e9858d8726e0a5f1ed56f41c2daee4e59cebff`
Documentation task: `0a70f3b71cd69db25dfaae0a62224a0f6eef087d`
Report introduction: `0a70f3b71cd69db25dfaae0a62224a0f6eef087d`

## Purpose

Reconcile generator, terminal, conversation, Book and generated documentation
guidance with delivered behavior and verify supported local workflows.

## Delivered Behavior

T6.1 uses the current source-build installation route: published v0.5.0 contains
the compatibility command but no `cmd/hm`. The learning path uses an explicit
parent directory and its generated child. Reference states CLI usage behavior,
playground prerequisites/overrides, Book release selection and compatibility,
managed documentation paths/markers, state retention, backend isolation and
post-mutation validation outcomes. Source and task identities are preserved.

T6.2 workflow receipts and the exact-head Slice 6 controller check are pending.

## Implementation Notes

The dedicated branch starts after verified PR #116 closure. The 451-row table
updates seven changed Slice 6 source/snippet identities; all 63 Slice 6 receipts
remain pending. Native-toolchain closure retains the generic approved wording.

Deterministic interpreter fixtures can verify the Hatmax kernel and real generated
artifacts, not production model interpretation or authenticated provider behavior.
Actual CLI state/usage checks and real project commands require separate evidence.

## Contracts Added or Changed

Documentation reflects existing contracts; toolkit runtime behavior and module
dependencies are unchanged. Generated documentation uses `hatmax:generated`
ownership markers. Its conformance checks local files, not external websites or
heading fragments. Stored user text has no general secret-redaction guarantee.

## Files of Interest

- [Generator reference](../../../../../docs/reference/generator/README.md)
- [Assisted generation](../../../../../docs/tutorials/user-guide/assisted-generation.md)
- [Coverage record](../../documentation-conformance-coverage.md)

## Validation

Focused T6.1 checks passed:

- `GOWORK=off make docs-check` — structure, links, compilation and whitespace.
- `GOWORK=off make source-license-check` — 930 headers and 113 content-preserving annotations.
- `GOWORK=off make lint-strict` — zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — 451 identities.
- `GOBIN="$PWD/.tmp/documentation-conformance/install" GOWORK=off go install ./cmd/hm` — exact published source-install command, with an owned binary destination.

The controller's required ordered final checks remain pending until T6.2 and
canonical PR references are pushed:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 6`
- `git diff --check`

## Risks and Follow-ups

T6.2, DC07/DC08 integrated acceptance and DC09 constrained native-toolchain
closure remain pending. Fixture interpretation cannot establish live provider
acceptance. Source, head, command, tool or configuration drift requires renewed
evidence.
