package hatmaxtui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestModelResizesEveryInteractiveSurface(t *testing.T) {
	initial := newModel()

	updated, _ := initial.Update(tea.WindowSizeMsg{Width: 112, Height: 38})
	value := updated.(model)

	if value.width != 112 || value.height != 38 {
		t.Fatalf("size = %dx%d, want 112x38", value.width, value.height)
	}

	if value.viewport.Width() != 112 || value.viewport.Height() != 29 {
		t.Fatalf("viewport = %dx%d, want 112x29", value.viewport.Width(), value.viewport.Height())
	}
}

func TestModelExposesAccessibleControlsAndRestorableView(t *testing.T) {
	value := newModel()
	view := value.View()

	if !view.AltScreen || view.WindowTitle != "Hatmax" {
		t.Fatalf("view terminal contract = alt %t title %q", view.AltScreen, view.WindowTitle)
	}

	for _, expected := range []string{"Hatmax", "Status: Ready", "enter", "send", "ctrl+c", "quit"} {
		if !strings.Contains(view.Content, expected) {
			t.Errorf("view does not contain %q:\n%s", expected, view.Content)
		}
	}
}

func TestEscapeClearsPendingInput(t *testing.T) {
	value := newModel()
	value.composer.SetValue("Create invoices")

	updated, _ := value.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := updated.(model)

	if result.composer.Value() != "" || result.status != "Cancelled" {
		t.Fatalf("escape left input %q status %q", result.composer.Value(), result.status)
	}
}
