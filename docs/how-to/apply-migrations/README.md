<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Apply Database Migrations

Use `db.Migrator` to apply embedded Postgres migrations during startup.

First complete [Connect to Postgres](../connect-postgres/README.md) with owned
disposable storage. The fragments extend that bootstrap and its embedded
`assetsFS`, `ctx`, router, validated `cfg` and logger.

## Add a migration

Create `assets/migration/postgres/202609270001-create_notes.sql`:

```sql
-- +migrate Up
CREATE TABLE notes (
    id text PRIMARY KEY,
    body text NOT NULL,
    created_at timestamptz NOT NULL
);

-- +migrate Down
DROP TABLE notes;
```

The filename must contain a datetime, a hyphen, a name, and the `.sql`
extension. The `Up` section must not be empty.

## Add the migrator

Create the database and migrator from the same embedded filesystem:

```go
database := db.New(assetsFS, db.Postgres, cfg, logger)
migrator := db.NewMigrator(database, assetsFS, db.Postgres, logger)

starts, stops, registrars := app.Setup(
	ctx,
	router,
	database,
	migrator,
)
```

The database must start first because the migrator reads its connection.

## Verify the migration

Start the application twice. The first start applies the migration and records
it in `migrations`. The second start skips it. Inspect the table with:

```sh
psql -h localhost -U dev -d myapp -c '\d notes'
```

Hatmax parses `Down` sections but does not execute them. See
[Database Reference](../../reference/database/README.md#migrations).

Use the same host, port, role, database and schema as the application when
inspecting results. Set `PGPORT` for a non-default port and `PGOPTIONS` to
`-c search_path=<schema>` for a non-default simple schema name. Supply any
password through your local connection mechanism. A second successful start
leaves one tracking row for this migration and preserves existing note rows.
Do not modify an applied file to request a replay; add a new unique fixed-width
prefix. Earlier successful files remain applied if a later file fails.
