<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Test with Postgres

Use `testhelper` when a package test needs a real Postgres connection.

## Create an isolated schema

```go
func TestStore(t *testing.T) {
	database, schema, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)

	_ = database
	_ = schema
}
```

When `DB_HOST` is set, the helper creates an isolated schema on that server.
Otherwise it starts a Postgres testcontainer. Cleanup drops the schema and, for
the container path, terminates the container.

Use `SetupTestDBWithConfig` when the code under test needs a Hatmax
`*config.Config`. Use `TestLogger` for a logger that emits only errors.

## Run the test

```sh
go test ./path/to/package
```

To reuse an existing server:

```sh
DB_HOST=localhost DB_USER=postgres DB_PASSWORD=postgres go test ./path/to/package
```

## Verify cleanup

After the test, the generated schema name beginning with `test_` must no longer
exist. See [Test Helper Reference](../../reference/testhelper/README.md) for all
environment defaults.
