# web

HTTP utilities for templates, forms, and htmx.

## Template Manager

```go
//go:embed assets
var assetsFS embed.FS

tm := web.NewTemplateManager(assetsFS, log)
tm.Start(ctx)

// Render full page (namespace="auth", template="login")
tm.Render(w, "auth", "login", data)

// Render partial (htmx response)
tm.RenderPartial(w, "tasks", "row", data)
```

### Directory Structure

Templates are loaded from `assets/templates/{namespace}/`:

```
assets/
└── templates/
    ├── auth/
    │   ├── login.html
    │   └── register.html
    ├── tasks/
    │   ├── list.html
    │   └── row.html
    └── shared/
        └── layout.html
```

Only `.html` files are processed. The namespace parameter in `Render()` maps to the subdirectory name.

## Utilities

```go
// Redirect (htmx-aware)
web.RedirectOrHXRedirect(w, r, "/dashboard")

// Parse form
if err := web.ParseForm(r); err != nil { ... }
name := web.FormValue(r, "name")
```

See `htmx/` for htmx-specific helpers.
