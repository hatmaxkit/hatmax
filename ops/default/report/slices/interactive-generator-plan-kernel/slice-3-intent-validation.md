# Slice 3: Intent Validation

Status: delivered
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Branch: `feat/generator-intent-validation`
PR: `#24` (merged)

## Purpose

Represent and semantically validate the first typed generator operations
without calling a model, expanding implementation obligations, or modifying a
project.

## Delivered Behavior

The new `generator/intent` package defines a strict, versioned YAML envelope
for `create_feature`, `add_field`, and `add_validation`. The envelope carries
the selected project fingerprint, Hatmax and Book versions, archetype,
feature, domain decisions, requested capabilities, explicit documentation
intent, and bounded exception requests. Unknown fields, multiple documents,
invalid operation shapes, malformed fingerprints, and duplicate capabilities
are rejected during decoding.

Semantic validation binds the typed intent to one inspected project,
fingerprint, and validated Book. It rejects stale state, incompatible Hatmax
versions, unavailable archetypes or operations, unsupported capabilities,
missing prerequisites, conflicting project state, invalid domain decisions,
and unexecutable exceptions with stable `HMGEN-*` diagnostics. Missing product
decisions produce field-specific clarification questions without asking the
caller to choose Hatmax-owned architecture.

Admitted results include the deterministic Book selection in canonical order.
Validation does not expand execution operations and does not read or write
project files.

## Implementation Notes

Schema validation is independent of project state. Semantic validation then
applies context checks, Book selection, capability rules, domain rules, and
exception handling in a stable order. Documentation defaults to
`not_requested`; no documentation work is inferred from a feature request.

The first delivery parses approved exceptions but deliberately returns
`HMGEN-EXCEPTION-EXECUTION-UNSUPPORTED`, because executing departures from the
Book is outside this planning-kernel slice.

One supporting fix force-tracks the project inventory's `AGENTS.md` fixture.
The file existed during Slice 2 validation but the machine-wide Git ignore
excluded it from the commit, causing project tests to fail in clean checkouts.

## Contracts Added or Changed

The public intent contract adds operation, documentation, validation-scope,
domain, field, business-rule, exception, result, diagnostic, clarification,
and validation-context types. `DecodeYAML`, `ValidateSchema`, and `Validate`
form the strict decoding and admission boundary for later deterministic plan
expansion.

## Files of Interest

- `generator/intent/types.go`
- `generator/intent/decode.go`
- `generator/intent/schema.go`
- `generator/intent/validate.go`
- `generator/intent/capabilities.go`
- `generator/intent/domain.go`
- `generator/intent/exceptions.go`
- `generator/intent/result.go`
- `generator/intent/testdata/`

## Validation

- `go test ./generator/intent/...` passed.
- `go test ./generator/book/... ./generator/project/... ./generator/intent/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Intent package tests report 84.8% statement coverage.

## Risks and Follow-ups

Typed intent construction remains provider-independent; no live model output
is accepted by this slice. Slice 4 owns deterministic expansion from admitted
intent and Book selection into an inspectable execution plan. Exception
execution remains excluded from the first delivery set.
