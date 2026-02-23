package ui

// StatusConfig maps a status value to its display configuration.
type StatusConfig struct {
	Variant Variant
	Label   string
	Icon    string
}

// statusRegistry holds the status-to-config mappings.
var statusRegistry = map[string]StatusConfig{
	"active":   {Variant: VariantSuccess, Label: "Active", Icon: "●"},
	"draft":    {Variant: VariantWarning, Label: "Draft", Icon: "○"},
	"inactive": {Variant: VariantMuted, Label: "Inactive", Icon: "◐"},
	"expired":  {Variant: VariantDanger, Label: "Expired", Icon: "✕"},
	"featured": {Variant: VariantInfo, Label: "Featured", Icon: "★"},
}

// RegisterStatus adds or overrides a status configuration.
func RegisterStatus(status string, cfg StatusConfig) {
	statusRegistry[status] = cfg
}

// StatusBadge creates a label configured for the given status.
// It uses the status registry to determine variant and display text.
func StatusBadge(status string) *Label {
	cfg, ok := statusRegistry[status]
	if !ok {
		return NewLabel(status)
	}

	label := NewLabel(cfg.Label)
	if cfg.Variant != "" {
		label.Variant(cfg.Variant)
	}

	return label
}

// StatusBadgeWithIcon creates a status label with an icon prefix.
func StatusBadgeWithIcon(status string) *Label {
	cfg, ok := statusRegistry[status]
	if !ok {
		return NewLabel(status)
	}

	text := cfg.Label
	if cfg.Icon != "" {
		text = cfg.Icon + " " + cfg.Label
	}

	label := NewLabel(text)
	if cfg.Variant != "" {
		label.Variant(cfg.Variant)
	}

	return label
}
