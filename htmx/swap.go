package htmx

import (
	"fmt"
	"strings"
	"time"
)

// Swap represents an HTMX swap strategy.
type Swap struct {
	strategy  string
	modifiers []string
}

// SwapInner returns a swap that replaces the inner HTML of the target.
func SwapInner() Swap {
	return Swap{strategy: "innerHTML"}
}

// SwapOuter returns a swap that replaces the entire target element.
func SwapOuter() Swap {
	return Swap{strategy: "outerHTML"}
}

// SwapBeforeEnd returns a swap that inserts content at the end of the target.
func SwapBeforeEnd() Swap {
	return Swap{strategy: "beforeend"}
}

// SwapAfterEnd returns a swap that inserts content after the target.
func SwapAfterEnd() Swap {
	return Swap{strategy: "afterend"}
}

// SwapBeforeBegin returns a swap that inserts content before the target.
func SwapBeforeBegin() Swap {
	return Swap{strategy: "beforebegin"}
}

// SwapAfterBegin returns a swap that inserts content at the start of the target.
func SwapAfterBegin() Swap {
	return Swap{strategy: "afterbegin"}
}

// SwapDelete returns a swap that deletes the target element.
func SwapDelete() Swap {
	return Swap{strategy: "delete"}
}

// SwapNone returns a swap that doesn't swap any content.
func SwapNone() Swap {
	return Swap{strategy: "none"}
}

// After adds a delay before the swap occurs.
func (s Swap) After(d time.Duration) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("swap:%s", formatDuration(d)))
	return s
}

// Settle adds a delay before the settle step.
func (s Swap) Settle(d time.Duration) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("settle:%s", formatDuration(d)))
	return s
}

// ScrollTop scrolls to the top of the target element after swap.
func (s Swap) ScrollTop() Swap {
	s.modifiers = append(s.modifiers, "scroll:top")
	return s
}

// ScrollBottom scrolls to the bottom of the target element after swap.
func (s Swap) ScrollBottom() Swap {
	s.modifiers = append(s.modifiers, "scroll:bottom")
	return s
}

// Scroll scrolls to a specific element after swap.
func (s Swap) Scroll(selector string) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("scroll:%s", selector))
	return s
}

// Show makes a specific element visible after swap.
func (s Swap) Show(selector string) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("show:%s", selector))
	return s
}

// ShowTop shows the top of an element after swap.
func (s Swap) ShowTop(selector string) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("show:%s:top", selector))
	return s
}

// ShowBottom shows the bottom of an element after swap.
func (s Swap) ShowBottom(selector string) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("show:%s:bottom", selector))
	return s
}

// FocusScroll enables or disables focus scrolling.
func (s Swap) FocusScroll(enabled bool) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("focus-scroll:%t", enabled))
	return s
}

// Transition enables or disables view transitions.
func (s Swap) Transition(enabled bool) Swap {
	s.modifiers = append(s.modifiers, fmt.Sprintf("transition:%t", enabled))
	return s
}

// IgnoreTitle prevents the title from being updated.
func (s Swap) IgnoreTitle() Swap {
	s.modifiers = append(s.modifiers, "ignoreTitle:true")
	return s
}

// String returns the hx-swap attribute value.
func (s Swap) String() string {
	if s.strategy == "" {
		return ""
	}

	if len(s.modifiers) == 0 {
		return s.strategy
	}

	return s.strategy + " " + strings.Join(s.modifiers, " ")
}

// IsEmpty returns true if no swap strategy is set.
func (s Swap) IsEmpty() bool {
	return s.strategy == ""
}
