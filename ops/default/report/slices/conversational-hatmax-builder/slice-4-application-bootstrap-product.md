# Slice 4: Application Bootstrap Product

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/application-scaffold-product`
PR: #59

## Purpose

Expose canonical Hatmax application creation through the existing interactive
product surface, from bounded interpretation through one approved and atomic
bootstrap operation.

## Delivered Behavior

The interpreter accepts `create_application` requests with application
identity, niche context, target evidence, and optional initial CRUD features.
It keeps paths and file ownership outside model control. PostgreSQL and the
canonical Hatmax web foundation are implicit product decisions rather than
clarifications that users must restate.

Application creation starts from a parent directory, derives the normalized
application slug, then inventories the exact child target before committing to
its module path and source fingerprint. Missing module identity is clarified
only after that exact inspection cannot establish it. Conflicting or stale
target evidence is rejected.

The coordinator presents one plan and requests one approval for the complete
bootstrap. A foundation-only request remains one application unit. A request
with initial features becomes one composite plan whose application, migrations,
models, stores, services, handlers, templates, SQL queries, wiring, tests, and
generated data-access code are prepared in one isolated workspace and
published atomically.

Terminal reporting distinguishes interpretation, clarification, approval,
execution, cancellation, stale targets, validation failures, and successful
application publication. It reports the application target and composite unit
count without exposing backend-owned paths during interpretation.

## Implementation Notes

Application interpretation uses contract and output schema version 2. The
request includes only bounded target facts; the evaluator constructs and
validates canonical application intent fields before deterministic planning.

The coordinator evaluates application intent twice. The first pass establishes
the normalized application identity. The second pass is bound to the exact
derived target inventory and supplies the authoritative module path or the
evidence needed for a clarification.

Composite rendering reuses the existing canonical CRUD recipes. It adds one
application-owned `sqlc.yaml`, combines feature assembly in
`internal/application/application.go`, and runs `sqlc generate`, dependency
resolution, compilation, and generated tests before publication. The final
manifest remains create-only and represents the complete approved application.

## Files of Interest

- `generator/eval/request.go`
- `generator/eval/schema.go`
- `generator/backend/codex/prompt.go`
- `generator/interaction/coordinator.go`
- `generator/interaction/execute.go`
- `generator/execute/render_application_features.go`
- `generator/execute/prepare_application.go`
- `internal/hatmaxcli/acceptance_test.go`
- `internal/hatmaxcli/report.go`

## Validation

- `go test ./generator/eval/... ./generator/backend/codex/... ./generator/interaction/... ./internal/hatmaxcli/...` passed.
- `go test -race ./generator/interaction/... ./generator/execute/...` passed.
- `go test ./generator/...` passed.
- `make generator-scaffold-acceptance` passed.
- The detailed composite terminal acceptance passed with real `go` and `sqlc`
  commands, including generated data access, compilation, and generated tests.
- `make vet` passed.
- `make lint-strict` passed.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 5 must persist bounded conversation state without turning the builder
  into a general-purpose harness.
- Existing compatible-project evolution continues through the established
  project inventory and feature-generation path; it does not reuse pre-project
  application target state.
- Real composite bootstrap requires `sqlc` on the execution path. Tool
  discovery and end-user presentation remain product-surface concerns for
  later slices.
