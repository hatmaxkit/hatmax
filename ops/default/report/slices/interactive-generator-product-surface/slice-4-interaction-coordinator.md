# Slice 4: Interaction Coordinator

Status: delivered
Delivery set: interactive-generator-product-surface
Plan: `ops/default/plan/interactive-generator-product-surface.md`
Tracker: `ops/default/tracker/interactive-generator-product-surface.md`
Branch: `feat/generator-interaction-coordinator`
PR: `#35` (merged)

## Purpose

Compose the delivered planning kernel and executor into one backend-neutral
interaction while keeping approval, rendering, mutation, conformance, and
repository validation under Hatmax control.

## Delivered Behavior

The `generator/interaction` package defines stable lifecycle states, terminal
outcomes, diagnostics, bounded provenance, and explicit clarification and
approval ports. A coordinator inspects the target project, loads the canonical
Book, computes a bounded fingerprint, obtains a typed interpretation, and
continues focused clarification through explicit answers only.

The initial fingerprint conservatively covers every surface and external
dependency admitted by the selected Book. This keeps the normal path to one
model turn while ensuring that every surface a resulting plan may select is
already protected by drift detection. Unrelated project content remains
outside the fingerprint.

An admitted intent expands through the deterministic planner. The coordinator
serializes the sealed plan as canonical YAML, supplies that exact projection,
plan digest, and source fingerprint to the approval port, and performs no edit
unless approval is explicitly granted. It reinspects the project after
approval and uses the single-use plan lifecycle to reject drift before handing
the plan to execution.

Approved plans dispatch only to `RenderCreateFeature`, `RenderAddField`, or
`RenderAddValidation`. The existing execution manifest, workspace staging,
atomic commit, conformance, and repository-command APIs own all project
effects. No model-produced source, command, dependency, path, repair, or
approval enters execution.

A structural application failure uses the workspace rollback behavior and
returns no retained changes. After a successful source commit, conformance or
repository-command failure retains and reports every applied change together
with the partial execution report and exact command evidence. The coordinator
does not run Git commands or create commits.

## Contracts Added or Changed

- `interaction.Config` binds one interpreter, approver, optional clarifier,
  Book, and project inspection limits.
- `interaction.Coordinator.Run` owns the complete backend-neutral lifecycle.
- `ApprovalRequest` binds one explicit decision to canonical plan YAML, its
  digest, and the source project fingerprint.
- `ClarificationRequest` and `ClarificationResponse` keep product answers
  explicit and bounded independently of backend thread history.
- `Result` carries the sealed plan, backend provenance, execution outcome,
  conformance report, diagnostics, drift evidence, and retained changes.

## Files of Interest

- `generator/interaction/types.go`
- `generator/interaction/coordinator.go`
- `generator/interaction/execute.go`
- `generator/interaction/coordinator_test.go`
- `generator/interaction/execute_test.go`

## Validation

- `go test ./generator/interaction/... ./generator/execute/...` passed.
- `go test -race ./generator/interaction/... ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed with zero issues.
- `git diff --check` passed.

The fixture workflow exercises `create_feature`, `add_field`, and
`add_validation` through planning, approval, canonical rendering, atomic
application, conformance, and sealed command execution. Separate cases cover
clarification continuation, unsupported intent, approval-time drift,
pre-mutation rendering failure, and post-commit validation failure with
retained changes and command evidence.

## Risks and Follow-ups

Slice 4 provides no terminal command or user-facing wording. Slice 5 must bind
the Codex interpreter and terminal ports, map stable outcomes to exit statuses,
and present the bounded evidence without exposing raw backend events.

The exact integrated candidate still requires the authenticated Codex smoke.
The standalone Codex daemon prerequisite observed in Slice 3 remains unresolved
on this host.
