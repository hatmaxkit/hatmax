// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package htmx

import (
	"bytes"
	"encoding/xml"
	"html/template"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestOOBEscape checks the final template boundary, not just escaped strings.
// Entity-looking input must round-trip once; quotes must not create attributes.
func TestOOBEscape(t *testing.T) {
	cases := []struct {
		name     string
		selector string
	}{
		{"id", "#counter"},
		{"quoted CSS", `[data-label="A&B"] > .row:first-child`},
		{"single quote", `[data-label='owner']`},
		{"event injection", `#target" onfocus="alert(1)`},
		{"extra attribute", `#target" data-review-injected="yes`},
		{"element injection", `#target"><script>alert(1)</script><div id="`},
		{"entities", `#target &quot; &#34; &amp; < >`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oob := OOBInner().Target(tc.selector)

			want := "innerHTML:" + tc.selector
			if oob.String() != want {
				t.Fatalf("String changed the selector: %q", oob.String())
			}

			t.Run("attribute", func(t *testing.T) {
				markup := renderOOB(t, `<div {{.}}></div>`, oob.Attr())
				checkOOBElement(t, markup, map[string]string{"hx-swap-oob": want})
			})
			t.Run("wrapper", func(t *testing.T) {
				id := `counter" onclick="bad & < >`
				class := `row" onfocus="bad &quot; < >`
				wrapper := OOBWrap(id).Swap(oob).Class("active", class)
				markup := renderOOB(t, `{{.Open}}{{.Close}}`, wrapper)
				checkOOBElement(t, markup, map[string]string{
					"id": id, "hx-swap-oob": want, "class": "active " + class,
				})
			})
		})
	}
}

// TestOOBTags preserves content/table wrappers and rejects tag injection or
// elements that would reinterpret ordinary template content as active markup.
func TestOOBTags(t *testing.T) {
	allowed := strings.Fields(`a abbr address article aside b bdi bdo
		blockquote button caption cite code colgroup data datalist dd del details
		dfn dialog div dl dt em fieldset figcaption figure footer form h1 h2 h3 h4
		h5 h6 header hgroup i ins kbd label legend li main mark menu meter nav ol
		optgroup option output p pre progress q rp rt ruby s samp section select
		small span strong sub summary sup table tbody td template tfoot th thead
		time tr u ul var`)
	for _, tag := range allowed {
		t.Run(tag, func(t *testing.T) {
			wrapper := OOBWrap("target").Tag(strings.ToUpper(tag))

			want := `<` + tag + ` id="target" hx-swap-oob="true"></` + tag + `>`
			if got := renderOOB(t, `{{.Open}}{{.Close}}`, wrapper); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}

	t.Run("empty defaults", func(t *testing.T) {
		wrapper := OOBWrap("target").Tag("nav").Tag("")
		checkOOBElement(t, renderOOB(t, `{{.Open}}{{.Close}}`, wrapper), map[string]string{
			"id": "target", "hx-swap-oob": "true",
		})

		var zero OOBWrapper

		checkOOBElement(t, renderOOB(t, `{{.Open}}{{.Close}}`, &zero), map[string]string{
			"id": "", "hx-swap-oob": "",
		})
	})

	cases := []struct {
		name string
		tag  string
	}{
		{"attribute injection", `div onclick="bad"`},
		{"element injection", `div><script>bad</script><div`},
		{"closing tag", "/div"},
		{"whitespace", " div "},
		{"newline", "div\nonclick"},
		{"script", "script"},
		{"uppercase script", "SCRIPT"},
		{"style", "style"},
		{"raw text", "xmp"},
		{"RCDATA", "textarea"},
		{"iframe", "iframe"},
		{"object", "object"},
		{"void", "img"},
		{"custom", "user-widget"},
		{"namespace", "svg:g"},
		{"foreign content", "svg"},
		{"null", "div\x00"},
		{"unknown", "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper := OOBWrap("target").Tag("nav")

			expectOOBPanic(t, func() { wrapper.Tag(tc.tag) })

			if wrapper.tag != "nav" {
				t.Fatal("invalid tag changed the wrapper")
			}

			wrapper.tag = tc.tag

			expectOOBPanic(t, func() { wrapper.Open() })
			expectOOBPanic(t, func() { wrapper.Close() })
		})
	}
}

func expectOOBPanic(t *testing.T, call func()) {
	t.Helper()

	defer func() {
		if got := recover(); got != "htmx: unsupported OOB wrapper tag" {
			t.Errorf("got panic %v, want unsupported OOB wrapper tag", got)
		}
	}()

	call()
}

func renderOOB(t *testing.T, source string, data any) string {
	t.Helper()

	tmpl, err := template.New("oob").Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, data)
	if err != nil {
		t.Fatal(err)
	}

	return output.String()
}

// These paired fragments use only XML-compatible HTML entities. A strict
// standard-library decoder checks attributes and structure without new modules.
func checkOOBElement(t *testing.T, markup string, want map[string]string) {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(markup))
	starts, ends := 0, 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("invalid rendered fragment %q: %v", markup, err)
		}

		switch token := token.(type) {
		case xml.StartElement:
			starts++

			if token.Name.Local != "div" || token.Name.Space != "" || len(token.Attr) != len(want) {
				t.Fatalf("unexpected element or attributes: %#v in %q", token, markup)
			}

			seen := make(map[string]bool)

			for _, attr := range token.Attr {
				value, exists := want[attr.Name.Local]
				if !exists || seen[attr.Name.Local] || attr.Name.Space != "" || attr.Value != value {
					t.Fatalf("unexpected attribute %#v in %q", attr, markup)
				}

				seen[attr.Name.Local] = true
			}
		case xml.EndElement:
			ends++
		default:
			t.Fatalf("unexpected content %#v in %q", token, markup)
		}
	}

	if starts != 1 || ends != 1 {
		t.Fatalf("got %d opening and %d closing tags in %q", starts, ends, markup)
	}
}

// FuzzOOBEscape checks that arbitrary XML-compatible data remains data in the
// template fragment. Browser normalization of invalid characters is out of scope.
func FuzzOOBEscape(f *testing.F) {
	f.Add("#counter", "counter", "row")
	f.Add(`#x" onclick="bad`, `x"><script>bad</script>`, `row" onfocus="bad`)
	f.Add(`[data-label="A&B"]`, "&#34;", "&quot; < > '")
	f.Fuzz(func(t *testing.T, selector, id, class string) {
		for _, value := range []string{selector, id, class} {
			if len(value) > 2048 || !utf8.ValidString(value) {
				t.Skip("input outside the fragment decoder's character domain")
			}

			for _, r := range value {
				if r < 0x20 || r == 0xfffe || r == 0xffff {
					t.Skip("input outside the fragment decoder's character domain")
				}
			}
		}

		oob := OOBInner().Target(selector)
		markup := renderOOB(t, `<div {{.}}></div>`, oob.Attr())
		checkOOBElement(t, markup, map[string]string{"hx-swap-oob": oob.String()})

		wrapper := OOBWrap(id).Swap(oob).Class(class)
		markup = renderOOB(t, `{{.Open}}{{.Close}}`, wrapper)
		checkOOBElement(t, markup, map[string]string{
			"id": id, "hx-swap-oob": oob.String(), "class": class,
		})
	})
}
