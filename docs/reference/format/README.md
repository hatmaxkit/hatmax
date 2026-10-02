<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Format

`format` prints numbers and prices in English. The implementation note is
[format/readme.md](../../../format/readme.md).

`Number` prints a decimal with grouping separators. `Integer` prints the same
value with no fraction digits, so a fractional amount is rounded.

## Prices

`SymbolBefore` places the symbol against the amount. `SymbolAfter` places the
symbol after the amount, separated by a space.

| Code | Symbol | Position |
| --- | --- | --- |
| `USD` | `$` | before |
| `EUR` | `€` | after |
| `GBP` | `£` | before |
| `PLN` | `zł` | after |
| `ARS` | `$` | before |
| `BRL` | `R$` | before |
| `MXN` | `$` | before |

`RegisterCurrency` overwrites one code. An unknown code uses the code itself
as the symbol, before the amount.

`Price` formats the amount with `Integer`. `PriceWithDecimals` formats it with
`Number`. `PriceRange` joins `Price(min)` and `Price(max)` with ` - `.
