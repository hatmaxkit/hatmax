<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# modal

Modal dialog configuration for templates.

## Usage

These are separate handler and template fragments. Carry `cfg` in the view's
`Modal` field in the [page/view composition](../docs/tutorials/user-guide/pages-and-partials.md).
The complete [guide companion](../examples/guide/main.go) demonstrates passing
application-owned view data to the template manager. `modal` itself does not
render a dialog or install browser behavior.

```go
// In handler
cfg := modal.DefaultConfig("delete-modal", "Confirm Delete")
cfg.Size = modal.SizeLarge

```

```html
<div id="{{.Modal.ID}}" class="modal {{.Modal.Size}}">
    <h2>{{.Modal.Title}}</h2>
    ...
</div>
```

Sizes: `SizeSmall`, `SizeMedium`, `SizeLarge`.
