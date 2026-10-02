<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Scaffold Execution

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/application-scaffold-rendering`
PR: #58

## Purpose

Render, stage, validate, and atomically publish one canonical compiling Hatmax
application from an approved standalone application plan.

## Delivered Behavior

Hatmax now renders every file owned by the canonical application archetype.
The foundation includes an explicit module and compatible Hatmax dependency,
bounded root metadata, a `main.go` containing only `main`, configuration,
Hatmax logging, Postgres lifecycle wiring without an ownerless migration,
application composition, and meaningful generated tests.

The web scaffold uses the Hatmax router, default middleware stack, template
manager, and lifecycle contracts. It embeds one layout, a neutral landing
page, and a stylesheet. The scaffold introduces no business model, JSON-first
API, client-side application state, README, Git initialization, or Diataxis
tree.

Application execution manifests use schema version 4 and bind the plan digest,
source fingerprint, absolute admitted target, preserved paths, exact recipes,
and validation commands. Rendering rejects missing, duplicate, unsupported,
or path-divergent recipes.

The application workspace keeps rendered content in memory, copies admitted
existing content into an isolated sibling staging directory, evaluates Book
conformance, resolves module sums, builds, and runs generated tests before
publication. It rechecks the target fingerprint after validation and publishes
the complete directory with a recoverable rename transaction. Cancellation,
concurrent target changes, collisions, incomplete staging, and swap failures
leave the admitted target unchanged. Repeating a commit through the same
workspace reports the already-satisfied result.

Test failures caused by generated behavior block publication. Recognized
missing external test infrastructure is reported separately as incomplete
validation and does not erase an otherwise compiling scaffold.

## Implementation Notes

`PrepareApplication` intentionally accepts only the standalone application
unit in this slice. Initial feature units remain visible in schema 6 plans but
their coordinated execution belongs to Slice 4, where the product layer can
apply the scaffold and dependent feature units under one approval.

Compilation and generated tests execute against the isolated staging tree.
`go mod tidy` completes `go.sum` before build so the published application is
reproducible with module-readonly testing.

Application conformance is independent from command success. It checks exact
plan-to-manifest file coverage, module identity, the thin-main AST boundary,
explicit Hatmax lifecycle assembly, Postgres wiring without a bootstrap
migrator, and the neutral embedded web surface.

## Files of Interest

- `generator/execute/render_application.go`
- `generator/execute/render_application_web.go`
- `generator/execute/prepare_application.go`
- `generator/execute/application_workspace.go`
- `generator/execute/conformance_application.go`
- `generator/execute/application_acceptance_test.go`

## Validation

- `go test ./generator/execute/... ./generator/project/... ./generator/plan/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make generator-scaffold-acceptance` passed.
- `make vet` passed.
- `make lint-strict` passed.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 4 must expose application interpretation and approval through the
  existing product surface instead of calling execution recipes directly.
- Slice 4 must coordinate initial feature units after the scaffold while
  preserving one approved composite plan and one observable operation.
- Later application evolution must inspect the published project through the
  normal compatible-project inventory rather than reusing pre-project target
  state.
