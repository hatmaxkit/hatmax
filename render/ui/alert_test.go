package ui

import (
	"strings"
	"testing"
)

func TestAlert(t *testing.T) {
	t.Run("basic alert", func(t *testing.T) {
		alert := NewAlert("Test message")
		html := string(alert.Render())

		if !strings.Contains(html, "Test message") {
			t.Error("alert should contain message")
		}
		if !strings.Contains(html, `class="alert alert--info"`) {
			t.Error("alert should have default info class")
		}
		if !strings.Contains(html, `role="alert"`) {
			t.Error("alert should have alert role")
		}
	})

	t.Run("alert variants", func(t *testing.T) {
		tests := []struct {
			name    string
			alert   *Alert
			variant string
		}{
			{"info", NewAlert("Test").Info(), "alert--info"},
			{"success", NewAlert("Test").Success(), "alert--success"},
			{"warning", NewAlert("Test").Warning(), "alert--warning"},
			{"danger", NewAlert("Test").Danger(), "alert--danger"},
			{"error", NewAlert("Test").Error(), "alert--danger"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				html := string(tt.alert.Render())
				if !strings.Contains(html, tt.variant) {
					t.Errorf("expected %s class", tt.variant)
				}
			})
		}
	})

	t.Run("alert with title", func(t *testing.T) {
		alert := NewAlert("Message").Title("Warning!")
		html := string(alert.Render())

		if !strings.Contains(html, "alert__title") {
			t.Error("alert should have title element")
		}
		if !strings.Contains(html, "Warning!") {
			t.Error("alert should show title")
		}
	})

	t.Run("dismissible alert", func(t *testing.T) {
		alert := NewAlert("Test").Dismissible()
		html := string(alert.Render())

		if !strings.Contains(html, "alert--dismissible") {
			t.Error("alert should have dismissible class")
		}
		if !strings.Contains(html, "alert__dismiss") {
			t.Error("alert should have dismiss button")
		}
	})

	t.Run("alert with icon", func(t *testing.T) {
		alert := NewAlert("Test").Icon("⚠")
		html := string(alert.Render())

		if !strings.Contains(html, "alert__icon") {
			t.Error("alert should have icon element")
		}
		if !strings.Contains(html, "⚠") {
			t.Error("alert should show icon")
		}
	})

	t.Run("alert with ID", func(t *testing.T) {
		alert := NewAlert("Test").ID("notification")
		html := string(alert.Render())

		if !strings.Contains(html, `id="notification"`) {
			t.Error("alert should have id")
		}
	})

	t.Run("shorthand constructors", func(t *testing.T) {
		info := AlertInfoMsg("Info")
		if !strings.Contains(string(info.Render()), "alert--info") {
			t.Error("AlertInfoMsg should create info alert")
		}

		success := AlertSuccessMsg("Success")
		if !strings.Contains(string(success.Render()), "alert--success") {
			t.Error("AlertSuccessMsg should create success alert")
		}

		warning := AlertWarningMsg("Warning")
		if !strings.Contains(string(warning.Render()), "alert--warning") {
			t.Error("AlertWarningMsg should create warning alert")
		}

		danger := AlertDangerMsg("Danger")
		if !strings.Contains(string(danger.Render()), "alert--danger") {
			t.Error("AlertDangerMsg should create danger alert")
		}

		err := AlertErrorMsg("Error")
		if !strings.Contains(string(err.Render()), "alert--danger") {
			t.Error("AlertErrorMsg should create danger alert")
		}
	})
}

func TestAlertDismissible(t *testing.T) {
	t.Run("with dismiss URL", func(t *testing.T) {
		alert := NewAlertDismissible("Test message").
			DismissURL("/dismiss/1")
		html := string(alert.Render())

		if !strings.Contains(html, `hx-delete="/dismiss/1"`) {
			t.Error("dismissible alert should have hx-delete")
		}
		if !strings.Contains(html, `hx-swap="outerHTML"`) {
			t.Error("dismissible alert should have hx-swap")
		}
	})

	t.Run("with ID targets itself", func(t *testing.T) {
		alert := NewAlertDismissible("Test").
			ID("alert-1")
		alert.DismissURL("/dismiss/1")
		html := string(alert.Render())

		if !strings.Contains(html, `hx-target="#alert-1"`) {
			t.Error("dismissible alert should target itself by ID")
		}
	})
}

func TestFlash(t *testing.T) {
	t.Run("basic flash", func(t *testing.T) {
		flash := NewFlash("Success!")
		html := string(flash.Render())

		if !strings.Contains(html, "flash") {
			t.Error("flash should have flash class")
		}
		if !strings.Contains(html, "Success!") {
			t.Error("flash should show message")
		}
	})

	t.Run("flash variants", func(t *testing.T) {
		success := NewFlash("Test").Success()
		html := string(success.Render())

		if !strings.Contains(html, "flash--success") {
			t.Error("flash should have variant class")
		}
	})

	t.Run("flash auto dismiss", func(t *testing.T) {
		flash := NewFlash("Test").AutoDismiss(5)
		html := string(flash.Render())

		if !strings.Contains(html, "hx-trigger") {
			t.Error("auto dismiss flash should have hx-trigger")
		}
		if !strings.Contains(html, "delay:5s") {
			t.Error("auto dismiss should have delay")
		}
	})

	t.Run("flash with icon", func(t *testing.T) {
		flash := NewFlash("Test").Icon("✓")
		html := string(flash.Render())

		if !strings.Contains(html, "flash__icon") {
			t.Error("flash should have icon element")
		}
		if !strings.Contains(html, "✓") {
			t.Error("flash should show icon")
		}
	})
}

func TestToast(t *testing.T) {
	t.Run("basic toast", func(t *testing.T) {
		toast := NewToast("Notification")
		html := string(toast.Render())

		if !strings.Contains(html, "toast") {
			t.Error("toast should have toast class")
		}
		if !strings.Contains(html, "Notification") {
			t.Error("toast should show message")
		}
		if !strings.Contains(html, "toast--top-right") {
			t.Error("toast should have default position")
		}
	})

	t.Run("toast variants", func(t *testing.T) {
		success := NewToast("Test").Success()
		html := string(success.Render())

		if !strings.Contains(html, "toast--success") {
			t.Error("toast should have variant class")
		}
	})

	t.Run("toast positions", func(t *testing.T) {
		positions := []string{"top-left", "bottom-right", "bottom-left"}

		for _, pos := range positions {
			toast := NewToast("Test").Position(pos)
			html := string(toast.Render())

			if !strings.Contains(html, "toast--"+pos) {
				t.Errorf("toast should have position class %s", pos)
			}
		}
	})

	t.Run("toast duration", func(t *testing.T) {
		toast := NewToast("Test").Duration(3000)
		html := string(toast.Render())

		if !strings.Contains(html, `data-duration="3000"`) {
			t.Error("toast should have duration data attribute")
		}
	})

	t.Run("persistent toast", func(t *testing.T) {
		toast := NewToast("Test").Persistent()
		html := string(toast.Render())

		if strings.Contains(html, "data-duration") {
			t.Error("persistent toast should not have duration")
		}
	})

	t.Run("toast with title", func(t *testing.T) {
		toast := NewToast("Message").Title("Alert")
		html := string(toast.Render())

		if !strings.Contains(html, "toast__title") {
			t.Error("toast should have title element")
		}
		if !strings.Contains(html, "Alert") {
			t.Error("toast should show title")
		}
	})
}
