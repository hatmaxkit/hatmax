package ui

import (
	"strings"
	"testing"
)

func TestButtonRender(t *testing.T) {
	tests := []struct {
		name     string
		button   *Button
		contains []string
	}{
		{
			name:     "basic button",
			button:   NewButton("Click"),
			contains: []string{`class="btn"`, `type="button"`, `Click`},
		},
		{
			name:     "button with emoji",
			button:   NewButton("Save").Emoji(EmojiSave),
			contains: []string{`class="btn__emoji"`, string(EmojiSave)},
		},
		{
			name:     "button with variant",
			button:   NewButton("Delete").Danger(),
			contains: []string{`btn--danger`},
		},
		{
			name:     "button small",
			button:   NewButton("Small").Small(),
			contains: []string{`btn--sm`},
		},
		{
			name:     "button large",
			button:   NewButton("Large").Large(),
			contains: []string{`btn--lg`},
		},
		{
			name:     "submit button",
			button:   NewButton("Submit").Submit(),
			contains: []string{`type="submit"`},
		},
		{
			name:     "disabled button",
			button:   NewButton("Disabled").Disabled(true),
			contains: []string{`disabled`},
		},
		{
			name:     "loading button",
			button:   NewButton("Loading").Loading(true),
			contains: []string{`btn--loading`, `btn__spinner`, `disabled`},
		},
		{
			name:     "button with id",
			button:   NewButton("Test").ID("my-btn"),
			contains: []string{`id="my-btn"`},
		},
		{
			name:     "button with name and value",
			button:   NewButton("Test").Name("action").Value("submit"),
			contains: []string{`name="action"`, `value="submit"`},
		},
		{
			name:     "button with class",
			button:   NewButton("Test").Class("custom"),
			contains: []string{`custom`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.button.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestButtonVariants(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Button) *Button
		variant Variant
	}{
		{"Primary", (*Button).Primary, VariantPrimary},
		{"Secondary", (*Button).Secondary, VariantSecondary},
		{"Success", (*Button).Success, VariantSuccess},
		{"Warning", (*Button).Warning, VariantWarning},
		{"Danger", (*Button).Danger, VariantDanger},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			btn := tt.method(NewButton("Test"))
			if btn.variant != tt.variant {
				t.Errorf("%s() variant = %v, want %v", tt.name, btn.variant, tt.variant)
			}
		})
	}
}

func TestButtonHX(t *testing.T) {
	tests := []struct {
		name     string
		button   *Button
		contains []string
	}{
		{
			name:     "hx get",
			button:   NewButton("Load").HX().Get("/data").Done(),
			contains: []string{`hx-get="/data"`},
		},
		{
			name:     "hx post with target",
			button:   NewButton("Save").HX().Post("/save").TargetID("result").Done(),
			contains: []string{`hx-post="/save"`, `hx-target="#result"`},
		},
		{
			name:     "hx delete with confirm",
			button:   NewButton("Delete").HX().Delete("/item/1").Confirm("Sure?").Done(),
			contains: []string{`hx-delete="/item/1"`, `hx-confirm="Sure?"`},
		},
		{
			name:     "hx with swap",
			button:   NewButton("Update").HX().Put("/update").SwapOuter().Done(),
			contains: []string{`hx-put="/update"`, `hx-swap="outerHTML"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.button.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestButtonHXRenderDirectly(t *testing.T) {
	html := string(NewButton("Test").HX().Get("/test").Render())
	if !strings.Contains(html, `hx-get="/test"`) {
		t.Errorf("HX().Render() missing hx-get")
	}
}

func TestButtonShorthands(t *testing.T) {
	btn := Btn("Test")
	if btn.text != "Test" {
		t.Error("Btn() text not set")
	}

	submit := BtnSubmit("Submit")
	if submit.type_ != "submit" {
		t.Error("BtnSubmit() type not submit")
	}

	danger := BtnDanger("Delete")
	if danger.variant != VariantDanger {
		t.Error("BtnDanger() variant not danger")
	}
}

func TestButtonHTMLEscaping(t *testing.T) {
	btn := NewButton("<script>").ID("<id>").Name("<name>").Value("<value>")
	html := string(btn.Render())

	if strings.Contains(html, "<script>") {
		t.Error("Render() did not escape text")
	}

	if strings.Contains(html, `id="<id>"`) {
		t.Error("Render() did not escape id")
	}
}

func TestButtonSwapInnerAndDelete(t *testing.T) {
	inner := NewButton("Test").HX().Post("/test").SwapInner().Done()

	html := string(inner.Render())
	if !strings.Contains(html, `hx-swap="innerHTML"`) {
		t.Errorf("SwapInner() missing innerHTML in %q", html)
	}

	del := NewButton("Test").HX().Delete("/test").SwapDelete().Done()

	html = string(del.Render())
	if !strings.Contains(html, `hx-swap="delete"`) {
		t.Errorf("SwapDelete() missing delete in %q", html)
	}
}
