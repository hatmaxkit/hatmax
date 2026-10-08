<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Coverage and Validation

Status: delivered
Delivery set: documentation-conformance
Plan: [Delivery plan](../../../plan/documentation-conformance.md)
Tracker: [Delivery tracker](../../../tracker/documentation-conformance.md)
Branch: `docs/hatmax-documentation-coverage`
PR: `#112`
Activation revision: `6715ab2912d27263a78f0bd1419db8b49c3d8089`

## Purpose

Establish reproducible documentation coverage and structural validation before
reconciling reader workflows against the implementation.

## Delivered Behavior

The coverage record identifies 443 independently discovered source, page and
example identities. It accounts for 41 public package directories, nested
implementation owners, four executable compositions, example configuration and
assets, both embedded Book releases, terminal help and dependency inputs.
It assigns 101 product pages, 218 fenced blocks and 28 walkthroughs to their
reader purpose and responsible slice. Every row binds an inspected revision
and SHA-256 content digest; behavioral receipts remain pending.

The Slice 1 gate runs documentation, source-license and strict-lint checks in
that order, reconciles coverage against source discovery, and validates local
heading anchors and reachability from each quadrant index. Negative controls
reject omitted surfaces, unaccounted pages, changed bindings, duplicate IDs,
false verification claims, broken anchors and circular orphan pages.

## Implementation Notes

The checker discovers source independently from the maintained table. It includes
staged and non-ignored new files, so newly introduced packages/pages cannot escape
coverage. Public product pages exclude historical operations, agent rules and
fixture documentation. Complete example bindings include migrations, templates,
configuration and browser test fixtures in addition to Go source.

The runner resolves tool overrides, requires Go 1.27.1 and records command exits,
head, tool versions, effective build flags and dependency/coverage digests in fresh
ignored fixture storage. It retains safe diagnostics and uses a worktree-local
cache. This slice starts no persistent server fixture. Go negative controls have
an explicit two-minute test bound.

## Contracts Added or Changed

`scripts/check-documentation-conformance.sh slice 1` is the implemented slice
mode. Integrated mode and future slice numbers fail with exit 2 before validation
starts. Inventory generation writes a new table to stdout; it does not preserve
later verification receipts. Changed paths or content invalidate reconciliation.
`inventoried` and `pending` establish coverage, not successful reader execution.

## Files of Interest

- [Coverage record](../../documentation-conformance-coverage.md)
- [Slice runner](../../../../../scripts/check-documentation-conformance.sh)
- [Source and navigation checker](../../../../../scripts/documentation-conformance/main.go)
- [Negative controls](../../../../../scripts/documentation-conformance/main_test.go)

## Validation

Focused local checks passed:

- `GOWORK=off make lint-strict` — passed with zero issues.
- `GOWORK=off go run ./scripts/documentation-conformance check` — passed for 443 identities and four-quadrant navigation.
- `GOWORK=off go test -count=1 -timeout=2m ./scripts/documentation-conformance` — passed, including the required negative controls and unavailable-mode checks.
- `bash -n scripts/check-documentation-conformance.sh` — passed.

The controller recorded these ordered checks as passed against clean pushed
head `4edfea516b25656b2ab4797163ddbc6523c54e33`:

- `GOWORK=off scripts/check-documentation-conformance.sh slice 1` — passed.
- `git diff --check` — passed.

Canonical PR #112 merged through Rebase + Fast-forward into `dev` at that same
commit. This is Slice 1 evidence, not integrated documentation acceptance or
the complete runtime suite.

## Risks and Follow-ups

Slices 2 through 6 must reconcile public contracts, specify owning compositions
for fragments and supply actual example/walkthrough execution receipts. Slice 7
owns complete acceptance. The initial coverage table must be reconciled when
source, pages or example content changes. No behavioral coverage row is verified
by this preparatory slice.
