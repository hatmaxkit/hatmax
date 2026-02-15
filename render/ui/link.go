package ui

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/hatmaxkit/hatmax/htmx"
)

// Link represents an interactive link component with optional HTMX support.
type Link struct {
	BaseInteractive
	text     string
	href     string
	target   string
	rel      string
	class    string
	id       string
	title    string
	download string
	boosted  bool
	external bool
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

// Blank opens the link in a new tab.
func (l *Link) Blank() *Link {
	l.target = "_blank"
	l.rel = "noopener noreferrer"
	l.external = true
	return l
}

// External marks the link as external and adds appropriate rel attribute.
func (l *Link) External() *Link {
	l.external = true
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
func (l *Link) WithAttrs(attrs *htmx.Attrs) InteractiveComponent {
	l.SetAttrs(attrs)
	return l
}

// HX starts building HTMX attributes for this link.
func (l *Link) HX() *LinkHX {
	return &LinkHX{link: l, attrs: htmx.NewAttrs()}
}

// Render renders the link to HTML.
func (l *Link) Render() template.HTML {
	var attrs []string

	attrs = append(attrs, fmt.Sprintf(`href="%s"`, escape(l.href)))

	if l.class != "" {
		attrs = append(attrs, fmt.Sprintf(`class="%s"`, escape(l.class)))
	}

	if l.id != "" {
		attrs = append(attrs, fmt.Sprintf(`id="%s"`, escape(l.id)))
	}

	if l.target != "" {
		attrs = append(attrs, fmt.Sprintf(`target="%s"`, escape(l.target)))
	}

	if l.rel != "" {
		attrs = append(attrs, fmt.Sprintf(`rel="%s"`, escape(l.rel)))
	}

	if l.title != "" {
		attrs = append(attrs, fmt.Sprintf(`title="%s"`, escape(l.title)))
	}

	if l.download != "" {
		attrs = append(attrs, fmt.Sprintf(`download="%s"`, escape(l.download)))
	}

	if l.boosted {
		attrs = append(attrs, `hx-boost="true"`)
	}

	if l.HasAttrs() {
		attrs = append(attrs, string(l.HXAttrs()))
	}

	return template.HTML(fmt.Sprintf(`<a %s>%s</a>`, strings.Join(attrs, " "), escape(l.text)))
}

// LinkHX provides a fluent interface for building HTMX-enabled links.
type LinkHX struct {
	link  *Link
	attrs *htmx.Attrs
}

// Get sets a GET action (replacing the href behavior).
func (lh *LinkHX) Get(url string) *LinkHX {
	lh.attrs.Get(url)
	return lh
}

// Post sets a POST action.
func (lh *LinkHX) Post(url string) *LinkHX {
	lh.attrs.Post(url)
	return lh
}

// Target sets the HTMX target selector.
func (lh *LinkHX) Target(target htmx.Target) *LinkHX {
	lh.attrs.Target(target)
	return lh
}

// TargetID sets a target by ID.
func (lh *LinkHX) TargetID(id string) *LinkHX {
	lh.attrs.TargetID(id)
	return lh
}

// Swap sets the swap strategy.
func (lh *LinkHX) Swap(swap htmx.Swap) *LinkHX {
	lh.attrs.Swap(swap)
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

// PushHref enables pushing the link's href to history.
func (lh *LinkHX) PushHref() *LinkHX {
	lh.attrs.PushURL(lh.link.href)
	return lh
}

// Select sets the selector for partial content.
func (lh *LinkHX) Select(selector string) *LinkHX {
	lh.attrs.Select(selector)
	return lh
}

// Trigger sets the trigger.
func (lh *LinkHX) Trigger(trigger htmx.Trigger) *LinkHX {
	lh.attrs.Trigger(trigger)
	return lh
}

// Done finalizes and returns the link.
func (lh *LinkHX) Done() *Link {
	lh.link.SetAttrs(lh.attrs)
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

// NavLink represents a navigation link with active state.
type NavLink struct {
	Link
	active bool
}

// NewNavLink creates a new navigation link.
func NewNavLink(text, href string) *NavLink {
	return &NavLink{
		Link: Link{
			text: text,
			href: href,
		},
	}
}

// Active sets the link as active.
func (n *NavLink) Active(active bool) *NavLink {
	n.active = active
	return n
}

// Render renders the navigation link to HTML.
func (n *NavLink) Render() template.HTML {
	var classes []string
	classes = append(classes, "nav-link")

	if n.active {
		classes = append(classes, "nav-link--active")
	}

	if n.class != "" {
		classes = append(classes, n.class)
	}

	n.class = strings.Join(classes, " ")
	return n.Link.Render()
}

// Nav creates a navigation link.
func Nav(text, href string) *NavLink {
	return NewNavLink(text, href)
}
