package ui

import (
	"html/template"
	"strings"
	"testing"
)

func TestPageRender(t *testing.T) {
	tests := []struct {
		name     string
		page     *Page
		contains []string
	}{
		{
			name:     "basic page",
			page:     NewPage("Home"),
			contains: []string{`class="page"`, `class="page__content"`},
		},
		{
			name:     "page with content",
			page:     NewPage("Test").Content(template.HTML("<p>Hello</p>")),
			contains: []string{`<p>Hello</p>`},
		},
		{
			name:     "page with class",
			page:     NewPage("Test").Class("my-page"),
			contains: []string{`my-page`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.page.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestPageTitle(t *testing.T) {
	page := NewPage("My Title")
	if page.Title() != "My Title" {
		t.Errorf("Title() = %q, want %q", page.Title(), "My Title")
	}
}

func TestPageHeaderRender(t *testing.T) {
	tests := []struct {
		name     string
		header   *PageHeader
		contains []string
	}{
		{
			name:     "basic header",
			header:   NewPageHeader("Dashboard"),
			contains: []string{`class="page-header"`, `Dashboard`},
		},
		{
			name:     "header with subtitle",
			header:   NewPageHeader("Settings").Subtitle("Configure app"),
			contains: []string{`Settings`, `Configure app`, `page-header__subtitle`},
		},
		{
			name:     "header with breadcrumbs",
			header:   NewPageHeader("Edit").Breadcrumbs(Breadcrumb{Label: "Home", Href: "/"}, Breadcrumb{Label: "Edit"}),
			contains: []string{`page-header__breadcrumbs`, `href="/"`, `Home`, `Edit`},
		},
		{
			name:     "header with class",
			header:   NewPageHeader("Test").Class("custom-header"),
			contains: []string{`custom-header`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.header.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

type mockRenderable struct {
	html string
}

func (m mockRenderable) Render() template.HTML {
	return template.HTML(m.html)
}

func TestPageHeaderWithActions(t *testing.T) {
	header := NewPageHeader("Test").Actions(mockRenderable{html: "<button>Save</button>"})
	html := string(header.Render())

	if !strings.Contains(html, "page-header__actions") {
		t.Error("Render() missing actions container")
	}

	if !strings.Contains(html, "<button>Save</button>") {
		t.Error("Render() missing action content")
	}
}

func TestPageWithHeaderAndFooter(t *testing.T) {
	header := mockRenderable{html: "<header>Header</header>"}
	footer := mockRenderable{html: "<footer>Footer</footer>"}

	page := NewPage("Test").Header(header).Footer(footer).Content(template.HTML("<p>Content</p>"))
	html := string(page.Render())

	if !strings.Contains(html, "<header>Header</header>") {
		t.Error("Render() missing header")
	}

	if !strings.Contains(html, "<footer>Footer</footer>") {
		t.Error("Render() missing footer")
	}

	if !strings.Contains(html, "<p>Content</p>") {
		t.Error("Render() missing content")
	}
}

func TestContainerRender(t *testing.T) {
	tests := []struct {
		name     string
		cont     *Container
		contains []string
	}{
		{
			name:     "basic container",
			cont:     NewContainer().Content(template.HTML("<p>Test</p>")),
			contains: []string{`class="container"`, `<p>Test</p>`},
		},
		{
			name:     "fluid container",
			cont:     NewContainer().Fluid(),
			contains: []string{`container--fluid`},
		},
		{
			name:     "container with class",
			cont:     NewContainer().Class("my-container"),
			contains: []string{`my-container`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := string(tt.cont.Render())
			for _, s := range tt.contains {
				if !strings.Contains(html, s) {
					t.Errorf("Render() missing %q in %q", s, html)
				}
			}
		})
	}
}

func TestPageDescription(t *testing.T) {
	page := NewPage("Test").Description("A test page")
	if page.description != "A test page" {
		t.Errorf("Description not set")
	}
}
