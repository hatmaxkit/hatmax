// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package htmx

import (
	"fmt"
	"html/template"
	"strings"
)

// OOB represents an out-of-band swap configuration.
type OOB struct {
	swap     string
	selector string
}

// OOBSwap creates an out-of-band swap with outerHTML (default).
func OOBSwap() OOB {
	return OOB{swap: "true"}
}

// OOBInner creates an out-of-band swap with innerHTML.
func OOBInner() OOB {
	return OOB{swap: "innerHTML"}
}

// OOBOuter creates an out-of-band swap with outerHTML.
func OOBOuter() OOB {
	return OOB{swap: "outerHTML"}
}

// OOBBeforeEnd creates an out-of-band swap that appends to the target.
func OOBBeforeEnd() OOB {
	return OOB{swap: "beforeend"}
}

// OOBAfterBegin creates an out-of-band swap that prepends to the target.
func OOBAfterBegin() OOB {
	return OOB{swap: "afterbegin"}
}

// OOBBeforeBegin creates an out-of-band swap that inserts before the target.
func OOBBeforeBegin() OOB {
	return OOB{swap: "beforebegin"}
}

// OOBAfterEnd creates an out-of-band swap that inserts after the target.
func OOBAfterEnd() OOB {
	return OOB{swap: "afterend"}
}

// OOBDelete creates an out-of-band delete.
func OOBDelete() OOB {
	return OOB{swap: "delete"}
}

// OOBNone creates an out-of-band swap that doesn't swap content.
func OOBNone() OOB {
	return OOB{swap: "none"}
}

// Target sets an explicit target selector for the OOB swap.
func (o OOB) Target(selector string) OOB {
	o.selector = selector

	return o
}

// String returns the hx-swap-oob attribute value.
func (o OOB) String() string {
	if o.selector != "" {
		return fmt.Sprintf("%s:%s", o.swap, o.selector)
	}

	return o.swap
}

// Attr returns the full hx-swap-oob attribute with its value HTML-escaped.
func (o OOB) Attr() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`hx-swap-oob="%s"`, template.HTMLEscapeString(o.String())))
}

// OOBWrapper wraps content with an element that has hx-swap-oob set.
type OOBWrapper struct {
	id      string
	tag     string
	oob     OOB
	classes []string
}

// OOBWrap creates a wrapper for OOB content targeting a specific element.
func OOBWrap(id string) *OOBWrapper {
	return &OOBWrapper{
		id:  id,
		tag: "div",
		oob: OOBSwap(),
	}
}

// Tag sets a supported paired HTML content tag, normalized to lowercase.
// An empty tag selects div. Unsupported tags panic before changing the wrapper.
// Raw-text, embedded, void, custom, and namespaced elements are not supported.
func (w *OOBWrapper) Tag(tag string) *OOBWrapper {
	w.tag = oobWrapperTag(tag)

	return w
}

// Swap sets the swap strategy for the OOB update.
func (w *OOBWrapper) Swap(oob OOB) *OOBWrapper {
	w.oob = oob

	return w
}

// Class adds CSS classes to the wrapper.
func (w *OOBWrapper) Class(classes ...string) *OOBWrapper {
	w.classes = append(w.classes, classes...)

	return w
}

// Open returns the opening tag with OOB attributes.
func (w *OOBWrapper) Open() template.HTML {
	classAttr := ""
	if len(w.classes) > 0 {
		classAttr = fmt.Sprintf(` class="%s"`, template.HTMLEscapeString(joinClasses(w.classes)))
	}

	return template.HTML(fmt.Sprintf(`<%s id="%s" hx-swap-oob="%s"%s>`,
		oobWrapperTag(w.tag),
		template.HTMLEscapeString(w.id),
		template.HTMLEscapeString(w.oob.String()),
		classAttr,
	))
}

// Close returns the closing tag.
func (w *OOBWrapper) Close() template.HTML {
	return template.HTML(fmt.Sprintf("</%s>", oobWrapperTag(w.tag)))
}

// oobWrapperTag keeps trusted fragments in normal HTML content contexts.
// Tag syntax alone is insufficient: script and style change how content is read.
func oobWrapperTag(tag string) string {
	if tag == "" {
		return "div"
	}

	tag = strings.ToLower(tag)

	switch tag {
	case "a", "abbr", "address", "article", "aside", "b", "bdi", "bdo",
		"blockquote", "button", "caption", "cite", "code", "colgroup",
		"data", "datalist", "dd", "del", "details", "dfn", "dialog", "div",
		"dl", "dt", "em", "fieldset", "figcaption", "figure", "footer", "form",
		"h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "i", "ins",
		"kbd", "label", "legend", "li", "main", "mark", "menu", "meter", "nav",
		"ol", "optgroup", "option", "output", "p", "pre", "progress", "q",
		"rp", "rt", "ruby", "s", "samp", "section", "select", "small", "span",
		"strong", "sub", "summary", "sup", "table", "tbody", "td", "template",
		"tfoot", "th", "thead", "time", "tr", "u", "ul", "var":
		return tag
	default:
		panic("htmx: unsupported OOB wrapper tag")
	}
}

// joinClasses joins CSS class names with spaces.
func joinClasses(classes []string) string {
	result := ""

	for i, c := range classes {
		if i > 0 {
			result += " "
		}

		result += c
	}

	return result
}
