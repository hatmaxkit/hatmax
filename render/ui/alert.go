package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// AlertVariant defines alert style variants.
type AlertVariant string

const (
	AlertInfo    AlertVariant = "info"
	AlertSuccess AlertVariant = "success"
	AlertWarning AlertVariant = "warning"
	AlertDanger  AlertVariant = "danger"
	AlertError   AlertVariant = "danger" // alias
)

// Alert represents an alert/notification component.
type Alert struct {
	BaseInteractive
	message     string
	title       string
	variant     AlertVariant
	dismissible bool
	icon        string
	class       string
	id          string
}

// NewAlert creates a new alert with the given message.
func NewAlert(message string) *Alert {
	return &Alert{
		message: message,
		variant: AlertInfo,
	}
}

// Title sets the alert title.
func (a *Alert) Title(title string) *Alert {
	a.title = title
	return a
}

// Variant sets the alert variant.
func (a *Alert) Variant(v AlertVariant) *Alert {
	a.variant = v
	return a
}

// Info sets the alert to info variant.
func (a *Alert) Info() *Alert {
	return a.Variant(AlertInfo)
}

// Success sets the alert to success variant.
func (a *Alert) Success() *Alert {
	return a.Variant(AlertSuccess)
}

// Warning sets the alert to warning variant.
func (a *Alert) Warning() *Alert {
	return a.Variant(AlertWarning)
}

// Danger sets the alert to danger variant.
func (a *Alert) Danger() *Alert {
	return a.Variant(AlertDanger)
}

