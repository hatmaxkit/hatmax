package htmx

import (
	"fmt"
	"strings"
	"time"
)

// Trigger represents an HTMX trigger configuration.
// It specifies when an HTMX request should be initiated.
type Trigger struct {
	event   string
	filters []string
	from    string
}

// OnClick creates a trigger that fires on click events.
func OnClick() Trigger {
	return Trigger{event: "click"}
}

// OnSubmit creates a trigger that fires on form submit events.
func OnSubmit() Trigger {
	return Trigger{event: "submit"}
}

// OnLoad creates a trigger that fires when the element is loaded.
func OnLoad() Trigger {
	return Trigger{event: "load"}
}

// OnRevealed creates a trigger that fires when the element is scrolled into view.
func OnRevealed() Trigger {
	return Trigger{event: "revealed"}
}

// OnIntersect creates a trigger that fires when the element intersects the viewport.
// The root parameter specifies the root element for the intersection observer.
func OnIntersect(root string) Trigger {
	t := Trigger{event: "intersect"}
	if root != "" {
		t.filters = append(t.filters, fmt.Sprintf("root:%s", root))
	}

	return t
}

// Every creates a trigger that fires at the specified interval.
func Every(interval time.Duration) Trigger {
	return Trigger{event: fmt.Sprintf("every %s", formatDuration(interval))}
}

// OnEvent creates a trigger for a custom event.
func OnEvent(name string) Trigger {
	return Trigger{event: name}
}

// OnChange creates a trigger that fires on change events.
func OnChange() Trigger {
	return Trigger{event: "change"}
}

// OnInput creates a trigger that fires on input events.
func OnInput() Trigger {
	return Trigger{event: "input"}
}

// OnKeyup creates a trigger that fires on keyup events.
func OnKeyup() Trigger {
	return Trigger{event: "keyup"}
}

// OnFocus creates a trigger that fires on focus events.
func OnFocus() Trigger {
	return Trigger{event: "focus"}
}

// OnBlur creates a trigger that fires on blur events.
func OnBlur() Trigger {
	return Trigger{event: "blur"}
}

// Once adds the once modifier, causing the trigger to fire only once.
func (t Trigger) Once() Trigger {
	t.filters = append(t.filters, "once")

	return t
}

// Changed adds the changed modifier, causing the trigger to fire only if the value has changed.
func (t Trigger) Changed() Trigger {
	t.filters = append(t.filters, "changed")

	return t
}

// Delay adds a delay before the trigger fires.
func (t Trigger) Delay(d time.Duration) Trigger {
	t.filters = append(t.filters, fmt.Sprintf("delay:%s", formatDuration(d)))

	return t
}

// Throttle adds throttling to limit how often the trigger can fire.
func (t Trigger) Throttle(d time.Duration) Trigger {
	t.filters = append(t.filters, fmt.Sprintf("throttle:%s", formatDuration(d)))

	return t
}

// From specifies a different element that should trigger this request.
func (t Trigger) From(selector string) Trigger {
	t.from = selector

	return t
}

// Queue specifies the queue behavior for requests.
// Valid values: "first", "last", "all", "none".
func (t Trigger) Queue(behavior string) Trigger {
	t.filters = append(t.filters, fmt.Sprintf("queue:%s", behavior))

	return t
}

// Target specifies a target element filter for the event.
func (t Trigger) Target(selector string) Trigger {
	t.filters = append(t.filters, fmt.Sprintf("target:%s", selector))

	return t
}

// Consume adds the consume modifier to stop event propagation.
func (t Trigger) Consume() Trigger {
	t.filters = append(t.filters, "consume")

	return t
}

// String returns the hx-trigger attribute value.
func (t Trigger) String() string {
	if t.event == "" {
		return ""
	}

	parts := []string{t.event}
	parts = append(parts, t.filters...)

	if t.from != "" {
		parts = append(parts, fmt.Sprintf("from:%s", t.from))
	}

	return strings.Join(parts, " ")
}

// formatDuration formats a duration for HTMX (e.g., "500ms", "1s").
func formatDuration(d time.Duration) string {
	if d >= time.Second && d%time.Second == 0 {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	return fmt.Sprintf("%dms", d.Milliseconds())
}

// Triggers combines multiple triggers into a comma-separated list.
func Triggers(triggers ...Trigger) string {
	strs := make([]string, 0, len(triggers))
	for _, t := range triggers {
		if s := t.String(); s != "" {
			strs = append(strs, s)
		}
	}

	return strings.Join(strs, ", ")
}
