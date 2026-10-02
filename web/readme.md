<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

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
form, err := web.ParseForm(r)
if err != nil { ... }
name := form.String("name")

// Present structured validation errors without exposing internal errors
formErrors := web.FormErrorsFrom(err, "Review the highlighted fields.")
formErrors.Add("email", "A contact with this email already exists.")
message := formErrors.First("email")
```

`FormErrors.General` contains the form-level summary. `FormErrors.Fields`
maps stable form field names to one or more user-facing messages. Applications
remain responsible for explicitly mapping business errors to safe messages.

See `htmx/` for htmx-specific helpers.
