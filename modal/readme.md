<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# modal

Modal dialog configuration for templates.

## Usage

```go
// In handler
cfg := modal.DefaultConfig("delete-modal", "Confirm Delete")
cfg.Size = modal.SizeLarge

// In template
<div id="{{.Modal.ID}}" class="modal {{.Modal.Size}}">
    <h2>{{.Modal.Title}}</h2>
    ...
</div>
```

Sizes: `SizeSmall`, `SizeMedium`, `SizeLarge`.
