package htmx

import "fmt"

// Target represents an HTMX target selector.
type Target struct {
	selector string
	relative string
}

// TargetSelf returns a target that refers to the element itself.
func TargetSelf() Target {
	return Target{selector: "this"}
}

// TargetID returns a target that selects an element by ID.
func TargetID(id string) Target {
	return Target{selector: fmt.Sprintf("#%s", id)}
}

// TargetSelector returns a target with a custom CSS selector.
func TargetSelector(sel string) Target {
	return Target{selector: sel}
}

// TargetClosest returns a target that finds the closest ancestor matching the selector.
func TargetClosest(sel string) Target {
	return Target{selector: sel, relative: "closest"}
}

// TargetNext returns a target that finds the next sibling matching the selector.
func TargetNext(sel string) Target {
	return Target{selector: sel, relative: "next"}
}

// TargetPrevious returns a target that finds the previous sibling matching the selector.
func TargetPrevious(sel string) Target {
	return Target{selector: sel, relative: "previous"}
}

// TargetFind returns a target that finds a descendant matching the selector.
func TargetFind(sel string) Target {
	return Target{selector: sel, relative: "find"}
}

// TargetBody returns a target for the document body.
func TargetBody() Target {
	return Target{selector: "body"}
}

// String returns the hx-target attribute value.
func (t Target) String() string {
	if t.selector == "" {
		return ""
	}

	if t.relative != "" {
		return fmt.Sprintf("%s %s", t.relative, t.selector)
	}

	return t.selector
}

// IsEmpty returns true if the target has no selector.
func (t Target) IsEmpty() bool {
	return t.selector == ""
}

// ItemTarget is a helper to create a target for a specific item by ID.
// It returns a target like "#item-123".
func ItemTarget(prefix string, id any) Target {
	return Target{selector: fmt.Sprintf("#%s-%v", prefix, id)}
}
