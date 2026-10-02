// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// This fixed template is only executed, never modified. Plain string inputs
// retain Go's URL filtering; callers cannot bypass it with template.URL.
var urlValueTemplate = template.Must(template.New("ui-url").Parse(`<a href="{{.}}">`))

// escapeURL returns a filtered, normalized, HTML-escaped attribute value.
// The surrounding element establishes URL context and is not part of the value.
func escapeURL(value string) string {
	var output strings.Builder

	err := urlValueTemplate.Execute(&output, value)
	if err != nil {
		// The fixed template, string input, and in-memory writer cannot fail.
		panic(fmt.Sprintf("ui: render URL attribute: %v", err))
	}

	markup := output.String()

	return markup[len(`<a href="`) : len(markup)-len(`">`)]
}
