package ui

import (
	"strings"
	"testing"

	"github.com/hatmaxkit/hatmax/htmx"
)

func TestLink(t *testing.T) {
	tests := []struct {
		name     string
		link     *Link
		wantAttr string
		wantText string
	}{
		{
			name:     "basic link",
			link:     NewLink("Click", "/page"),
			wantAttr: `href="/page"`,
			wantText: "Click",
		},
		{
			name:     "blank target",
			link:     NewLink("External", "https://example.com").Blank(),
			wantAttr: `target="_blank"`,
		},
		{
			name:     "blank has noopener",
			link:     NewLink("External", "https://example.com").Blank(),
			wantAttr: `rel="noopener noreferrer"`,
		},
		{
			name:     "external",
			link:     NewLink("Ext", "/").External(),
			wantAttr: `rel="noopener noreferrer"`,
		},
		{
			name:     "custom rel",
			link:     NewLink("Link", "/").Rel("nofollow"),
			wantAttr: `rel="nofollow"`,
		},
		{
			name:     "with class",
			link:     NewLink("Link", "/").Class("btn"),
			wantAttr: `class="btn"`,
		},
		{
			name:     "with id",
			link:     NewLink("Link", "/").ID("my-link"),
			wantAttr: `id="my-link"`,
		},
		{
			name:     "with title",
			link:     NewLink("Link", "/").Title("Tooltip"),
			wantAttr: `title="Tooltip"`,
		},
		{
			name:     "with download",
			link:     NewLink("Download", "/file.pdf").Download("report.pdf"),
			wantAttr: `download="report.pdf"`,
		},
		{
			name:     "with target",
			link:     NewLink("Link", "/").Target("_parent"),
			wantAttr: `target="_parent"`,
		},
		{
			name:     "boosted",
			link:     NewLink("Link", "/page").Boost(),
			wantAttr: `hx-boost="true"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.link.Render())

			if tt.wantAttr != "" && !strings.Contains(got, tt.wantAttr) {
				t.Errorf("Link.Render() = %q, want attr %q", got, tt.wantAttr)
			}

			if tt.wantText != "" && !strings.Contains(got, tt.wantText) {
				t.Errorf("Link.Render() = %q, want text %q", got, tt.wantText)
			}
		})
	}
}

func TestLinkWithAttrs(t *testing.T) {
	attrs := htmx.NewAttrs().Get("/api/data").TargetID("content")
	link := NewLink("Load", "/").WithAttrs(attrs)
	got := string(link.Render())

	if !strings.Contains(got, `hx-get="/api/data"`) {
		t.Errorf("Link.WithAttrs() = %q, want hx-get", got)
	}

	if !strings.Contains(got, `hx-target="#content"`) {
		t.Errorf("Link.WithAttrs() = %q, want hx-target", got)
	}
}

func TestLinkHX(t *testing.T) {
	tests := []struct {
		name     string
		link     *Link
		wantAttr string
	}{
		{
			name:     "hx get",
			link:     NewLink("Load", "/").HX().Get("/api").Done(),
			wantAttr: `hx-get="/api"`,
		},
		{
			name:     "hx post",
			link:     NewLink("Send", "/").HX().Post("/api").Done(),
			wantAttr: `hx-post="/api"`,
		},
		{
			name:     "hx target id",
			link:     NewLink("Load", "/").HX().Get("/api").TargetID("box").Done(),
			wantAttr: `hx-target="#box"`,
		},
		{
			name:     "hx swap outer",
			link:     NewLink("Replace", "/").HX().Get("/api").SwapOuter().Done(),
			wantAttr: `hx-swap="outerHTML"`,
		},
		{
			name:     "hx swap inner",
			link:     NewLink("Fill", "/").HX().Get("/api").SwapInner().Done(),
			wantAttr: `hx-swap="innerHTML"`,
		},
		{
			name:     "hx push url",
			link:     NewLink("Nav", "/").HX().Get("/page").PushURL("/page").Done(),
			wantAttr: `hx-push-url="/page"`,
		},
		{
			name:     "hx select",
			link:     NewLink("Load", "/").HX().Get("/api").Select("#main").Done(),
			wantAttr: `hx-select="#main"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.link.Render())

			if !strings.Contains(got, tt.wantAttr) {
				t.Errorf("LinkHX = %q, want attr %q", got, tt.wantAttr)
			}
		})
	}
}

func TestLinkHXRender(t *testing.T) {
	got := string(NewLink("Test", "/").HX().Get("/api").Render())
	if !strings.Contains(got, `hx-get="/api"`) {
		t.Errorf("LinkHX.Render() = %q, want hx-get", got)
	}
}

func TestLinkShorthands(t *testing.T) {
	tests := []struct {
		name     string
		link     *Link
		wantAttr string
	}{
		{
			name:     "A",
			link:     A("Click", "/page"),
			wantAttr: `href="/page"`,
		},
		{
			name:     "ABlank",
			link:     ABlank("External", "https://example.com"),
			wantAttr: `target="_blank"`,
		},
		{
			name:     "ABoosted",
			link:     ABoosted("Nav", "/page"),
			wantAttr: `hx-boost="true"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.link.Render())

			if !strings.Contains(got, tt.wantAttr) {
				t.Errorf("%s() = %q, want attr %q", tt.name, got, tt.wantAttr)
			}
		})
	}
}

