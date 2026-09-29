# Connect to Postgres

This procedure adds the Hatmax database component to an existing application.

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

## Verify the connection

Run the process. A successful start logs `Database connection established`.
An unavailable server returns `cannot ping database` before routes are
registered.

See the [Database Reference](../../reference/database/README.md) for connection
and schema behavior.
