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

`Form.Open` escapes action, method, class, id, enctype, the CSRF field name,
and the token. `Render` is `Open` followed by `Close`. `HX` on a form, link,
button, or delete button returns that component's HTMX builder. `Done` returns
the component.

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
