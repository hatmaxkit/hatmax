# Slice 3: Feature Anatomy

Status: reviewing
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-feature-anatomy`
PR: pending

## Purpose

Establish one canonical Hatmax feature shape from model through tests, then
show how responsibilities, values, errors, and changes cross its boundaries
without depending on the generator for the reader's mental model.

## Delivered Behavior

The User Guide now presents a cohesive feature package with explicit model,
store, Postgres adapter, service, feature-owned handler, templates, wiring,
and tests. It connects the package to its migration, SQLC queries, embedded
templates, and `main.go` composition root.

The chapter follows one create workflow from the browser to durable Postgres
state and back to a page or partial. It identifies the representation at each
boundary, preserves classified error meaning, and shows why a field or
validation change must expand across every affected surface.

Testing guidance now follows the domain, application, HTTP, persistence, and
composition contracts instead of file count. Current examples are classified
as evidence: Ticked demonstrates most working surfaces but predates the
feature-owned handler layout, while the guide executable is deliberately
compressed rather than canonical product structure.

## Implementation Notes

The chapter uses an invoice vocabulary and small connected fragments. It does
not instruct the reader to build a cumulative project, reproduce generator
operations, or copy reference tables.

Generator Book policy and the approved server-rendered CRUD archetype were
used to check the documented shape. The reader-facing chapter remains
generator-independent and links exact package behavior to reference
documentation.

## Contracts Added or Changed

- A feature is one cohesive vertical slice under `internal/feat/<feature>`.
- Models own durable rules without depending on HTTP, SQLC, templates, or
  database handles.
- Stores own Postgres adaptation and stable not-found translation; services
  own workflows; handlers own HTTP translation.
- Full-page and partial templates share feature vocabulary and server-owned
  state.
- `main.go` remains a single-function visible composition root.
- Feature changes expand across every affected schema, query, domain,
  transport, presentation, wiring, and test surface.
- Examples provide evidence but do not override the canonical architecture.

## Files of Interest

- `docs/tutorials/user-guide/feature-anatomy.md`
- `docs/tutorials/user-guide/README.md`
- `ops/default/spec/server-rendered-crud.md`
- `generator/book/archetypes/server-rendered-crud.yaml`
- `examples/ticked/internal/feat/list/`

## Validation

- `make docs-check` passed.
- Canonical surface and dependency-order audit passed against the approved
  server-rendered CRUD archetype and current generator renderers.
- Example classification audit passed for `examples/ticked` and
  `examples/guide`.
- The chapter remains understandable without generator execution or generator
  internals.
- `main.go` is documented only as the entrypoint and visible assembly root.
- `git diff --check` passed.

## Risks and Follow-ups

The chapter names Postgres, migrations, SQLC, and store lifecycle only to
place them in the complete feature. Slice 4 owns their full progression and
the replacement of the current data exercises.
