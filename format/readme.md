<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# format

Formatting utilities for prices and numbers. Pure functions without HTML output.

## Exact Locale-Aware Display

Construct an immutable `Formatter` with an explicit canonical BCP 47 locale.
The following contextual calls preserve exact digits and fractional zeros:

```go
formatter, err := format.NewFormatter("de")
if err != nil {
	return err
}
count := formatter.Integer(1234567)                 // "1.234.567"
quantity, err := formatter.Decimal("1234567.8900") // "1.234.567,8900"
if err != nil {
	return err
}
amount, err := formatter.MinorUnits(123450, 2)      // "1.234,50"
if err != nil {
	return err
}
label := "EUR " + amount                          // "EUR 1.234,50"
```

Applications own currency identity, scale, locale defaults and arithmetic.
These methods only prepare display text; they never convert through floating
point or round. A formatter is safe to share between concurrent callers.
Locale behavior follows the pinned `golang.org/x/text` data, including regional
fallback, grouping, digit shapes and signs. There is no fixed language enum.

See the [reference](../docs/reference/format/README.md) for admission limits,
the [presentation guide](../docs/tutorials/user-guide/presentation-primitives.md)
for template composition and the [executable example](locale_example_test.go)
for all three operations, existing helpers and escaped application text.

## Usage

Import `hatmax.adrianpk.com/format` in the handler/view composition described in
[Presentation Primitives](../docs/tutorials/user-guide/presentation-primitives.md).
These calls are contextual expressions; the comments state their observable
results. The [guide companion](../examples/guide/main.go) shows the complete
page/view boundary where formatted values belong.

```go
// Numbers
format.Number(1234567)    // "1,234,567"
format.Integer(1234.56)   // "1,235"

// Prices
format.Price(150000, "USD")          // "$150,000"
format.Price(150000, "EUR")          // "150,000 €"
format.PriceWithDecimals(99.99, "USD") // "$99.99"
format.PriceRange(100, 500, "USD")   // "$100 - $500"

// Register custom currencies
format.RegisterCurrency("CLP", "$", format.SymbolBefore)
```

## Template Usage

Available via `ui.FuncMap()`:

```html
<span class="price">{{ formatPrice .Amount .Currency }}</span>
<span class="count">{{ formatNumber .Count }}</span>
```

The existing UI functions keep English number formatting. Install application
names bound to your formatter's methods to opt into localized exact display.
The new methods return ordinary strings; let `html/template` escape them.
