package ui

import (
	"strings"
	"testing"

	"github.com/hatmaxkit/hatmax/htmx"
)

func TestButton(t *testing.T) {
	t.Run("basic button", func(t *testing.T) {
		btn := NewButton("Click me")
		html := string(btn.Render())

		if !strings.Contains(html, "Click me") {
			t.Error("button should contain text")
		}
		if !strings.Contains(html, `class="btn btn--primary"`) {
			t.Error("button should have default classes")
		}
		if !strings.Contains(html, `type="button"`) {
			t.Error("button should have default type")
		}
	})

	t.Run("button variants", func(t *testing.T) {
		tests := []struct {
			name    string
			btn     *Button
			variant string
		}{
			{"primary", NewButton("Test").Primary(), "btn--primary"},
			{"secondary", NewButton("Test").Secondary(), "btn--secondary"},
			{"danger", NewButton("Test").Danger(), "btn--danger"},
			{"success", NewButton("Test").Success(), "btn--success"},
			{"warning", NewButton("Test").Warning(), "btn--warning"},
			{"ghost", NewButton("Test").Ghost(), "btn--ghost"},
			{"link", NewButton("Test").LinkStyle(), "btn--link"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				html := string(tt.btn.Render())
				if !strings.Contains(html, tt.variant) {
					t.Errorf("expected %s class", tt.variant)
				}
			})
		}
	})

	t.Run("button sizes", func(t *testing.T) {
		small := NewButton("Test").Small()
		large := NewButton("Test").Large()

		if !strings.Contains(string(small.Render()), "btn--sm") {
			t.Error("small button should have btn--sm class")
		}
		if !strings.Contains(string(large.Render()), "btn--lg") {
			t.Error("large button should have btn--lg class")
		}
	})

	t.Run("disabled button", func(t *testing.T) {
		btn := NewButton("Test").Disabled(true)
		html := string(btn.Render())

		if !strings.Contains(html, "disabled") {
			t.Error("disabled button should have disabled attribute")
		}
	})

	t.Run("loading button", func(t *testing.T) {
		btn := NewButton("Test").Loading(true)
		html := string(btn.Render())

		if !strings.Contains(html, "btn--loading") {
			t.Error("loading button should have loading class")
		}
		if !strings.Contains(html, "btn__spinner") {
			t.Error("loading button should have spinner")
		}
		if !strings.Contains(html, "disabled") {
			t.Error("loading button should be disabled")
		}
	})

	t.Run("submit button", func(t *testing.T) {
		btn := NewButton("Submit").Submit()
		html := string(btn.Render())

		if !strings.Contains(html, `type="submit"`) {
			t.Error("submit button should have type=submit")
		}
	})

	t.Run("button with icon", func(t *testing.T) {
		btn := NewButton("Save").Icon("💾")
		html := string(btn.Render())

		if !strings.Contains(html, "💾") {
			t.Error("button should contain icon")
		}
		if !strings.Contains(html, "btn__icon") {
			t.Error("button should have icon class")
		}
	})

	t.Run("button with HTMX", func(t *testing.T) {
		btn := NewButton("Delete").HX().
			Delete("/items/1").
			TargetID("item-1").
			SwapOuter().
			Confirm("Are you sure?").
			Done()

		html := string(btn.Render())

		if !strings.Contains(html, `hx-delete="/items/1"`) {
			t.Error("button should have hx-delete")
		}
		if !strings.Contains(html, `hx-target="#item-1"`) {
			t.Error("button should have hx-target")
		}
		if !strings.Contains(html, `hx-swap="outerHTML"`) {
			t.Error("button should have hx-swap")
		}
		if !strings.Contains(html, `hx-confirm="Are you sure?"`) {
			t.Error("button should have hx-confirm")
		}
	})

	t.Run("shorthand constructors", func(t *testing.T) {
		btn := Btn("Test")
		if btn == nil {
			t.Error("Btn should return button")
		}

		submit := BtnSubmit("Submit")
		if !strings.Contains(string(submit.Render()), `type="submit"`) {
			t.Error("BtnSubmit should create submit button")
		}

		danger := BtnDanger("Delete")
		if !strings.Contains(string(danger.Render()), "btn--danger") {
			t.Error("BtnDanger should create danger button")
		}
	})
}

func TestButtonHX(t *testing.T) {
	t.Run("fluent API", func(t *testing.T) {
		html := NewButton("Test").HX().
			Post("/api/action").
			TargetID("result").
			SwapInner().
			WithVal("id", "123").
			Render()

		s := string(html)
		if !strings.Contains(s, `hx-post="/api/action"`) {
			t.Error("should have hx-post")
		}
		if !strings.Contains(s, `hx-target="#result"`) {
			t.Error("should have hx-target")
		}
		if !strings.Contains(s, `hx-swap="innerHTML"`) {
			t.Error("should have hx-swap")
		}
		if !strings.Contains(s, `hx-vals`) {
			t.Error("should have hx-vals")
		}
	})

	t.Run("with attrs directly", func(t *testing.T) {
		attrs := htmx.HX().Post("/test").TargetID("target")
		btn := NewButton("Test").WithAttrs(attrs)
		html := string(btn.Render())

		if !strings.Contains(html, `hx-post="/test"`) {
			t.Error("should have hx-post from attrs")
		}
	})
}
