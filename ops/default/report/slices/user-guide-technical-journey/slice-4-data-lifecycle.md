# Slice 4: Data Lifecycle

Status: reviewing
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-data-lifecycle`
PR: #51

## Purpose

Explain the complete Hatmax data path from Postgres connection and schema
startup through migrations, SQLC, feature stores, domain models, services,
presentation data, and durable verification.

## Delivered Behavior

The User Guide now presents Postgres as an explicit lifecycle dependency. It
explains connection startup, transactional forward migrations, schema and
query ownership, SQLC generation, adapter mapping, stable errors,
transactions, dependency-order wiring, seed execution, and real Postgres
tests.

The model chapter distinguishes form, feature input, domain, generated DAL,
and view representations. It follows identity, UTC timestamps,
normalization, validation, optionality, row mapping, service workflows,
aggregates, derived slugs, seed references, and request-context propagation
through the complete round trip.

The previous Add Postgres and Save a Record exercises were removed after
their durable startup, persistence-boundary, restart, and recovery material
was incorporated into the new journey. Their inbound links now point to the
new chapters.

## Implementation Notes

The chapters explain architecture and evolution rather than asking readers to
enable a repository example. SQL and Go fragments clarify ownership without
reproducing database or package reference tables.

The guide records the current migration limitation explicitly: `Down`
sections are parsed but not executed by the current migrator. It also
separates mandatory schema and data migrations from tracked application-aware
seeders.

## Contracts Added or Changed

- The application owns one managed Postgres connection and one ordered
  migration lifecycle.
- Features own schema requirements, named SQLC queries, explicit mappings,
  stable persistence errors, and transaction boundaries.
- Domain and generated DAL representations remain separate.
- Durable rules align across model validation and applicable Postgres
  constraints.
- Seed data is tracked bootstrap behavior, not schema migration history.
- Stored fields are traced in both directions across every affected
  representation and test surface.

## Files of Interest

- `docs/tutorials/user-guide/persistence-and-migrations.md`
- `docs/tutorials/user-guide/models-and-data-flow.md`
- `docs/tutorials/user-guide/feature-anatomy.md`
- `docs/tutorials/user-guide/README.md`

## Validation

- `make docs-check` passed.
- Data capability audit passed for `db`, `model`, `validation`, `seed`,
  `slug`, SQLC, and `testhelper`.
- Database startup, migration, and seed behavior was checked against current
  package implementations and reference contracts.
- Superseded `postgres.md` and `records.md` link audit passed.
- `git diff --check` passed.

## Risks and Follow-ups

Authentication models and session persistence are deliberately deferred to
Slice 5. Pubsub, scheduler, mail, image, and telemetry persistence belong to
their supporting-service chapters rather than this feature data flow.
