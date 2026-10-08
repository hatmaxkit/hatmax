<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# render

Base template FuncMap with string, math, and i18n utilities.

For UI components, use `ui.FuncMap()` which extends this.

## Usage

These construction fragments belong in the composition root described in
[Presentation Primitives](../docs/tutorials/user-guide/presentation-primitives.md).
The [guide companion](../examples/guide/main.go) shows a complete template-manager
lifecycle. `localesFS` is an embedded filesystem with YAML under `locales`.

```go
// Base functions only
baseTemplate := template.New("").Funcs(render.FuncMap())

// With HTMX helpers
htmxTemplate := template.New("").Funcs(render.FuncMapWithHTMX())

// Full UI kit (recommended)
uiTemplate := template.New("").Funcs(ui.FuncMap())

// Merge with your own
customFuncs := render.MergeFuncMaps(render.FuncMap(), myFuncMap)

// With i18n support
translator := i18n.New()
if err := translator.LoadFromFS(localesFS, "locales"); err != nil {
    return err
}
localizedFuncs := render.MergeFuncMaps(render.FuncMap(), render.I18nFuncMap(translator))
```

The base `t` function returns its key unchanged. Load translations and merge
`render.I18nFuncMap(translator)` after the base map to perform lookups. Later
maps replace existing functions; `ui.FuncMap()` also starts with the base `t`.

## Available Functions

```html
<!-- String -->
{{upper "hello"}}  <!-- HELLO -->
{{lower "HELLO"}}  <!-- hello -->

<!-- i18n -->
{{t .Locale "common.search"}}  <!-- common.search in the base map -->

<!-- Math -->
{{add 1 2}}        <!-- 3 -->
{{sub 5 2}}        <!-- 3 -->
{{range seq 1 5}}{{.}} {{end}}  <!-- 1 2 3 4 5 -->
```

## UI Components

For UI components (chips, buttons, alerts, forms, etc.), use `ui.FuncMap()`:

```go
tmpl := template.New("").Funcs(ui.FuncMap())
```

See `ui/readme.md` for component documentation.
