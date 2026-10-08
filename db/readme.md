<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# db

PostgreSQL connection with lifecycle management.

## Usage

This fragment belongs in a bootstrap startup function with `ctx`, `cfg` and a
logger. Import `embed` and `db`; keep at least one file below `assets/`.

```go
//go:embed assets
var assetsFS embed.FS

database := db.New(assetsFS, db.Postgres, cfg, logger)

// Implements app.Startable/Stoppable
err := database.Start(ctx)
if err != nil {
    _ = database.Stop(ctx)
    return err
}
defer database.Stop(ctx)

// Use the connection
sqlDB := database.GetDB()
```

Creates schema automatically if `cfg.Database.Schema` is set.
The caller closes a pool retained after a schema error. `Stop` closes the pool,
not the schema. See [Database](../docs/reference/database/README.md).

## Directory Structure

Migrations are loaded from `assets/migration/{engine}/` by default:

```
assets/
└── migration/
    └── postgres/
        ├── 20240101120000-create_users.sql
        ├── 20240102090000-add_email_index.sql
        └── 20240115143000-create_orders.sql
```

Filenames must follow `{datetime}-{name}.sql` format. The datetime prefix determines execution order.

## Migration File Format

Each file uses `-- +migrate Up` and `-- +migrate Down` markers:

```sql
-- +migrate Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE
);

-- +migrate Down
DROP TABLE users;
```

Use `SetPath()` on the Migrator to override the default path.
