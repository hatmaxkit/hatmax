# HatMax

<p align="center">
  <img src="docs/img/hero.png" width="800">
</p>

[![Go Reference](https://pkg.go.dev/badge/hatmax.adrianpk.com.svg)](https://pkg.go.dev/hatmax.adrianpk.com)
[![CI](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/ci.json)](https://codeberg.org/hatmax/hatmax)
[![coverage](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/coverage.json)](https://codeberg.org/hatmax/hatmax)

**A composable Go toolkit for building web applications with consistent wiring, clear configuration boundaries, and predictable package integration.**

## Overview

HatMax provides practical, composable packages for building web applications in Go.

It includes:

- Consistent constructors and dependency wiring across packages.
- Practical building blocks that work together out of the box.
- Composable roles and interfaces instead of hidden global state.
- Postgres-first primitives for auth, scheduling, pubsub, and app lifecycle.

## Quick Start

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
	"hatmax.adrianpk.com/mailer"
	"hatmax.adrianpk.com/pubsub/postgres"
)

//go:embed assets/*
var assetsFS embed.FS

func main() {
	cfg, err := config.Load("config.yml", "APP_", os.Args) // your env prefix
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	logger := log.NewLogger(cfg)
	router := app.NewRouter(logger, app.WithPing(), app.WithDebugRoutes())

	// Infrastructure
	database := db.New(assetsFS, "postgres", cfg, logger)
	events := postgres.New(database, cfg, logger)
	mail := mailer.New(cfg, logger)

	// Application layer
	service := featname.NewService(database, events, mail, cfg, logger)
	handler := featname.NewHandler(service, cfg, logger)

	deps := []any{
		database,
		events,
		mail,
		service,
		handler,
	}

	starts, stops, registrars := app.Setup(ctx, router, deps...)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("cannot start app: %v", err)
		os.Exit(1)
	}
}
```

## Package Map

| Package               | Role                                                                   |
| --------------------- | ---------------------------------------------------------------------- |
| `app`                 | Lifecycle orchestration and component startup/shutdown                 |
| `config`              | Static configuration loading and structure                             |
| `settings`            | Dynamic runtime settings and attributes                                |
| `db`                  | Postgres connection, migration helpers, DB wiring                      |
| `auth`                | Authentication, sessions, auth middleware primitives                   |
| `mailer`              | Pluggable mail delivery providers (SES, SendGrid, Mailgun, SMTP, Noop) |
| `scheduler`           | Background job scheduling and execution                                |
| `pubsub`              | Event publication/subscription (including Postgres implementation)     |
| `web` / `htmx` / `ui` | HTTP, template rendering, htmx helpers, UI primitives                  |

## Interfaces

Small interfaces make components swappable:

| Component | Interface | Implementations |
|-----------|-----------|-----------------|
| Mail | `Mailer` | SMTP, SES, SendGrid, Mailgun |
| Events | `Publisher`, `Subscriber` | Postgres |
| Jobs | `JobStore` | Postgres |

## Key Patterns

- **Transactional startup**: If component N fails to start, components 0→N-1 stop in reverse order automatically.
- **Two config layers**: Static config at startup, dynamic settings with schema validation at runtime. Use what you need.
- **Just Use Postgres**: PubSub, scheduler, and sessions run on Postgres. Add Redis or RabbitMQ if you need them, but you probably don't.

## Docs

- [Features](docs/features.md)
- [Gallery](docs/gallery.md)

## License

MIT
