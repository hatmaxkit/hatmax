# Serve a Page

Serve one HTML page and replace one element on that page with an HTMX
response.

This chapter keeps the Postgres component from Add Postgres. It does not
accept a form.

The contracts are in
[HTTP](../../reference/http/index.md),
[HTMX](../../reference/htmx/index.md), and
[Rendering](../../reference/rendering/index.md).
This page does not use a template function. The rendering reference is where
those functions are defined.

## Add the templates

Add `assets/templates/home/page.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Home</title>
  <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
  <div id="status"><p>Waiting</p></div>
  <button type="button" hx-get="/status" hx-target="#status" hx-swap="innerHTML">
    Update
  </button>
</body>
</html>
```

Add `assets/templates/home/status.html`:

```html
<p>Updated</p>
```

`web.NewTemplateManager` parses every `.html` file when it starts. `Render`
and `RenderPartial` execute
`assets/templates/<namespace>/<template>.html`.

## Register the page

Keep the database setup from Add Postgres. Add a page component and start
the template manager before the routes are registered:

```go
package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/app"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/db"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/web"
)

//go:embed assets
var assetsFS embed.FS

type pages struct {
	templates *web.TemplateManager
}

func (p *pages) RegisterRoutes(r chi.Router) {
	r.Get("/", p.home)
	r.Get("/status", p.status)
}

func (p *pages) home(w http.ResponseWriter, r *http.Request) {
	p.templates.Render(w, "home", "page", nil)
}

func (p *pages) status(w http.ResponseWriter, r *http.Request) {
	p.templates.RenderPartial(w, "home", "status", nil)
}

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
	templates := web.NewTemplateManager(assetsFS, logger)
	site := &pages{templates: templates}

	ctx := context.Background()
	starts, stops, registrars := app.Setup(ctx, router, database, templates, site)

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

`app.Start` runs `database.Start`, then `templates.Start`, and only then
`RegisterRoutes`. A template parse error stops the database that already
started and the process exits before it listens.

## Check the result

Run the process against the Postgres from Add Postgres.

```sh
curl -sS localhost:8080/ | grep -F 'hx-get="/status"'
curl -sS localhost:8080/status
```

The first command finds the button. The second prints `<p>Updated</p>`.

Open `http://localhost:8080/` in a browser. The page shows `Waiting`. Choose
Update. The element `#status` shows `Updated`. The process is still running.
Stop it with Ctrl+C.
