<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Test with Postgres

Use `testhelper` when a package test needs a real Postgres connection.

## Create an isolated schema

Put this function in a `_test.go` file in your package, importing `testing`,
`context` and `hatmax.adrianpk.com/testhelper`:

```go
// TestStore verifies the isolated schema selected by the connection pool.
func TestStore(t *testing.T) {
	database, schema, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)

	var selected string
	if err := database.QueryRowContext(context.Background(), "SELECT current_schema()").Scan(&selected); err != nil {
		t.Fatal(err)
	}
	if selected != schema {
		t.Fatalf("selected %q, want %q", selected, schema)
	}
}
```

When `DB_HOST` is set, the helper creates an isolated schema on that server.
Use a disposable test server and a role permitted to create and drop schemas.
Otherwise it starts a `postgres:16-alpine` testcontainer using `public`; that
path requires a working container runtime. Server cleanup closes the pool and
attempts to drop the schema; container cleanup closes the pool and terminates
the container. Schema-drop errors are ignored, so verify removal.

Use `SetupTestDBWithConfig` when the code under test needs a Hatmax
`*config.Config`; close any connection you open from it before invoking its
cleanup. Use `TestLogger` for a logger that emits only errors.

## Run the test

```sh
go test -count=1 ./...
```

To reuse an owned server, export `DB_HOST`, `DB_USER`, `DB_PASSWORD`, and optional
`DB_PORT`/`DB_NAME` for that server before running from the package's module:

```sh
DB_HOST="$DB_HOST" DB_USER="$DB_USER" DB_PASSWORD="$DB_PASSWORD" go test -count=1 ./...
```

## Verify cleanup

For the server path, after the test, the generated schema name beginning with
`test_` must no longer exist. For the container path, confirm that the owned
container has stopped. See [Test Helper Reference](../../reference/testhelper/README.md)
for all environment defaults.
