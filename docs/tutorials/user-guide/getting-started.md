# Getting Started

Load configuration, create a logger and a router, and reach `app.Start`.
The process stays up, and `GET /ping` answers.

This chapter does not open Postgres, serve a page, or sign anyone in.

The contracts are in
[Application Lifecycle](../../reference/application-lifecycle/index.md),
[Configuration](../../reference/configuration/index.md), and
[Logging](../../reference/logging/index.md).

## Create the process

In a module that depends on `hatmax.adrianpk.com`, add `config.yaml`:

```yaml
log:
  level: info
server:
  port: ":8080"
```

`config.Load` requires that file. Fields you omit keep the defaults from
`config.New`, including the database fields that `Validate` checks. This
chapter does not connect to that database.

Add `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

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

	ctx := context.Background()
	starts, stops, registrars := app.Setup(ctx, router)

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

`app.Setup` receives no components, so `app.Start` only registers routes.
`app.WithPing` registers `GET /ping`. `app.Serve` blocks until the process
stops.

## Check the result

Run the process from the directory that contains `config.yaml`.

In another terminal:

```sh
curl -sS localhost:8080/ping
```

The command prints `{"status":"ok"}`. The process is still running. Stop it
with Ctrl+C.
