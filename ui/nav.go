package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// NavGrid renders a grid navigation with icon and label cards.
type NavGrid struct {
	items []NavItem
	cols  int
	class string
}

// NavItem represents a navigation item.
type NavItem struct {
	Emoji Emoji
	Label string
	Href  string
	Badge string
}

// NewNavGrid creates a new navigation grid.
func NewNavGrid() *NavGrid {
	return &NavGrid{cols: 3}
}

// Items sets the navigation items.
func (n *NavGrid) Items(items ...NavItem) *NavGrid {
	n.items = items
	return n
}

// AddItem adds a navigation item.
func (n *NavGrid) AddItem(emoji Emoji, label, href string) *NavGrid {
	n.items = append(n.items, NavItem{Emoji: emoji, Label: label, Href: href})
	return n
}

// AddItemWithBadge adds a navigation item with a badge.
func (n *NavGrid) AddItemWithBadge(emoji Emoji, label, href, badge string) *NavGrid {
	n.items = append(n.items, NavItem{Emoji: emoji, Label: label, Href: href, Badge: badge})
	return n
}

// Cols sets the number of columns.
func (n *NavGrid) Cols(cols int) *NavGrid {
	n.cols = cols
	return n
}

// Class adds custom CSS classes.
func (n *NavGrid) Class(class string) *NavGrid {
	n.class = class
	return n
}

// Render renders the navigation grid to HTML.
func (n *NavGrid) Render() template.HTML {
	var classes []string
	classes = append(classes, "nav-grid")
	if n.class != "" {
		classes = append(classes, n.class)
	}

	var html strings.Builder
	fmt.Fprintf(&html, `<nav class="%s" style="--nav-cols: %d">`, strings.Join(classes, " "), n.cols)

	for _, item := range n.items {
		fmt.Fprintf(&html, `<a href="%s" class="nav-grid__item">`, template.HTMLEscapeString(item.Href))

		if item.Emoji != "" {
			fmt.Fprintf(&html, `<span class="nav-grid__emoji">%s</span>`, item.Emoji)
		}

		fmt.Fprintf(&html, `<span class="nav-grid__label">%s</span>`, template.HTMLEscapeString(item.Label))

		if item.Badge != "" {
			fmt.Fprintf(&html, `<span class="nav-grid__badge">%s</span>`, template.HTMLEscapeString(item.Badge))
		}

		html.WriteString(`</a>`)
	}

	html.WriteString(`</nav>`)

	return template.HTML(html.String())
}

// Nav represents a navigation menu.
type Nav struct {
	items   []NavLink
	class   string
	vertical bool
}

// NavLink represents a navigation link.
type NavLink struct {
	Label  string
	Href   string
	Active bool
	Emoji  Emoji
}

// NewNav creates a new navigation menu.
func NewNav() *Nav {
	return &Nav{}
}

// Items sets the navigation links.
func (n *Nav) Items(items ...NavLink) *Nav {
	n.items = items
	return n
}

// AddLink adds a navigation link.
func (n *Nav) AddLink(label, href string, active bool) *Nav {
	n.items = append(n.items, NavLink{Label: label, Href: href, Active: active})
	return n
}

// Vertical makes the navigation vertical.
func (n *Nav) Vertical() *Nav {
	n.vertical = true
	return n
}

// Class adds custom CSS classes.
func (n *Nav) Class(class string) *Nav {
	n.class = class
	return n
}

// Render renders the navigation menu to HTML.
func (n *Nav) Render() template.HTML {
	var classes []string
	classes = append(classes, "nav")
	if n.vertical {
		classes = append(classes, "nav--vertical")
	}
	if n.class != "" {
		classes = append(classes, n.class)
	}

	var html strings.Builder
	fmt.Fprintf(&html, `<nav class="%s">`, strings.Join(classes, " "))

	for _, item := range n.items {
		linkClass := "nav__link"
		if item.Active {
			linkClass += " nav__link--active"
		}

		fmt.Fprintf(&html, `<a href="%s" class="%s">`,
			template.HTMLEscapeString(item.Href),
			linkClass)

		if item.Emoji != "" {
			fmt.Fprintf(&html, `<span class="nav__emoji">%s</span>`, item.Emoji)
		}

		fmt.Fprintf(&html, `%s</a>`, template.HTMLEscapeString(item.Label))
	}

	html.WriteString(`</nav>`)

	return template.HTML(html.String())
}
