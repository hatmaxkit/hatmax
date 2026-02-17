package telemetry

import "github.com/hatmaxkit/hatmax/settings"

const (
	KeyMode       = "telemetry.mode"
	KeyInstanceID = "telemetry.instance_id"
)

const (
	ModeOff   = "off"
	ModeBasic = "basic"
	ModeFull  = "full"
	ModeDebug = "debug"
)

var Schemas = []settings.Schema{
	{
		Key:         KeyMode,
		Type:        settings.Enum,
		Default:     ModeBasic,
		Options:     []string{ModeOff, ModeBasic, ModeFull, ModeDebug},
		Labels:      []string{"Off", "Basic (ping only)", "Full (with request count)", "Debug (with crash reports)"},
		Label:       "Telemetry Mode",
		Description: "Controls what anonymous data is sent to help improve the product",
	},
	{
		Key:         KeyInstanceID,
		Type:        settings.String,
		Label:       "Instance ID",
		Description: "Unique identifier for this installation",
	},
}

// RegisterSchemas registers telemetry schemas with the given registry.
func RegisterSchemas(r *settings.Registry) {
	for _, s := range Schemas {
		r.Register(s)
	}
}
