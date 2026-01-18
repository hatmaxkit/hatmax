package ui

import (
	"fmt"
	"html/template"
)

// StatusVariant maps status values to CSS modifier classes.
var StatusVariant = map[string]string{
	"active":   "success",
	"draft":    "warning",
	"inactive": "muted",
	"expired":  "danger",
	"featured": "info",
}

// StatusIcon maps status values to icons.
var StatusIcon = map[string]string{
	"active":   "●",
	"draft":    "○",
	"inactive": "◐",
	"expired":  "✕",
	"featured": "★",
}

// StatusLabel maps status values to display labels.
// This will be replaced by i18n in Phase 3.
var StatusLabel = map[string]string{
	"active":   "Active",
	"draft":    "Draft",
	"inactive": "Inactive",
	"expired":  "Expired",
	"featured": "Featured",
}

// Badge renders a badge component.
func Badge(label string) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="badge">%s</span>`, escape(label)))
}

// BadgeWithVariant renders a badge with a variant class.
func BadgeWithVariant(label, variant string) template.HTML {
	if variant == "" {
		return Badge(label)
	}
	return template.HTML(fmt.Sprintf(`<span class="badge badge--%s">%s</span>`, escape(variant), escape(label)))
}

// StatusBadge renders a badge for a status value with automatic styling.
// It uses StatusVariant for the CSS class and StatusLabel for the display text.
func StatusBadge(status string) template.HTML {
	variant := StatusVariant[status]
	if variant == "" {
		variant = status
	}

	label := StatusLabel[status]
	if label == "" {
		label = status
	}

	return template.HTML(fmt.Sprintf(
		`<span class="badge badge--%s">%s</span>`,
		escape(variant),
		escape(label),
	))
}

// StatusBadgeWithIcon renders a status badge with an icon prefix.
func StatusBadgeWithIcon(status string) template.HTML {
	variant := StatusVariant[status]
	if variant == "" {
		variant = status
	}

	label := StatusLabel[status]
	if label == "" {
		label = status
	}

	icon := StatusIcon[status]

	if icon != "" {
		return template.HTML(fmt.Sprintf(
			`<span class="badge badge--%s">%s %s</span>`,
			escape(variant),
			icon,
			escape(label),
		))
	}

	return StatusBadge(status)
}

// RegisterStatusVariant adds or overrides a status variant mapping.
func RegisterStatusVariant(status, variant string) {
	StatusVariant[status] = variant
}

// RegisterStatusLabel adds or overrides a status label mapping.
func RegisterStatusLabel(status, label string) {
	StatusLabel[status] = label
}

// RegisterStatusIcon adds or overrides a status icon mapping.
func RegisterStatusIcon(status, icon string) {
	StatusIcon[status] = icon
}
