<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# testhelper

Test utilities for database and logging. This complete test verifies the
selected schema. Set `DB_HOST` to use an owned PostgreSQL server, or provide a
working container runtime for the container path:

## Usage

```go
package example

import (
	"context"
	"testing"

	"hatmax.adrianpk.com/testhelper"
)

// TestDatabase verifies the schema selected by the helper's connection pool.
func TestDatabase(t *testing.T) {
	database, schema, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)

	var selected string
	if err := database.QueryRowContext(context.Background(), "SELECT current_schema()").Scan(&selected); err != nil {
		t.Fatal(err)
	}
	if selected != schema {
		t.Fatalf("selected %q, want %q", selected, schema)
	}
	testhelper.TestLogger().Info("database ready")
}
```

Selection depends on `DB_HOST`, not whether execution is local or in CI. The
server path creates a unique schema; cleanup closes the pool and attempts to
drop it. The container path starts `postgres:16-alpine`, uses `public`, and
cleanup closes the pool and terminates the owned container. Always register
cleanup and confirm removal when isolation is an acceptance requirement:
schema-drop errors are ignored by the helper.

`SetupTestDBWithConfig` instead returns connection configuration and cleanup;
the caller owns any connection opened from it. Close that connection before
schema or container cleanup. `TestLogger` emits only errors. See the
[Test Helper Reference](../docs/reference/testhelper/README.md) for defaults and
privileges.
