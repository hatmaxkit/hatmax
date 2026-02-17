package ui

import (
	"strings"
	"testing"
)

func TestFormRender(t *testing.T) {
	tests := []struct {
		name     string
		form     *Form
		contains []string
	}{
		{
			name:     "basic form",
			form:     NewForm().Action("/submit"),
			contains: []string{`<form`, `action="/submit"`, `method="post"`},
		},
		{
			name:     "form with csrf",
			form:     NewForm().Action("/submit").CSRFToken("abc123"),
			contains: []string{`name="_token"`, `value="abc123"`, `type="hidden"`},
		},
		{
			name:     "form with custom csrf field",
			form:     NewForm().CSRFToken("xyz").CSRFField("csrf_token"),
			contains: []string{`name="csrf_token"`, `value="xyz"`},
		},
		{
			name:     "form get method",
			form:     NewForm().Get(),
			contains: []string{`method="get"`},
		},
		{
			name:     "form with class",
			form:     NewForm().Class("my-form"),
			contains: []string{`class="my-form"`},
		},
		{
			name:     "form with id",
			form:     NewForm().ID("login-form"),
			contains: []string{`id="login-form"`},
		},
		{
			name:     "form multipart",
			form:     NewForm().Multipart(),
			contains: []string{`enctype="multipart/form-data"`},
		},
		{
			name:     "form novalidate",
			form:     NewForm().NoValidate(),
			contains: []string{`novalidate`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.form.Open())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Open() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestFormClose(t *testing.T) {
	form := NewForm()
	if string(form.Close()) != "</form>" {
		t.Errorf("Close() = %q, want </form>", form.Close())
	}
}

func TestFormRenderComplete(t *testing.T) {
	form := NewForm().Action("/test")
	html := string(form.Render())

	if !strings.Contains(html, "<form") {
		t.Error("Render() missing opening tag")
	}
	if !strings.Contains(html, "</form>") {
		t.Error("Render() missing closing tag")
	}
}

func TestFormHX(t *testing.T) {
	tests := []struct {
		name     string
		form     *Form
		contains []string
	}{
		{
			name:     "hx post",
			form:     NewForm().HX().Post("/api/submit").Done(),
			contains: []string{`hx-post="/api/submit"`},
		},
		{
			name:     "hx put with target",
			form:     NewForm().HX().Put("/api/update").TargetID("result").Done(),
			contains: []string{`hx-put="/api/update"`, `hx-target="#result"`},
		},
		{
			name:     "hx delete with swap",
			form:     NewForm().HX().Delete("/api/item").SwapDelete().Done(),
			contains: []string{`hx-delete="/api/item"`, `hx-swap="delete"`},
		},
		{
			name:     "hx with confirm",
			form:     NewForm().HX().Post("/danger").Confirm("Sure?").Done(),
			contains: []string{`hx-confirm="Sure?"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.form.Open())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Open() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestFormHXSwapInnerAndOuter(t *testing.T) {
	inner := NewForm().HX().Post("/test").SwapInner().Done()
	html := string(inner.Open())
	if !strings.Contains(html, `hx-swap="innerHTML"`) {
		t.Errorf("SwapInner() missing innerHTML")
	}

	outer := NewForm().HX().Post("/test").SwapOuter().Done()
	html = string(outer.Open())
	if !strings.Contains(html, `hx-swap="outerHTML"`) {
		t.Errorf("SwapOuter() missing outerHTML")
	}
}

func TestDeleteButtonRender(t *testing.T) {
	tests := []struct {
		name     string
		btn      *DeleteButton
		contains []string
	}{
		{
			name:     "basic delete button",
			btn:      NewDeleteButton("Delete", "/items/1"),
			contains: []string{`<form`, `action="/items/1"`, `_method`, `DELETE`, `btn--danger`},
		},
		{
			name:     "delete with csrf",
			btn:      NewDeleteButton("Delete", "/items/1").CSRFToken("token123"),
			contains: []string{`name="_token"`, `value="token123"`},
		},
		{
			name:     "delete with confirm",
			btn:      NewDeleteButton("Delete", "/items/1").Confirm("Are you sure?"),
			contains: []string{`onclick="return confirm`, `Are you sure?`},
		},
		{
			name:     "delete with emoji",
			btn:      NewDeleteButton("Delete", "/items/1").Emoji(EmojiTrash),
			contains: []string{`btn__emoji`, string(EmojiTrash)},
		},
		{
			name:     "delete with custom class",
			btn:      NewDeleteButton("Delete", "/items/1").Class("my-delete"),
			contains: []string{`my-delete`},
		},
		{
			name:     "delete with custom variant",
			btn:      NewDeleteButton("Remove", "/items/1").Variant(VariantWarning),
			contains: []string{`btn--warning`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.btn.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestDeleteButtonCSRFField(t *testing.T) {
	btn := NewDeleteButton("Delete", "/items/1").CSRFToken("xyz").CSRFField("authenticity_token")
	html := string(btn.Render())

	if !strings.Contains(html, `name="authenticity_token"`) {
		t.Error("CSRFField() not applied")
	}
}

func TestDeleteButtonHX(t *testing.T) {
	tests := []struct {
		name     string
		btn      *DeleteButton
		contains []string
	}{
		{
			name:     "hx delete",
			btn:      NewDeleteButton("Delete", "").HX().Delete("/api/items/1").Done(),
			contains: []string{`hx-delete="/api/items/1"`},
		},
		{
			name:     "hx delete with target",
			btn:      NewDeleteButton("Delete", "").HX().Delete("/api/items/1").TargetID("item-1").Done(),
			contains: []string{`hx-target="#item-1"`},
		},
		{
			name:     "hx delete with swap delete",
			btn:      NewDeleteButton("Delete", "").HX().Delete("/api/items/1").SwapDelete().Done(),
			contains: []string{`hx-swap="delete"`},
		},
		{
			name:     "hx delete with confirm",
			btn:      NewDeleteButton("Delete", "").HX().Delete("/api/items/1").Confirm("Sure?").Done(),
			contains: []string{`hx-confirm="Sure?"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.btn.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestDeleteButtonHXSwapOuter(t *testing.T) {
	btn := NewDeleteButton("Delete", "").HX().Delete("/test").SwapOuter().Done()
	html := string(btn.Render())
	if !strings.Contains(html, `hx-swap="outerHTML"`) {
		t.Errorf("SwapOuter() missing outerHTML")
	}
}

func TestDeleteButtonHXRenderDirectly(t *testing.T) {
	html := string(NewDeleteButton("Delete", "").HX().Delete("/test").Render())
	if !strings.Contains(html, `hx-delete="/test"`) {
		t.Error("HX().Render() missing hx-delete")
	}
}

func TestFormHTMLEscaping(t *testing.T) {
	form := NewForm().Action("/test?a=1&b=2").CSRFToken("<script>")
	html := string(form.Open())

	if strings.Contains(html, "<script>") {
		t.Error("Open() did not escape CSRF token")
	}
}

func TestDeleteButtonHTMLEscaping(t *testing.T) {
	btn := NewDeleteButton("<script>", "/delete").CSRFToken("<token>")
	html := string(btn.Render())

	if strings.Contains(html, "<script>") {
		t.Error("Render() did not escape text")
	}
	if strings.Contains(html, "<token>") {
		t.Error("Render() did not escape token")
	}
}
