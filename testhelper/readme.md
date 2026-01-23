# testhelper

Test utilities for database and logging.

## Usage

```go
func TestSomething(t *testing.T) {
    // Get isolated database (testcontainer locally, unique schema in CI)
    db, schema, cleanup := testhelper.SetupTestDB(t)
    defer cleanup()

    // Or get config for services that need it
    cfg, cleanup := testhelper.SetupTestDBWithConfig(t)
    defer cleanup()

    // Quiet logger for tests
    log := testhelper.TestLogger()
}
```

Uses testcontainers locally, shared Postgres with unique schemas in CI (set `DB_HOST`).
