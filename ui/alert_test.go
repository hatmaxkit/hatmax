package ui

import (
	"strings"
	"testing"
)

func TestAlert(t *testing.T) {
	tests := []struct {
		name       string
		alert      *Alert
		wantClass  string
		wantText   string
		wantTitle  string
		wantDismis bool
	}{
		{
			name:      "basic info alert",
			alert:     NewAlert("Test message"),
			wantClass: "alert--info",
			wantText:  "Test message",
		},
		{
			name:      "success alert",
			alert:     NewAlert("Success!").Success(),
			wantClass: "alert--success",
			wantText:  "Success!",
		},
		{
			name:      "warning alert",
			alert:     NewAlert("Warning!").Warning(),
			wantClass: "alert--warning",
		},
		{
			name:      "danger alert",
			alert:     NewAlert("Danger!").Danger(),
			wantClass: "alert--danger",
		},
		{
			name:      "error alias",
			alert:     NewAlert("Error!").Error(),
			wantClass: "alert--danger",
		},
		{
			name:       "dismissible",
			alert:      NewAlert("Dismiss me").Dismissible(),
			wantClass:  "alert--dismissible",
			wantDismis: true,
		},
		{
			name:      "with title",
			alert:     NewAlert("Body").Title("Header"),
			wantTitle: "Header",
		},
		{
			name:      "with icon",
			alert:     NewAlert("Info").Icon("ℹ️"),
			wantText:  "ℹ️",
		},
		{
			name:      "with id",
			alert:     NewAlert("Test").ID("alert-1"),
			wantText:  `id="alert-1"`,
		},
		{
			name:      "with class",
			alert:     NewAlert("Test").Class("custom-class"),
			wantClass: "custom-class",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.alert.Render())

			if tt.wantClass != "" && !strings.Contains(got, tt.wantClass) {
				t.Errorf("Alert.Render() = %q, want class %q", got, tt.wantClass)
			}

			if tt.wantText != "" && !strings.Contains(got, tt.wantText) {
				t.Errorf("Alert.Render() = %q, want text %q", got, tt.wantText)
			}

			if tt.wantTitle != "" && !strings.Contains(got, tt.wantTitle) {
				t.Errorf("Alert.Render() = %q, want title %q", got, tt.wantTitle)
			}

			hasDismiss := strings.Contains(got, "alert__dismiss")
			if tt.wantDismis != hasDismiss {
				t.Errorf("Alert.Render() dismiss button = %v, want %v", hasDismiss, tt.wantDismis)
			}

			if !strings.Contains(got, `role="alert"`) {
				t.Errorf("Alert.Render() missing role=alert")
			}
		})
	}
}

func TestAlertShorthands(t *testing.T) {
	tests := []struct {
		name      string
		alert     *Alert
		wantClass string
	}{
		{"AlertInfo", AlertInfo("msg"), "alert--info"},
		{"AlertSuccess", AlertSuccess("msg"), "alert--success"},
		{"AlertWarning", AlertWarning("msg"), "alert--warning"},
		{"AlertDanger", AlertDanger("msg"), "alert--danger"},
		{"AlertError", AlertError("msg"), "alert--danger"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.alert.Render())
			if !strings.Contains(got, tt.wantClass) {
				t.Errorf("%s() = %q, want class %q", tt.name, got, tt.wantClass)
			}
		})
	}
}

func TestFlash(t *testing.T) {
	tests := []struct {
		name        string
		flash       *Flash
		wantClass   string
		wantText    string
		wantTrigger bool
	}{
		{
			name:      "basic flash",
			flash:     NewFlash("Saved!"),
			wantClass: "flash--info",
			wantText:  "Saved!",
		},
		{
			name:      "success flash",
			flash:     NewFlash("Done!").Success(),
			wantClass: "flash--success",
		},
		{
			name:        "auto dismiss",
			flash:       NewFlash("Bye!").AutoDismiss(5),
			wantTrigger: true,
		},
		{
			name:      "with icon",
			flash:     NewFlash("Info").Icon("✓"),
			wantText:  "✓",
		},
		{
			name:      "with id",
			flash:     NewFlash("Test").ID("flash-1"),
			wantText:  `id="flash-1"`,
		},
		{
			name:      "warning variant",
			flash:     NewFlash("Warn").Warning(),
			wantClass: "flash--warning",
		},
		{
			name:      "danger variant",
			flash:     NewFlash("Bad").Danger(),
			wantClass: "flash--danger",
		},
		{
			name:      "error alias",
			flash:     NewFlash("Err").Error(),
			wantClass: "flash--danger",
		},
		{
			name:      "with class",
			flash:     NewFlash("Test").Class("my-flash"),
			wantClass: "my-flash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.flash.Render())

			if tt.wantClass != "" && !strings.Contains(got, tt.wantClass) {
				t.Errorf("Flash.Render() = %q, want class %q", got, tt.wantClass)
			}

			if tt.wantText != "" && !strings.Contains(got, tt.wantText) {
				t.Errorf("Flash.Render() = %q, want text %q", got, tt.wantText)
			}

			hasTrigger := strings.Contains(got, "hx-trigger")
			if tt.wantTrigger != hasTrigger {
				t.Errorf("Flash.Render() hx-trigger = %v, want %v", hasTrigger, tt.wantTrigger)
			}
		})
	}
}

func TestToast(t *testing.T) {
	tests := []struct {
		name       string
		toast      *Toast
		wantClass  string
		wantText   string
		noDuration bool
	}{
		{
			name:      "basic toast",
			toast:     NewToast("Hello"),
			wantClass: "toast--top-right",
			wantText:  "Hello",
		},
		{
			name:      "success toast",
			toast:     NewToast("Done!").Success(),
			wantClass: "toast--success",
		},
		{
			name:     "with title",
			toast:    NewToast("Body").Title("Header"),
			wantText: "Header",
		},
		{
			name:      "custom position",
			toast:     NewToast("Test").Position("bottom-left"),
			wantClass: "toast--bottom-left",
		},
		{
			name:       "persistent",
			toast:      NewToast("Stay").Persistent(),
			noDuration: true,
		},
		{
			name:     "custom duration",
			toast:    NewToast("Quick").Duration(3000),
			wantText: `data-duration="3000"`,
		},
		{
			name:     "with icon",
			toast:    NewToast("Info").Icon("🔔"),
			wantText: "🔔",
		},
		{
			name:     "with id",
			toast:    NewToast("Test").ID("toast-1"),
			wantText: `id="toast-1"`,
		},
		{
			name:      "warning variant",
			toast:     NewToast("Warn").Warning(),
			wantClass: "toast--warning",
		},
		{
			name:      "danger variant",
			toast:     NewToast("Bad").Danger(),
			wantClass: "toast--danger",
		},
		{
			name:      "error alias",
			toast:     NewToast("Err").Error(),
			wantClass: "toast--danger",
		},
		{
			name:      "with class",
			toast:     NewToast("Test").Class("my-toast"),
			wantClass: "my-toast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.toast.Render())

			if tt.wantClass != "" && !strings.Contains(got, tt.wantClass) {
				t.Errorf("Toast.Render() = %q, want class %q", got, tt.wantClass)
			}

			if tt.wantText != "" && !strings.Contains(got, tt.wantText) {
				t.Errorf("Toast.Render() = %q, want text %q", got, tt.wantText)
			}

			hasDuration := strings.Contains(got, "data-duration")
			if tt.noDuration && hasDuration {
				t.Errorf("Toast.Render() has data-duration but should not")
			}
		})
	}
}
