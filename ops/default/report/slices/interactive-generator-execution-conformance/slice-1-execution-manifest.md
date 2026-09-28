# Slice 1: Execution Manifest

Status: reviewing
Delivery set: interactive-generator-execution-conformance
Plan: `ops/default/plan/interactive-generator-execution-conformance.md`
Tracker: `ops/default/tracker/interactive-generator-execution-conformance.md`
Branch: `feat/generator-execution-manifest`
PR: pending

## Purpose

Turn a sealed semantic plan into a complete, inspectable set of typed edits
without modifying the project.

## Delivered Behavior

Plan schema version 2 preserves the normalized domain decisions required for
execution. Domain fields, validation requests, and business rules are deep
copied from admitted intent, validated again when a plan is sealed, serialized
in stable order, and included in plan digest identity.

Plans now also preserve the exact selected paths, dependencies, and surfaces
used to compute their project fingerprint. Execution reproduces that bounded
fingerprint and rejects stale project state before resolving any target.

The new `generator/execute` package defines versioned, sealed manifests with
typed edits, Book obligation attribution, project-relative targets,
implementation slots, preconditions, postconditions, bounded repository
commands, and stable diagnostics. Validation rejects path escapes, undeclared
surfaces, duplicate targets, invalid ordering, malformed slots, and unknown
edit or condition kinds.

`Prepare` produces deterministic manifests for `create_feature`, `add_field`,
and `add_validation`. It uses one established project layout when present and
the canonical Hatmax layout otherwise. It computes the next migration name,
selects the composition root, includes only repository-owned commands, and
keeps client-only validation out of durable migration edits.

Preparation rejects Book or fingerprint drift, missing canonical structure,
ambiguous layout roots, generated files, protected paths, dirty overlap,
occupied create targets, missing update targets, and missing SQLC generation
configuration. It performs no project writes or command execution.

## Implementation Notes

One edit can satisfy several Book obligations because a single file can own
domain, validation, HTTP, and HTMX behavior. Each obligation retains its plan
operation, owner, and rule IDs, while edit dependencies are derived from plan
operation ordering.

Project inventory now exposes detached file metadata without exposing file
content. Persistence classification recognizes canonical SQL query files,
singular migration directories, and `*_store.go` adapters so fingerprints
cover the surfaces execution can target.

Existing singular SQLC, migration, query, template, feature, and composition
roots are reused. Ambiguous roots fail closed instead of selecting one
heuristically.

## Contracts Added or Changed

The plan schema advances to version 2 and adds sealed domain and fingerprint
inputs. The public execution contract adds manifest, edit, obligation,
condition, implementation-slot, command, diagnostic, and error types together
with `Validate`, `Seal`, `VerifyDigest`, `MarshalYAML`, and `Prepare`.

Project inventory adds detached `File` metadata through `Files` and `File`.

## Files of Interest

- `generator/execute/types.go`
- `generator/execute/validate.go`
- `generator/execute/prepare.go`
- `generator/execute/catalog.go`
- `generator/plan/types.go`
- `generator/plan/expand.go`
- `generator/project/files.go`
- `generator/project/layout.go`
- `generator/execute/prepare_test.go`

## Validation

- `go test ./generator/plan/... ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Execution package tests report 81.0% statement coverage.

## Risks and Follow-ups

This slice prepares and validates an execution manifest but does not render,
stage, apply, roll back, or run its commands. Slice 2 owns atomic filesystem
application and idempotency. Slice 3 owns the Book-backed CRUD render recipes.
