<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Format

`format` provides exact locale-selected display and historical English number
and price helpers. The implementation note is
[format/readme.md](../../../format/readme.md).

## Exact Locale-Aware Display

| API | Contract |
| --- | --- |
| `NewFormatter(locale string) (Formatter, error)` | Construct an immutable formatter from an explicit canonical BCP 47 tag. |
| `Formatter.Integer(value int64) string` | Group a signed integer without fraction digits or loss of precision. |
| `Formatter.Decimal(value string) (string, error)` | Localize exact canonical decimal text and preserve every fractional digit. |
| `Formatter.MinorUnits(value int64, scale int) (string, error)` | Display a signed amount with exactly `scale` fractional digits. |

Construction accepts 1–63 ASCII bytes recognized by the pinned
`golang.org/x/text/language` parser. Tags must already be canonical: use `en-US`,
not `en_US` or `EN-us`. Malformed or unrecognized tags return an error. The
caller chooses defaults before construction. The locale data supplies normal
language/region fallback and digit, grouping, separator and sign conventions;
additional supported locales need no application language enum. The zero-value
formatter uses the root locale, with English-style decimal grouping.

Decimal grammar is `-?(0|[1-9][0-9]*)(\.[0-9]+)?`, bounded to 128 ASCII digits
excluding the sign and point. Whitespace, plus signs, exponents, grouping,
non-ASCII digits, NaN and infinity are invalid. Negative zero becomes unsigned
zero with the same fractional scale. Invalid decimals and scales return an
empty string and a bounded error that excludes the input.

Minor-unit scale must be from zero through nine. The caller supplies this
scale explicitly; it is not inferred from a currency. Zero, negative amounts
and the minimum signed 64-bit integer retain their exact value and scale.
No exact operation converts through floating point or performs rounding,
arithmetic, currency conversion or tax calculations.

For example, the contextual calls for a formatter constructed with `de` are:

```go
formatter.Integer(1234567)          // "1.234.567"
formatter.Decimal("1234567.8900")   // "1.234.567,8900", nil
formatter.MinorUnits(123450, 2)     // "1.234,50", nil
formatter.MinorUnits(-1, 2)         // "-0,01", nil
formatter.Decimal("-0.00")         // "0,00", nil
```

Instances are immutable and safe for concurrent use. They do not change a
global current locale or the currency registry. Results are plain strings,
subject to normal Go template escaping. Keep the ISO currency code in the
application view, for example `EUR 1.234,50`.

The [presentation guide](../../tutorials/user-guide/presentation-primitives.md)
shows opt-in template functions. The
[executable example](../../../format/locale_example_test.go) composes the real
render/UI maps, verifies all three methods and shows escaped application text.

## Historical Number Helpers

`Number` prints a decimal with grouping separators. `Integer` prints the same
value with no fraction digits, so a fractional amount is rounded.
Both keep their English printer and existing signatures and semantics.

## Historical Prices

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

Configure currency registrations before concurrent request handling.
`RegisterCurrency` mutates a package map and does not synchronize concurrent
reads/writes. Currency codes are case-sensitive; registering a code does not
install locale-aware number formatting.
