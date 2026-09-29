# Add Postgres

This chapter enables the database-backed note store and pubsub components. By
the end, startup creates the required tables and the page can list notes.

## Before You Begin

Complete [Run the First Process](getting-started.md). A Postgres server must
accept the values in `examples/guide/config.yaml`.

Create the default database when necessary:

```sh
createdb -h localhost -U dev dev
```

## Enable database components

From `examples/guide`:

```sh
GUIDE_DATABASE_ENABLED=true go run .
```

The entrypoint assembles components in this order:

1. `db.Database` opens and verifies the connection.
2. `noteStore` creates `guide_notes`.
3. The Postgres broker creates its schema.
4. `eventListener` subscribes after the broker is ready.
5. The template manager parses templates.
6. The page registrar installs routes.

Every startup component in this list also implements `Stoppable`, keeping the
current lifecycle rollback slices aligned.

## Check failure containment

Stop the process, set `database.port` to an unused port, and run it again. The
process reports `cannot ping database` and does not listen on port `8080`.
Restore the port and start it again.

## Verify the result

```sh
curl -fsS http://localhost:8080/ | grep -F '<form method="post" action="/notes">'
psql -h localhost -U dev -d dev -c '\d guide_notes'
```

Both commands must succeed. Stop the process with Ctrl+C.

For a focused integration procedure, see
[Connect to Postgres](../../how-to/connect-postgres/README.md). For exact
behavior, see [Database](../../reference/database/README.md).

Continue with [Serve a Page](pages.md).

---

[Previous: Run the First Process](getting-started.md) · [User Guide](README.md) ·
[Next: Serve a Page](pages.md)
