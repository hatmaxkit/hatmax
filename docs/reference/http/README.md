<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# HTTP

`web` parses forms, presents validation feedback, renders templates from an
embedded filesystem, and redirects HTMX and ordinary requests. The
implementation note is [web/readme.md](../../../web/readme.md).

## Forms

`ParseForm` calls `Request.ParseForm` and returns the error unchanged.
`FormValues` reads the parsed form:

| Method | Result |
| --- | --- |
| `String` | `FormValue`. An absent field is `""`. |
| `StringOr` | `FormValue`, or the supplied default when that value is empty. |
| `Bool` | True only when the value is `true` or `on`. |
| `UUID` | `uuid.Parse` of `FormValue`. An invalid value returns the parse error. |

`ParseIDParam` parses the chi URL parameter with `uuid.Parse`.

## Form errors

`FormErrors` has `General` and `Fields`. `Fields` maps a field name to zero or
more messages.

`NewFormErrors` sets `General` and an empty field map.

`Add` ignores an empty message. An empty field replaces `General`. Any other
field appends the message.

`First` returns the first message for a field, or `""`. `For` returns that
field's slice. `Has` reports whether the slice is non-empty. `Any` reports
whether `General` is non-empty or any field has a message.

`FormErrorsFrom(err, general)` always sets `General` to the supplied string.
It copies `Field` and `Message` from a `validation.ValidationErrors` value or
pointer found with `errors.As`. An error that is not that type adds no field
messages. The source error text is not copied into `General`.

## Templates

`NewTemplateManager(fs, logger, options...)` stores the embedded filesystem
and logger. `WithFuncMap` installs a `template.FuncMap` used while parsing.

`Start` walks the filesystem. It parses each `.html` file. The template name
is the path from the first `assets` segment, or the whole path when `assets`
is absent. A read or parse error is logged and returned. Success stores the
parsed set and logs `templates loaded successfully`.

`Stop` returns nil.

`Render(w, namespace, template, data)` executes
`assets/templates/<namespace>/<template>.html`. It sets
`Content-Type: text/html; charset=utf-8` before execution. A render error is
logged and written as `500` with `Template rendering error`.

`RenderPartial` executes the same path and does not set `Content-Type`. A
render error is logged and written as `500` with `Partial rendering error`.

`RenderTemplate` and `RenderPartial` in this package execute a caller-supplied
`*template.Template` by name. `RenderTemplate` sets the HTML content type.
Both log the template name on failure and write `500`.

## Redirect

`RedirectOrHXRedirect` writes `HX-Redirect` and status `200` when
`HX-Request` is `true`. Otherwise it sends `303` to the URL.
