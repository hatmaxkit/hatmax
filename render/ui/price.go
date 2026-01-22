package ui

import (
	"fmt"
	"html/template"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// CurrencySymbol maps currency codes to symbols.
var CurrencySymbol = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"PLN": "zł",
	"ARS": "$",
	"BRL": "R$",
	"MXN": "$",
}

// CurrencyPosition indicates where the symbol appears.
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

// CurrencyFormats maps currency codes to their format preferences.
var CurrencyFormats = map[string]CurrencyFormat{
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
	CurrencySymbol[code] = symbol
	CurrencyFormats[code] = CurrencyFormat{Symbol: symbol, Position: position}
}

// defaultPrinter is used for number formatting.
// Will be replaced by locale-aware printer in Phase 2.
var defaultPrinter = message.NewPrinter(language.English)

// FormatPrice formats a price amount with the given currency.
// Returns formatted string like "$150,000" or "150.000 €".
func FormatPrice(amount float64, currency string) string {
	format, ok := CurrencyFormats[currency]
	if !ok {
		format = CurrencyFormat{Symbol: currency, Position: SymbolBefore}
	}

	formatted := defaultPrinter.Sprintf("%v", number.Decimal(amount, number.MaxFractionDigits(0)))

	if format.Position == SymbolAfter {
		return fmt.Sprintf("%s %s", formatted, format.Symbol)
	}
	return fmt.Sprintf("%s%s", format.Symbol, formatted)
}

// PriceTag renders a price tag component.
func PriceTag(amount float64, currency string) template.HTML {
	formatted := FormatPrice(amount, currency)
	return template.HTML(fmt.Sprintf(`<span class="price">%s</span>`, escape(formatted)))
}

// PriceTagNegotiable renders a price tag with negotiable indicator.
func PriceTagNegotiable(amount float64, currency string, negotiable bool) template.HTML {
	formatted := FormatPrice(amount, currency)
	if negotiable {
		return template.HTML(fmt.Sprintf(
			`<span class="price">%s <small class="price__tag">Negotiable</small></span>`,
			escape(formatted),
		))
	}
	return PriceTag(amount, currency)
}

// PriceRange renders a price range.
func PriceRange(min, max float64, currency string) template.HTML {
	minFormatted := FormatPrice(min, currency)
	maxFormatted := FormatPrice(max, currency)
	return template.HTML(fmt.Sprintf(
		`<span class="price-range">%s - %s</span>`,
		escape(minFormatted),
		escape(maxFormatted),
	))
}
