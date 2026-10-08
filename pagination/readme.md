<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# pagination

Generic pagination for lists.

## Usage

These are handler and template fragments in the
[page/view composition](../docs/tutorials/user-guide/pages-and-partials.md).
`page` and `pageSize` are parsed integers; `items` and `totalCount` come from the
application's store. The complete [guide companion](../examples/guide/main.go)
shows how a handler builds a view before rendering it.

```go
// Parse from request
params := pagination.NewParams(page, pageSize)

// Pass params.Limit() and params.Offset() to the application-owned query.

// Build result
result := pagination.NewResult(items, totalCount, params)

```

```html
{{range .Items}} ... {{end}}
{{if .HasMore}} <a href="?page={{.NextPage}}">Next</a> {{end}}
{{.StartIndex}} - {{.EndIndex}} of {{.TotalCount}}
```

Defaults: page size 20, max 100.

`NewParams` normalizes bounds, but `NewResult` uses the supplied parameters
without revalidating them. Build parameters through `NewParams`; a manually
supplied zero page size panics. The caller owns total-count and item consistency.
