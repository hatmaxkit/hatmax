# format

Formatting utilities for prices and numbers. Pure functions without HTML output.

## Usage

```go
import "github.com/hatmaxkit/hatmax/format"

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
