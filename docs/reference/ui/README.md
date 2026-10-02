<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# UI

`ui` renders HTML components and serves static assets. The implementation
note is [ui/readme.md](../../../ui/readme.md).

## Kit

`New(cfg, logger, options...)` stores the config and logger. Options are
`WithSettings`, `WithEmbeds`, `WithOverlay`, and `WithCSRFFunc`.

`CSRFToken` returns `""` when no CSRF function is set. Otherwise it returns
that function's result. `Config`, `Logger`, and `Settings` return the stored
values. `Settings` is nil until `WithSettings` is used.

`Assets` builds an `Assets` value from the kit embeds and overlay. A kit
without embeds still returns an `Assets` value whose embedded filesystem is
empty.

## Assets

`NewAssets` uses the prefix `/static`. `WithPrefix` replaces it. `URL` joins
the prefix and the path, adding a leading slash to the path when it has none.

`Open` checks the overlay first. An overlay error, including not found, falls
through to the embedded filesystem. `Read` reads the opened file.

`Handler` strips the prefix. An empty path, a missing file, or a directory
responds `404`. A read error responds `500` with `Internal Server Error`. A
successful file is written with a content type from its extension: `css`,
`js`, `json`, `html`, `htm`, `svg`, `png`, `jpg`, `jpeg`, `gif`, `ico`,
`woff`, `woff2`, and `ttf`. Any other extension is
`application/octet-stream`.

## Variants

| Name | Value |
| --- | --- |
| `VariantDefault` | empty |
| `VariantPrimary` | `primary` |
| `VariantSecondary` | `secondary` |
| `VariantSuccess` | `success` |
| `VariantWarning` | `warning` |
| `VariantDanger` | `danger` |
| `VariantInfo` | `info` |
| `VariantMuted` | `muted` |

## Components

Each component below has `Render() template.HTML`. Builders return the same
value so calls can be chained.

| Constructor | Starts from |
| --- | --- |
| `NewChip`, `NewLabel` | The supplied text. |
| `NewButton`, `Btn` | The supplied text. |
| `BtnSubmit` | A submit button. |
| `BtnDanger` | A danger button. |
| `NewPage` | The supplied title. |
| `NewPageHeader` | The supplied title. |
| `NewContainer` | An empty container. `Fluid` changes its width class. |
| `NewNavGrid`, `NewNav` | Empty navigation. `NewNav` is horizontal until `Vertical`. |
| `NewTable` | No columns and no rows. `EmptyText` is the text used when there are no rows. |
| `NewForm` | Method `post` and CSRF field `_token`. A token is emitted only when `CSRFToken` is set. |
| `NewDeleteButton` | The supplied text and action, variant `danger`. |
| `NewSettingsForm` | Method `post` and submit text `Save`. |
| `NewAlert` | Variant `info`. |
| `AlertInfo`, `AlertSuccess`, `AlertWarning`, `AlertDanger` | That variant. |
| `AlertError` | Variant `danger`. |
| `NewFlash`, `NewToast` | Variant `info`. |
| `NewLink`, `A` | The supplied text and href. |
| `ABlank` | `target="_blank"` and `rel="noopener noreferrer"`. |
| `ABoosted` | HTMX boost enabled. |

`Text` HTML-escapes its string. `HTML` and `CellWithClass` keep the supplied
`template.HTML`. `Col` builds a column from a key and a label. `NewRow` builds
a row from cells.

`StatusBadge` uses the status registry. Unknown statuses render the raw
status with no variant. The built-in statuses are `active`, `draft`,
`inactive`, `expired`, and `featured`. `RegisterStatus` overwrites one entry.
`StatusBadgeWithIcon` prefixes the registered icon and a space when the icon
is non-empty.

`NewSettingsForm` shows `Schema.Default` when the supplied value is empty. A
non-empty error marks the field. `bool` renders a checkbox. `enum` renders a
select from `Options` and `Labels`. `Secret` renders a password input. Other
types render a text input.

