package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// ButtonVariant defines button style variants.
type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
	ButtonSuccess   ButtonVariant = "success"
	ButtonWarning   ButtonVariant = "warning"
	ButtonGhost     ButtonVariant = "ghost"
	ButtonLink      ButtonVariant = "link"
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
	BaseInteractive
	text      string
	variant   ButtonVariant
	size      ButtonSize
	disabled  bool
	loading   bool
	icon      string
	iconRight string
	class     string
	id        string
	type_     string
	name      string
	value     string
}

// NewButton creates a new button with the given text.
func NewButton(text string) *Button {
	return &Button{
		text:    text,
		variant: ButtonPrimary,
		size:    ButtonSizeMedium,
		type_:   "button",
	}
}

// Variant sets the button variant.
func (b *Button) Variant(v ButtonVariant) *Button {
	b.variant = v
	return b
}

// Primary sets the button to primary variant.
func (b *Button) Primary() *Button {
	return b.Variant(ButtonPrimary)
}

// Secondary sets the button to secondary variant.
func (b *Button) Secondary() *Button {
	return b.Variant(ButtonSecondary)
}

// Danger sets the button to danger variant.
func (b *Button) Danger() *Button {
	return b.Variant(ButtonDanger)
}

// Success sets the button to success variant.
func (b *Button) Success() *Button {
	return b.Variant(ButtonSuccess)
}

// Warning sets the button to warning variant.
func (b *Button) Warning() *Button {
	return b.Variant(ButtonWarning)
}

// Ghost sets the button to ghost variant.
func (b *Button) Ghost() *Button {
	return b.Variant(ButtonGhost)
}

// LinkStyle sets the button to look like a link.
func (b *Button) LinkStyle() *Button {
	return b.Variant(ButtonLink)
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

// Icon sets an icon to display before the text.
func (b *Button) Icon(icon string) *Button {
	b.icon = icon
	return b
}

// IconRight sets an icon to display after the text.
func (b *Button) IconRight(icon string) *Button {
	b.iconRight = icon
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

// Reset sets the button type to reset.
func (b *Button) Reset() *Button {
	return b.Type("reset")
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
func (b *Button) WithAttrs(attrs *htmx.Attrs) InteractiveComponent {
	b.SetAttrs(attrs)
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
	attrs = append(attrs, fmt.Sprintf(`class="%s"`, escape(strings.Join(classes, " "))))

	if b.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(b.id)))
	}

	attrs = append(attrs, fmt.Sprintf(`type="%s"`, escape(b.type_)))

	if b.name != "" {
		attrs = append(attrs, fmt.Sprintf(`name="%s"`, escape(b.name)))
	}

	if b.value != "" {
		attrs = append(attrs, fmt.Sprintf(`value="%s"`, escape(b.value)))
	}

	if b.disabled || b.loading {
		attrs = append(attrs, "disabled")
	}

	if b.HasAttrs() {
		attrs = append(attrs, string(b.HXAttrs()))
	}

	var content strings.Builder

	if b.icon != "" {
		content.WriteString(fmt.Sprintf(`<span class="btn__icon">%s</span>`, b.icon))
	}

	if b.loading {
		content.WriteString(`<span class="btn__spinner"></span>`)
	}

	content.WriteString(fmt.Sprintf(`<span class="btn__text">%s</span>`, escape(b.text)))

	if b.iconRight != "" {
		content.WriteString(fmt.Sprintf(`<span class="btn__icon btn__icon--right">%s</span>`, b.iconRight))
	}

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

// Patch sets a PATCH action.
func (bh *ButtonHX) Patch(url string) *ButtonHX {
	bh.attrs.Patch(url)
	return bh
}

// Delete sets a DELETE action.
func (bh *ButtonHX) Delete(url string) *ButtonHX {
	bh.attrs.Delete(url)
	return bh
}

// Target sets the target selector.
func (bh *ButtonHX) Target(target htmx.Target) *ButtonHX {
	bh.attrs.Target(target)
	return bh
}

// TargetID sets a target by ID.
func (bh *ButtonHX) TargetID(id string) *ButtonHX {
	bh.attrs.TargetID(id)
	return bh
}

// TargetThis sets the target to the button itself.
func (bh *ButtonHX) TargetThis() *ButtonHX {
	bh.attrs.TargetThis()
	return bh
}

// Swap sets the swap strategy.
func (bh *ButtonHX) Swap(swap htmx.Swap) *ButtonHX {
	bh.attrs.Swap(swap)
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

// Trigger sets the trigger.
func (bh *ButtonHX) Trigger(trigger htmx.Trigger) *ButtonHX {
	bh.attrs.Trigger(trigger)
	return bh
}

// DisableDuring disables the button during the request.
func (bh *ButtonHX) DisableDuring() *ButtonHX {
	bh.attrs.DisableSelf()
	return bh
}

// Indicator sets the indicator element.
func (bh *ButtonHX) Indicator(selector string) *ButtonHX {
	bh.attrs.Indicator(selector)
	return bh
}

// WithVal adds a value to hx-vals.
func (bh *ButtonHX) WithVal(key string, value any) *ButtonHX {
	bh.attrs.WithVal(key, value)
	return bh
}

// WithVals adds multiple values to hx-vals.
func (bh *ButtonHX) WithVals(vals map[string]any) *ButtonHX {
	bh.attrs.WithVals(vals)
	return bh
}

// Done finalizes and returns the button.
func (bh *ButtonHX) Done() *Button {
	bh.button.SetAttrs(bh.attrs)
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
