# Slice 2: Project Inventory

Status: reviewing
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Branch: `feat/generator-project-inventory`
PR: pending

## Purpose

Provide a bounded, read-only semantic inventory and relevant-state fingerprint
for Hatmax project planning without ingesting or modifying unrelated project
content.

## Delivered Behavior

The new `generator/project` package inspects a project root and records its Go
module and Go version, Hatmax version and effective source, declared
dependencies, optional Git revision and dirty paths, executable entrypoints and
composition roots, canonical layouts, repository instructions, protected
paths, and declared validation, formatting, and generation commands.

Hatmax resolution distinguishes versioned modules, local `replace` directives,
`go.work` workspaces, the Hatmax main module, and missing dependencies. The
inventory can evaluate the discovered version against a validated Hatmax Book
without executing project-owned commands.

Fingerprinting selects a Book version, explicit input paths, relevant module
dependencies, and planned surfaces. It hashes only those inputs plus applicable
repository rules, command definitions, Hatmax resolution, and overlapping dirty
state. Comparing fingerprints classifies added, modified, and removed relevant
observations. Unselected files and dependencies do not invalidate a plan.

## Implementation Notes

Inspection is bounded by file-count and file-size limits, skips dependency and
temporary trees, does not follow symbolic links, and parses Go source only for
semantic entrypoint and generated-file observations. Git commands are limited
to read-only revision and porcelain-status queries. Make targets and SQLC
configuration are observed but never executed.

One supporting fix keeps a selected input's observation identity stable when a
previously missing path appears, so drift is reported as one modification.

## Contracts Added or Changed

The public project contract adds inventory, module source, repository state,
layout, command, compatibility, fingerprint, observation, and change types. It
also adds `golang.org/x/mod` for direct parsing of `go.mod` and `go.work`.

## Files of Interest

- `generator/project/types.go`
- `generator/project/inspect.go`
- `generator/project/module.go`
- `generator/project/repository.go`
- `generator/project/layout.go`
- `generator/project/commands.go`
- `generator/project/compatibility.go`
- `generator/project/fingerprint.go`
- `generator/project/change.go`
- `generator/project/testdata/`

## Validation

- `go test ./generator/project/...` passed.
- `go test ./generator/book/... ./generator/project/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

## Risks and Follow-ups

Inventory reports observations rather than deciding whether an incomplete or
incompatible project admits an operation. Slice 3 owns those semantic intent
diagnostics. Drift checks must use a freshly inspected inventory immediately
before execution, as required by the planning lifecycle.
