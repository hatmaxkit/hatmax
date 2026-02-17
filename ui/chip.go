package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// Variant defines color variants for chips and labels.
type Variant string

const (
	VariantDefault   Variant = ""
	VariantPrimary   Variant = "primary"
	VariantSecondary Variant = "secondary"
	VariantSuccess   Variant = "success"
	VariantWarning   Variant = "warning"
	VariantDanger    Variant = "danger"
	VariantInfo      Variant = "info"
	VariantMuted     Variant = "muted"
)

// Chip renders a chip component with rounded laterals.
type Chip struct {
	text    string
	emoji   Emoji
	variant Variant
	class   string
}

// NewChip creates a new chip with the given text.
func NewChip(text string) *Chip {
	return &Chip{text: text}
}

// Emoji sets the chip emoji.
func (c *Chip) Emoji(e Emoji) *Chip {
	c.emoji = e
	return c
}

// Variant sets the chip variant.
func (c *Chip) Variant(v Variant) *Chip {
	c.variant = v
	return c
}

// Primary sets the chip to primary variant.
func (c *Chip) Primary() *Chip {
	return c.Variant(VariantPrimary)
}

// Secondary sets the chip to secondary variant.
func (c *Chip) Secondary() *Chip {
	return c.Variant(VariantSecondary)
}

// Success sets the chip to success variant.
func (c *Chip) Success() *Chip {
	return c.Variant(VariantSuccess)
}

// Warning sets the chip to warning variant.
func (c *Chip) Warning() *Chip {
	return c.Variant(VariantWarning)
}

// Danger sets the chip to danger variant.
func (c *Chip) Danger() *Chip {
	return c.Variant(VariantDanger)
}

// Info sets the chip to info variant.
func (c *Chip) Info() *Chip {
	return c.Variant(VariantInfo)
}

// Muted sets the chip to muted variant.
func (c *Chip) Muted() *Chip {
	return c.Variant(VariantMuted)
}

// Class adds custom CSS classes.
func (c *Chip) Class(class string) *Chip {
	c.class = class
	return c
}

// Render renders the chip to HTML.
func (c *Chip) Render() template.HTML {
	var classes []string
	classes = append(classes, "chip")

	if c.variant != "" {
		classes = append(classes, fmt.Sprintf("chip--%s", c.variant))
	}

	if c.class != "" {
		classes = append(classes, c.class)
	}

	var content strings.Builder
	if c.emoji != "" {
		fmt.Fprintf(&content, `<span class="chip__emoji">%s</span>`, c.emoji)
	}
	fmt.Fprintf(&content, `<span class="chip__text">%s</span>`, template.HTMLEscapeString(c.text))

	return template.HTML(fmt.Sprintf(`<span class="%s">%s</span>`,
		strings.Join(classes, " "),
		content.String(),
	))
}

// Label renders a label component with squared corners.
type Label struct {
	text    string
	emoji   Emoji
	variant Variant
	class   string
}

// NewLabel creates a new label with the given text.
func NewLabel(text string) *Label {
	return &Label{text: text}
}

// Emoji sets the label emoji.
func (l *Label) Emoji(e Emoji) *Label {
	l.emoji = e
	return l
}

// Variant sets the label variant.
func (l *Label) Variant(v Variant) *Label {
	l.variant = v
	return l
}

// Primary sets the label to primary variant.
func (l *Label) Primary() *Label {
	return l.Variant(VariantPrimary)
}

// Secondary sets the label to secondary variant.
func (l *Label) Secondary() *Label {
	return l.Variant(VariantSecondary)
}

// Success sets the label to success variant.
func (l *Label) Success() *Label {
	return l.Variant(VariantSuccess)
}

// Warning sets the label to warning variant.
func (l *Label) Warning() *Label {
	return l.Variant(VariantWarning)
}

// Danger sets the label to danger variant.
func (l *Label) Danger() *Label {
	return l.Variant(VariantDanger)
}

// Info sets the label to info variant.
func (l *Label) Info() *Label {
	return l.Variant(VariantInfo)
}

// Muted sets the label to muted variant.
func (l *Label) Muted() *Label {
	return l.Variant(VariantMuted)
}

// Class adds custom CSS classes.
func (l *Label) Class(class string) *Label {
	l.class = class
	return l
}

// Render renders the label to HTML.
func (l *Label) Render() template.HTML {
	var classes []string
	classes = append(classes, "label")

	if l.variant != "" {
		classes = append(classes, fmt.Sprintf("label--%s", l.variant))
	}

	if l.class != "" {
		classes = append(classes, l.class)
	}

	var content strings.Builder
	if l.emoji != "" {
		fmt.Fprintf(&content, `<span class="label__emoji">%s</span>`, l.emoji)
	}
	fmt.Fprintf(&content, `<span class="label__text">%s</span>`, template.HTMLEscapeString(l.text))

	return template.HTML(fmt.Sprintf(`<span class="%s">%s</span>`,
		strings.Join(classes, " "),
		content.String(),
	))
}
