package ui

import (
	"strings"
	"testing"
)

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantText     string
		wantVariant  string
	}{
		{"active", "active", "Active", "success"},
		{"draft", "draft", "Draft", "warning"},
		{"inactive", "inactive", "Inactive", "muted"},
		{"expired", "expired", "Expired", "danger"},
		{"featured", "featured", "Featured", "info"},
		{"unknown uses raw status", "custom", "custom", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StatusBadge(tt.status).Render())

			if !strings.Contains(got, tt.wantText) {
				t.Errorf("StatusBadge(%q) = %q, want text %q", tt.status, got, tt.wantText)
			}

			if tt.wantVariant != "" {
				wantClass := "label--" + tt.wantVariant
				if !strings.Contains(got, wantClass) {
					t.Errorf("StatusBadge(%q) = %q, want class %q", tt.status, got, wantClass)
				}
			}
		})
	}
}

func TestStatusBadgeWithIcon(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantIcon string
		wantText string
	}{
		{"active has dot", "active", "●", "Active"},
		{"draft has circle", "draft", "○", "Draft"},
		{"expired has x", "expired", "✕", "Expired"},
		{"unknown has no icon", "custom", "", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StatusBadgeWithIcon(tt.status).Render())

			if !strings.Contains(got, tt.wantText) {
				t.Errorf("StatusBadgeWithIcon(%q) = %q, want text %q", tt.status, got, tt.wantText)
			}

			if tt.wantIcon != "" && !strings.Contains(got, tt.wantIcon) {
				t.Errorf("StatusBadgeWithIcon(%q) = %q, want icon %q", tt.status, got, tt.wantIcon)
			}
		})
	}
}

func TestRegisterStatus(t *testing.T) {
	RegisterStatus("pending", StatusConfig{
		Variant: VariantWarning,
		Label:   "Pending Review",
		Icon:    "⏳",
	})

	got := string(StatusBadge("pending").Render())

	if !strings.Contains(got, "Pending Review") {
		t.Errorf("StatusBadge after RegisterStatus = %q, want text 'Pending Review'", got)
	}

	if !strings.Contains(got, "label--warning") {
		t.Errorf("StatusBadge after RegisterStatus = %q, want class 'label--warning'", got)
	}

	gotWithIcon := string(StatusBadgeWithIcon("pending").Render())
	if !strings.Contains(gotWithIcon, "⏳") {
		t.Errorf("StatusBadgeWithIcon after RegisterStatus = %q, want icon '⏳'", gotWithIcon)
	}
}
