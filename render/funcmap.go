package render

import (
	"html/template"

	"github.com/hatmaxkit/hatmax/render/ui"
)

// FuncMap returns a template.FuncMap with all render functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// Chips
		"chip":          ui.Chip,
		"chipMuted":     ui.ChipMuted,
		"chipWithClass": ui.ChipWithClass,

		// Pills
		"pill":          ui.Pill,
		"pillMuted":     ui.PillMuted,
		"pillWithClass": ui.PillWithClass,

		// Badges
		"badge":               ui.Badge,
		"badgeWithVariant":    ui.BadgeWithVariant,
		"statusBadge":         ui.StatusBadge,
		"statusBadgeWithIcon": ui.StatusBadgeWithIcon,

		// Prices
		"formatPrice":        ui.FormatPrice,
		"priceTag":           ui.PriceTag,
		"priceTagNegotiable": ui.PriceTagNegotiable,
		"priceRange":         ui.PriceRange,

		// Stats
		"formatNumber": ui.FormatNumber,
		"stat":         ui.Stat,
		"statWithIcon": ui.StatWithIcon,
		"statCompact":  ui.StatCompact,
	}
}

// MergeFuncMaps merges multiple FuncMaps into one.
// Later maps override earlier ones for duplicate keys.
func MergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	result := make(template.FuncMap)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
