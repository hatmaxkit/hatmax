# render

Template FuncMap with UI components and utilities.

## Usage

```go
tmpl := template.New("").Funcs(render.FuncMap())

// With HTMX helpers
tmpl := template.New("").Funcs(render.FuncMapWithHTMX())

// Or merge with your own
funcs := render.MergeFuncMaps(render.FuncMap(), myFuncMap)

// With i18n support
translator := i18n.New()
translator.LoadFromFS(localesFS, "locales")
funcs := render.MergeFuncMaps(render.FuncMap(), render.I18nFuncMap(translator))
```

In templates:

```html
<!-- String -->
{{upper "hello"}}  <!-- HELLO -->
{{lower "HELLO"}}  <!-- hello -->

<!-- i18n -->
{{t .Locale "common.search"}}  <!-- Search / Buscar -->

<!-- Math -->
{{add 1 2}}        <!-- 3 -->
{{sub 5 2}}        <!-- 3 -->
{{range seq 1 5}}  <!-- 1, 2, 3, 4, 5 -->

<!-- UI Components -->
{{chip "Active"}}
{{badge "New" "success"}}
{{statusBadge "published"}}
{{formatPrice 1999 "USD"}}
{{stat "Users" 1234}}

<!-- Buttons -->
{{btn "Save"}}
{{btnSubmit "Submit"}}
{{btnDanger "Delete"}}

<!-- Links -->
{{link "Home" "/"}}
{{linkBlank "Docs" "https://docs.example.com"}}
{{linkBoosted "Page" "/page"}}

<!-- Forms -->
{{(text "username").Placeholder "Enter username"}}
{{(email "email").Required}}
{{(field "Email" "email").Error .Errors.email}}

<!-- Alerts -->
{{alertSuccess "Saved!"}}
{{alertError "Something went wrong"}}
{{(flash "Done!").AutoDismiss 5}}
```

Components: chips, pills, badges, prices, stats, buttons, links, forms, alerts. See `ui/` for details.
