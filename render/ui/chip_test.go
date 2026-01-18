package ui

import (
	"html/template"
	"testing"
)

func TestChip(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  template.HTML
	}{
		{
			name:  "simple label",
			label: "Apartment",
			want:  `<span class="chip">Apartment</span>`,
		},
		{
			name:  "empty label",
			label: "",
			want:  `<span class="chip"></span>`,
		},
		{
			name:  "escapes html",
			label: "<script>alert('xss')</script>",
			want:  `<span class="chip">&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Chip(tt.label)
			if got != tt.want {
				t.Errorf("Chip(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestChipMuted(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  template.HTML
	}{
		{
			name:  "simple label",
			label: "Used",
			want:  `<span class="chip chip--muted">Used</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChipMuted(tt.label)
			if got != tt.want {
				t.Errorf("ChipMuted(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestChipWithClass(t *testing.T) {
	tests := []struct {
		name  string
		label string
		class string
		want  template.HTML
	}{
		{
			name:  "with custom class",
			label: "Sale",
			class: "chip--primary",
			want:  `<span class="chip chip--primary">Sale</span>`,
		},
		{
			name:  "empty class falls back to chip",
			label: "Rent",
			class: "",
			want:  `<span class="chip">Rent</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChipWithClass(tt.label, tt.class)
			if got != tt.want {
				t.Errorf("ChipWithClass(%q, %q) = %q, want %q", tt.label, tt.class, got, tt.want)
			}
		})
	}
}

func TestPill(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  template.HTML
	}{
		{
			name:  "simple label",
			label: "Sale",
			want:  `<span class="pill">Sale</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Pill(tt.label)
			if got != tt.want {
				t.Errorf("Pill(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestPillMuted(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  template.HTML
	}{
		{
			name:  "simple label",
			label: "Draft",
			want:  `<span class="pill pill--muted">Draft</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PillMuted(tt.label)
			if got != tt.want {
				t.Errorf("PillMuted(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestPillWithClass(t *testing.T) {
	tests := []struct {
		name  string
		label string
		class string
		want  template.HTML
	}{
		{
			name:  "with custom class",
			label: "Active",
			class: "pill--success",
			want:  `<span class="pill pill--success">Active</span>`,
		},
		{
			name:  "empty class falls back to pill",
			label: "Pending",
			class: "",
			want:  `<span class="pill">Pending</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PillWithClass(tt.label, tt.class)
			if got != tt.want {
				t.Errorf("PillWithClass(%q, %q) = %q, want %q", tt.label, tt.class, got, tt.want)
			}
		})
	}
}
