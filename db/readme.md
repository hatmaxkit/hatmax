# db

PostgreSQL connection with lifecycle management.

## Usage

```go
//go:embed migrations/*.sql
var migrations embed.FS

database := db.New(migrations, "postgres", cfg, log)

// Implements app.Startable/Stoppable
if err := database.Start(ctx); err != nil {
    log.Fatal(err)
}
defer database.Stop(ctx)

// Use the connection
sqlDB := database.GetDB()
```

Creates schema automatically if `cfg.Database.Schema` is set.

For migrations, see `migrate.go`.
