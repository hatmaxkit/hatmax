package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// Form represents a form component with CSRF support.
type Form struct {
	action     string
	method     string
	csrfToken  string
	csrfField  string
	class      string
	id         string
	enctype    string
	novalidate bool
	attrs      *htmx.Attrs
}

// NewForm creates a new form.
func NewForm() *Form {
	return &Form{
		method:    "post",
		csrfField: "_token",
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

// CSRFToken sets the CSRF token to include in the form.
func (f *Form) CSRFToken(token string) *Form {
	f.csrfToken = token

	return f
}

// CSRFField sets the name of the CSRF token field.
func (f *Form) CSRFField(name string) *Form {
	f.csrfField = name

	return f
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
func (f *Form) WithAttrs(attrs *htmx.Attrs) *Form {
	f.attrs = attrs

	return f
}

// HX starts building HTMX attributes for this form.
func (f *Form) HX() *FormHX {
	return &FormHX{form: f, attrs: htmx.NewAttrs()}
}

// Open renders the opening form tag with CSRF token.
func (f *Form) Open() template.HTML {
	var attrs []string

	if f.action != "" {
		attrs = append(attrs, fmt.Sprintf(`action="%s"`, template.HTMLEscapeString(f.action)))
	}

	attrs = append(attrs, fmt.Sprintf(`method="%s"`, template.HTMLEscapeString(f.method)))

	if f.class != "" {
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, template.HTMLEscapeString(f.class)))
	}

	if f.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(f.id)))
	}

	if f.enctype != "" {
		attrs = append(attrs, fmt.Sprintf(`enctype="%s"`, template.HTMLEscapeString(f.enctype)))
	}

	if f.novalidate {
		attrs = append(attrs, "novalidate")
	}

	if f.attrs != nil {
		attrs = append(attrs, string(f.attrs.HTML()))
	}

	var html strings.Builder
	fmt.Fprintf(&html, `<form %s>`, strings.Join(attrs, " "))

	if f.csrfToken != "" {
		fmt.Fprintf(&html, `<input type="hidden" name="%s" value="%s">`,
			template.HTMLEscapeString(f.csrfField),
			template.HTMLEscapeString(f.csrfToken))
	}

	return template.HTML(html.String())
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

// Delete sets a DELETE action.
func (fh *FormHX) Delete(url string) *FormHX {
	fh.attrs.Delete(url)

	return fh
}

