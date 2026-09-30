<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 2: Documentation Inventory and Planning

Status: delivered
Delivery set: interactive-generator-boxed-documentation
Plan: `ops/default/plan/interactive-generator-boxed-documentation.md`
Tracker: `ops/default/tracker/interactive-generator-boxed-documentation.md`
Branch: `feat/generator-documentation-planning`
PR: #42

## Purpose

Turn explicit documentation intent into bounded project evidence, exact
Diataxis paths, freshness inputs, and inspectable execution-manifest effects
without rendering or committing Markdown.

## Delivered Behavior

Project inspection now inventories only the canonical `docs/` root, quadrant
indexes, subject documents, local links, managed markers, documentation
validation commands, and protected documentation paths. Other Markdown paths
remain visible as unstructured paths and never become project authority.

For an existing canonical CRUD feature, planning extracts bounded evidence
from the model, service, handler, templates, persistence mapping, validation,
wiring, and tests. It reconciles model, form, and schema fields, then seals
source roles and digests into the plan. Planned feature creation, field
addition, and validation changes produce the corresponding planned evidence.
Every existing evidence source must match a project-fingerprint observation.

Documentation planning derives canonical slugs, reader-oriented titles,
target paths, root and quadrant index effects, repository documentation gates,
and managed-file snapshots. Existing unmanaged, malformed, unreadable, or
protected targets fail before execution. Existing managed files are bound to
their inspected digest; absent files remain explicit create effects.

Execution preparation accepts documentation-only and combined plans. Pure
documentation manifests contain only the requested Markdown target, its index
chain, and documentation validation commands. Combined plans retain their
implementation edits and add the same documentation effects under one sealed
manifest. Target and index postconditions require managed markers, titles,
and planned links, while explicit dependencies order content before
navigation.

## Implementation Notes

The plan schema is now version 5 and carries `DocumentationEvidence` plus a
derived `DocumentationPlan`. The execution-manifest schema is version 2 and
adds `update_markdown`; this slice plans that edit kind but does not render or
stage Markdown.

The interaction coordinator already fingerprints the complete bounded Book
surface. Documentation snapshots and evidence digests are checked against
those observations, so source or documentation drift makes execution stale.

## Contracts Added or Changed

- `project.DocumentationInventory` describes canonical managed documentation
  structure without exposing arbitrary prose.
- `project.FeatureEvidence` records canonical feature identity, fields,
  validations, capabilities, source roles, and digests.
- `plan.DocumentationPlan` records exact target and index effects, snapshots,
  links, titles, and repository documentation gates.
- Documentation-only plans select no capability implementation obligations or
  runtime dependency effects.
- `execute.EditUpdateMarkdown` represents a bounded managed-section update.
- Documentation-only manifests omit Go, SQL, template, configuration, test,
  generation, formatting, and unrelated validation work.

## Files of Interest

- `generator/project/documentation.go`
- `generator/project/feature_evidence.go`
- `generator/plan/documentation.go`
- `generator/plan/expand.go`
- `generator/plan/validate.go`
- `generator/execute/catalog.go`
- `generator/execute/prepare.go`
- `generator/execute/documentation_prepare_test.go`

## Validation

- `go test ./generator/project/... ./generator/plan/... ./generator/execute/...`
  passed.
- `go test -race ./generator/project/... ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

This slice prepares exact Markdown effects but deliberately provides no
Markdown renderer or staged mutation. Slice 3 must implement managed-section
editors, the four distinct Diataxis renderers, and independent documentation
conformance. Interactive coordination, Codex interpretation, terminal
presentation, and authenticated documentation smoke coverage remain in Slice
4.
