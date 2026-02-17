package format

import "fmt"

// CurrencyPosition indicates where the symbol appears relative to the amount.
type CurrencyPosition int

const (
	SymbolBefore CurrencyPosition = iota
	SymbolAfter
)

// CurrencyFormat holds formatting preferences for a currency.
type CurrencyFormat struct {
	Symbol   string
	Position CurrencyPosition
}

// currencies maps currency codes to their format preferences.
var currencies = map[string]CurrencyFormat{
	"USD": {Symbol: "$", Position: SymbolBefore},
	"EUR": {Symbol: "€", Position: SymbolAfter},
	"GBP": {Symbol: "£", Position: SymbolBefore},
	"PLN": {Symbol: "zł", Position: SymbolAfter},
	"ARS": {Symbol: "$", Position: SymbolBefore},
	"BRL": {Symbol: "R$", Position: SymbolBefore},
	"MXN": {Symbol: "$", Position: SymbolBefore},
}

// RegisterCurrency adds or updates a currency format.
func RegisterCurrency(code, symbol string, position CurrencyPosition) {
	currencies[code] = CurrencyFormat{Symbol: symbol, Position: position}
}

// Price formats a price amount with the given currency.
// Examples:
//
//	Price(150000, "USD")  // "$150,000"
//	Price(150000, "EUR")  // "150,000 €"
//	Price(99.99, "GBP")   // "£100"
func Price(amount float64, currency string) string {
	format, ok := currencies[currency]
	if !ok {
		format = CurrencyFormat{Symbol: currency, Position: SymbolBefore}
	}

	formatted := Integer(amount)

	if format.Position == SymbolAfter {
		return fmt.Sprintf("%s %s", formatted, format.Symbol)
	}
	return fmt.Sprintf("%s%s", format.Symbol, formatted)
}

// PriceWithDecimals formats a price with decimal places.
// Examples:
//
//	PriceWithDecimals(99.99, "USD")  // "$99.99"
//	PriceWithDecimals(99.99, "EUR")  // "99.99 €"
func PriceWithDecimals(amount float64, currency string) string {
	format, ok := currencies[currency]
	if !ok {
		format = CurrencyFormat{Symbol: currency, Position: SymbolBefore}
	}

	formatted := Number(amount)

	if format.Position == SymbolAfter {
		return fmt.Sprintf("%s %s", formatted, format.Symbol)
	}
	return fmt.Sprintf("%s%s", format.Symbol, formatted)
}

// PriceRange formats a price range.
// Examples:
//
//	PriceRange(100, 500, "USD")  // "$100 - $500"
//	PriceRange(100, 500, "EUR")  // "100 € - 500 €"
func PriceRange(min, max float64, currency string) string {
	return fmt.Sprintf("%s - %s", Price(min, currency), Price(max, currency))
}
