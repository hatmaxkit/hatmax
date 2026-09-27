# Test Helper

`testhelper` builds a Postgres database for a Go test. The implementation
note is [testhelper/readme.md](../../../testhelper/readme.md).

`SetupTestDB(t)` returns a `*sql.DB`, a schema name, and a cleanup function.
When `DB_HOST` is set, it uses that server. `DB_PORT` defaults to `5432`,
`DB_USER` and `DB_PASSWORD` default to `postgres`, and `DB_NAME` defaults to
`postgres`. The schema name starts with `test_`, the current Unix time in
nanoseconds, and 8 random characters. Cleanup drops that schema.

When `DB_HOST` is empty, `SetupTestDB` starts a Postgres testcontainer and
returns its database. Cleanup terminates the container.

`SetupTestDBWithConfig(t)` returns a `*config.Config` pointed at the same
kind of database, plus a cleanup function.

`TestLogger` returns `log.NewTestLogger("error")`.
