package modal

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("test-modal", "Test Title")

	if cfg.ID != "test-modal" {
		t.Errorf("ID = %q, want %q", cfg.ID, "test-modal")
	}
	if cfg.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", cfg.Title, "Test Title")
	}
	if cfg.Size != SizeMedium {
		t.Errorf("Size = %q, want %q", cfg.Size, SizeMedium)
	}
	if !cfg.CloseOnEsc {
		t.Error("CloseOnEsc should be true by default")
	}
	if !cfg.CloseOnClick {
		t.Error("CloseOnClick should be true by default")
	}
}

func TestSizeConstants(t *testing.T) {
	tests := []struct {
		name string
		size Size
		want string
	}{
		{"small", SizeSmall, "modal-sm"},
		{"medium", SizeMedium, "modal-md"},
		{"large", SizeLarge, "modal-lg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.size) != tt.want {
				t.Errorf("Size = %q, want %q", tt.size, tt.want)
			}
		})
	}
}
