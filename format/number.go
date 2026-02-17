package format

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// defaultPrinter is used for number formatting.
var defaultPrinter = message.NewPrinter(language.English)

// Number formats a number with thousand separators.
// Examples:
//
//	Number(1234)      // "1,234"
//	Number(1234567)   // "1,234,567"
//	Number(1234.56)   // "1,234.56"
func Number(n any) string {
	return defaultPrinter.Sprintf("%v", number.Decimal(n))
}

// Integer formats a number as an integer with thousand separators.
// Examples:
//
//	Integer(1234.56)  // "1,235"
//	Integer(1234567)  // "1,234,567"
func Integer(n any) string {
	return defaultPrinter.Sprintf("%v", number.Decimal(n, number.MaxFractionDigits(0)))
}
