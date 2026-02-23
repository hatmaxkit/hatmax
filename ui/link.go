package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// Link represents an interactive link component with optional HTMX support.
type Link struct {
	text     string
	href     string
	target   string
	rel      string
	class    string
	id       string
	title    string
	download string
	boosted  bool
	attrs    *htmx.Attrs
}

// NewLink creates a new link with the given text and href.
func NewLink(text, href string) *Link {
	return &Link{
		text: text,
		href: href,
	}
}

// Target sets the link target attribute.
func (l *Link) Target(target string) *Link {
	l.target = target

	return l
}

// Blank opens the link in a new tab with security attributes.
func (l *Link) Blank() *Link {
	l.target = "_blank"
	l.rel = "noopener noreferrer"

	return l
}

// External marks the link as external and adds appropriate rel attribute.
func (l *Link) External() *Link {
	if l.rel == "" {
		l.rel = "noopener noreferrer"
	}

	return l
}

// Rel sets the rel attribute.
func (l *Link) Rel(rel string) *Link {
	l.rel = rel

	return l
}

// Class adds custom CSS classes.
func (l *Link) Class(class string) *Link {
	l.class = class

	return l
}

// ID sets the link ID.
func (l *Link) ID(id string) *Link {
	l.id = id

	return l
}

// Title sets the title attribute.
func (l *Link) Title(title string) *Link {
	l.title = title

	return l
}

// Download sets the download attribute.
func (l *Link) Download(filename string) *Link {
	l.download = filename

	return l
}

// Boost enables hx-boost for the link.
func (l *Link) Boost() *Link {
	l.boosted = true

	return l
}

// WithAttrs sets HTMX attributes for the link.
func (l *Link) WithAttrs(attrs *htmx.Attrs) *Link {
	l.attrs = attrs

	return l
}

// HX starts building HTMX attributes for this link.
func (l *Link) HX() *LinkHX {
	return &LinkHX{link: l, attrs: htmx.NewAttrs()}
}

// Render renders the link to HTML.
func (l *Link) Render() template.HTML {
	var attrs []string

	attrs = append(attrs, fmt.Sprintf(`href="%s"`, template.HTMLEscapeString(l.href)))

	if l.class != "" {
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, template.HTMLEscapeString(l.class)))
	}

	if l.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, template.HTMLEscapeString(l.id)))
	}

	if l.target != "" {
		attrs = append(attrs, fmt.Sprintf(`target="%s"`, template.HTMLEscapeString(l.target)))
	}

	if l.rel != "" {
		attrs = append(attrs, fmt.Sprintf(`rel="%s"`, template.HTMLEscapeString(l.rel)))
	}

	if l.title != "" {
		attrs = append(attrs, fmt.Sprintf(`title="%s"`, template.HTMLEscapeString(l.title)))
	}

	if l.download != "" {
		attrs = append(attrs, fmt.Sprintf(`download="%s"`, template.HTMLEscapeString(l.download)))
	}

	if l.boosted {
		attrs = append(attrs, `hx-boost="true"`)
	}

	if l.attrs != nil {
		attrs = append(attrs, string(l.attrs.HTML()))
	}

	return template.HTML(fmt.Sprintf(`<a %s>%s</a>`, strings.Join(attrs, " "), template.HTMLEscapeString(l.text)))
}

// LinkHX provides a fluent interface for building HTMX-enabled links.
type LinkHX struct {
	link  *Link
	attrs *htmx.Attrs
}

// Get sets a GET action.
func (lh *LinkHX) Get(url string) *LinkHX {
	lh.attrs.Get(url)

	return lh
}

// Post sets a POST action.
func (lh *LinkHX) Post(url string) *LinkHX {
	lh.attrs.Post(url)

	return lh
}

// TargetID sets a target by ID.
func (lh *LinkHX) TargetID(id string) *LinkHX {
	lh.attrs.TargetID(id)

	return lh
}

// SwapOuter sets outer HTML swap.
func (lh *LinkHX) SwapOuter() *LinkHX {
	lh.attrs.SwapOuter()

	return lh
}

// SwapInner sets inner HTML swap.
func (lh *LinkHX) SwapInner() *LinkHX {
	lh.attrs.SwapInner()

	return lh
}

// PushURL enables pushing the URL to history.
func (lh *LinkHX) PushURL(url string) *LinkHX {
	lh.attrs.PushURL(url)

	return lh
}

// Select sets the selector for partial content.
func (lh *LinkHX) Select(selector string) *LinkHX {
	lh.attrs.Select(selector)

	return lh
}

// Done finalizes and returns the link.
func (lh *LinkHX) Done() *Link {
	lh.link.attrs = lh.attrs

	return lh.link
}

// Render renders the link to HTML.
func (lh *LinkHX) Render() template.HTML {
	return lh.Done().Render()
}

// A is a shorthand to create a link.
func A(text, href string) *Link {
	return NewLink(text, href)
}

// ABlank creates an external link that opens in a new tab.
func ABlank(text, href string) *Link {
	return NewLink(text, href).Blank()
}

// ABoosted creates a boosted link.
func ABoosted(text, href string) *Link {
	return NewLink(text, href).Boost()
}
