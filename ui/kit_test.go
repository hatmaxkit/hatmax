package ui

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/hatmaxkit/hatmax/config"
	"github.com/hatmaxkit/hatmax/log"
	"github.com/hatmaxkit/hatmax/settings"
)

func TestNew(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")

	k := New(cfg, logger)

	if k.cfg != cfg {
		t.Error("New() cfg not set")
	}
	if k.log != logger {
		t.Error("New() log not set")
	}
}

func TestWithSettings(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")
	settingsSvc := &settings.Service{}

	k := New(cfg, logger, WithSettings(settingsSvc))

	if k.settings != settingsSvc {
		t.Error("WithSettings() settings not set")
	}
}

func TestWithOverlay(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")
	overlay := fstest.MapFS{
		"test.txt": &fstest.MapFile{Data: []byte("test")},
	}

	k := New(cfg, logger, WithOverlay(overlay))

	if k.overlay == nil {
		t.Error("WithOverlay() overlay not set")
	}
}

func TestKitAccessors(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")
	settingsSvc := &settings.Service{}

	k := New(cfg, logger, WithSettings(settingsSvc))

	if k.Config() != cfg {
		t.Error("Config() returned wrong value")
	}
	if k.Logger() != logger {
		t.Error("Logger() returned wrong value")
	}
	if k.Settings() != settingsSvc {
		t.Error("Settings() returned wrong value")
	}
}

func TestWithCSRFFunc(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")

	csrfFunc := func(ctx context.Context) string {
		return "test-token"
	}

	k := New(cfg, logger, WithCSRFFunc(csrfFunc))

	token := k.CSRFToken(context.Background())
	if token != "test-token" {
		t.Errorf("CSRFToken() = %q, want %q", token, "test-token")
	}
}

func TestCSRFTokenWithoutFunc(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")

	k := New(cfg, logger)

	token := k.CSRFToken(context.Background())
	if token != "" {
		t.Errorf("CSRFToken() without func = %q, want empty", token)
	}
}

