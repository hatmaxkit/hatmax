package ui

import (
	"strings"
	"testing"
)

func TestChipRender(t *testing.T) {
	tests := []struct {
		name     string
		chip     *Chip
		contains []string
	}{
		{
			name:     "basic chip",
			chip:     NewChip("Active"),
			contains: []string{`class="chip"`, `<span class="chip__text">Active</span>`},
		},
		{
			name:     "chip with emoji",
			chip:     NewChip("Done").Emoji(EmojiCheck),
			contains: []string{`class="chip__emoji"`, string(EmojiCheck), `Done`},
		},
		{
			name:     "chip with variant",
			chip:     NewChip("Success").Success(),
			contains: []string{`chip--success`},
		},
		{
			name:     "chip with custom class",
			chip:     NewChip("Custom").Class("my-class"),
			contains: []string{`my-class`},
		},
		{
			name:     "chip with custom emoji",
			chip:     NewChip("Coffee").Emoji("☕"),
			contains: []string{`☕`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.chip.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestChipVariants(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Chip) *Chip
		variant Variant
	}{
		{"Primary", (*Chip).Primary, VariantPrimary},
		{"Secondary", (*Chip).Secondary, VariantSecondary},
		{"Success", (*Chip).Success, VariantSuccess},
		{"Warning", (*Chip).Warning, VariantWarning},
		{"Danger", (*Chip).Danger, VariantDanger},
		{"Info", (*Chip).Info, VariantInfo},
		{"Muted", (*Chip).Muted, VariantMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chip := tt.method(NewChip("Test"))
			if chip.variant != tt.variant {
				t.Errorf("%s() variant = %v, want %v", tt.name, chip.variant, tt.variant)
			}
		})
	}
}

func TestLabelRender(t *testing.T) {
	tests := []struct {
		name     string
		label    *Label
		contains []string
	}{
		{
			name:     "basic label",
			label:    NewLabel("Important"),
			contains: []string{`class="label"`, `<span class="label__text">Important</span>`},
		},
		{
			name:     "label with emoji",
			label:    NewLabel("Warning").Emoji(EmojiWarning),
			contains: []string{`class="label__emoji"`, string(EmojiWarning), `Warning`},
		},
		{
			name:     "label with variant",
			label:    NewLabel("Error").Danger(),
			contains: []string{`label--danger`},
		},
		{
			name:     "label with custom class",
			label:    NewLabel("Custom").Class("my-label"),
			contains: []string{`my-label`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.label.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestLabelVariants(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Label) *Label
		variant Variant
	}{
		{"Primary", (*Label).Primary, VariantPrimary},
		{"Secondary", (*Label).Secondary, VariantSecondary},
		{"Success", (*Label).Success, VariantSuccess},
		{"Warning", (*Label).Warning, VariantWarning},
		{"Danger", (*Label).Danger, VariantDanger},
		{"Info", (*Label).Info, VariantInfo},
		{"Muted", (*Label).Muted, VariantMuted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := tt.method(NewLabel("Test"))
			if label.variant != tt.variant {
				t.Errorf("%s() variant = %v, want %v", tt.name, label.variant, tt.variant)
			}
		})
	}
}

func TestChipHTMLEscaping(t *testing.T) {
	chip := NewChip("<script>alert('xss')</script>")
	html := string(chip.Render())

	if strings.Contains(html, "<script>") {
		t.Error("Render() did not escape HTML")
	}

	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("Render() did not properly escape HTML entities")
	}
}

func TestLabelHTMLEscaping(t *testing.T) {
	label := NewLabel("<script>alert('xss')</script>")
	html := string(label.Render())

	if strings.Contains(html, "<script>") {
		t.Error("Render() did not escape HTML")
	}

	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("Render() did not properly escape HTML entities")
	}
}
