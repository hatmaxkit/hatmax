# render

Base template FuncMap with string, math, and i18n utilities.

For UI components, use `ui.FuncMap()` which extends this.

## Usage

```go
// Base functions only
tmpl := template.New("").Funcs(render.FuncMap())

// With HTMX helpers
tmpl := template.New("").Funcs(render.FuncMapWithHTMX())

// Full UI kit (recommended)
tmpl := template.New("").Funcs(ui.FuncMap())

// Merge with your own
funcs := render.MergeFuncMaps(render.FuncMap(), myFuncMap)

// With i18n support
translator := i18n.New()
translator.LoadFromFS(localesFS, "locales")
funcs := render.MergeFuncMaps(render.FuncMap(), render.I18nFuncMap(translator))
```

## Available Functions

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
```

## UI Components

For UI components (chips, buttons, alerts, forms, etc.), use `ui.FuncMap()`:

```go
tmpl := template.New("").Funcs(ui.FuncMap())
```

See `ui/readme.md` for component documentation.
