<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Modal

`modal` holds the data for a dialog. It does not render HTML. The
implementation note is [modal/readme.md](../../../modal/readme.md).

## Config

| Field | Meaning |
| --- | --- |
| `ID` | Dialog identifier. |
| `Title` | Dialog title. |
| `Size` | Width token. |
| `CloseOnEsc` | Whether Escape closes the dialog. |
| `CloseOnClick` | Whether a click outside the dialog closes it. |

`SizeSmall` is `modal-sm`. `SizeMedium` is `modal-md`. `SizeLarge` is
`modal-lg`.

`DefaultConfig(id, title)` sets those two fields, `SizeMedium`, and both close
flags to true.
