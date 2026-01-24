# db

PostgreSQL connection with lifecycle management.

## Usage

```go
//go:embed assets
var assetsFS embed.FS

database := db.New(assetsFS, "postgres", cfg, log)

// Implements app.Startable/Stoppable
if err := database.Start(ctx); err != nil {
    log.Fatal(err)
}
defer database.Stop(ctx)

// Use the connection
sqlDB := database.GetDB()
```

Creates schema automatically if `cfg.Database.Schema` is set.

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
