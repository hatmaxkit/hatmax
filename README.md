# HatMax

<p align="center">
  <img src="docs/img/hero.png" width="800">
</p>

[![Go Reference](https://pkg.go.dev/badge/hatmax.adrianpk.com.svg)](https://pkg.go.dev/hatmax.adrianpk.com)
[![CI](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/ci.json)](https://codeberg.org/hatmax/hatmax)
[![coverage](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/coverage.json)](https://codeberg.org/hatmax/hatmax)

**A composable Go toolkit for server-rendered web applications with explicit
wiring, clear configuration boundaries, and Postgres-first infrastructure.**

## Overview

Hatmax provides practical, composable packages for building web applications
in Go.

It includes:

- Explicit constructors and dependency wiring across packages.
- Practical building blocks that work together out of the box.
- Composable roles and interfaces instead of hidden global state.
- Postgres-first primitives for authentication, scheduling, and pubsub.
- Interactive generation that turns a product request into an inspectable,
  canonical Hatmax plan before changing the project.

## Quick Start

Create a module and add Hatmax:

```sh
mkdir myapp
cd myapp
go mod init example.com/myapp
go get hatmax.adrianpk.com
```

Create `config.yaml`:

```yaml
log:
  level: info
server:
  port: ":8080"
```

Create `main.go`:

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
	cfg, err := config.Load("config.yaml", "MYAPP_", os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load config: %v\n", err)
		os.Exit(1)
	}

	if err = cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot validate config: %v\n", err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)
	router := app.NewRouter(logger, app.WithPing())
	ctx := context.Background()
	starts, stops, registrars := app.Setup(ctx, router)

	err = app.Start(ctx, logger, starts, stops, registrars, router)
	if err != nil {
		logger.Errorf("cannot start app: %v", err)
		os.Exit(1)
	}

	if err = app.Serve(router, cfg.Server.Port); err != nil {
		logger.Errorf("cannot serve: %v", err)
		os.Exit(1)
	}
}
```

Run `go run .`, then request `http://localhost:8080/ping`. Continue with the
[User Guide](docs/tutorials/user-guide/index.md) to add Postgres, templates,
forms, authentication, records, background work, and runtime settings.

## Generate a Feature

Install the Hatmax command and run it from the root of an existing Hatmax
project:

```sh
go install hatmax.adrianpk.com/cmd/hatmax@v0.5.0
cd myapp
hatmax generate "Create a property feature with a required name."
```

Hatmax asks Codex to interpret the request, presents the complete typed plan,
and waits for approval before applying deterministic Book-owned changes. See
[Generate a Feature](docs/tutorials/user-guide/generation.md) for the guided
workflow and [Generator](docs/reference/generator/index.md) for the exact
contract and prerequisites.

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

- **Ordered lifecycle**: Components start in declared order and routes register
  only after startup succeeds. Startup rollback requires aligned start and stop
  capabilities.
- **Two configuration layers**: Static process configuration and schema-checked
  runtime settings have separate lifetimes.
- **Postgres-first infrastructure**: Pubsub, scheduler, and sessions can share
  Postgres while remaining behind explicit interfaces.

## Docs

- [User Guide](docs/tutorials/user-guide/index.md)
- [Documentation index](docs/index.md)

## Repos

The official repo is available at [https://forge.adrianpk.com/hatmax/hatmax](https://forge.adrianpk.com/hatmax/hatmax).

Public mirrors are available at:

- [https://codeberg.org/hatmax/hatmax](https://codeberg.org/hatmax/hatmax)
- [https://github.com/hatmaxkit/hatmax](https://github.com/hatmaxkit/hatmax)

During the Forge migration, the `hatmax.adrianpk.com` vanity import page is served from Codeberg Pages and points to the official Forge repo.

## License

MIT
