package ui

import (
	"strings"
	"testing"
)

func TestNavGridRender(t *testing.T) {
	tests := []struct {
		name     string
		nav      *NavGrid
		contains []string
	}{
		{
			name:     "basic nav grid",
			nav:      NewNavGrid().AddItem(EmojiHome, "Home", "/"),
			contains: []string{`class="nav-grid"`, `href="/"`, `Home`, string(EmojiHome)},
		},
		{
			name:     "nav grid with badge",
			nav:      NewNavGrid().AddItemWithBadge(EmojiMail, "Messages", "/messages", "5"),
			contains: []string{`nav-grid__badge`, `5`},
		},
		{
			name:     "nav grid columns",
			nav:      NewNavGrid().Cols(4),
			contains: []string{`--nav-cols: 4`},
		},
		{
			name:     "nav grid with class",
			nav:      NewNavGrid().Class("my-nav"),
			contains: []string{`my-nav`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.nav.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestNavGridItems(t *testing.T) {
	items := []NavItem{
		{Emoji: EmojiHome, Label: "Home", Href: "/"},
		{Emoji: EmojiSettings, Label: "Settings", Href: "/settings"},
	}

	nav := NewNavGrid().Items(items...)
	html := string(nav.Render())

	if !strings.Contains(html, "Home") {
		t.Error("Render() missing Home")
	}

	if !strings.Contains(html, "Settings") {
		t.Error("Render() missing Settings")
	}
}

func TestNavRender(t *testing.T) {
	tests := []struct {
		name     string
		nav      *Nav
		contains []string
	}{
		{
			name:     "basic nav",
			nav:      NewNav().AddLink("Home", "/", true),
			contains: []string{`class="nav"`, `href="/"`, `Home`, `nav__link--active`},
		},
		{
			name:     "vertical nav",
			nav:      NewNav().Vertical(),
			contains: []string{`nav--vertical`},
		},
		{
			name:     "nav with class",
			nav:      NewNav().Class("sidebar"),
			contains: []string{`sidebar`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.nav.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestNavItems(t *testing.T) {
	items := []NavLink{
		{Label: "Home", Href: "/", Active: true},
		{Label: "About", Href: "/about", Active: false, Emoji: EmojiInfo},
	}

	nav := NewNav().Items(items...)
	html := string(nav.Render())

	if !strings.Contains(html, "Home") {
		t.Error("Render() missing Home")
	}

	if !strings.Contains(html, "About") {
		t.Error("Render() missing About")
	}

	if !strings.Contains(html, string(EmojiInfo)) {
		t.Error("Render() missing emoji")
	}
}
