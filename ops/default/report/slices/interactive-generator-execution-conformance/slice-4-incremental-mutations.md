# Slice 4: Incremental Mutations

Status: reviewing
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Branch: `feat/generator-incremental-mutations`
PR: pending

## Purpose

Apply `add_field` and `add_validation` plans across an existing canonical CRUD
feature without duplicating or silently repairing noncanonical structure.

## Delivered Behavior

Incremental execution now inspects the existing feature before rendering. It
parses the model and tests, validates named SQLC queries and templates,
discovers the entity, table, route, label, field order, field types, required
attributes, schema types, and composition wiring, and rejects missing semantic
anchors through `execution_feature_structure_invalid`.

`RenderAddField` reconstructs the canonical feature context, rejects duplicate
fields, adds one reversible column migration, and deterministically rerenders
the model, store mapping, queries, service, handler, templates, and behavioral
tests. The returned 14 mutations retain manifest order and use creation only
for the new migration; existing surfaces use complete bounded replacement.

`RenderAddValidation` resolves the target field and validates the rule against
its domain type. Durable rules produce reversible Postgres constraints,
applicable domain validation with the requested user-facing message, HTML
constraints, and domain and handler coverage. Client-only rules produce HTML interaction
constraints and handler coverage without model or schema mutations.

Supported canonical rules are `required`, `minimum`, `maximum`, `min_length`,
`max_length`, `pattern`, and durable `unique`. Missing fields, invalid numeric
values, malformed patterns, unsupported type combinations, and client-only
durable constraints fail closed with stable execution errors.

Integration tests first generate and atomically apply a complete feature, then
prepare, render, stage, and commit incremental field and validation manifests.
Negative coverage verifies duplicate fields, missing fields, and missing
canonical anchors.

## Implementation Notes

Incremental rendering does not infer a second architecture. It reuses the same
Book-owned renderers as `create_feature`, so domain, persistence, HTTP, HTMX,
and test surfaces remain aligned after a field is added.

Pattern values are validated as regular expressions before rendering and HTML
attributes are escaped. Postgres constraint names are deterministic and every
durable migration includes its inverse operation.

Repository-owned generation and formatting commands remain sealed in the
manifest but are not executed by this slice. Slice 5 owns bounded command
execution after independent conformance succeeds.

## Contracts Added or Changed

The public execution contract adds `RenderAddField` and
`RenderAddValidation`. Both require a sealed incremental plan, matching
manifest, fresh inventory, and an intact canonical feature.

## Files of Interest

- `generator/execute/inspect_feature.go`
- `generator/execute/render_add_field.go`
- `generator/execute/render_add_validation.go`
- `generator/execute/render_domain.go`
- `generator/execute/render_transport.go`
- `generator/execute/inspect_feature_test.go`
- `generator/execute/render_add_field_test.go`
- `generator/execute/render_add_validation_test.go`

## Validation

- `go test ./generator/execute/...` passed.
- `go test -race ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Execution package tests report 82.5% statement coverage.

## Risks and Follow-ups

This slice validates the structure needed to render incremental mutations but
does not independently evaluate every selected Book rule after application.
Slice 5 owns that conformance pass, bounded repository command execution, and
final execution evidence.
