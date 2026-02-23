package htmx

import (
	"fmt"
	"html/template"
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

// OOBAttr returns the full hx-swap-oob attribute.
func (o OOB) Attr() template.HTMLAttr {
	return template.HTMLAttr(fmt.Sprintf(`hx-swap-oob="%s"`, o.String()))
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

// Tag sets the HTML tag for the wrapper (default: div).
func (w *OOBWrapper) Tag(tag string) *OOBWrapper {
	w.tag = tag

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
		w.tag,
		template.HTMLEscapeString(w.id),
		w.oob.String(),
		classAttr,
	))
}

// Close returns the closing tag.
func (w *OOBWrapper) Close() template.HTML {
	return template.HTML(fmt.Sprintf("</%s>", w.tag))
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
