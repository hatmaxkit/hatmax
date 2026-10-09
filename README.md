<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# HatMax

<p align="center">
  <img src="assets/img/docs/brand/hero.png" width="800">
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

With Go 1.27.1, clone the source and bind the application to that revision.
Hatmax v0.6.0 provides the caller-owned server API shown below:

```sh
git clone --branch v0.6.0 https://forge.adrianpk.com/hatmax/hatmax.git hatmax
mkdir myapp
cd myapp
go mod init example.com/myapp
go mod edit -replace=hatmax.adrianpk.com=../hatmax
go get hatmax.adrianpk.com
git -C ../hatmax rev-parse HEAD
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
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

func main() {
	cfg, err := config.Load("config.yaml", "MYAPP_", os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err = cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := log.NewLogger(cfg)
	router := app.NewRouter(logger, app.WithPing())
	ctx := context.Background()
	starts, stops, registrars := app.Setup(ctx, router)

	if err = app.Start(ctx, logger, starts, stops, registrars, router); err != nil {
		logger.Errorf("cannot start: %v", err)
		os.Exit(1)
	}

	server := &http.Server{Addr: cfg.Server.Port, Handler: router}
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- app.Serve(server)
	}()

	stopContext, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-stopContext.Done():
		app.Shutdown(server, logger, stops)
	case err = <-serveErrors:
		app.Shutdown(server, logger, stops)
		if err != nil {
			logger.Errorf("cannot serve: %v", err)
			os.Exit(1)
		}
	}
}
```

Run `go mod tidy` after creating `main.go`, then `go run .`. Request
`http://localhost:8080/ping`; the response is `{"status":"ok"}`. Ctrl+C
stops the server and components. The [bootstrap procedure](docs/how-to/bootstrap-application/README.md)
gives the complete setup and verification. The
[User Guide](docs/tutorials/user-guide/README.md) explains application assembly
and the supporting package boundaries.

## Build with Hatmax

Build `hm` from the source checkout. Add Go's binary directory to `PATH`;
authenticate a compatible Codex CLI and resident App Server before requesting
generation. Run `hm` from a parent
directory to create an application, or from a compatible existing project to
evolve it:

```sh
cd ../hatmax
go install ./cmd/hm
cd ..
hm
```

Describe the application or feature conversationally. Hatmax uses Codex only
for bounded interpretation, presents the complete typed plan, and waits for
explicit approval before applying deterministic Book-owned changes. For a
single headless request, use:

```sh
hm generate "Create an invoice feature with a required number."
```

See
[Assisted Generation](docs/tutorials/user-guide/assisted-generation.md) for the
guided workflow and [Generator](docs/reference/generator/README.md) for the
exact contract and prerequisites.

After creating an application, continue its conversation to add features,
fields, validation and explicitly requested documentation. Each change requires
its own plan approval and project validation.

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
  only after startup succeeds. Each completed startup step carries its own optional rollback; a failing
  start owns its partial cleanup.
- **Two configuration layers**: Static process configuration and schema-checked
  runtime settings have separate lifetimes.
- **Postgres-first infrastructure**: Pubsub, scheduler, and sessions can share
  Postgres while remaining behind explicit interfaces.

## Docs

- [User Guide](docs/tutorials/user-guide/README.md)
- [Documentation index](docs/README.md)

## Repos

The official repo is available at [https://forge.adrianpk.com/hatmax/hatmax](https://forge.adrianpk.com/hatmax/hatmax).

Public mirrors are available at:

- [https://codeberg.org/hatmax/hatmax](https://codeberg.org/hatmax/hatmax)
- [https://github.com/hatmaxkit/hatmax](https://github.com/hatmaxkit/hatmax)

During the Forge migration, the `hatmax.adrianpk.com` vanity import page is served from Codeberg Pages and points to the official Forge repo.

## License

Copyright 2026 Adrian PK. Licensed under the Apache License, Version 2.0
(`Apache-2.0`). See [LICENSE](LICENSE).

Adrian PK, as the sole author and copyright holder, makes Hatmax's libraries,
generator, examples, and templates available under Apache-2.0 for use in
open-source and proprietary applications. This licensing decision is effective
2026-10-02. Third-party dependencies remain subject to their own licenses.
