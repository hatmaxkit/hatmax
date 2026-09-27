# Add Postgres

Add a Postgres component. When that component cannot reach the database, the
process exits before it listens. When it can, the process stays up and
`GET /ping` still answers.

This chapter does not run migrations or serve a page.

The contracts are in
[Database](../../reference/database/index.md) and
[Application Lifecycle](../../reference/application-lifecycle/index.md).

## Point the configuration at Postgres

Keep the file from Getting Started and set the database fields to a server
you can reach:

```yaml
log:
  level: info
server:
  port: ":8080"
database:
  host: localhost
  port: 5432
  user: dev
  password: dev
  database: dev
  sslmode: disable
```

`db.New` reads those fields through `cfg.Database.ConnectionString`. It does
not open the connection.

Add `assets/keep.txt` so the embedded filesystem has a file. The database
component stores that filesystem and does not read it in this chapter.

## Start the database component

Replace `main.go` with:

```go
package main

import (
	"context"
	"embed"
	"fmt"
	"os"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
)

//go:embed assets
var assetsFS embed.FS

func main() {
	cfg, err := config.Load("config.yaml", "APP_", os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load config: %v\n", err)
		os.Exit(1)
	}

	err = cfg.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot validate config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)
	router := app.NewRouter(logger, app.WithPing())
	database := db.New(assetsFS, db.Postgres, cfg, logger)

	ctx := context.Background()
	starts, stops, registrars := app.Setup(ctx, router, database)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("cannot start: %v", err)
		os.Exit(1)
	}

	logger.Info("listening")

	err = app.Serve(router, cfg.Server.Port)
	if err != nil {
		logger.Errorf("server stopped: %v", err)
		os.Exit(1)
	}
}
```

`app.Setup` collects `database.Start` because `*db.Database` implements
`Start`. `app.Start` calls it before `app.Serve`. A failed ping closes the
connection and returns `cannot ping database`. This program has no earlier
component to stop, so the process exits and nothing listens.

## Check a failed start

Set `database.port` to a port where Postgres is not listening, then run the
process.

The log contains `cannot ping database`. The process exits. This command
fails because nothing is listening:

```sh
curl -sS localhost:8080/ping
```

## Check a successful start

Set `database.port` back to the port of a Postgres you can reach, with the
user, password, and database name that server accepts. Run the process again.

The log contains `Database connection established`, then `listening`. In
another terminal:

```sh
curl -sS localhost:8080/ping
```

The command prints `{"status":"ok"}`. The process is still running. Stop it
with Ctrl+C.