// TargetID sets a target by ID.
func (fh *FormHX) TargetID(id string) *FormHX {
	fh.attrs.TargetID(id)

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

// SwapDelete sets delete swap (removes element).
func (fh *FormHX) SwapDelete() *FormHX {
	fh.attrs.SwapDelete()

	return fh
}

// Confirm sets a confirmation message.
func (fh *FormHX) Confirm(message string) *FormHX {
	fh.attrs.Confirm(message)

	return fh
}

// Done finalizes and returns the form.
func (fh *FormHX) Done() *Form {
	fh.form.attrs = fh.attrs

	return fh.form
}

// DeleteButton renders a delete action as a form (not a link).
type DeleteButton struct {
	action    string
	text      string
	csrfToken string
	csrfField string
	confirm   string
	class     string
	variant   Variant
	emoji     Emoji
	attrs     *htmx.Attrs
}

// NewDeleteButton creates a delete button that submits as a form.
func NewDeleteButton(text, action string) *DeleteButton {
	return &DeleteButton{
		text:      text,
		action:    action,
		csrfField: "_token",
		variant:   VariantDanger,
	}
}

// CSRFToken sets the CSRF token.
func (d *DeleteButton) CSRFToken(token string) *DeleteButton {
	d.csrfToken = token

	return d
}

// CSRFField sets the name of the CSRF token field.
func (d *DeleteButton) CSRFField(name string) *DeleteButton {
	d.csrfField = name

	return d
}

// Confirm sets a confirmation message.
func (d *DeleteButton) Confirm(message string) *DeleteButton {
	d.confirm = message

	return d
}

// Class adds custom CSS classes.
func (d *DeleteButton) Class(class string) *DeleteButton {
	d.class = class

	return d
}

// Variant sets the button variant.
func (d *DeleteButton) Variant(v Variant) *DeleteButton {
	d.variant = v

	return d
}

// Emoji sets the button emoji.
func (d *DeleteButton) Emoji(e Emoji) *DeleteButton {
	d.emoji = e

	return d
}

// HX starts building HTMX attributes for this delete button.
func (d *DeleteButton) HX() *DeleteButtonHX {
	return &DeleteButtonHX{btn: d, attrs: htmx.NewAttrs()}
}

// Render renders the delete button as a form.
func (d *DeleteButton) Render() template.HTML {
	var html strings.Builder

	formAttrs := `method="post" class="inline-form"`
	if d.action != "" {
		formAttrs += fmt.Sprintf(` action="%s"`, template.HTMLEscapeString(d.action))
	}

	if d.attrs != nil {
		formAttrs += " " + string(d.attrs.HTML())
	}

	fmt.Fprintf(&html, `<form %s>`, formAttrs)
	html.WriteString(`<input type="hidden" name="_method" value="DELETE">`)

	if d.csrfToken != "" {
		fmt.Fprintf(&html, `<input type="hidden" name="%s" value="%s">`,
			template.HTMLEscapeString(d.csrfField),
			template.HTMLEscapeString(d.csrfToken))
	}

	var btnClasses []string

	btnClasses = append(btnClasses, "btn")
	if d.variant != "" {
		btnClasses = append(btnClasses, fmt.Sprintf("btn--%s", d.variant))
	}

	if d.class != "" {
		btnClasses = append(btnClasses, d.class)
	}

	btnAttrs := fmt.Sprintf(`type="submit" class="%s"`, strings.Join(btnClasses, " "))
	if d.confirm != "" {
		btnAttrs += fmt.Sprintf(` onclick="return confirm('%s')"`, template.JSEscapeString(d.confirm))
	}

	fmt.Fprintf(&html, `<button %s>`, btnAttrs)

	if d.emoji != "" {
		fmt.Fprintf(&html, `<span class="btn__emoji">%s</span>`, d.emoji)
	}

	fmt.Fprintf(&html, `<span class="btn__text">%s</span>`, template.HTMLEscapeString(d.text))
	html.WriteString(`</button>`)
	html.WriteString(`</form>`)

	return template.HTML(html.String())
}

// DeleteButtonHX provides a fluent interface for HTMX-enabled delete buttons.
type DeleteButtonHX struct {
	btn   *DeleteButton
	attrs *htmx.Attrs
}

// Delete sets a DELETE action via HTMX.
func (dh *DeleteButtonHX) Delete(url string) *DeleteButtonHX {
	dh.attrs.Delete(url)

	return dh
}

// TargetID sets a target by ID.
func (dh *DeleteButtonHX) TargetID(id string) *DeleteButtonHX {
	dh.attrs.TargetID(id)

	return dh
}

// SwapDelete sets delete swap (removes element).
func (dh *DeleteButtonHX) SwapDelete() *DeleteButtonHX {
	dh.attrs.SwapDelete()

	return dh
}

// SwapOuter sets outer HTML swap.
func (dh *DeleteButtonHX) SwapOuter() *DeleteButtonHX {
	dh.attrs.SwapOuter()

	return dh
}

// Confirm sets a confirmation message.
func (dh *DeleteButtonHX) Confirm(message string) *DeleteButtonHX {
	dh.attrs.Confirm(message)

	return dh
}

// Done finalizes and returns the delete button.
func (dh *DeleteButtonHX) Done() *DeleteButton {
	dh.btn.attrs = dh.attrs

	return dh.btn
}

// Render renders the delete button.
func (dh *DeleteButtonHX) Render() template.HTML {
	return dh.Done().Render()
}
