package ui

import (
	"html/template"

	"github.com/hatmaxkit/hatmax/htmx"
)

// InteractiveComponent extends Component with HTMX interaction capabilities.
type InteractiveComponent interface {
	Component
	WithAttrs(attrs *htmx.Attrs) InteractiveComponent
}

// BaseInteractive provides common HTMX functionality for interactive components.
type BaseInteractive struct {
	attrs *htmx.Attrs
}

// HXAttrs returns the HTMX attributes for rendering.
func (b *BaseInteractive) HXAttrs() template.HTMLAttr {
	if b.attrs == nil {
		return ""
	}
	return b.attrs.HTML()
}

// HasAttrs returns true if HTMX attributes are set.
func (b *BaseInteractive) HasAttrs() bool {
	return b.attrs != nil
}

// SetAttrs sets the HTMX attributes.
func (b *BaseInteractive) SetAttrs(attrs *htmx.Attrs) {
	b.attrs = attrs
}
