package telemetry

import (
	"testing"

	"github.com/hatmaxkit/hatmax/settings"
)

func TestSchemasKeyConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		wantKey  string
	}{
		{"KeyMode matches schema", KeyMode, "telemetry.mode"},
		{"KeyInstanceID matches schema", KeyInstanceID, "telemetry.instance_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.wantKey {
				t.Errorf("constant = %q, want %q", tt.constant, tt.wantKey)
			}
		})
	}
}

func TestSchemasModeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ModeOff", ModeOff, "off"},
		{"ModeBasic", ModeBasic, "basic"},
		{"ModeFull", ModeFull, "full"},
		{"ModeDebug", ModeDebug, "debug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("constant = %q, want %q", tt.constant, tt.want)
			}
		})
	}
}

func TestSchemasValidation(t *testing.T) {
	registry := settings.NewRegistry()
	RegisterSchemas(registry)

	modeSchema, ok := registry.Get(KeyMode)
	if !ok {
		t.Fatal("mode schema not registered")
	}

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid off", ModeOff, false},
		{"valid basic", ModeBasic, false},
		{"valid full", ModeFull, false},
		{"valid debug", ModeDebug, false},
		{"invalid value", "invalid", true},
		{"empty value", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := modeSchema.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestRegisterSchemas(t *testing.T) {
	registry := settings.NewRegistry()

	RegisterSchemas(registry)

	if _, ok := registry.Get(KeyMode); !ok {
		t.Error("KeyMode not registered")
	}

	if _, ok := registry.Get(KeyInstanceID); !ok {
		t.Error("KeyInstanceID not registered")
	}
}

func TestSchemasCount(t *testing.T) {
	if len(Schemas) != 2 {
		t.Errorf("Schemas has %d entries, want 2", len(Schemas))
	}
}

func TestModeSchemaOptions(t *testing.T) {
	var modeSchema settings.Schema

	for _, s := range Schemas {
		if s.Key == KeyMode {
			modeSchema = s

			break
		}
	}

	if modeSchema.Key == "" {
		t.Fatal("mode schema not found")
	}

	if len(modeSchema.Options) != 4 {
		t.Errorf("Options has %d entries, want 4", len(modeSchema.Options))
	}

	if len(modeSchema.Labels) != 4 {
		t.Errorf("Labels has %d entries, want 4", len(modeSchema.Labels))
	}

	if modeSchema.Default != ModeBasic {
		t.Errorf("Default = %q, want %q", modeSchema.Default, ModeBasic)
	}
}