// Error sets the alert to error (danger) variant.
func (a *Alert) Error() *Alert {
	return a.Variant(AlertError)
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

// WithAttrs sets HTMX attributes for the alert.
func (a *Alert) WithAttrs(attrs *htmx.Attrs) InteractiveComponent {
	a.SetAttrs(attrs)
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
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(a.id)))
	}

	if a.HasAttrs() {
		attrs = append(attrs, string(a.HXAttrs()))
	}

	var content strings.Builder

	if a.icon != "" {
		content.WriteString(fmt.Sprintf(`<span class="alert__icon">%s</span>`, a.icon))
	}

	content.WriteString(`<div class="alert__content">`)

	if a.title != "" {
		content.WriteString(fmt.Sprintf(`<strong class="alert__title">%s</strong>`, escape(a.title)))
	}

	content.WriteString(fmt.Sprintf(`<span class="alert__message">%s</span>`, escape(a.message)))
	content.WriteString(`</div>`)

	if a.dismissible {
		content.WriteString(`<button type="button" class="alert__dismiss" aria-label="Dismiss">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// AlertDismissible creates an alert with the ability to dismiss via HTMX.
type AlertDismissible struct {
	Alert
	dismissURL string
}

// NewAlertDismissible creates a new dismissible alert.
func NewAlertDismissible(message string) *AlertDismissible {
	return &AlertDismissible{
		Alert: Alert{
			message:     message,
			variant:     AlertInfo,
			dismissible: true,
		},
	}
}

// DismissURL sets the URL to call when dismissing.
func (a *AlertDismissible) DismissURL(url string) *AlertDismissible {
	a.dismissURL = url
	return a
}

// ID sets the alert ID.
func (a *AlertDismissible) ID(id string) *AlertDismissible {
	a.id = id
	return a
}

// Title sets the alert title.
func (a *AlertDismissible) Title(title string) *AlertDismissible {
	a.title = title
	return a
}

// Variant sets the alert variant.
func (a *AlertDismissible) Variant(v AlertVariant) *AlertDismissible {
	a.variant = v
	return a
}

// Info sets the alert to info variant.
func (a *AlertDismissible) Info() *AlertDismissible {
	return a.Variant(AlertInfo)
}

// Success sets the alert to success variant.
func (a *AlertDismissible) Success() *AlertDismissible {
	return a.Variant(AlertSuccess)
}

// Warning sets the alert to warning variant.
func (a *AlertDismissible) Warning() *AlertDismissible {
	return a.Variant(AlertWarning)
}

// Danger sets the alert to danger variant.
func (a *AlertDismissible) Danger() *AlertDismissible {
	return a.Variant(AlertDanger)
}

// Error sets the alert to error (danger) variant.
func (a *AlertDismissible) Error() *AlertDismissible {
	return a.Variant(AlertError)
}

// Icon sets the alert icon.
func (a *AlertDismissible) Icon(icon string) *AlertDismissible {
	a.icon = icon
	return a
}

// Class adds custom CSS classes.
func (a *AlertDismissible) Class(class string) *AlertDismissible {
	a.class = class
	return a
}

// Render renders the dismissible alert with HTMX support.
func (a *AlertDismissible) Render() template.HTML {
	var classes []string
	classes = append(classes, "alert")

	if a.variant != "" {
		classes = append(classes, fmt.Sprintf("alert--%s", a.variant))
	}

	classes = append(classes, "alert--dismissible")

	if a.class != "" {
		classes = append(classes, a.class)
	}

	var attrs []string
	attrs = append(attrs, fmt.Sprintf(`class="%s"`, strings.Join(classes, " ")))
	attrs = append(attrs, `role="alert"`)

	if a.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(a.id)))
	}

	var content strings.Builder

	if a.icon != "" {
		content.WriteString(fmt.Sprintf(`<span class="alert__icon">%s</span>`, a.icon))
	}

	content.WriteString(`<div class="alert__content">`)

	if a.title != "" {
		content.WriteString(fmt.Sprintf(`<strong class="alert__title">%s</strong>`, escape(a.title)))
	}

	content.WriteString(fmt.Sprintf(`<span class="alert__message">%s</span>`, escape(a.message)))
	content.WriteString(`</div>`)

	// Dismiss button with HTMX
	dismissAttrs := []string{
		`type="button"`,
		`class="alert__dismiss"`,
		`aria-label="Dismiss"`,
	}

	if a.dismissURL != "" {
		dismissAttrs = append(dismissAttrs,
			fmt.Sprintf(`hx-delete="%s"`, escape(a.dismissURL)),
			`hx-swap="outerHTML"`,
		)
		if a.id != "" {
			dismissAttrs = append(dismissAttrs, fmt.Sprintf(`hx-target="#%s"`, escape(a.id)))
		} else {
			dismissAttrs = append(dismissAttrs, `hx-target="closest .alert"`)
		}
	}

	content.WriteString(fmt.Sprintf(`<button %s>×</button>`, strings.Join(dismissAttrs, " ")))

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// Flash represents a flash message (similar to Alert but with auto-dismiss).
type Flash struct {
	Alert
	autoDismiss int // seconds, 0 = no auto-dismiss
}

// NewFlash creates a new flash message.
func NewFlash(message string) *Flash {
	return &Flash{
		Alert: Alert{
			message:     message,
			variant:     AlertInfo,
			dismissible: true,
		},
	}
}

// AutoDismiss sets auto-dismiss timeout in seconds.
func (f *Flash) AutoDismiss(seconds int) *Flash {
	f.autoDismiss = seconds
	return f
}

// Variant sets the flash variant.
func (f *Flash) Variant(v AlertVariant) *Flash {
	f.variant = v
	return f
}

// Info sets the flash to info variant.
func (f *Flash) Info() *Flash {
	return f.Variant(AlertInfo)
}

// Success sets the flash to success variant.
func (f *Flash) Success() *Flash {
	return f.Variant(AlertSuccess)
}

// Warning sets the flash to warning variant.
func (f *Flash) Warning() *Flash {
	return f.Variant(AlertWarning)
}

// Danger sets the flash to danger variant.
func (f *Flash) Danger() *Flash {
	return f.Variant(AlertDanger)
}

// Error sets the flash to error (danger) variant.
func (f *Flash) Error() *Flash {
	return f.Variant(AlertError)
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
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(f.id)))
	}

	// Auto-dismiss via HTMX if set
	if f.autoDismiss > 0 {
		attrs = append(attrs,
			fmt.Sprintf(`hx-get="data:," hx-trigger="load delay:%ds" hx-swap="outerHTML"`, f.autoDismiss),
		)
	}

	var content strings.Builder

	if f.icon != "" {
		content.WriteString(fmt.Sprintf(`<span class="flash__icon">%s</span>`, f.icon))
	}

	content.WriteString(fmt.Sprintf(`<span class="flash__message">%s</span>`, escape(f.message)))

	if f.dismissible {
		content.WriteString(`<button type="button" class="flash__dismiss" onclick="this.parentElement.remove()">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// Toast represents a toast notification.
type Toast struct {
	Alert
	position string // top-right, top-left, bottom-right, bottom-left
	duration int    // auto-dismiss duration in ms
}

// NewToast creates a new toast notification.
func NewToast(message string) *Toast {
	return &Toast{
		Alert: Alert{
			message:     message,
			variant:     AlertInfo,
			dismissible: true,
		},
		position: "top-right",
		duration: 5000,
	}
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

// Title sets the toast title.
func (t *Toast) Title(title string) *Toast {
	t.title = title
	return t
}

// Variant sets the toast variant.
func (t *Toast) Variant(v AlertVariant) *Toast {
	t.variant = v
	return t
}

// Info sets the toast to info variant.
func (t *Toast) Info() *Toast {
	return t.Variant(AlertInfo)
}

// Success sets the toast to success variant.
func (t *Toast) Success() *Toast {
	return t.Variant(AlertSuccess)
}

// Warning sets the toast to warning variant.
func (t *Toast) Warning() *Toast {
	return t.Variant(AlertWarning)
}

// Danger sets the toast to danger variant.
func (t *Toast) Danger() *Toast {
	return t.Variant(AlertDanger)
}

// Error sets the toast to error (danger) variant.
func (t *Toast) Error() *Toast {
	return t.Variant(AlertError)
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
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(t.id)))
	}

	if t.duration > 0 {
		attrs = append(attrs, fmt.Sprintf(`data-duration="%d"`, t.duration))
	}

	var content strings.Builder

	if t.icon != "" {
		content.WriteString(fmt.Sprintf(`<span class="toast__icon">%s</span>`, t.icon))
	}

	content.WriteString(`<div class="toast__content">`)

	if t.title != "" {
		content.WriteString(fmt.Sprintf(`<strong class="toast__title">%s</strong>`, escape(t.title)))
	}

	content.WriteString(fmt.Sprintf(`<span class="toast__message">%s</span>`, escape(t.message)))
	content.WriteString(`</div>`)

	if t.dismissible {
		content.WriteString(`<button type="button" class="toast__dismiss" aria-label="Dismiss">×</button>`)
	}

	return template.HTML(fmt.Sprintf(`<div %s>%s</div>`, strings.Join(attrs, " "), content.String()))
}

// Shorthand constructors

// AlertInfo creates an info alert.
func AlertInfoMsg(message string) *Alert {
	return NewAlert(message).Info()
}

// AlertSuccess creates a success alert.
func AlertSuccessMsg(message string) *Alert {
	return NewAlert(message).Success()
}

// AlertWarning creates a warning alert.
func AlertWarningMsg(message string) *Alert {
	return NewAlert(message).Warning()
}

// AlertDangerMsg creates a danger alert.
func AlertDangerMsg(message string) *Alert {
	return NewAlert(message).Danger()
}

// AlertErrorMsg creates an error alert.
func AlertErrorMsg(message string) *Alert {
	return NewAlert(message).Error()
}
