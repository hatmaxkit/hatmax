package ui

import (
	"testing"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

func TestUIFuncMap(t *testing.T) {
	cfg := config.New()
	logger := log.NewTestLogger("error")
	u := New(cfg, logger)

	fm := u.FuncMap()

	funcs := []string{
		"chip", "label",
		"btn", "btnSubmit", "btnDanger", "button",
		"page", "pageHeader", "container",
		"navGrid", "nav",
		"table", "col", "row", "text", "html",
		"settingsForm",
		"form", "deleteButton",
	}

	for _, name := range funcs {
		if fm[name] == nil {
			t.Errorf("FuncMap() missing %q", name)
		}
	}
}

func TestStandaloneFuncMap(t *testing.T) {
	fm := FuncMap()

	funcs := []string{
		"chip", "label",
		"btn", "btnSubmit", "btnDanger", "button",
		"page", "pageHeader", "container",
		"navGrid", "nav",
		"table", "col", "row", "text", "html",
		"settingsForm",
		"form", "deleteButton",
	}

	for _, name := range funcs {
		if fm[name] == nil {
			t.Errorf("FuncMap() missing %q", name)
		}
	}
}

func TestMergeFuncMaps(t *testing.T) {
	fm1 := map[string]any{"a": func() {}}
	fm2 := map[string]any{"b": func() {}}

	merged := MergeFuncMaps(fm1, fm2)

	if merged["a"] == nil {
		t.Error("MergeFuncMaps() missing 'a'")
	}

	if merged["b"] == nil {
		t.Error("MergeFuncMaps() missing 'b'")
	}
}

func TestHX(t *testing.T) {
	attrs := HX()
	if attrs == nil {
		t.Error("HX() returned nil")
	}
}
