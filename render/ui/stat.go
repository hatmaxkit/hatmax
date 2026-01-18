package ui

import (
	"fmt"
	"html/template"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// statPrinter is used for number formatting in stats.
var statPrinter = message.NewPrinter(language.English)

// FormatNumber formats a number with thousand separators.
func FormatNumber(n any) string {
	return statPrinter.Sprintf("%v", number.Decimal(n))
}

// Stat renders a stat component with value and label.
func Stat(value any, label string) template.HTML {
	formatted := FormatNumber(value)
	return template.HTML(fmt.Sprintf(
		`<div class="stat"><span class="stat__value">%s</span><span class="stat__label">%s</span></div>`,
		escape(formatted),
		escape(label),
	))
}

// StatWithIcon renders a stat component with an icon.
func StatWithIcon(value any, label, icon string) template.HTML {
	formatted := FormatNumber(value)
	return template.HTML(fmt.Sprintf(
		`<div class="stat"><span class="stat__icon">%s</span><span class="stat__value">%s</span><span class="stat__label">%s</span></div>`,
		icon,
		escape(formatted),
		escape(label),
	))
}

// StatCompact renders a compact inline stat.
func StatCompact(value any, label string) template.HTML {
	formatted := FormatNumber(value)
	return template.HTML(fmt.Sprintf(
		`<span class="stat-compact"><strong>%s</strong> %s</span>`,
		escape(formatted),
		escape(label),
	))
}
