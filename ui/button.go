package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// ButtonSize defines button size variants.
type ButtonSize string

const (
	ButtonSizeSmall  ButtonSize = "sm"
	ButtonSizeMedium ButtonSize = "md"
	ButtonSizeLarge  ButtonSize = "lg"
)

// Button represents an interactive button component.
type Button struct {
	text     string
	emoji    Emoji
	variant  Variant
	size     ButtonSize
	disabled bool
	loading  bool
	class    string
	id       string
	type_    string
	name     string
	value    string
	attrs    *htmx.Attrs
}

// NewButton creates a new button with the given text.
func NewButton(text string) *Button {
	return &Button{
		text:  text,
		size:  ButtonSizeMedium,
		type_: "button",
	}
}

// Emoji sets the button emoji.
func (b *Button) Emoji(e Emoji) *Button {
	b.emoji = e

	return b
}

// Variant sets the button variant.
func (b *Button) Variant(v Variant) *Button {
	b.variant = v

	return b
}

// Primary sets the button to primary variant.
func (b *Button) Primary() *Button {
	return b.Variant(VariantPrimary)
}

// Secondary sets the button to secondary variant.
func (b *Button) Secondary() *Button {
	return b.Variant(VariantSecondary)
}

// Success sets the button to success variant.
func (b *Button) Success() *Button {
	return b.Variant(VariantSuccess)
}

// Warning sets the button to warning variant.
func (b *Button) Warning() *Button {
	return b.Variant(VariantWarning)
}

// Danger sets the button to danger variant.
func (b *Button) Danger() *Button {
	return b.Variant(VariantDanger)
}

// Size sets the button size.
func (b *Button) Size(s ButtonSize) *Button {
	b.size = s

	return b
}

// Small sets the button to small size.
func (b *Button) Small() *Button {
	return b.Size(ButtonSizeSmall)
}

// Large sets the button to large size.
func (b *Button) Large() *Button {
	return b.Size(ButtonSizeLarge)
}

// Disabled sets the button as disabled.
func (b *Button) Disabled(disabled bool) *Button {
	b.disabled = disabled

	return b
}

// Loading sets the button to loading state.
func (b *Button) Loading(loading bool) *Button {
	b.loading = loading

	return b
}

// Class adds custom CSS classes.
func (b *Button) Class(class string) *Button {
	b.class = class

	return b
}

// ID sets the button ID.
func (b *Button) ID(id string) *Button {
	b.id = id

	return b
}

// Type sets the button type (button, submit, reset).
func (b *Button) Type(t string) *Button {
	b.type_ = t

	return b
}

// Submit sets the button type to submit.
func (b *Button) Submit() *Button {
	return b.Type("submit")
}

// Name sets the button name attribute.
func (b *Button) Name(name string) *Button {
	b.name = name

	return b
}

// Value sets the button value attribute.
func (b *Button) Value(value string) *Button {
	b.value = value

	return b
}

// WithAttrs sets HTMX attributes for the button.
func (b *Button) WithAttrs(attrs *htmx.Attrs) *Button {
	b.attrs = attrs

	return b
}

// HX starts building HTMX attributes for this button.
func (b *Button) HX() *ButtonHX {
	return &ButtonHX{button: b, attrs: htmx.NewAttrs()}
}

// Render renders the button to HTML.
func (b *Button) Render() template.HTML {
	var classes []string

	classes = append(classes, "btn")

	if b.variant != "" {
		classes = append(classes, fmt.Sprintf("btn--%s", b.variant))
	}

	if b.size != "" && b.size != ButtonSizeMedium {
		classes = append(classes, fmt.Sprintf("btn--%s", b.size))
	}

	if b.loading {
		classes = append(classes, "btn--loading")
	}

	if b.class != "" {
		classes = append(classes, b.class)
	}

	var attrs []string

	attrs = append(attrs, fmt.Sprintf(`class="%s"`, template.HTMLEscapeString(strings.Join(classes, " "))))

	if b.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(b.id)))
	}

	attrs = append(attrs, fmt.Sprintf(`type="%s"`, template.HTMLEscapeString(b.type_)))

	if b.name != "" {
		attrs = append(attrs, fmt.Sprintf(`name="%s"`, template.HTMLEscapeString(b.name)))
	}

	if b.value != "" {
		attrs = append(attrs, fmt.Sprintf(`value="%s"`, template.HTMLEscapeString(b.value)))
	}

	if b.disabled || b.loading {
		attrs = append(attrs, "disabled")
	}

	if b.attrs != nil {
		attrs = append(attrs, string(b.attrs.HTML()))
	}

	var content strings.Builder
	if b.emoji != "" {
		fmt.Fprintf(&content, `<span class="btn__emoji">%s</span>`, b.emoji)
	}

	if b.loading {
		content.WriteString(`<span class="btn__spinner"></span>`)
	}

	fmt.Fprintf(&content, `<span class="btn__text">%s</span>`, template.HTMLEscapeString(b.text))

	return template.HTML(fmt.Sprintf(`<button %s>%s</button>`, strings.Join(attrs, " "), content.String()))
}

// ButtonHX provides a fluent interface for building HTMX-enabled buttons.
type ButtonHX struct {
	button *Button
	attrs  *htmx.Attrs
}

// Get sets a GET action.
func (bh *ButtonHX) Get(url string) *ButtonHX {
	bh.attrs.Get(url)

	return bh
}

// Post sets a POST action.
func (bh *ButtonHX) Post(url string) *ButtonHX {
	bh.attrs.Post(url)

	return bh
}

// Put sets a PUT action.
func (bh *ButtonHX) Put(url string) *ButtonHX {
	bh.attrs.Put(url)

	return bh
}

// Delete sets a DELETE action.
func (bh *ButtonHX) Delete(url string) *ButtonHX {
	bh.attrs.Delete(url)

	return bh
}

// TargetID sets a target by ID.
func (bh *ButtonHX) TargetID(id string) *ButtonHX {
	bh.attrs.TargetID(id)

	return bh
}

// SwapOuter sets outer HTML swap.
func (bh *ButtonHX) SwapOuter() *ButtonHX {
	bh.attrs.SwapOuter()

	return bh
}

// SwapInner sets inner HTML swap.
func (bh *ButtonHX) SwapInner() *ButtonHX {
	bh.attrs.SwapInner()

	return bh
}

// SwapDelete sets delete swap.
func (bh *ButtonHX) SwapDelete() *ButtonHX {
	bh.attrs.SwapDelete()

	return bh
}

// Confirm sets a confirmation message.
func (bh *ButtonHX) Confirm(message string) *ButtonHX {
	bh.attrs.Confirm(message)

	return bh
}

// Done finalizes and returns the button.
func (bh *ButtonHX) Done() *Button {
	bh.button.attrs = bh.attrs

	return bh.button
}

// Render renders the button to HTML.
func (bh *ButtonHX) Render() template.HTML {
	return bh.Done().Render()
}

// Btn is a shorthand to create a button.
func Btn(text string) *Button {
	return NewButton(text)
}

// BtnSubmit creates a submit button.
func BtnSubmit(text string) *Button {
	return NewButton(text).Submit()
}

// BtnDanger creates a danger button.
func BtnDanger(text string) *Button {
	return NewButton(text).Danger()
}
