package ui

import "html/template"

// Component renders to HTML.
type Component interface {
	Render() template.HTML
}

// escape escapes a string for safe HTML output.
func escape(s string) string {
	return template.HTMLEscapeString(s)
}
