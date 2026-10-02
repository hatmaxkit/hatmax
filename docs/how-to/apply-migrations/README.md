<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Apply Database Migrations

Use `db.Migrator` to apply embedded Postgres migrations during startup.

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
psql myapp -c '\d notes'
```

Hatmax parses `Down` sections but does not execute them. See
[Database Reference](../../reference/database/README.md#migrations).
