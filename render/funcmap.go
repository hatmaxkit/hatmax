package render

import (
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
	"github.com/hatmaxkit/hatmax/i18n"
	"github.com/hatmaxkit/hatmax/render/ui"
)

// FuncMap returns a template.FuncMap with all render functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// String
		"upper": strings.ToUpper,
		"lower": strings.ToLower,

		// i18n (default no-op, override with I18nFuncMap)
		"t": func(locale, key string) string { return key },

		// Math
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"seq": func(start, end int) []int {
			if end < start {
				return nil
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},

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

		// Buttons
		"btn":       ui.Btn,
		"btnSubmit": ui.BtnSubmit,
		"btnDanger": ui.BtnDanger,

		// Links
		"link":        ui.A,
		"linkBlank":   ui.ABlank,
		"linkBoosted": ui.ABoosted,
		"navLink":     ui.Nav,

		// Forms
		"form":     ui.NewForm,
		"input":    ui.NewInput,
		"text":     ui.Text,
		"email":    ui.Email,
		"password": ui.Password,
		"number":   ui.Number,
		"hidden":   ui.Hidden,
		"search":   ui.Search,
		"field":    ui.NewField,

		// Alerts
		"alert":        ui.NewAlert,
		"alertInfo":    ui.AlertInfoMsg,
		"alertSuccess": ui.AlertSuccessMsg,
		"alertWarning": ui.AlertWarningMsg,
		"alertDanger":  ui.AlertDangerMsg,
		"alertError":   ui.AlertErrorMsg,
		"flash":        ui.NewFlash,
		"toast":        ui.NewToast,
	}
}

// FuncMapWithHTMX returns a template.FuncMap with all render and HTMX functions.
func FuncMapWithHTMX() template.FuncMap {
	return MergeFuncMaps(FuncMap(), htmx.FuncMap())
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

// I18nFuncMap returns template functions for internationalization.
func I18nFuncMap(translator *i18n.Translator) template.FuncMap {
	return template.FuncMap{
		"t": func(locale, key string) string {
			if translator == nil {
				return key
			}
			return translator.Get(locale, key)
		},
	}
}
