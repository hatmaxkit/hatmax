# Slice 2: Application Planning Kernel

Status: delivered
Delivery set: conversational-hatmax-builder
Plan: [Conversational Hatmax Builder Delivery Plan](../../../plan/conversational-hatmax-builder.md)
Tracker: [Conversational Hatmax Builder Tracker](../../../tracker/conversational-hatmax-builder.md)
Branch: `feat/application-scaffold-kernel`
PR: #57

## Purpose

Inventory proposed application targets, validate `create_application`, and
produce sealed canonical scaffold plans without writing project files.

## Delivered Behavior

Hatmax can inspect an authorized parent and child target before a project
exists. The inventory classifies absent, empty, preservable, compatible, and
incompatible targets; records exact preserved entries and collisions; derives
credential-free module evidence from Git remotes; and fingerprints target,
remote, repository-rule, and planned-path state.

Intent schema version 3 now represents application identity, target,
enrichment, and bounded initial features through `source_fingerprint`. Display
names deterministically produce project slugs and default child directories.
Module paths come from unambiguous remote evidence or a focused clarification,
never from an invented hosting owner. Schema version 3 also carries
`source_fingerprint` for existing-project operations while schema version 2
remains compatible.

Book release 2 adds the canonical
`server_rendered_hatmax_application` archetype, scaffold rules, operation-bound
dependencies, and exact file-effect templates. Application planning emits one
sealed application plan with ordered visible units for the scaffold and each
initial CRUD feature. The plan records creates, updates, preserved files,
dependencies, rules, validations, preconditions, observations, and a digest
bound to the inspected target.

## Implementation Notes

`book.LoadDefault` still selects delivered Book release 1 for existing product
surfaces. `book.LoadRelease(2)` overlays the application and compatible CRUD
contracts without duplicating unchanged release-1 entries.

The application plan uses schema version 6 so its source fingerprint, target,
units, and file effects remain structurally distinct from delivered
existing-project plans. Initial features depend on the application unit,
migrations receive stable sequence names, shared configuration paths become
ordered create-then-update effects, and a neutral scaffold contains no
bootstrap migration, README, Git initialization, or documentation tree.

This slice performs no rendering or mutation. Scaffold source generation,
atomic publication, compilation, and conformance belong to Slice 3.

## Files of Interest

- `generator/project/target.go`
- `generator/project/target_fingerprint.go`
- `generator/intent/application.go`
- `generator/book/release2/manifest.yaml`
- `generator/book/release2/archetypes/server-rendered-hatmax-application.yaml`
- `generator/plan/application.go`
- `generator/plan/validate_application.go`

## Validation

- `go test ./generator/book/... ./generator/project/... ./generator/intent/... ./generator/plan/...` passed.
- `go test -race ./generator/project/... ./generator/intent/... ./generator/plan/...` passed.
- `go test ./generator/...` passed.
- `make vet` passed.
- `make lint-strict` passed.
- `git diff --check` passed.

## Risks and Follow-ups

- Slice 3 must render every Book-owned file effect and reject any divergence
  between the sealed plan and staged output.
- Slice 3 must extend execution freshness and manifests to consume
  application-plan schema version 6 and `source_fingerprint`.
- Slice 4 must perform the final exact-path target inventory after intent
  collection and before presenting the plan for approval.
