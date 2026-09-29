# Persistence and Migrations

Hatmax treats Postgres as an explicit application dependency. The composition
root owns connection and migration lifecycle; each feature owns its schema,
queries, mappings, and persistence rules.

```text
config -> database -> migrator -> feature stores -> services
```

This order makes failure containment predictable. No store receives traffic
until the connection is live and every pending migration has succeeded.

## Start One Managed Connection

Construct the database without opening a connection:

```go
database := db.New(assetsFS, db.Postgres, cfg, logger)
```

`db.Database.Start` opens the configured connection through the Postgres
driver and verifies it with a ping. When the application selects a schema, it
also ensures that schema exists. `GetDB` exposes the connection only after a
successful start; `Stop` closes it during reverse shutdown.

Code that needs SQL depends on a narrow provider:

```go
type DBProvider interface {
	GetDB() *sql.DB
}
```

The provider keeps constructors free of I/O. A store retains it during
construction, then acquires the live `*sql.DB` in `Start`. This is why stores
follow the database and migrator in the lifecycle list.

Connection pool sizing, credentials, host selection, and schema selection are
startup configuration. Feature packages consume the managed connection; they
do not open their own pools or read configuration independently.

## Make Schema History Executable

Embed migrations with the application and construct one migrator:

```go
migrator := db.NewMigrator(
	database,
	assetsFS,
	db.Postgres,
	logger,
)
```

The default location is `assets/migration/postgres`. Files use an ordered
identifier and descriptive name:

```text
assets/migration/postgres/004-invoices.sql
```

Each file separates forward and reverse definitions:

```sql
-- +migrate Up
CREATE TABLE invoices (
    id TEXT PRIMARY KEY,
    number TEXT NOT NULL UNIQUE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX invoices_updated_at_idx
    ON invoices (updated_at DESC, id DESC);

-- +migrate Down
DROP TABLE invoices;
```

Migration order is data, not decoration. Once an identifier has been applied
to a shared database, add a new migration instead of rewriting its history.
Choose nullability, uniqueness, foreign keys, checks, and indexes from the
feature invariants and supported queries, not from Go field types alone.

During startup the migrator reads its files, compares them with the
`migrations` table, and applies each pending `Up` section in its own
transaction. It records the migration only after the SQL succeeds. A failure
rolls back that file, stops subsequent migrations, and prevents route
registration.

The current migrator parses `Down` sections but does not execute them. They
still document the reverse operation and keep migration files structurally
complete; application rollback planning must account for the deployed schema
explicitly.

## Keep Queries Named and Generated

Feature SQL lives in `db/queries/<feature>.sql` as named SQLC operations:

```sql
-- name: CreateInvoice :exec
INSERT INTO invoices (
    id, number, notes, created_at, updated_at
)
VALUES (
    sqlc.arg(id), sqlc.arg(number), sqlc.arg(notes),
    sqlc.arg(created_at), sqlc.arg(updated_at)
);

-- name: GetInvoice :one
SELECT id, number, notes, created_at, updated_at
FROM invoices
WHERE id = sqlc.arg(id);
```

SQLC makes selected columns, arguments, and row cardinality part of the Go
build. Regenerate the DAL whenever a migration changes a queried column, then
compile before adapting feature code. Do not silently replace the project
SQLC pipeline with ad hoc SQL because generation or compilation reveals a
mismatch.

The query file expresses persistence operations. The feature store remains
responsible for domain mapping, stable errors, and transactions that combine
multiple operations.

## Adapt Rows in the Feature Store

`PostgresStore.Start` builds `dal.Queries` from the managed connection. Store
methods then translate in both directions:

```text
domain model -> SQLC parameters -> Postgres
domain model <- SQLC row        <- Postgres
```

Keep generated DAL types inside this adapter. A service receives `Invoice`,
not `dal.Invoice`; the model never imports the generated package. Conversion
code is intentionally explicit so nullability, names, timestamps, and enum
representations can be reviewed.

Translate `sql.ErrNoRows` and zero affected-row results into the feature's
stable `ErrNotFound`. Wrap other failures with the failed operation while
preserving the cause. The handler can then use `errors.Is` without learning
database details.

Use a transaction when several statements protect one application invariant.
Begin it in the store method, derive a transaction-bound SQLC query set, defer
rollback, and commit only after every operation succeeds. Transaction
boundaries follow the durable operation, not the number of repository
methods.

## Wire Persistence in Dependency Order

The relevant composition remains visible in `main`:

```go
database := db.New(assetsFS, db.Postgres, cfg, logger)
migrator := db.NewMigrator(database, assetsFS, db.Postgres, logger)
invoiceStore := invoicefeat.NewPostgresStore(database)
invoiceService := invoicefeat.NewService(invoiceStore)
invoiceHandler := invoicefeat.NewHandler(invoiceService, tmplMgr, logger)

deps := []any{
	database,
	migrator,
	tmplMgr,
	invoiceStore,
	invoiceHandler,
}
```

The migrator follows the database because it needs a live connection. The
store follows the migrator because its queries require the current schema.
The handler registers routes only after all startup functions succeed.
Shutdown walks stoppable components in reverse order.

## Separate Migrations from Seed Data

A migration defines schema and mandatory data transitions. Seeders create
named development, bootstrap, or reference data through application-aware
code. They are not a substitute for a required schema or data migration.

`seed.Runner` executes registered seeders in order after it obtains the live
database. It records successful names in `_seeds` and skips them on later
runs. A seeder must therefore have a stable name and tolerate the recovery
semantics of its external effects: if its work succeeds but tracking fails,
it can run again at the next start.

Use `seed.RefMap` when several seeders need stable symbolic references to IDs
created earlier in the same run. Keep seed runners after the database and
migrator, and before components that require the seeded records.

## Verify the Durable Boundary

Persistence tests use `testhelper.SetupTestDB` and exercise real Postgres
behavior. Cover migrations, SQLC queries, row mapping, constraints,
transactions, and missing-row translation. Service fakes cannot prove any of
those contracts.

Before delivery, regenerate SQLC output and run the project build, tests, and
lint gates. A migration that parses but disagrees with generated queries is
not a complete change.

For exact behavior, see [Database](../../reference/database/README.md),
[Seed](../../reference/seed/README.md), and
[Test Helper](../../reference/testhelper/README.md). For the architectural
default and its tradeoffs, read
[Postgres-First Infrastructure](../../explanation/postgres-first/README.md).

---

[Previous: Feature Anatomy](feature-anatomy.md) ·
[User Guide](README.md)
