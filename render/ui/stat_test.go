package ui

import (
	"html/template"
	"testing"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    any
		want string
	}{
		{
			name: "integer",
			n:    1234567,
			want: "1,234,567",
		},
		{
			name: "float",
			n:    1234.56,
			want: "1,234.56",
		},
		{
			name: "small number",
			n:    42,
			want: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatNumber(tt.n)
			if got != tt.want {
				t.Errorf("FormatNumber(%v) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestStat(t *testing.T) {
	tests := []struct {
		name  string
		value any
		label string
		want  template.HTML
	}{
		{
			name:  "views stat",
			value: 12500,
			label: "Views",
			want:  `<div class="stat"><span class="stat__value">12,500</span><span class="stat__label">Views</span></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Stat(tt.value, tt.label)
			if got != tt.want {
				t.Errorf("Stat(%v, %q) = %q, want %q", tt.value, tt.label, got, tt.want)
			}
		})
	}
}

func TestStatWithIcon(t *testing.T) {
	tests := []struct {
		name  string
		value any
		label string
		icon  string
		want  template.HTML
	}{
		{
			name:  "stat with eye icon",
			value: 5000,
			label: "Views",
			icon:  "👁",
			want:  `<div class="stat"><span class="stat__icon">👁</span><span class="stat__value">5,000</span><span class="stat__label">Views</span></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatWithIcon(tt.value, tt.label, tt.icon)
			if got != tt.want {
				t.Errorf("StatWithIcon(%v, %q, %q) = %q, want %q", tt.value, tt.label, tt.icon, got, tt.want)
			}
		})
	}
}

func TestStatCompact(t *testing.T) {
	tests := []struct {
		name  string
		value any
		label string
		want  template.HTML
	}{
		{
			name:  "compact stat",
			value: 42,
			label: "items",
			want:  `<span class="stat-compact"><strong>42</strong> items</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatCompact(tt.value, tt.label)
			if got != tt.want {
				t.Errorf("StatCompact(%v, %q) = %q, want %q", tt.value, tt.label, got, tt.want)
			}
		})
	}
}
