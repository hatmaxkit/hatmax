# render

Template FuncMap with UI components.

## Usage

```go
tmpl := template.New("").Funcs(render.FuncMap())

// Or merge with your own
funcs := render.MergeFuncMaps(render.FuncMap(), myFuncMap)
```

In templates:

```html
{{chip "Active"}}
{{badge "New" "success"}}
{{statusBadge "published"}}
{{formatPrice 1999 "USD"}}
{{stat "Users" 1234}}
```

Components: chips, pills, badges, prices, stats. See `ui/` for details.
