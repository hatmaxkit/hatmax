package book

import "testing"

func TestLoadDefault(t *testing.T) {
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	supported, err := loaded.SupportsHatmax("v0.4.0")
	if err != nil {
		t.Fatalf("SupportsHatmax() error = %v", err)
	}

	if !supported {
		t.Fatal("SupportsHatmax() = false, want true")
	}

	selection, err := loaded.Select(
		"server_rendered_crud",
		[]string{"postgres_persistence", "htmx_form", "runtime_validation"},
	)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}

	if len(selection.Capabilities) != 3 {
		t.Fatalf("len(Selection.Capabilities) = %d, want 3", len(selection.Capabilities))
	}

	if len(selection.Rules) != 9 {
		t.Fatalf("len(Selection.Rules) = %d, want 9", len(selection.Rules))
	}
}
