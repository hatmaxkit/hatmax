package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// Renderable can render itself to HTML.
type Renderable interface {
	Render() template.HTML
}

// Page represents a full page layout.
type Page struct {
	title       string
	description string
	header      Renderable
	content     template.HTML
	footer      Renderable
	class       string
}

// NewPage creates a new page with the given title.
func NewPage(title string) *Page {
	return &Page{title: title}
}

// Description sets the page description.
func (p *Page) Description(desc string) *Page {
	p.description = desc
	return p
}

// Header sets the page header component.
func (p *Page) Header(h Renderable) *Page {
	p.header = h
	return p
}

// Content sets the page content.
func (p *Page) Content(c template.HTML) *Page {
	p.content = c
	return p
}

// Footer sets the page footer component.
func (p *Page) Footer(f Renderable) *Page {
	p.footer = f
	return p
}

// Class adds custom CSS classes.
func (p *Page) Class(class string) *Page {
	p.class = class
	return p
}

// Title returns the page title.
func (p *Page) Title() string {
	return p.title
}

// Render renders the page to HTML.
func (p *Page) Render() template.HTML {
	var classes []string
	classes = append(classes, "page")
	if p.class != "" {
		classes = append(classes, p.class)
	}

	var html strings.Builder
	fmt.Fprintf(&html, `<div class="%s">`, strings.Join(classes, " "))

	if p.header != nil {
		html.WriteString(string(p.header.Render()))
	}

	fmt.Fprintf(&html, `<main class="page__content">%s</main>`, p.content)

	if p.footer != nil {
		html.WriteString(string(p.footer.Render()))
	}

	html.WriteString(`</div>`)

	return template.HTML(html.String())
}

// PageHeader renders a page title section.
type PageHeader struct {
	title       string
	subtitle    string
	actions     []Renderable
	breadcrumbs []Breadcrumb
	class       string
}

// Breadcrumb represents a breadcrumb item.
type Breadcrumb struct {
	Label string
	Href  string
}

// NewPageHeader creates a new page header.
func NewPageHeader(title string) *PageHeader {
	return &PageHeader{title: title}
}

// Subtitle sets the header subtitle.
func (h *PageHeader) Subtitle(sub string) *PageHeader {
	h.subtitle = sub
	return h
}

// Actions sets the header action buttons.
func (h *PageHeader) Actions(actions ...Renderable) *PageHeader {
	h.actions = actions
	return h
}

// Breadcrumbs sets the breadcrumb trail.
func (h *PageHeader) Breadcrumbs(crumbs ...Breadcrumb) *PageHeader {
	h.breadcrumbs = crumbs
	return h
}

// Class adds custom CSS classes.
func (h *PageHeader) Class(class string) *PageHeader {
	h.class = class
	return h
}

// Render renders the page header to HTML.
func (h *PageHeader) Render() template.HTML {
	var classes []string
	classes = append(classes, "page-header")
	if h.class != "" {
		classes = append(classes, h.class)
	}

	var html strings.Builder
	fmt.Fprintf(&html, `<header class="%s">`, strings.Join(classes, " "))

	if len(h.breadcrumbs) > 0 {
		html.WriteString(`<nav class="page-header__breadcrumbs">`)
		for i, crumb := range h.breadcrumbs {
			if i > 0 {
				html.WriteString(`<span class="page-header__separator">/</span>`)
			}
			if crumb.Href != "" {
				fmt.Fprintf(&html, `<a href="%s">%s</a>`,
					template.HTMLEscapeString(crumb.Href),
					template.HTMLEscapeString(crumb.Label))
			} else {
				fmt.Fprintf(&html, `<span>%s</span>`, template.HTMLEscapeString(crumb.Label))
			}
		}
		html.WriteString(`</nav>`)
	}

	html.WriteString(`<div class="page-header__main">`)
	html.WriteString(`<div class="page-header__titles">`)
	fmt.Fprintf(&html, `<h1 class="page-header__title">%s</h1>`, template.HTMLEscapeString(h.title))
	if h.subtitle != "" {
		fmt.Fprintf(&html, `<p class="page-header__subtitle">%s</p>`, template.HTMLEscapeString(h.subtitle))
	}
	html.WriteString(`</div>`)

	if len(h.actions) > 0 {
		html.WriteString(`<div class="page-header__actions">`)
		for _, action := range h.actions {
			html.WriteString(string(action.Render()))
		}
		html.WriteString(`</div>`)
	}

	html.WriteString(`</div>`)
	html.WriteString(`</header>`)

	return template.HTML(html.String())
}

// Container wraps content with standard padding/margins.
type Container struct {
	content template.HTML
	class   string
	fluid   bool
}

// NewContainer creates a new container.
func NewContainer() *Container {
	return &Container{}
}

// Content sets the container content.
func (c *Container) Content(content template.HTML) *Container {
	c.content = content
	return c
}

// Class adds custom CSS classes.
func (c *Container) Class(class string) *Container {
	c.class = class
	return c
}

// Fluid makes the container full-width.
func (c *Container) Fluid() *Container {
	c.fluid = true
	return c
}

// Render renders the container to HTML.
func (c *Container) Render() template.HTML {
	var classes []string
	classes = append(classes, "container")
	if c.fluid {
		classes = append(classes, "container--fluid")
	}
	if c.class != "" {
		classes = append(classes, c.class)
	}

	return template.HTML(fmt.Sprintf(`<div class="%s">%s</div>`,
		strings.Join(classes, " "),
		c.content,
	))
}
