package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/settings"
)

// SettingsForm generates a form from setting schemas.
type SettingsForm struct {
	schemas   []settings.Schema
	values    map[string]string
	errors    map[string]string
	action    string
	method    string
	class     string
	submitBtn string
}

// NewSettingsForm creates a form for the given schemas.
func NewSettingsForm(schemas []settings.Schema) *SettingsForm {
	return &SettingsForm{
		schemas:   schemas,
		values:    make(map[string]string),
		errors:    make(map[string]string),
		method:    "post",
		submitBtn: "Save",
	}
}

// Values sets current values for the form.
func (f *SettingsForm) Values(vals map[string]string) *SettingsForm {
	f.values = vals

	return f
}

// Errors sets validation errors.
func (f *SettingsForm) Errors(errs map[string]string) *SettingsForm {
	f.errors = errs

	return f
}

// Action sets the form submission URL.
func (f *SettingsForm) Action(url string) *SettingsForm {
	f.action = url

	return f
}

// Method sets the form method.
func (f *SettingsForm) Method(method string) *SettingsForm {
	f.method = method

	return f
}

// Class adds custom CSS classes.
func (f *SettingsForm) Class(class string) *SettingsForm {
	f.class = class

	return f
}

// SubmitButton sets the submit button text.
func (f *SettingsForm) SubmitButton(text string) *SettingsForm {
	f.submitBtn = text

	return f
}

// Render renders the settings form to HTML.
func (f *SettingsForm) Render() template.HTML {
	var classes []string

	classes = append(classes, "settings-form")
	if f.class != "" {
		classes = append(classes, f.class)
	}

	var html strings.Builder

	fmt.Fprintf(&html, `<form class="%s" method="%s"`,
		strings.Join(classes, " "),
		template.HTMLEscapeString(f.method))

	if f.action != "" {
		fmt.Fprintf(&html, ` action="%s"`, template.HTMLEscapeString(f.action))
	}

	html.WriteString(`>`)

	for _, schema := range f.schemas {
		f.renderField(&html, schema)
	}

	fmt.Fprintf(&html, `<div class="settings-form__actions"><button type="submit" class="btn btn--primary">%s</button></div>`,
		template.HTMLEscapeString(f.submitBtn))

	html.WriteString(`</form>`)

	return template.HTML(html.String())
}

func (f *SettingsForm) renderField(html *strings.Builder, schema settings.Schema) {
	value := f.values[schema.Key]
	if value == "" {
		value = schema.Default
	}

	err := f.errors[schema.Key]

	fieldClass := "field"
	if err != "" {
		fieldClass += " field--error"
	}

	fmt.Fprintf(html, `<div class="%s">`, fieldClass)

	label := schema.DisplayLabel()

	requiredMark := ""
	if schema.Required {
		requiredMark = `<span class="field__required">*</span>`
	}

	fmt.Fprintf(html, `<label class="field__label" for="%s">%s%s</label>`,
		template.HTMLEscapeString(schema.Key),
		template.HTMLEscapeString(label),
		requiredMark)

	switch schema.Type {
	case settings.Bool:
		f.renderCheckbox(html, schema, value)
	case settings.Enum:
		f.renderSelect(html, schema, value)
	default:
		f.renderInput(html, schema, value)
	}

	if schema.Description != "" && err == "" {
		fmt.Fprintf(html, `<span class="field__hint">%s</span>`, template.HTMLEscapeString(schema.Description))
	}

	if err != "" {
		fmt.Fprintf(html, `<span class="field__error">%s</span>`, template.HTMLEscapeString(err))
	}

	html.WriteString(`</div>`)
}

func (f *SettingsForm) renderInput(html *strings.Builder, schema settings.Schema, value string) {
	inputType := "text"
	if schema.Type == settings.Int {
		inputType = "number"
	}

	if schema.Secret {
		inputType = "password"
	}

	fmt.Fprintf(html, `<input type="%s" id="%s" name="%s" value="%s" class="input"`,
		inputType,
		template.HTMLEscapeString(schema.Key),
		template.HTMLEscapeString(schema.Key),
		template.HTMLEscapeString(value))

	if schema.Required {
		html.WriteString(` required`)
	}

	if schema.Type == settings.Int {
		if schema.Min != nil {
			fmt.Fprintf(html, ` min="%d"`, *schema.Min)
		}

		if schema.Max != nil {
			fmt.Fprintf(html, ` max="%d"`, *schema.Max)
		}
	}

	if schema.MaxLength > 0 {
		fmt.Fprintf(html, ` maxlength="%d"`, schema.MaxLength)
	}

	html.WriteString(`>`)
}

func (f *SettingsForm) renderCheckbox(html *strings.Builder, schema settings.Schema, value string) {
	checked := ""
	if value == "true" || value == "1" {
		checked = " checked"
	}

	fmt.Fprintf(html, `<input type="checkbox" id="%s" name="%s" value="true" class="checkbox"%s>`,
		template.HTMLEscapeString(schema.Key),
		template.HTMLEscapeString(schema.Key),
		checked)
}

func (f *SettingsForm) renderSelect(html *strings.Builder, schema settings.Schema, value string) {
	fmt.Fprintf(html, `<select id="%s" name="%s" class="select"`,
		template.HTMLEscapeString(schema.Key),
		template.HTMLEscapeString(schema.Key))

	if schema.Required {
		html.WriteString(` required`)
	}

	html.WriteString(`>`)

	for i, opt := range schema.Options {
		selected := ""
		if opt == value {
			selected = " selected"
		}

		optLabel := opt
		if i < len(schema.Labels) && schema.Labels[i] != "" {
			optLabel = schema.Labels[i]
		}

		fmt.Fprintf(html, `<option value="%s"%s>%s</option>`,
			template.HTMLEscapeString(opt),
			selected,
			template.HTMLEscapeString(optLabel))
	}

	html.WriteString(`</select>`)
}
