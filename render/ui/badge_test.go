package ui

import (
	"html/template"
	"testing"
)

func TestBadge(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  template.HTML
	}{
		{
			name:  "simple label",
			label: "New",
			want:  `<span class="badge">New</span>`,
		},
		{
			name:  "escapes html",
			label: "<b>Bold</b>",
			want:  `<span class="badge">&lt;b&gt;Bold&lt;/b&gt;</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Badge(tt.label)
			if got != tt.want {
				t.Errorf("Badge(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestBadgeWithVariant(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		variant string
		want    template.HTML
	}{
		{
			name:    "with variant",
			label:   "Active",
			variant: "success",
			want:    `<span class="badge badge--success">Active</span>`,
		},
		{
			name:    "empty variant falls back to badge",
			label:   "Pending",
			variant: "",
			want:    `<span class="badge">Pending</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BadgeWithVariant(tt.label, tt.variant)
			if got != tt.want {
				t.Errorf("BadgeWithVariant(%q, %q) = %q, want %q", tt.label, tt.variant, got, tt.want)
			}
		})
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   template.HTML
	}{
		{
			name:   "active status",
			status: "active",
			want:   `<span class="badge badge--success">Active</span>`,
		},
		{
			name:   "draft status",
			status: "draft",
			want:   `<span class="badge badge--warning">Draft</span>`,
		},
		{
			name:   "expired status",
			status: "expired",
			want:   `<span class="badge badge--danger">Expired</span>`,
		},
		{
			name:   "unknown status uses status as both variant and label",
			status: "custom",
			want:   `<span class="badge badge--custom">custom</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusBadge(tt.status)
			if got != tt.want {
				t.Errorf("StatusBadge(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusBadgeWithIcon(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   template.HTML
	}{
		{
			name:   "active status with icon",
			status: "active",
			want:   `<span class="badge badge--success">● Active</span>`,
		},
		{
			name:   "featured status with icon",
			status: "featured",
			want:   `<span class="badge badge--info">★ Featured</span>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusBadgeWithIcon(tt.status)
			if got != tt.want {
				t.Errorf("StatusBadgeWithIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRegisterStatusVariant(t *testing.T) {
	RegisterStatusVariant("custom", "primary")
	if StatusVariant["custom"] != "primary" {
		t.Error("RegisterStatusVariant did not set variant")
	}
	delete(StatusVariant, "custom")
}

func TestRegisterStatusLabel(t *testing.T) {
	RegisterStatusLabel("custom", "Custom Label")
	if StatusLabel["custom"] != "Custom Label" {
		t.Error("RegisterStatusLabel did not set label")
	}
	delete(StatusLabel, "custom")
}

func TestRegisterStatusIcon(t *testing.T) {
	RegisterStatusIcon("custom", "⚡")
	if StatusIcon["custom"] != "⚡" {
		t.Error("RegisterStatusIcon did not set icon")
	}
	delete(StatusIcon, "custom")
}