`Form.Open` filters and escapes action, and HTML-escapes method, class, id,
enctype, the CSRF field name, and the token. `Render` is `Open` followed by
`Close`. `HX` on a form, link, button, or delete button returns that component's
HTMX builder. `Done` returns the component.

### URL attributes

`Link`, `Nav`, `NavGrid`, and `PageHeader` breadcrumbs filter native `href`
values. `Form`, `DeleteButton`, and `SettingsForm` apply the same policy to
native `action` values. Filtering occurs when rendering, before returning
trusted `template.HTML`.

These values are passed as plain strings through `html/template`'s quoted URL
context. It permits relative URLs and case-insensitive `http`, `https`, and
`mailto` schemes. Unsupported schemes, including `javascript`, `data`, `file`,
and `tel`, become the inert `#ZgotmplZ` fragment. Scheme detection is
conservative: a colon before the first slash is treated as a candidate scheme,
even in fragment or query text. For example, `#section:1` is rejected.

Allowed URLs receive standard template URL normalization and HTML escaping.
Spaces, quotes, controls, and non-ASCII bytes are percent-encoded; query
ampersands are HTML-escaped once. Existing valid percent escapes are preserved.
Encoded or entity-looking input is not recursively decoded into a scheme.

Empty form actions remain omitted. Empty breadcrumb URLs still render labels
without links; other links retain an empty `href`. External and network-relative
HTTP destinations remain allowed. This is not an origin allowlist, URL validity
check, or authorization policy. Caller-supplied trusted HTML and custom HTMX
attributes remain application-owned and are not sanitized by this policy.

### Common builder groups

| Components | Builder concerns |
| --- | --- |
| Button | emoji, variant, size, disabled, loading, class, id, type, name, value, HTMX |
| Link | target, external relation, class, id, title, download, boost, HTMX |
| Form | action, method, CSRF token and field, class, id, encoding, validation, HTMX |
| Alert, Flash, Toast | message, variant, classes, and component-specific dismissal behavior |
| Page, PageHeader, Container | page title, breadcrumbs, actions, content, and layout classes |
| NavGrid, Nav | items, links, active state, and layout |
| Table | columns, rows, empty text, classes, and row rendering |

`Renderable` requires `Render() template.HTML`. `CSRFFunc` is
`func(context.Context) string`. `Option` configures a `Kit`; the exported
options are `WithSettings`, `WithEmbeds`, `WithOverlay`, and `WithCSRFFunc`.

`ButtonSize` values are `sm`, `md`, and `lg`. `StatusConfig` defines the label,
variant, and icon stored by `RegisterStatus`.

## Template functions

`FuncMap` and `Kit.FuncMap` start from `render.FuncMapWithHTMX` and add:

`formatPrice`, `formatNumber`, `chip`, `label`, `statusBadge`,
`statusBadgeWithIcon`, `btn`, `btnSubmit`, `btnDanger`, `button`, `page`,
`pageHeader`, `container`, `navGrid`, `nav`, `table`, `col`, `row`, `text`,
`html`, `settingsForm`, `form`, `deleteButton`, `alert`, `alertInfo`,
`alertSuccess`, `alertWarning`, `alertDanger`, `alertError`, `flash`,
`toast`, `link`, `linkBlank`, and `linkBoosted`.

`MergeFuncMaps` uses `render.MergeFuncMaps`. `HX` returns `htmx.HX()`.

## Emoji

`Emoji` is a string. `String` returns that string. The preset constants are
`EmojiCheck`, `EmojiCross`, `EmojiWarning`, `EmojiInfo`, `EmojiStar`,
`EmojiHeart`, `EmojiPin`, `EmojiFolder`, `EmojiEdit`, `EmojiTrash`,
`EmojiSearch`, `EmojiSettings`, `EmojiUser`, `EmojiLock`, `EmojiUnlock`,
`EmojiMail`, `EmojiPhone`, `EmojiCalendar`, `EmojiClock`, `EmojiHome`,
`EmojiLink`, `EmojiPlus`, `EmojiMinus`, `EmojiRefresh`, `EmojiDownload`,
`EmojiUpload`, `EmojiSave`, `EmojiFire`, and `EmojiSparkle`.
