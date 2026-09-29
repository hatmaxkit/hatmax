# Bootstrap a Hatmax Application

Use this procedure to create a minimal Hatmax process with a health endpoint.

## Create the module

Hatmax currently declares Go 1.24 in its module. Create a directory and add the
dependency:

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

	if err = app.Serve(router, cfg.Server.Port); err != nil {
		logger.Errorf("cannot serve: %v", err)
		os.Exit(1)
	}
}
```

## Verify the process

Run the application:

```sh
go run .
```

In another terminal:

```sh
curl -fsS http://localhost:8080/ping
```

The response is `{"status":"ok"}`. Stop the application with Ctrl+C.

For component ordering and shutdown behavior, see
[Application Lifecycle](../../reference/application-lifecycle/README.md).
