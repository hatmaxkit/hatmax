package ui

import (
	"fmt"
	"html/template"
)

// Chip renders a chip component.
func Chip(label string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="chip">%s</span>`, escape(label)))
}

// ChipMuted renders a muted chip component.
func ChipMuted(label string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="chip chip--muted">%s</span>`, escape(label)))
}

// ChipWithClass renders a chip with custom class.
func ChipWithClass(label, class string) template.HTML {
	if class == "" {
		return Chip(label)
	}
	return template.HTML(fmt.Sprintf(`<span class="chip %s">%s</span>`, escape(class), escape(label)))
}

// Pill renders a pill component (rounded variant).
func Pill(label string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="pill">%s</span>`, escape(label)))
}

// PillMuted renders a muted pill component.
func PillMuted(label string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="pill pill--muted">%s</span>`, escape(label)))
}

// PillWithClass renders a pill with custom class.
func PillWithClass(label, class string) template.HTML {
	if class == "" {
		return Pill(label)
	}
	return template.HTML(fmt.Sprintf(`<span class="pill %s">%s</span>`, escape(class), escape(label)))
}
