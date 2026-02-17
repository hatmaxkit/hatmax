package ui

import (
	"html/template"

	"github.com/hatmaxkit/hatmax/format"
	"github.com/hatmaxkit/hatmax/htmx"
	"github.com/hatmaxkit/hatmax/render"
)

// FuncMap returns a template.FuncMap with all kit functions.
func (k *Kit) FuncMap() template.FuncMap {
	fm := render.FuncMapWithHTMX()
	addUIFuncs(fm)
	return fm
}

// FuncMap returns a standalone FuncMap without kit instance.
func FuncMap() template.FuncMap {
	fm := render.FuncMapWithHTMX()
	addUIFuncs(fm)
	return fm
}

func addUIFuncs(fm template.FuncMap) {
	// Format
	fm["formatPrice"] = format.Price
	fm["formatNumber"] = format.Number

	// Chips and Labels
	fm["chip"] = NewChip
	fm["label"] = NewLabel

	// Badges
	fm["statusBadge"] = StatusBadge
	fm["statusBadgeWithIcon"] = StatusBadgeWithIcon

	// Buttons
	fm["btn"] = Btn
	fm["btnSubmit"] = BtnSubmit
	fm["btnDanger"] = BtnDanger
	fm["button"] = NewButton

	// Layout
	fm["page"] = NewPage
	fm["pageHeader"] = NewPageHeader
	fm["container"] = NewContainer

	// Navigation
	fm["navGrid"] = NewNavGrid
	fm["nav"] = NewNav

	// Tables
	fm["table"] = NewTable
	fm["col"] = Col
	fm["row"] = NewRow
	fm["text"] = Text
	fm["html"] = HTML

	// Settings
	fm["settingsForm"] = NewSettingsForm

	// Forms
	fm["form"] = NewForm
	fm["deleteButton"] = NewDeleteButton

	// Alerts
	fm["alert"] = NewAlert
	fm["alertInfo"] = AlertInfo
	fm["alertSuccess"] = AlertSuccess
	fm["alertWarning"] = AlertWarning
	fm["alertDanger"] = AlertDanger
	fm["alertError"] = AlertError
	fm["flash"] = NewFlash
	fm["toast"] = NewToast

	// Links
	fm["link"] = NewLink
	fm["linkBlank"] = ABlank
	fm["linkBoosted"] = ABoosted
}

// MergeFuncMaps merges multiple FuncMaps into one.
func MergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	return render.MergeFuncMaps(maps...)
}

// HX returns an htmx.Attrs builder for templates.
func HX() *htmx.Attrs {
	return htmx.HX()
}
