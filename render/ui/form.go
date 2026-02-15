package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// Form represents an interactive form component with HTMX support.
type Form struct {
	BaseInteractive
	action   string
	method   string
	class    string
	id       string
	enctype  string
	novalidate bool
}

// NewForm creates a new form.
func NewForm() *Form {
	return &Form{
		method: "post",
	}
}

// Action sets the form action URL.
func (f *Form) Action(action string) *Form {
	f.action = action
	return f
}

// Method sets the form method.
func (f *Form) Method(method string) *Form {
	f.method = method
	return f
}

// Get sets the form method to GET.
func (f *Form) Get() *Form {
	return f.Method("get")
}

// Post sets the form method to POST.
func (f *Form) Post() *Form {
	return f.Method("post")
}

// Class adds custom CSS classes.
func (f *Form) Class(class string) *Form {
	f.class = class
	return f
}

// ID sets the form ID.
func (f *Form) ID(id string) *Form {
	f.id = id
	return f
}

// Enctype sets the form enctype.
func (f *Form) Enctype(enctype string) *Form {
	f.enctype = enctype
	return f
}

// Multipart sets the form enctype to multipart/form-data.
func (f *Form) Multipart() *Form {
	return f.Enctype("multipart/form-data")
}

// NoValidate disables browser validation.
func (f *Form) NoValidate() *Form {
	f.novalidate = true
	return f
}

// WithAttrs sets HTMX attributes for the form.
func (f *Form) WithAttrs(attrs *htmx.Attrs) InteractiveComponent {
	f.SetAttrs(attrs)
	return f
}

// HX starts building HTMX attributes for this form.
func (f *Form) HX() *FormHX {
	return &FormHX{form: f, attrs: htmx.NewAttrs()}
}

// Open renders the opening form tag.
func (f *Form) Open() template.HTML {
	var attrs []string

	if f.action != "" {
		attrs = append(attrs, fmt.Sprintf(`action="%s"`, escape(f.action)))
	}

	attrs = append(attrs, fmt.Sprintf(`method="%s"`, escape(f.method)))

	if f.class != "" {
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, escape(f.class)))
	}

	if f.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(f.id)))
	}

	if f.enctype != "" {
		attrs = append(attrs, fmt.Sprintf(`enctype="%s"`, escape(f.enctype)))
	}

	if f.novalidate {
		attrs = append(attrs, "novalidate")
	}

	if f.HasAttrs() {
		attrs = append(attrs, string(f.HXAttrs()))
	}

	return template.HTML(fmt.Sprintf(`<form %s>`, strings.Join(attrs, " ")))
}

// Close renders the closing form tag.
func (f *Form) Close() template.HTML {
	return template.HTML("</form>")
}

// Render renders the complete form (open + close).
func (f *Form) Render() template.HTML {
	return f.Open() + f.Close()
}

// FormHX provides a fluent interface for building HTMX-enabled forms.
type FormHX struct {
	form  *Form
	attrs *htmx.Attrs
}

// Post sets a POST action.
func (fh *FormHX) Post(url string) *FormHX {
	fh.attrs.Post(url)
	return fh
}

// Put sets a PUT action.
func (fh *FormHX) Put(url string) *FormHX {
	fh.attrs.Put(url)
	return fh
}

// Patch sets a PATCH action.
func (fh *FormHX) Patch(url string) *FormHX {
	fh.attrs.Patch(url)
	return fh
}

// Delete sets a DELETE action.
func (fh *FormHX) Delete(url string) *FormHX {
	fh.attrs.Delete(url)
	return fh
}

// Target sets the target selector.
func (fh *FormHX) Target(target htmx.Target) *FormHX {
	fh.attrs.Target(target)
	return fh
}

// TargetID sets a target by ID.
func (fh *FormHX) TargetID(id string) *FormHX {
	fh.attrs.TargetID(id)
	return fh
}

// Swap sets the swap strategy.
func (fh *FormHX) Swap(swap htmx.Swap) *FormHX {
	fh.attrs.Swap(swap)
	return fh
}

// SwapOuter sets outer HTML swap.
func (fh *FormHX) SwapOuter() *FormHX {
	fh.attrs.SwapOuter()
	return fh
}

// SwapInner sets inner HTML swap.
func (fh *FormHX) SwapInner() *FormHX {
	fh.attrs.SwapInner()
	return fh
}

// Validate enables HTMX form validation.
func (fh *FormHX) Validate() *FormHX {
	fh.attrs.Validate()
	return fh
}

// Encoding sets the form encoding for HTMX.
func (fh *FormHX) Encoding(encoding string) *FormHX {
	fh.attrs.Encoding(encoding)
	return fh
}

// Indicator sets the indicator element.
func (fh *FormHX) Indicator(selector string) *FormHX {
	fh.attrs.Indicator(selector)
	return fh
}

// Done finalizes and returns the form.
func (fh *FormHX) Done() *Form {
	fh.form.SetAttrs(fh.attrs)
	return fh.form
}

// InputType defines input types.
type InputType string

