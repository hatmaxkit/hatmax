package ui

import (
	"strings"
	"testing"

	"hatmax.adrianpk.com/settings"
)

func TestSettingsFormRender(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "app.name", Type: settings.String, Label: "App Name", Required: true},
		{Key: "app.debug", Type: settings.Bool, Label: "Debug Mode"},
	}

	form := NewSettingsForm(schemas).Action("/settings").SubmitButton("Update")
	html := string(form.Render())

	if !strings.Contains(html, `class="settings-form"`) {
		t.Error("Render() missing form class")
	}

	if !strings.Contains(html, `action="/settings"`) {
		t.Error("Render() missing action")
	}

	if !strings.Contains(html, `App Name`) {
		t.Error("Render() missing label")
	}

	if !strings.Contains(html, `Update`) {
		t.Error("Render() missing submit button text")
	}
}

func TestSettingsFormWithValues(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "site.title", Type: settings.String, Default: "Default"},
	}

	form := NewSettingsForm(schemas).Values(map[string]string{"site.title": "My Site"})
	html := string(form.Render())

	if !strings.Contains(html, `value="My Site"`) {
		t.Error("Render() missing value")
	}
}

func TestSettingsFormDefaultValue(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "site.title", Type: settings.String, Default: "Default Title"},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `value="Default Title"`) {
		t.Error("Render() missing default value")
	}
}

func TestSettingsFormWithErrors(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "email", Type: settings.String},
	}

	form := NewSettingsForm(schemas).Errors(map[string]string{"email": "Invalid email"})
	html := string(form.Render())

	if !strings.Contains(html, `field--error`) {
		t.Error("Render() missing error class")
	}

	if !strings.Contains(html, `Invalid email`) {
		t.Error("Render() missing error message")
	}
}

func TestSettingsFormIntField(t *testing.T) {
	min, max := 1, 100
	schemas := []settings.Schema{
		{Key: "limit", Type: settings.Int, Min: &min, Max: &max},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `type="number"`) {
		t.Error("Render() missing number type")
	}

	if !strings.Contains(html, `min="1"`) {
		t.Error("Render() missing min")
	}

	if !strings.Contains(html, `max="100"`) {
		t.Error("Render() missing max")
	}
}

func TestSettingsFormBoolField(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "enabled", Type: settings.Bool},
	}

	form := NewSettingsForm(schemas).Values(map[string]string{"enabled": "true"})
	html := string(form.Render())

	if !strings.Contains(html, `type="checkbox"`) {
		t.Error("Render() missing checkbox type")
	}

	if !strings.Contains(html, `checked`) {
		t.Error("Render() missing checked attribute")
	}
}

func TestSettingsFormEnumField(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "theme", Type: settings.Enum, Options: []string{"light", "dark"}, Labels: []string{"Light", "Dark"}},
	}

	form := NewSettingsForm(schemas).Values(map[string]string{"theme": "dark"})
	html := string(form.Render())

	if !strings.Contains(html, `<select`) {
		t.Error("Render() missing select")
	}

	if !strings.Contains(html, `value="light"`) {
		t.Error("Render() missing light option")
	}

	if !strings.Contains(html, `value="dark" selected`) {
		t.Error("Render() missing selected dark option")
	}

	if !strings.Contains(html, `>Light</option>`) {
		t.Error("Render() missing Light label")
	}
}

func TestSettingsFormSecretField(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "api_key", Type: settings.String, Secret: true},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `type="password"`) {
		t.Error("Render() missing password type for secret")
	}
}

func TestSettingsFormRequired(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "name", Type: settings.String, Required: true},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `required`) {
		t.Error("Render() missing required attribute")
	}

	if !strings.Contains(html, `field__required`) {
		t.Error("Render() missing required marker")
	}
}

func TestSettingsFormMaxLength(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "code", Type: settings.String, MaxLength: 10},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `maxlength="10"`) {
		t.Error("Render() missing maxlength")
	}
}

func TestSettingsFormDescription(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "timeout", Type: settings.Int, Description: "Timeout in seconds"},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `Timeout in seconds`) {
		t.Error("Render() missing description")
	}

	if !strings.Contains(html, `field__hint`) {
		t.Error("Render() missing hint class")
	}
}

func TestSettingsFormDisplayLabel(t *testing.T) {
	schemas := []settings.Schema{
		{Key: "app.setting", Type: settings.String},
	}

	form := NewSettingsForm(schemas)
	html := string(form.Render())

	if !strings.Contains(html, `app.setting`) {
		t.Error("Render() missing fallback label (key)")
	}
}

func TestSettingsFormMethod(t *testing.T) {
	schemas := []settings.Schema{}
	form := NewSettingsForm(schemas).Method("put")
	html := string(form.Render())

	if !strings.Contains(html, `method="put"`) {
		t.Error("Method() not applied")
	}
}

func TestSettingsFormClass(t *testing.T) {
	schemas := []settings.Schema{}
	form := NewSettingsForm(schemas).Class("custom-form")
	html := string(form.Render())

	if !strings.Contains(html, `custom-form`) {
		t.Error("Class() not applied")
	}
}
