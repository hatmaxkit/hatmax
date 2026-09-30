<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Slice 3: CRUD Generation

Status: delivered
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Branch: `feat/generator-crud-generation`
PR: `#29` (merged)

## Purpose

Render and wire one complete canonical `server_rendered_crud` feature from a
sealed `create_feature` plan and its prepared execution manifest.

## Delivered Behavior

The execution package now renders every create-feature edit through Book-owned
recipes. `RenderCreateFeature` verifies plan and manifest identity, requires a
renderer for every declared edit, preserves manifest order, and returns typed
create or replace mutations without changing the inspected project.

Domain rendering produces the feature model and input, domain validation,
consumer-owned store contract, SQLC-backed Postgres adapter, service workflows,
and model, service, and Postgres tests. Every admitted field type has one
deterministic Go, SQL, form, and parsing representation. Unresolved business
rules fail closed instead of producing placeholders or improvised behavior.

Transport rendering produces a reversible migration, named SQLC list, get,
create, update, and delete queries, a feature-owned Chi handler, full-page and
HTMX response paths, Hatmax-backed form parsing and validation, and page, form,
and row templates using Hatmax HTMX helpers. Handler and persistence tests are
part of the same manifest.

Composition-root rendering parses `main.go` structurally. It requires the
canonical `database`, `migrator`, `tmplMgr`, `logger`, and `deps := []any{...}`
prerequisites, adds a deterministic feature import, constructs the Postgres
store, service, and handler explicitly, and inserts lifecycle and route
components into the dependency list after their required infrastructure.
Missing prerequisites and identifier or import conflicts produce stable
execution errors rather than textual guesses.

An integration test renders all 15 manifest edits, stages them through the
bounded workspace, commits the transaction, and verifies that every target and
the explicit composition wiring were applied.

## Implementation Notes

Generated IDs and timestamps use Hatmax model primitives. Durable validation
stays in the model, handlers use Hatmax web, validation, logging, and HTMX
primitives, and Postgres construction follows the provider and lifecycle
contracts established by the Book.

Rendering is deterministic and side-effect free. Repository-owned generation
and formatting commands remain sealed in the manifest but are not executed by
this slice. Slice 5 owns bounded command execution after independent
conformance succeeds, as required by the execution specification.

## Contracts Added or Changed

The public execution contract adds `RenderCreateFeature`, which turns one
validated create-feature plan, manifest, and inventory into the complete
ordered mutation set. Renderer failures preserve stable execution error codes,
and update recipes produce replacement mutations while create-file recipes
produce creation mutations.

## Files of Interest

- `generator/execute/render.go`
- `generator/execute/render_domain.go`
- `generator/execute/render_transport.go`
- `generator/execute/render_wiring.go`
- `generator/execute/render_domain_test.go`
- `generator/execute/render_transport_test.go`
- `generator/execute/render_wiring_test.go`

## Validation

- `go test ./generator/execute/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Execution package tests report 84.4% statement coverage.

## Risks and Follow-ups

This slice implements complete `create_feature` rendering only. Slice 4 owns
semantic discovery and cross-surface `add_field` and `add_validation`
mutations. Slice 5 owns independent Hatmax conformance, bounded repository
command execution, and final execution evidence.
