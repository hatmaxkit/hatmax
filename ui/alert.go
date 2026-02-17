package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// Alert represents an alert/notification component.
type Alert struct {
	message     string
	title       string
	variant     Variant
	dismissible bool
	icon        string
	class       string
	id          string
}

// NewAlert creates a new alert with the given message.
func NewAlert(message string) *Alert {
	return &Alert{
		message: message,
		variant: VariantInfo,
	}
}

// Title sets the alert title.
func (a *Alert) Title(title string) *Alert {
	a.title = title
	return a
}

// Variant sets the alert variant.
func (a *Alert) Variant(v Variant) *Alert {
	a.variant = v
	return a
}

// Info sets the alert to info variant.
func (a *Alert) Info() *Alert {
	return a.Variant(VariantInfo)
}

// Success sets the alert to success variant.
func (a *Alert) Success() *Alert {
	return a.Variant(VariantSuccess)
}

// Warning sets the alert to warning variant.
func (a *Alert) Warning() *Alert {
	return a.Variant(VariantWarning)
}

// Danger sets the alert to danger variant.
func (a *Alert) Danger() *Alert {
	return a.Variant(VariantDanger)
}

// Error sets the alert to error (danger) variant.
func (a *Alert) Error() *Alert {
	return a.Variant(VariantDanger)
}

// Dismissible makes the alert dismissible.
func (a *Alert) Dismissible() *Alert {
	a.dismissible = true
	return a
}

// Icon sets the alert icon.
func (a *Alert) Icon(icon string) *Alert {
	a.icon = icon
	return a
}

// Class adds custom CSS classes.
func (a *Alert) Class(class string) *Alert {
	a.class = class
	return a
}

// ID sets the alert ID.
func (a *Alert) ID(id string) *Alert {
	a.id = id
	return a
}