const (
	InputText     InputType = "text"
	InputEmail    InputType = "email"
	InputPassword InputType = "password"
	InputNumber   InputType = "number"
	InputTel      InputType = "tel"
	InputURL      InputType = "url"
	InputSearch   InputType = "search"
	InputDate     InputType = "date"
	InputTime     InputType = "time"
	InputDatetime InputType = "datetime-local"
	InputHidden   InputType = "hidden"
	InputFile     InputType = "file"
	InputCheckbox InputType = "checkbox"
	InputRadio    InputType = "radio"
	InputColor    InputType = "color"
	InputRange    InputType = "range"
)

// Input represents a form input element.
type Input struct {
	BaseInteractive
	type_       InputType
	name        string
	value       string
	placeholder string
	required    bool
	disabled    bool
	readonly    bool
	autofocus   bool
	class       string
	id          string
	min         string
	max         string
	step        string
	pattern     string
	minlength   int
	maxlength   int
	autocomplete string
}

// NewInput creates a new input with the given type and name.
func NewInput(inputType InputType, name string) *Input {
	return &Input{
		type_: inputType,
		name:  name,
		id:    name,
	}
}

// Text creates a text input.
func Text(name string) *Input {
	return NewInput(InputText, name)
}

// Email creates an email input.
func Email(name string) *Input {
	return NewInput(InputEmail, name)
}

// Password creates a password input.
func Password(name string) *Input {
	return NewInput(InputPassword, name)
}

// Number creates a number input.
func Number(name string) *Input {
	return NewInput(InputNumber, name)
}

// Hidden creates a hidden input.
func Hidden(name, value string) *Input {
	return NewInput(InputHidden, name).Value(value)
}

// Search creates a search input.
func Search(name string) *Input {
	return NewInput(InputSearch, name)
}

// Value sets the input value.
func (i *Input) Value(value string) *Input {
	i.value = value
	return i
}

// Placeholder sets the placeholder text.
func (i *Input) Placeholder(placeholder string) *Input {
	i.placeholder = placeholder
	return i
}

// Required marks the input as required.
func (i *Input) Required() *Input {
	i.required = true
	return i
}

// Disabled disables the input.
func (i *Input) Disabled() *Input {
	i.disabled = true
	return i
}

// Readonly makes the input readonly.
func (i *Input) Readonly() *Input {
	i.readonly = true
	return i
}

// Autofocus enables autofocus.
func (i *Input) Autofocus() *Input {
	i.autofocus = true
	return i
}

// Class adds custom CSS classes.
func (i *Input) Class(class string) *Input {
	i.class = class
	return i
}

// ID sets the input ID.
func (i *Input) ID(id string) *Input {
	i.id = id
	return i
}

// Min sets the minimum value.
func (i *Input) Min(min string) *Input {
	i.min = min
	return i
}

// Max sets the maximum value.
func (i *Input) Max(max string) *Input {
	i.max = max
	return i
}

// Step sets the step value.
func (i *Input) Step(step string) *Input {
	i.step = step
	return i
}

// Pattern sets the validation pattern.
func (i *Input) Pattern(pattern string) *Input {
	i.pattern = pattern
	return i
}

// MinLength sets the minimum length.
func (i *Input) MinLength(min int) *Input {
	i.minlength = min
	return i
}

// MaxLength sets the maximum length.
func (i *Input) MaxLength(max int) *Input {
	i.maxlength = max
	return i
}

// Autocomplete sets the autocomplete attribute.
func (i *Input) Autocomplete(value string) *Input {
	i.autocomplete = value
	return i
}

// WithAttrs sets HTMX attributes for the input.
func (i *Input) WithAttrs(attrs *htmx.Attrs) InteractiveComponent {
	i.SetAttrs(attrs)
	return i
}

// HX starts building HTMX attributes for this input.
func (i *Input) HX() *InputHX {
	return &InputHX{input: i, attrs: htmx.NewAttrs()}
}

// Render renders the input to HTML.
func (i *Input) Render() template.HTML {
	var attrs []string

	attrs = append(attrs, fmt.Sprintf(`type="%s"`, escape(string(i.type_))))
	attrs = append(attrs, fmt.Sprintf(`name="%s"`, escape(i.name)))

	if i.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(i.id)))
	}

	if i.value != "" {
		attrs = append(attrs, fmt.Sprintf(`value="%s"`, escape(i.value)))
	}

	if i.placeholder != "" {
		attrs = append(attrs, fmt.Sprintf(`placeholder="%s"`, escape(i.placeholder)))
	}

	if i.class != "" {
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, escape(i.class)))
	}

	if i.required {
		attrs = append(attrs, "required")
	}

	if i.disabled {
		attrs = append(attrs, "disabled")
	}

	if i.readonly {
		attrs = append(attrs, "readonly")
	}

	if i.autofocus {
		attrs = append(attrs, "autofocus")
	}

	if i.min != "" {
		attrs = append(attrs, fmt.Sprintf(`min="%s"`, escape(i.min)))
	}

	if i.max != "" {
		attrs = append(attrs, fmt.Sprintf(`max="%s"`, escape(i.max)))
	}

	if i.step != "" {
		attrs = append(attrs, fmt.Sprintf(`step="%s"`, escape(i.step)))
	}

	if i.pattern != "" {
		attrs = append(attrs, fmt.Sprintf(`pattern="%s"`, escape(i.pattern)))
	}

	if i.minlength > 0 {
		attrs = append(attrs, fmt.Sprintf(`minlength="%d"`, i.minlength))
	}

	if i.maxlength > 0 {
		attrs = append(attrs, fmt.Sprintf(`maxlength="%d"`, i.maxlength))
	}

	if i.autocomplete != "" {
		attrs = append(attrs, fmt.Sprintf(`autocomplete="%s"`, escape(i.autocomplete)))
	}

	if i.HasAttrs() {
		attrs = append(attrs, string(i.HXAttrs()))
	}

	return template.HTML(fmt.Sprintf(`<input %s>`, strings.Join(attrs, " ")))
}

