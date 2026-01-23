# web

HTTP utilities for templates, forms, and htmx.

## Usage

```go
// Render full page
web.RenderTemplate(w, tmpl, "page.html", data, log)

// Render partial (htmx response)
web.RenderPartial(w, tmpl, "row.html", data, log)

// Redirect (htmx-aware)
web.RedirectOrHXRedirect(w, r, "/dashboard")

// Parse form
if err := web.ParseForm(r); err != nil { ... }
name := web.FormValue(r, "name")
```

See `htmx/` for htmx-specific helpers.
