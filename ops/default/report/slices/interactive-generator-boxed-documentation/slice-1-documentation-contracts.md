<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 1: Documentation Contracts

Status: delivered
Delivery set: interactive-generator-boxed-documentation
Plan: `ops/default/plan/interactive-generator-boxed-documentation.md`
Tracker: `ops/default/tracker/interactive-generator-boxed-documentation.md`
Branch: `feat/generator-documentation-contracts`
PR: #41

## Purpose

Define documentation as a strict typed Hatmax intent and Book-owned planning
capability without inspecting documentation content or mutating Markdown.

## Delivered Behavior

The intent contract now supports `document_feature` for existing behavior and
one to four typed `documentation_targets`. Each target contains only a
Diataxis quadrant, stable subject, and reader goal. The contract rejects
implicit targets, unsupported quadrants, empty or oversized goals, duplicate
quadrant-and-subject pairs, documentation-only domain changes, and invalid
authorization modes.

The Book owns two documentation obligations and four required rules covering
explicit intent, Diataxis semantics, managed sections, and index reachability.
The planner adds those obligations to combined implementation requests and
produces documentation-only effects for `document_feature`. A pure
documentation plan cannot add runtime dependencies or affect a non-document
surface.

The deterministic corpus covers pure, combined, ambiguous, equivalent,
invalid, and ordinary no-documentation requests. Equivalent target order is
normalized before plan sealing, so it produces the same digest.

## Implementation Notes

The intent schema is now version 2 and the plan schema is version 3. The
interpreter output schema exposes the new operation and target structure but
still forbids paths, Markdown, commands, plans, and edits.

Documentation obligations are declared by the archetype's `document_feature`
operation. Combined operations reuse those same obligations instead of
duplicating a second documentation contract. Capability implementation
obligations and dependency effects are excluded from pure documentation
plans.

## Contracts Added or Changed

- `intent.OperationDocumentFeature` identifies documentation-only requests.
- `intent.DocumentationTarget` carries a quadrant, subject, and reader goal.
- `intent.CurrentSchemaVersion` is 2.
- `plan.Plan` carries normalized documentation targets.
- `plan.CurrentSchemaVersion` is 3.
- The `server_rendered_crud` Book archetype owns documentation content and
  navigation obligations on the `documentation` surface.
- Three new Book rules require distinct Diataxis intent, managed-section
  preservation, and canonical index reachability.

## Files of Interest

- `generator/intent/types.go`
- `generator/intent/schema.go`
- `generator/book/archetypes/server-rendered-crud.yaml`
- `generator/book/rules/documentation-*.yaml`
- `generator/plan/expand.go`
- `generator/plan/validate.go`
- `generator/eval/schema.go`
- `generator/eval/testdata/corpus/v1/`

## Validation

- `go test ./generator/intent/... ./generator/book/... ./generator/plan/... ./generator/eval/...`
  passed.
- `go test -race ./generator/intent/... ./generator/plan/... ./generator/eval/...`
  passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

This slice defines intent and plan contracts only. Slice 2 must inventory
documentation files, collect bounded feature evidence, derive target paths and
index effects, and include those inputs in freshness checks. Rendering,
managed-section mutation, conformance, and live Codex interpretation remain in
Slices 3 and 4.
