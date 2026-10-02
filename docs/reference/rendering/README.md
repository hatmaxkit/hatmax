<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Rendering

`render` supplies template functions. UI components are not in this package.
The implementation note is [render/readme.md](../../../render/readme.md).

## Functions

`FuncMap` registers:

| Name | Result |
| --- | --- |
| `upper` | `strings.ToUpper` |
| `lower` | `strings.ToLower` |
| `t` | The key argument, ignoring the locale. |
| `add` | The sum of two `int` values. |
| `sub` | The difference of two `int` values. |
| `seq` | Integers from `start` through `end`, or nil when `end < start`. |

`FuncMapWithHTMX` merges `FuncMap` with `htmx.FuncMap`. Later keys win.

`MergeFuncMaps` copies each map in order. A later map replaces a duplicate
key.

`I18nFuncMap(translator)` registers `t`. A nil translator returns the key.
Otherwise `t` returns `translator.Get(locale, key)`.
