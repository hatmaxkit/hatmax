package ui

import (
	"strings"
	"testing"
)

func TestForm(t *testing.T) {
	t.Run("basic form", func(t *testing.T) {
		form := NewForm()
		html := string(form.Open())

		if !strings.Contains(html, `method="post"`) {
			t.Error("form should have default post method")
		}
		if !strings.HasPrefix(html, "<form") {
			t.Error("Open should return form tag")
		}
	})

	t.Run("form with action", func(t *testing.T) {
		form := NewForm().Action("/submit")
		html := string(form.Open())

		if !strings.Contains(html, `action="/submit"`) {
			t.Error("form should have action")
		}
	})

	t.Run("form with GET method", func(t *testing.T) {
		form := NewForm().Get()
		html := string(form.Open())

		if !strings.Contains(html, `method="get"`) {
			t.Error("form should have get method")
		}
	})

	t.Run("multipart form", func(t *testing.T) {
		form := NewForm().Multipart()
		html := string(form.Open())

		if !strings.Contains(html, `enctype="multipart/form-data"`) {
			t.Error("form should have multipart enctype")
		}
	})

	t.Run("novalidate form", func(t *testing.T) {
		form := NewForm().NoValidate()
		html := string(form.Open())

		if !strings.Contains(html, "novalidate") {
			t.Error("form should have novalidate")
		}
	})

	t.Run("form with HTMX", func(t *testing.T) {
		form := NewForm().HX().
			Post("/api/submit").
			TargetID("result").
			SwapOuter().
			Done()

		html := string(form.Open())

		if !strings.Contains(html, `hx-post="/api/submit"`) {
			t.Error("form should have hx-post")
		}
		if !strings.Contains(html, `hx-target="#result"`) {
			t.Error("form should have hx-target")
		}
	})

	t.Run("form close", func(t *testing.T) {
		form := NewForm()
		close := string(form.Close())

		if close != "</form>" {
			t.Error("Close should return closing tag")
		}
	})
}

func TestInput(t *testing.T) {
	t.Run("text input", func(t *testing.T) {
		input := Text("username")
		html := string(input.Render())

		if !strings.Contains(html, `type="text"`) {
			t.Error("input should have text type")
		}
		if !strings.Contains(html, `name="username"`) {
			t.Error("input should have name")
		}
		if !strings.Contains(html, `id="username"`) {
			t.Error("input should have id matching name")
		}
	})

	t.Run("email input", func(t *testing.T) {
		input := Email("email")
		html := string(input.Render())

		if !strings.Contains(html, `type="email"`) {
			t.Error("input should have email type")
		}
	})

	t.Run("password input", func(t *testing.T) {
		input := Password("password")
		html := string(input.Render())

		if !strings.Contains(html, `type="password"`) {
			t.Error("input should have password type")
		}
	})

	t.Run("number input", func(t *testing.T) {
		input := Number("quantity").Min("0").Max("100").Step("1")
		html := string(input.Render())

		if !strings.Contains(html, `type="number"`) {
			t.Error("input should have number type")
		}
		if !strings.Contains(html, `min="0"`) {
			t.Error("input should have min")
		}
		if !strings.Contains(html, `max="100"`) {
			t.Error("input should have max")
		}
		if !strings.Contains(html, `step="1"`) {
			t.Error("input should have step")
		}
	})

	t.Run("hidden input", func(t *testing.T) {
		input := Hidden("csrf", "token123")
		html := string(input.Render())

		if !strings.Contains(html, `type="hidden"`) {
			t.Error("input should have hidden type")
		}
		if !strings.Contains(html, `value="token123"`) {
			t.Error("input should have value")
		}
	})

	t.Run("search input", func(t *testing.T) {
		input := Search("q").Placeholder("Search...")
		html := string(input.Render())

		if !strings.Contains(html, `type="search"`) {
			t.Error("input should have search type")
		}
		if !strings.Contains(html, `placeholder="Search..."`) {
			t.Error("input should have placeholder")
		}
	})

	t.Run("input with validation", func(t *testing.T) {
		input := Text("username").Required().MinLength(3).MaxLength(20)
		html := string(input.Render())

		if !strings.Contains(html, "required") {
			t.Error("input should be required")
		}
		if !strings.Contains(html, `minlength="3"`) {
			t.Error("input should have minlength")
		}
		if !strings.Contains(html, `maxlength="20"`) {
			t.Error("input should have maxlength")
		}
	})

	t.Run("input states", func(t *testing.T) {
		disabled := Text("test").Disabled()
		if !strings.Contains(string(disabled.Render()), "disabled") {
			t.Error("disabled input should have disabled attribute")
		}

		readonly := Text("test").Readonly()
		if !strings.Contains(string(readonly.Render()), "readonly") {
			t.Error("readonly input should have readonly attribute")
		}

		autofocus := Text("test").Autofocus()
		if !strings.Contains(string(autofocus.Render()), "autofocus") {
			t.Error("autofocus input should have autofocus attribute")
		}
	})

	t.Run("input with pattern", func(t *testing.T) {
		input := Text("phone").Pattern(`\d{3}-\d{4}`)
		html := string(input.Render())

		if !strings.Contains(html, "pattern=") {
			t.Error("input should have pattern")
		}
	})

	t.Run("input with HTMX", func(t *testing.T) {
		input := Search("q").HX().
			Get("/search").
			TargetID("results").
			OnKeyup().
			Done()

		html := string(input.Render())

		if !strings.Contains(html, `hx-get="/search"`) {
			t.Error("input should have hx-get")
		}
		if !strings.Contains(html, `hx-target="#results"`) {
			t.Error("input should have hx-target")
		}
		if !strings.Contains(html, `hx-trigger="keyup"`) {
			t.Error("input should have hx-trigger")
		}
	})
}

func TestField(t *testing.T) {
	t.Run("basic field", func(t *testing.T) {
		field := NewField("Username", "username").
			Input(Text("username"))
		html := string(field.Render())

		if !strings.Contains(html, `class="field"`) {
			t.Error("field should have field class")
		}
		if !strings.Contains(html, "Username") {
			t.Error("field should have label")
		}
		if !strings.Contains(html, `for="username"`) {
			t.Error("label should have for attribute")
		}
		if !strings.Contains(html, `<input`) {
			t.Error("field should contain input")
		}
	})

	t.Run("required field", func(t *testing.T) {
		field := NewField("Email", "email").Required()
		html := string(field.Render())

		if !strings.Contains(html, "field__required") {
			t.Error("required field should have required mark")
		}
	})

	t.Run("field with hint", func(t *testing.T) {
		field := NewField("Password", "password").
			Hint("At least 8 characters")
		html := string(field.Render())

		if !strings.Contains(html, "field__hint") {
			t.Error("field should have hint")
		}
		if !strings.Contains(html, "At least 8 characters") {
			t.Error("field should show hint text")
		}
	})

	t.Run("field with error", func(t *testing.T) {
		field := NewField("Email", "email").
			Error("Invalid email format")
		html := string(field.Render())

		if !strings.Contains(html, "field--error") {
			t.Error("field should have error class")
		}
		if !strings.Contains(html, "field__error") {
			t.Error("field should have error element")
		}
		if !strings.Contains(html, "Invalid email format") {
			t.Error("field should show error text")
		}
	})

	t.Run("field with error hides hint", func(t *testing.T) {
		field := NewField("Email", "email").
			Hint("Enter your email").
			Error("Invalid email")
		html := string(field.Render())

		if strings.Contains(html, "field__hint") {
			t.Error("error should hide hint")
		}
	})
}
