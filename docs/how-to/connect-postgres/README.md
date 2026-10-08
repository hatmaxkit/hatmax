<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Connect to Postgres

This procedure adds the Hatmax database component to an existing application.

Use the [bootstrap application](../bootstrap-application/README.md) and a
disposable PostgreSQL database owned by this procedure. Provision the database
and role before startup; the component creates a schema, not the database or
role. The role needs schema creation privileges if `database.schema` is set.

## Add configuration

Add the connection fields to `config.yaml`:

```yaml
database:
  host: localhost
  port: 5432
  user: dev
  password: dev
  database: myapp
  sslmode: disable
```

Use environment variables such as `MYAPP_DATABASE_PASSWORD` for deployed
secrets. Do not commit live credentials.

## Create and order the component

`db.New` needs an embedded filesystem even when this step does not run
migrations:

```go
//go:embed assets
var assetsFS embed.FS

database := db.New(assetsFS, db.Postgres, cfg, logger)
starts, stops, registrars := app.Setup(ctx, router, database)
```

Ensure `assets/` contains at least one file so the embed pattern matches.
Place components that call `database.GetDB()` during `Start` after `database`.

This fragment adds imports `embed` and `db` to the bootstrap. Pass `starts`,
`stops` and `registrars` to its existing startup/shutdown functions. If schema
creation fails after a successful ping, close the pool on the startup error
path before exiting. Rollback closes only earlier successful components:

```go
err := app.Start(ctx, logger, starts, stops, registrars, router)
if err != nil {
    _ = database.Stop(ctx)
    return err
}
```

## Verify the connection

Run the process. A successful start logs `Database connection established`.
An unavailable server returns `cannot ping database` before routes are
registered.

See the [Database Reference](../../reference/database/README.md) for connection
and schema behavior.