// Render renders the alert to HTML.
func (a *Alert) Render() template.HTML {
	var classes []string
	classes = append(classes, "alert")

	if a.variant != "" {
		classes = append(classes, fmt.Sprintf("alert--%s", a.variant))
	}

	if a.dismissible {
		classes = append(classes, "alert--dismissible")
	}

	if a.class != "" {
		classes = append(classes, a.class)
	}

	var attrs []string
	attrs = append(attrs, fmt.Sprintf(`class="%s"`, strings.Join(classes, " ")))
	attrs = append(attrs, `role="alert"`)

	if a.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(a.id)))
	}

	var content strings.Builder

	if a.icon != "" {
		fmt.Fprintf(&content, `<span class="alert__icon">%s</span>`, a.icon)
	}

	content.WriteString(`<div class="alert__content">`)

	if a.title != "" {
		fmt.Fprintf(&content, `<strong class="alert__title">%s</strong>`, template.HTMLEscapeString(a.title))
	}

	fmt.Fprintf(&content, `<span class="alert__message">%s</span>`, template.HTMLEscapeString(a.message))
	content.WriteString(`</div>`)

	if a.dismissible {
		content.WriteString(`<button type="button" class="alert__dismiss" aria-label="Dismiss">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// Flash represents a flash message with auto-dismiss capability.
type Flash struct {
	message     string
	variant     Variant
	dismissible bool
	icon        string
	class       string
	id          string
	autoDismiss int
}

// NewFlash creates a new flash message.
func NewFlash(message string) *Flash {
	return &Flash{
		message:     message,
		variant:     VariantInfo,
		dismissible: true,
	}
}

// Variant sets the flash variant.
func (f *Flash) Variant(v Variant) *Flash {
	f.variant = v
	return f
}

// Info sets the flash to info variant.
func (f *Flash) Info() *Flash {
	return f.Variant(VariantInfo)
}

// Success sets the flash to success variant.
func (f *Flash) Success() *Flash {
	return f.Variant(VariantSuccess)
}

// Warning sets the flash to warning variant.
func (f *Flash) Warning() *Flash {
	return f.Variant(VariantWarning)
}

// Danger sets the flash to danger variant.
func (f *Flash) Danger() *Flash {
	return f.Variant(VariantDanger)
}

// Error sets the flash to error (danger) variant.
func (f *Flash) Error() *Flash {
	return f.Variant(VariantDanger)
}

// AutoDismiss sets auto-dismiss timeout in seconds.
func (f *Flash) AutoDismiss(seconds int) *Flash {
	f.autoDismiss = seconds
	return f
}

// Icon sets the flash icon.
func (f *Flash) Icon(icon string) *Flash {
	f.icon = icon
	return f
}

// ID sets the flash ID.
func (f *Flash) ID(id string) *Flash {
	f.id = id
	return f
}

// Class adds custom CSS classes.
func (f *Flash) Class(class string) *Flash {
	f.class = class
	return f
}

// Render renders the flash message.
func (f *Flash) Render() template.HTML {
	var classes []string
	classes = append(classes, "flash")

	if f.variant != "" {
		classes = append(classes, fmt.Sprintf("flash--%s", f.variant))
	}

	if f.class != "" {
		classes = append(classes, f.class)
	}

	var attrs []string
	attrs = append(attrs, fmt.Sprintf(`class="%s"`, strings.Join(classes, " ")))
	attrs = append(attrs, `role="alert"`)

	if f.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(f.id)))
	}

	if f.autoDismiss > 0 {
		attrs = append(attrs,
			fmt.Sprintf(`hx-get="data:," hx-trigger="load delay:%ds" hx-swap="outerHTML"`, f.autoDismiss),
		)
	}

	var content strings.Builder

	if f.icon != "" {
		fmt.Fprintf(&content, `<span class="flash__icon">%s</span>`, f.icon)
	}

	fmt.Fprintf(&content, `<span class="flash__message">%s</span>`, template.HTMLEscapeString(f.message))

	if f.dismissible {
		content.WriteString(`<button type="button" class="flash__dismiss" onclick="this.parentElement.remove()">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// Toast represents a toast notification with positioning.
type Toast struct {
	message     string
	title       string
	variant     Variant
	dismissible bool
	icon        string
	class       string
	id          string
	position    string
	duration    int
}

// NewToast creates a new toast notification.
func NewToast(message string) *Toast {
	return &Toast{
		message:     message,
		variant:     VariantInfo,
		dismissible: true,
		position:    "top-right",
		duration:    5000,
	}
}

// Title sets the toast title.
func (t *Toast) Title(title string) *Toast {
	t.title = title
	return t
}

// Variant sets the toast variant.
func (t *Toast) Variant(v Variant) *Toast {
	t.variant = v
	return t
}

// Info sets the toast to info variant.
func (t *Toast) Info() *Toast {
	return t.Variant(VariantInfo)
}

// Success sets the toast to success variant.
func (t *Toast) Success() *Toast {
	return t.Variant(VariantSuccess)
}

// Warning sets the toast to warning variant.
func (t *Toast) Warning() *Toast {
	return t.Variant(VariantWarning)
}

// Danger sets the toast to danger variant.
func (t *Toast) Danger() *Toast {
	return t.Variant(VariantDanger)
}

// Error sets the toast to error (danger) variant.
func (t *Toast) Error() *Toast {
	return t.Variant(VariantDanger)
}

// Position sets the toast position.
func (t *Toast) Position(position string) *Toast {
	t.position = position
	return t
}

// Duration sets the auto-dismiss duration in milliseconds.
func (t *Toast) Duration(ms int) *Toast {
	t.duration = ms
	return t
}

// Persistent makes the toast stay until dismissed.
func (t *Toast) Persistent() *Toast {
	t.duration = 0
	return t
}

// Icon sets the toast icon.
func (t *Toast) Icon(icon string) *Toast {
	t.icon = icon
	return t
}

// ID sets the toast ID.
func (t *Toast) ID(id string) *Toast {
	t.id = id
	return t
}

// Class adds custom CSS classes.
func (t *Toast) Class(class string) *Toast {
	t.class = class
	return t
}

// Render renders the toast notification.
func (t *Toast) Render() template.HTML {
	var classes []string
	classes = append(classes, "toast")

	if t.variant != "" {
		classes = append(classes, fmt.Sprintf("toast--%s", t.variant))
	}

	if t.position != "" {
		classes = append(classes, fmt.Sprintf("toast--%s", t.position))
	}

	if t.class != "" {
		classes = append(classes, t.class)
	}

	var attrs []string
	attrs = append(attrs, fmt.Sprintf(`class="%s"`, strings.Join(classes, " ")))
	attrs = append(attrs, `role="alert"`)

	if t.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(t.id)))
	}

	if t.duration > 0 {
		attrs = append(attrs, fmt.Sprintf(`data-duration="%d"`, t.duration))
	}

	var content strings.Builder

	if t.icon != "" {
		fmt.Fprintf(&content, `<span class="toast__icon">%s</span>`, t.icon)
	}

	content.WriteString(`<div class="toast__content">`)

	if t.title != "" {
		fmt.Fprintf(&content, `<strong class="toast__title">%s</strong>`, template.HTMLEscapeString(t.title))
	}

	fmt.Fprintf(&content, `<span class="toast__message">%s</span>`, template.HTMLEscapeString(t.message))
	content.WriteString(`</div>`)

	if t.dismissible {
		content.WriteString(`<button type="button" class="toast__dismiss" aria-label="Dismiss">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// AlertInfo creates an info alert.
func AlertInfo(message string) *Alert {
	return NewAlert(message).Info()
}

// AlertSuccess creates a success alert.
func AlertSuccess(message string) *Alert {
	return NewAlert(message).Success()
}

// AlertWarning creates a warning alert.
func AlertWarning(message string) *Alert {
	return NewAlert(message).Warning()
}

// AlertDanger creates a danger alert.
func AlertDanger(message string) *Alert {
	return NewAlert(message).Danger()
}

// AlertError creates an error alert.
func AlertError(message string) *Alert {
	return NewAlert(message).Error()
}
