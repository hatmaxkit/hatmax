<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Presentation Primitives

Pages and forms establish the application's HTML contract. Hatmax presentation
packages remove repeated mechanics inside that contract without taking
ownership of routes, product language, or feature behavior.

| Package | Role in a page |
| --- | --- |
| `ui` | Render reusable components and serve UI assets. |
| `modal` | Carry dialog configuration as presentation data. |
| `format` | Format numbers and prices for display. |
| `pagination` | Calculate page bounds and result metadata. |
| `i18n` | Load translations and resolve locale keys. |

These primitives support application-owned view models and templates. They
are not a separate frontend framework.

The Go blocks below are contextual fragments, not standalone programs. The
[guide companion](../../../examples/guide/main.go) is the complete runnable
composition for lifecycle, routes, pages and forms. Invoice-specific fields,
handlers and services are application-owned illustrative names; they require
that feature's implementation. Use the
[bootstrap procedure](../../how-to/bootstrap-application/README.md) for a
complete signal-aware process with a health endpoint.

## Compose One Template Function Map

Install template functions when constructing the template manager:

```go
translator := i18n.New()
funcs := ui.MergeFuncMaps(
	ui.FuncMap(),
	render.I18nFuncMap(translator),
)

templates := web.NewTemplateManager(
	assetsFS,
	logger,
	web.WithFuncMap(funcs),
)
```

The shown translator has no loaded locales yet. Load embedded locale files
before serving requests; otherwise `t` returns the untranslated key. The
[Internationalization reference](../../reference/i18n/README.md) states loading
and fallback behavior.

Later function maps replace duplicate names. Build the map deliberately in
the composition root so every parsed template sees one predictable set. A
`ui.Kit` can provide the UI function map when the application also needs
embedded assets, an overlay, settings, or a CSRF token function.

The base rendering map supplies simple string, sequence, arithmetic, and HTMX
helpers. The UI map adds components such as buttons, forms, links, alerts,
navigation, tables, chips, and page containers. Prefer one shared component
for repeated interaction and accessibility behavior, while keeping
feature-specific markup with its feature templates.

## Use UI Components as Trusted Building Blocks

UI component builders return renderable values and support focused options:

```html
{{with btnSubmit "Save"}}
  {{with .Primary}}{{.Render}}{{end}}
{{end}}
```

Forms, buttons, links, and delete buttons can also construct HTMX attributes
through their `HX` builder. This keeps attribute names and encoding in the
Hatmax primitive while the application still chooses the URL, target, swap,
and visible language.

Ordinary string values remain escaped by Go templates. `ui.Text` escapes its
input, while `ui.HTML` and component `Render` methods produce trusted HTML.
Use the trusted forms only for markup assembled by the application or Hatmax,
never to pass through untrusted request or stored content.

## Keep Dialog State in the View Model

`modal.Config` describes a dialog identifier, title, size, and close behavior.
It does not render the dialog or manage browser state. Put the configuration
in a view model, then let the application template and JavaScript implement
the presentation contract.

```go
view.ConfirmDelete = modal.DefaultConfig(
	"delete-invoice",
	"Delete invoice",
)
```

This separation lets a feature decide why a dialog appears while the template
decides how its configured presentation is rendered.

## Format at the Presentation Edge

Keep stored and domain values typed. Apply `format.Number`, `format.Integer`,
`format.Price`, or `format.PriceWithDecimals` only while building a view or
rendering a template. The built-in number printer uses English formatting;
currency registration controls the symbol and its position, not a full locale
formatting policy.

If product requirements need locale-aware dates, measurements, or currency
rules beyond these helpers, keep that policy in an application-owned
presentation adapter. Do not turn preformatted strings into domain values.

## Translate with the Request Locale

An `i18n.Translator` loads YAML files from an embedded filesystem and flattens
nested keys. Missing keys fall back to the default locale and then to the key
itself. This makes a missing translation visible without crashing a page.

`middleware.Locale` selects a locale from the cookie or `Accept-Language` and
stores it in the request context. The handler can carry that locale into the
view, or select a locale-bound translation function for rendering. Translation
keys belong to application copy; Hatmax supplies loading, fallback, and
lookup.

Do not use translation output as trusted HTML. Keep translations as text and
let templates escape them.

## Paginate Across the Store and Page Boundary

`pagination.NewParams(page, pageSize)` normalizes browser-provided bounds and
calculates `Offset` and `Limit`. The store uses those values and separately
returns the total count. The handler then builds the page result:

```go
params := pagination.NewParams(page, pageSize)
items, total, err := h.service.List(r.Context(), params.Offset(), params.Limit())
if err != nil {
	// Translate the failure at the request boundary.
}

result := pagination.NewResult(items, total, params)
```

The result exposes total pages, previous and next page numbers, visible index
bounds, and whether more data exists. The package does not read query
parameters or issue database queries; those remain handler and feature
responsibilities.

For HTMX pagination, render the same list or table partial used by the full
page and decide whether the URL should be pushed or replaced. The current page
therefore remains meaningful as both server state and browser navigation.

## Choose the Smallest Primitive

Use these packages when they remove a Hatmax-owned presentation mechanic. Do
not force a feature-specific widget, copy rule, or workflow into a generic
component merely because it appears in HTML. The application should remain
readable in this direction:

```text
handler -> view model -> template
                    \-> Hatmax presentation primitives
```

Exact builders and edge cases are documented in the
[UI](../../reference/ui/README.md), [Modal](../../reference/modal/README.md),
[Format](../../reference/format/README.md),
[Pagination](../../reference/pagination/README.md), and
[Internationalization](../../reference/i18n/README.md) references.

---

[Previous: Forms and Validation](forms-and-validation.md) ·
[User Guide](README.md) ·
[Next: Feature Anatomy](feature-anatomy.md)
