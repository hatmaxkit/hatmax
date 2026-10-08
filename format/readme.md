<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# format

Formatting utilities for prices and numbers. Pure functions without HTML output.

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
