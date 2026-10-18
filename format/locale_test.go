// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package format

import (
	"math"
	"strings"
	"testing"
)

// TestFormatter protects exact digit/scale preservation, locale independence and
// finite admission. Parallel cases share instances as concurrent callers do.
func TestFormatter(t *testing.T) {
	formatters := map[string]Formatter{}
	longLocale := "en-x-" + strings.Repeat("abcdefgh-", 6) + "abcd"

	for _, locale := range []string{"en", "es", "pl", "fr", "de", "hi-IN", "ar", "fa", "de-AT", "en-XX", longLocale} {
		formatter, err := NewFormatter(locale)
		if err != nil {
			t.Fatal(err)
		}

		formatters[locale] = formatter
	}

	t.Run("display", func(t *testing.T) {
		tests := []struct {
			name      string
			locale    string
			operation string
			decimal   string
			amount    int64
			scale     int
			want      string
		}{
			{name: "English", locale: "en", decimal: "1234567.8900", want: "1,234,567.8900"},
			{name: "Spanish", locale: "es", decimal: "1234567.8900", want: "1.234.567,8900"},
			{name: "Polish", locale: "pl", decimal: "1234567.8900", want: "1\u00a0234\u00a0567,8900"},
			{name: "French", locale: "fr", decimal: "1234567.8900", want: "1\u00a0234\u00a0567,8900"},
			{name: "German", locale: "de", decimal: "1234567.8900", want: "1.234.567,8900"},
			{name: "Indian grouping", locale: "hi-IN", decimal: "1234567.8900", want: "12,34,567.8900"},
			{name: "Arabic digits and sign", locale: "ar", decimal: "-1234567.8900", want: "\u061c-١٬٢٣٤٬٥٦٧٫٨٩٠٠"},
			{name: "Persian digits and sign", locale: "fa", decimal: "-1234567.8900", want: "\u200e−۱٬۲۳۴٬۵۶۷٫۸۹۰۰"},
			{name: "regional conventions", locale: "de-AT", decimal: "1234567.8900", want: "1\u00a0234\u00a0567,8900"},
			{name: "region fallback", locale: "en-XX", decimal: "1234567.8900", want: "1,234,567.8900"},
			{name: "maximum locale bytes", locale: longLocale, decimal: "1234.00", want: "1,234.00"},
			{name: "maximum integer", locale: "en", operation: "integer", amount: math.MaxInt64, want: "9,223,372,036,854,775,807"},
			{name: "minimum integer", locale: "de", operation: "integer", amount: math.MinInt64, want: "-9.223.372.036.854.775.808"},
			{name: "integer zero", locale: "pl", operation: "integer", want: "0"},
			{name: "whole decimal", locale: "en", decimal: "9007199254740993", want: "9,007,199,254,740,993"},
			{name: "negative decimal", locale: "de", decimal: "-1234.00010", want: "-1.234,00010"},
			{name: "small fraction", locale: "en", decimal: "0.00000000000000000100", want: "0.00000000000000000100"},
			{name: "negative zero", locale: "en", decimal: "-0.0000", want: "0.0000"},
			{name: "negative whole zero", locale: "en", decimal: "-0", want: "0"},
			{name: "maximum digits", locale: "en", decimal: "-" + strings.Repeat("1", 128), want: "-11" + strings.Repeat(",111", 42)},
			{name: "maximum fraction", locale: "de", decimal: "-0." + strings.Repeat("1", 127), want: "-0," + strings.Repeat("1", 127)},
			{name: "minor-unit amount", locale: "es", operation: "minor", amount: 123456789, scale: 2, want: "1.234.567,89"},
			{name: "minor-unit zero", locale: "fr", operation: "minor", scale: 2, want: "0,00"},
			{name: "minor-unit small negative", locale: "de", operation: "minor", amount: -1, scale: 2, want: "-0,01"},
			{name: "minor-unit trailing zeros", locale: "en", operation: "minor", amount: 1200, scale: 2, want: "12.00"},
			{name: "minor-unit minimum", locale: "en", operation: "minor", amount: math.MinInt64, scale: 2, want: "-92,233,720,368,547,758.08"},
			{name: "minor-unit zero scale", locale: "en", operation: "minor", amount: math.MinInt64, want: "-9,223,372,036,854,775,808"},
			{name: "minor-unit maximum scale", locale: "en", operation: "minor", amount: math.MinInt64, scale: 9, want: "-9,223,372,036.854775808"},
			{name: "minor-unit maximum", locale: "de", operation: "minor", amount: math.MaxInt64, scale: 9, want: "9.223.372.036,854775807"},
			{name: "minor-unit tiny", locale: "en", operation: "minor", amount: 1, scale: 9, want: "0.000000001"},
			{name: "root zero value", operation: "integer", amount: 1234, want: "1,234"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				formatter := formatters[tt.locale]

				for range 32 {
					var got string

					var err error

					switch tt.operation {
					case "integer":
						got = formatter.Integer(tt.amount)
					case "minor":
						got, err = formatter.MinorUnits(tt.amount, tt.scale)
					default:
						got, err = formatter.Decimal(tt.decimal)
					}

					if err != nil || got != tt.want {
						t.Fatalf("got (%q, %v), want (%q, nil)", got, err, tt.want)
					}
				}
			})
		}
	})

	t.Run("invalid locales", func(t *testing.T) {
		tests := []struct{ name, value string }{
			{"empty", ""}, {"whitespace", " en"}, {"underscore", "en_US"},
			{"noncanonical case", "EN-us"}, {"canonicalized alias", "iw"},
			{"unknown language", "zz"},
			{"malformed", "en--US"}, {"non-ASCII", "é"},
			{"over limit", "en-x-" + strings.Repeat("a", 59)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewFormatter(tt.value)
				if err == nil {
					t.Fatal("accepted invalid locale")
				}
			})
		}
	})

	t.Run("invalid decimals", func(t *testing.T) {
		tests := []struct{ name, value string }{
			{"empty", ""}, {"minus only", "-"}, {"plus", "+1"},
			{"whitespace", " 1"}, {"trailing whitespace", "1 "},
			{"leading zeros", "01"}, {"negative leading zeros", "-00.1"},
			{"missing integer", ".1"}, {"missing fraction", "1."},
			{"repeated point", "1.2.3"}, {"exponent", "1e3"},
			{"grouping", "1,234"}, {"NaN", "NaN"}, {"infinity", "Inf"},
			{"embedded minus", "1-2"}, {"non-ASCII digits", "١٢"},
			{"over digit limit", strings.Repeat("1", 129)},
			{"over fraction limit", "-0." + strings.Repeat("1", 128)},
			{"over byte limit", strings.Repeat("1", 131)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := formatters["en"].Decimal(tt.value)
				if err == nil || got != "" {
					t.Fatalf("got (%q, %v), want empty result and error", got, err)
				}
			})
		}
	})

	t.Run("invalid scales", func(t *testing.T) {
		tests := []struct {
			name  string
			scale int
		}{{"negative", -1}, {"above limit", 10}, {"maximum int", math.MaxInt}}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := formatters["en"].MinorUnits(1, tt.scale)
				if err == nil || got != "" {
					t.Fatalf("got (%q, %v), want empty result and error", got, err)
				}
			})
		}
	})
}
