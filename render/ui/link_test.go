package ui

import (
	"strings"
	"testing"
)

func TestLink(t *testing.T) {
	t.Run("basic link", func(t *testing.T) {
		link := NewLink("Home", "/")
		html := string(link.Render())

		if !strings.Contains(html, "Home") {
			t.Error("link should contain text")
		}
		if !strings.Contains(html, `href="/"`) {
			t.Error("link should have href")
		}
	})

	t.Run("link with target blank", func(t *testing.T) {
		link := NewLink("External", "https://example.com").Blank()
		html := string(link.Render())

		if !strings.Contains(html, `target="_blank"`) {
			t.Error("link should have target blank")
		}
		if !strings.Contains(html, `rel="noopener noreferrer"`) {
			t.Error("link should have secure rel")
		}
	})

	t.Run("boosted link", func(t *testing.T) {
		link := NewLink("Page", "/page").Boost()
		html := string(link.Render())

		if !strings.Contains(html, `hx-boost="true"`) {
			t.Error("link should have hx-boost")
		}
	})

	t.Run("link with class and id", func(t *testing.T) {
		link := NewLink("Test", "/test").Class("nav-item").ID("nav-home")
		html := string(link.Render())

		if !strings.Contains(html, `class="nav-item"`) {
			t.Error("link should have class")
		}
		if !strings.Contains(html, `id="nav-home"`) {
			t.Error("link should have id")
		}
	})

	t.Run("download link", func(t *testing.T) {
		link := NewLink("Download", "/file.pdf").Download("report.pdf")
		html := string(link.Render())

		if !strings.Contains(html, `download="report.pdf"`) {
			t.Error("link should have download attribute")
		}
	})

	t.Run("link with HTMX", func(t *testing.T) {
		link := NewLink("Load Content", "/content").HX().
			Get("/api/content").
			TargetID("main").
			SwapInner().
			PushHref().
			Done()

		html := string(link.Render())

		if !strings.Contains(html, `hx-get="/api/content"`) {
			t.Error("link should have hx-get")
		}
		if !strings.Contains(html, `hx-target="#main"`) {
			t.Error("link should have hx-target")
		}
		if !strings.Contains(html, `hx-swap="innerHTML"`) {
			t.Error("link should have hx-swap")
		}
		if !strings.Contains(html, `hx-push-url="/content"`) {
			t.Error("link should have hx-push-url")
		}
	})

	t.Run("shorthand constructors", func(t *testing.T) {
		basic := A("Test", "/test")
		if basic == nil {
			t.Error("A should return link")
		}

		blank := ABlank("External", "https://example.com")
		if !strings.Contains(string(blank.Render()), `target="_blank"`) {
			t.Error("ABlank should create blank link")
		}

		boosted := ABoosted("Page", "/page")
		if !strings.Contains(string(boosted.Render()), `hx-boost="true"`) {
			t.Error("ABoosted should create boosted link")
		}
	})
}

func TestNavLink(t *testing.T) {
	t.Run("basic nav link", func(t *testing.T) {
		nav := NewNavLink("Home", "/")
		html := string(nav.Render())

		if !strings.Contains(html, "nav-link") {
			t.Error("nav link should have nav-link class")
		}
	})

	t.Run("active nav link", func(t *testing.T) {
		nav := NewNavLink("Home", "/").Active(true)
		html := string(nav.Render())

		if !strings.Contains(html, "nav-link--active") {
			t.Error("active nav link should have active class")
		}
	})

	t.Run("shorthand constructor", func(t *testing.T) {
		nav := Nav("Home", "/")
		if nav == nil {
			t.Error("Nav should return nav link")
		}
	})
}
