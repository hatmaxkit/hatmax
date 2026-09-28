# Slice 4: Deterministic Planning

Status: delivered
Delivery set: interactive-generator-plan-kernel
Plan: `ops/default/plan/interactive-generator-plan-kernel.md`
Tracker: `ops/default/tracker/interactive-generator-plan-kernel.md`
Branch: `feat/generator-deterministic-planning`
PR: `#25` (merged)

## Purpose

Expand an admitted typed intent and validated Book selection into one stable,
inspectable, single-use plan without reading, editing, or executing project
content.

## Delivered Behavior

The new `generator/plan` package defines a versioned plan contract containing
the admitted operation, canonical capabilities and affected surfaces, selected
rules, ordered logical operations, project preconditions and expected
observations, allowed surface and dependency effects, conformance obligations,
documentation intent, exceptions, and a deterministic digest.

`Expand` accepts only an admitted intent result, a validated Book, and the
exact selected project fingerprint. It recomputes the Book selection rather
than trusting model-authored architecture. Archetype and capability
obligations become namespaced logical operations with explicit owners, rules,
surfaces, and dependencies. Stable topological ordering preserves Book order
when multiple operations are available, and affected surfaces retain the
archetype's canonical order.

Every expanded plan is sealed over canonical JSON while remaining available
as stable, user-visible YAML. Digest verification detects semantic mutation.
Fingerprint checks compare the plan with freshly inspected relevant state and
return `HMGEN-PLAN-STALE` plus classified observation changes when drift is
present.

The lifecycle wrapper detaches plan storage and enforces one atomic claim. A
fresh plan becomes `consumed` when claimed for a later executor; rejection or
drift makes it permanently unavailable. This slice performs no execution or
project mutation.

## Implementation Notes

Plan validation rejects missing attribution, forward or unknown dependencies,
incomplete effects, missing conformance checks, version or fingerprint
mismatches, and invalid preconditions. Capability dependencies remain bounded
to the Book allowlist and record their owning capability and purpose.

The lifecycle uses a mutex so concurrent claim attempts produce exactly one
successful transition. Retained plans are returned as deep copies, preventing
callers from mutating the verified internal artifact.

## Contracts Added or Changed

The public plan contract adds plan, rule, owner, operation, precondition,
allowed-effect, dependency-effect, validation-obligation, diagnostic,
freshness, lifecycle-state, and transition-result types. `Expand`, `Seal`,
`VerifyDigest`, `MarshalYAML`, `CheckFingerprint`, and `Activate` define the
planning and lifecycle boundary for later evaluation and execution work.

## Files of Interest

- `generator/plan/types.go`
- `generator/plan/validate.go`
- `generator/plan/serialize.go`
- `generator/plan/expand.go`
- `generator/plan/digest.go`
- `generator/plan/freshness.go`
- `generator/plan/lifecycle.go`

## Validation

- `go test ./generator/plan/...` passed.
- `go test ./generator/book/... ./generator/project/... ./generator/intent/... ./generator/plan/...` passed.
- `go test -race ./generator/plan/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.
- Plan package tests report 92.1% statement coverage.

## Risks and Follow-ups

Logical operations deliberately stop at Book obligations; exact file edits,
content generation, command execution, and conformance evaluation remain
outside this delivery set. Slice 5 owns the deterministic corpus and the
provider-neutral interpreter-evaluation boundary. Live model output remains
unevaluated.
