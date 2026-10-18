// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package format_test

import (
	"fmt"
	"html/template"
	"strings"

	"hatmax.adrianpk.com/format"
	"hatmax.adrianpk.com/render"
	"hatmax.adrianpk.com/ui"
)

// ExampleFormatter composes exact and historical helpers through the real
// render/UI function maps while preserving ordinary template escaping.
func ExampleFormatter() {
	formatter, err := format.NewFormatter("de")
	if err != nil {
		panic(err)
	}

	funcs := render.MergeFuncMaps(ui.FuncMap(), template.FuncMap{
		"exactInteger": formatter.Integer,
		"exactDecimal": formatter.Decimal,
		"exactMinor":   formatter.MinorUnits,
	})
	page := template.Must(template.New("amount").Funcs(funcs).Parse(
		`<p>{{.Currency}} {{exactMinor .Amount .Scale}}; {{exactInteger .Count}}; {{exactDecimal .Quantity}}; {{.Note}}; {{formatNumber .Count}}; {{formatPrice .LegacyAmount "USD"}}</p>`,
	))
	view := struct {
		Currency     string
		Amount       int64
		Scale        int
		Count        int64
		Quantity     string
		Note         string
		LegacyAmount float64
	}{"EUR", 123450, 2, 1234, "12.3400", "<net>", 1234}

	var output strings.Builder

	err = page.Execute(&output, view)
	if err != nil {
		panic(err)
	}

	fmt.Println(output.String())
	// Output:
	// <p>EUR 1.234,50; 1.234; 12,3400; &lt;net&gt;; 1,234; $1,234</p>
}
