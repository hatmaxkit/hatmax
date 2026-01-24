# render

Template FuncMap with UI components and utilities.

## Usage

```go
tmpl := template.New("").Funcs(render.FuncMap())

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
```

Components: chips, pills, badges, prices, stats. See `ui/` for details.
