// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package ui

import (
	"encoding/xml"
	"html/template"
	"io"
	"strings"
	"testing"
)

type urlComponent struct {
	name     string
	attr     string
	optional bool
	render   func(string) template.HTML
}

func urlComponents() []urlComponent {
	return []urlComponent{
		{"link", "href", false, func(url string) template.HTML {
			return NewLink("Open", url).Render()
		}},
		{"navigation", "href", false, func(url string) template.HTML {
			return NewNav().AddLink("Open", url, false).Render()
		}},
		{"grid", "href", false, func(url string) template.HTML {
			return NewNavGrid().AddItem("", "Open", url).Render()
		}},
		{"breadcrumb", "href", true, func(url string) template.HTML {
			return NewPageHeader("Page").Breadcrumbs(Breadcrumb{Label: "Open", Href: url}).Render()
		}},
		{"form", "action", true, func(url string) template.HTML {
			return NewForm().Action(url).Render()
		}},
		{"delete form", "action", true, func(url string) template.HTML {
			return NewDeleteButton("Delete", url).Render()
		}},
		{"settings form", "action", true, func(url string) template.HTML {
			return NewSettingsForm(nil).Action(url).Render()
		}},
	}
}

// TestURLPolicy checks every native UI URL boundary through html/template.
// A direct template supplies the URL-context oracle; policy decisions are named.
func TestURLPolicy(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		blocked bool
	}{
		{"root relative", "/invoices?q=paid&sort=number", false},
		{"relative", "invoices/42", false},
		{"parent relative", "../invoices#recent", false},
		{"relative colon", "./item:42", false},
		{"anchor", "#recent", false},
		{"query", "?next=/items&x=1", false},
		{"network relative", "//example.invalid/invoices", false},
		{"http", "http://example.invalid/invoices", false},
		{"https", "https://example.invalid/invoices", false},
		{"mixed case HTTPS", "hTtPs://example.invalid/invoices", false},
		{"mail", "mailto:person@example.invalid?subject=Hi there", false},
		{"encoded colon", "javascript%3Aalert(1)", false},
		{"entity-looking colon", "javascript&#58;alert(1)", false},
		{"quoted query", `/search?q=" onfocus="bad&x=<tag>`, false},
		{"unicode path", "/café?q=two words", false},
		{"javascript", "javascript:alert(1)", true},
		{"mixed case script", "JaVaScRiPt:alert(1)", true},
		{"leading space", " javascript:alert(1)", true},
		{"leading controls", "\t\nJavascript:alert(1)", true},
		{"embedded tab", "java\tscript:alert(1)", true},
		{"embedded newline", "java\nscript:alert(1)", true},
		{"embedded return", "java\rscript:alert(1)", true},
		{"embedded null", "java\x00script:alert(1)", true},
		{"entity-looking scheme", "java&#x73;cript:alert(1)", true},
		{"data", "data:text/html,<script>alert(1)</script>", true},
		{"vbscript", "vbscript:msgbox(1)", true},
		{"file", "file:///etc/passwd", true},
		{"blob", "blob:https://example.invalid/id", true},
		{"ftp", "ftp://example.invalid/file", true},
		{"custom scheme", "custom:open", true},
		{"phone", "tel:123456", true},
		{"colon anchor", "#javascript:alert(1)", true},
	}
	reference := template.Must(template.New("reference").Parse(`<a href="{{.}}"></a>`))
	wrapper := template.Must(template.New("wrapper").Parse(`{{.}}`))

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := readURLAttrs(t, executeURLTemplate(t, reference, tc.url), "href")
			if len(want) != 1 || (want[0] == "#ZgotmplZ") != tc.blocked {
				t.Fatalf("reference URL %q does not match the declared policy", want)
			}

			for _, component := range urlComponents() {
				t.Run(component.name, func(t *testing.T) {
					markup := executeURLTemplate(t, wrapper, component.render(tc.url))

					got := readURLAttrs(t, markup, component.attr)
					if len(got) != 1 || got[0] != want[0] {
						t.Fatalf("got URL attributes %q, want %q in %q", got, want, markup)
					}
				})
			}
		})
	}
}

// TestURLEmpty preserves missing actions and breadcrumb labels without links.
func TestURLEmpty(t *testing.T) {
	for _, component := range urlComponents() {
		t.Run(component.name, func(t *testing.T) {
			got := readURLAttrs(t, string(component.render("")), component.attr)
			if component.optional {
				if len(got) != 0 {
					t.Fatalf("empty optional URL produced %q", got)
				}
			} else if len(got) != 1 || got[0] != "" {
				t.Fatalf("empty link URL produced %q", got)
			}
		})
	}
}

func executeURLTemplate(t *testing.T, tmpl *template.Template, data any) string {
	t.Helper()

	var output strings.Builder

	err := tmpl.Execute(&output, data)
	if err != nil {
		t.Fatal(err)
	}

	return output.String()
}

// The standard-library decoder handles paired tags and native void inputs.
// Tests check URL attributes and reject injected event attributes or elements.
func readURLAttrs(t *testing.T, markup, attribute string) []string {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(markup))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var values []string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return values
		}

		if err != nil {
			t.Fatalf("invalid rendered fragment %q: %v", markup, err)
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if start.Name.Local == "script" || start.Name.Local == "iframe" {
			t.Fatalf("injected element in %q", markup)
		}

		for _, attr := range start.Attr {
			if strings.HasPrefix(strings.ToLower(attr.Name.Local), "on") {
				t.Fatalf("injected event attribute in %q", markup)
			}

			if attr.Name.Local == attribute {
				values = append(values, attr.Value)
			}
		}
	}
}

// FuzzUIURLs requires every primitive to match Go's URL-context filtering for
// arbitrary strings, including malformed bytes and browser-significant controls.
func FuzzUIURLs(f *testing.F) {
	f.Add("/invoices?number=A&B")
	f.Add("java\tscript:alert(1)")
	f.Add(`/search?q=" onclick="bad`)
	f.Add("javascript&#58;alert(1)")
	f.Add("https://example.invalid/\x00\xff")

	reference := template.Must(template.New("reference").Parse(`<a href="{{.}}"></a>`))
	wrapper := template.Must(template.New("wrapper").Parse(`{{.}}`))

	f.Fuzz(func(t *testing.T, url string) {
		if len(url) > 8192 {
			t.Skip("input exceeds this bounded fuzz run")
		}

		want := readURLAttrs(t, executeURLTemplate(t, reference, url), "href")
		for _, component := range urlComponents() {
			got := readURLAttrs(t, executeURLTemplate(t, wrapper, component.render(url)), component.attr)
			if url == "" && component.optional {
				if len(got) != 0 {
					t.Fatalf("%s emitted an empty optional URL", component.name)
				}

				continue
			}

			if len(got) != 1 || got[0] != want[0] {
				t.Fatalf("%s URLs %q differ from reference %q", component.name, got, want)
			}
		}
	})
}