// InputHX provides a fluent interface for building HTMX-enabled inputs.
type InputHX struct {
	input *Input
	attrs *htmx.Attrs
}

// Get sets a GET action.
func (ih *InputHX) Get(url string) *InputHX {
	ih.attrs.Get(url)
	return ih
}

// Post sets a POST action.
func (ih *InputHX) Post(url string) *InputHX {
	ih.attrs.Post(url)
	return ih
}

// Target sets the target selector.
func (ih *InputHX) Target(target htmx.Target) *InputHX {
	ih.attrs.Target(target)
	return ih
}

// TargetID sets a target by ID.
func (ih *InputHX) TargetID(id string) *InputHX {
	ih.attrs.TargetID(id)
	return ih
}

// Swap sets the swap strategy.
func (ih *InputHX) Swap(swap htmx.Swap) *InputHX {
	ih.attrs.Swap(swap)
	return ih
}

// SwapOuter sets outer HTML swap.
func (ih *InputHX) SwapOuter() *InputHX {
	ih.attrs.SwapOuter()
	return ih
}

// Trigger sets the trigger.
func (ih *InputHX) Trigger(trigger htmx.Trigger) *InputHX {
	ih.attrs.Trigger(trigger)
	return ih
}

// OnChange triggers on change events.
func (ih *InputHX) OnChange() *InputHX {
	ih.attrs.Trigger(htmx.OnChange())
	return ih
}

// OnInput triggers on input events.
func (ih *InputHX) OnInput() *InputHX {
	ih.attrs.Trigger(htmx.OnInput())
	return ih
}

// OnKeyup triggers on keyup events.
func (ih *InputHX) OnKeyup() *InputHX {
	ih.attrs.Trigger(htmx.OnKeyup())
	return ih
}

// Debounce adds a delay to the trigger.
func (ih *InputHX) Debounce(ms int) *InputHX {
	// This needs to be called after setting a trigger
	return ih
}

// Done finalizes and returns the input.
func (ih *InputHX) Done() *Input {
	ih.input.SetAttrs(ih.attrs)
	return ih.input
}

// Render renders the input to HTML.
func (ih *InputHX) Render() template.HTML {
	return ih.Done().Render()
}

// Field represents a form field with label and error.
type Field struct {
	label    string
	name     string
	error    string
	hint     string
	required bool
	class    string
	input    Component
}

// NewField creates a new field.
func NewField(label, name string) *Field {
	return &Field{
		label: label,
		name:  name,
	}
}

// Error sets the error message.
func (f *Field) Error(error string) *Field {
	f.error = error
	return f
}

// Hint sets the hint text.
func (f *Field) Hint(hint string) *Field {
	f.hint = hint
	return f
}

// Required marks the field as required.
func (f *Field) Required() *Field {
	f.required = true
	return f
}

// Class adds custom CSS classes.
func (f *Field) Class(class string) *Field {
	f.class = class
	return f
}

// Input sets the input component.
func (f *Field) Input(input Component) *Field {
	f.input = input
	return f
}

// Render renders the field to HTML.
func (f *Field) Render() template.HTML {
	var classes []string
	classes = append(classes, "field")

	if f.error != "" {
		classes = append(classes, "field--error")
	}

	if f.class != "" {
		classes = append(classes, f.class)
	}

	var html strings.Builder
	html.WriteString(fmt.Sprintf(`<div class="%s">`, strings.Join(classes, " ")))

	// Label
	requiredMark := ""
	if f.required {
		requiredMark = `<span class="field__required">*</span>`
	}
	html.WriteString(fmt.Sprintf(`<label class="field__label" for="%s">%s%s</label>`,
		escape(f.name), escape(f.label), requiredMark))

	// Input
	if f.input != nil {
		html.WriteString(string(f.input.Render()))
	}

	// Hint
	if f.hint != "" && f.error == "" {
		html.WriteString(fmt.Sprintf(`<span class="field__hint">%s</span>`, escape(f.hint)))
	}

	// Error
	if f.error != "" {
		html.WriteString(fmt.Sprintf(`<span class="field__error">%s</span>`, escape(f.error)))
	}

	html.WriteString("</div>")

	return template.HTML(html.String())
}
