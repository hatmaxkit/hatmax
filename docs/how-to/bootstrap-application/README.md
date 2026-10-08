<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Bootstrap a Hatmax Application

Use this procedure to create a minimal Hatmax process with a health endpoint.

## Create the module

This procedure targets the current Hatmax `dev` checkout with Go 1.27.1.
The published `v0.5.0` module uses the older `Serve(router, port)` contract;
`go get` alone does not select the caller-owned-server API shown here.
Clone the source and bind the application dependency to that checkout:


```sh
git clone --branch dev https://forge.adrianpk.com/hatmax/hatmax.git hatmax
mkdir myapp
cd myapp
go mod init example.com/myapp
go mod edit -replace=hatmax.adrianpk.com=../hatmax
go get hatmax.adrianpk.com
git -C ../hatmax rev-parse HEAD
```

Keep that source revision with the application when reproducing this procedure.
The replacement is local to this application's module; dependency packages use
ordinary published Go modules. Run the following commands from `myapp`.

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

## Verify the process

Resolve the imported dependencies after creating `main.go`, then run the application:

```sh
go mod tidy
go run .
```

In another terminal:

```sh
curl -fsS http://localhost:8080/ping
```

The response is `{"status":"ok"}`. Stop the application with Ctrl+C; it
shuts down the same server before stopping components. `app.Serve` defaults
missing header and idle limits to 5 and 60 seconds without imposing a
whole-response deadline. Set positive values on `server` to customize them.

For component ordering and shutdown behavior, see
[Application Lifecycle](../../reference/application-lifecycle/README.md).
