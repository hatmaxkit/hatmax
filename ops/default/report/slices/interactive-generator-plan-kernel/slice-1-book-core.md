# Slice 1: Book Core

Status: reviewing
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Branch: `feat/generator-book-core`
PR: pending

## Purpose

Establish the versioned, embedded Hatmax Book that later generator slices use
to select canonical capabilities, archetypes, obligations, and rules.

## Delivered Behavior

The `generator/book` package strictly loads and validates one embedded Book.
It rejects unsupported schemas, invalid Hatmax version ranges, malformed or
unreferenced entries, duplicate IDs and paths, missing references, capability
cycles, and invalid rule diagnostics before the Book can be selected.

Callers can check Hatmax version compatibility and deterministically select the
`server_rendered_crud` archetype with `postgres_persistence`, `htmx_form`, and
`runtime_validation`. Required capability prerequisites and applicable rules
are expanded in manifest order, and returned values do not expose mutable Book
state.

## Implementation Notes

The initial Book is embedded from YAML and text fixtures under
`generator/book/`. Its compatibility range is Hatmax `0.4.0` inclusive through
`0.5.0` exclusive. Manifest validation runs before entry loading so incompatible
schemas and invalid or duplicate paths fail at the owning boundary.

## Contracts Added or Changed

The new public `generator/book` package defines the Book schema, stable
validation diagnostics, Hatmax version compatibility, immutable-in-practice
accessors, and deterministic archetype and capability selection. The initial
Book defines one canonical server-rendered CRUD assembly and nine rules.

## Files of Interest

- `generator/book/manifest.yaml`
- `generator/book/types.go`
- `generator/book/load.go`
- `generator/book/validate.go`
- `generator/book/select.go`
- `generator/book/archetypes/server-rendered-crud.yaml`
- `generator/book/capabilities/`
- `generator/book/rules/`
- `generator/book/examples/`

## Validation

- `go test ./generator/book/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

This slice provides selection and validation only. Project inventory, typed
intent admission, deterministic planning, and model-interpreter evaluation
remain isolated in later slices of this delivery set.
