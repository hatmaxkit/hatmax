package htmx

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"
)

// Attrs holds a collection of HTMX attributes that can be rendered to HTML.
type Attrs struct {
	action  *Action
	trigger *Trigger
	target  *Target
	swap    *Swap

	confirm    string
	disable    string
	indicator  string
	sync       string
	validate   bool
	boost      *bool
	pushURL    string
	replaceURL string
	select_    string
	selectOOB  string
	ext        string
	encoding   string
	preserve   bool
	history    string

	extra map[string]string
}

// NewAttrs creates a new empty Attrs.
func NewAttrs() *Attrs {
	return &Attrs{}
}

// Action sets the HTTP action (GET, POST, etc.).
func (a *Attrs) Action(action Action) *Attrs {
	a.action = &action
	return a
}

// Get sets a GET action.
func (a *Attrs) Get(url string) *Attrs {
	action := Get(url)
	a.action = &action
	return a
}

// Post sets a POST action.
func (a *Attrs) Post(url string) *Attrs {
	action := Post(url)
	a.action = &action
	return a
}

// Put sets a PUT action.
func (a *Attrs) Put(url string) *Attrs {
	action := Put(url)
	a.action = &action
	return a
}

// Patch sets a PATCH action.
func (a *Attrs) Patch(url string) *Attrs {
	action := Patch(url)
	a.action = &action
	return a
}

// Delete sets a DELETE action.
func (a *Attrs) Delete(url string) *Attrs {
	action := Delete(url)
	a.action = &action
	return a
}

// Trigger sets the trigger configuration.
func (a *Attrs) Trigger(trigger Trigger) *Attrs {
	a.trigger = &trigger
	return a
}

// Target sets the target selector.
func (a *Attrs) Target(target Target) *Attrs {
	a.target = &target
	return a
}

// TargetID sets a target by ID.
func (a *Attrs) TargetID(id string) *Attrs {
	t := TargetID(id)
	a.target = &t
	return a
}

// TargetThis sets the target to the element itself.
func (a *Attrs) TargetThis() *Attrs {
	t := TargetSelf()
	a.target = &t
	return a
}

// Swap sets the swap strategy.
func (a *Attrs) Swap(swap Swap) *Attrs {
	a.swap = &swap
	return a
}

// SwapOuter sets outer HTML swap.
func (a *Attrs) SwapOuter() *Attrs {
	s := SwapOuter()
	a.swap = &s
	return a
}

// SwapInner sets inner HTML swap.
func (a *Attrs) SwapInner() *Attrs {
	s := SwapInner()
	a.swap = &s
	return a
}

// SwapDelete sets delete swap.
func (a *Attrs) SwapDelete() *Attrs {
	s := SwapDelete()
	a.swap = &s
	return a
}

// Confirm sets a confirmation message.
func (a *Attrs) Confirm(message string) *Attrs {
	a.confirm = message
	return a
}

// Disable sets the disable selector while request is in flight.
func (a *Attrs) Disable(selector string) *Attrs {
	a.disable = selector
	return a
}

// DisableSelf disables the element itself while request is in flight.
func (a *Attrs) DisableSelf() *Attrs {
	a.disable = "this"
	return a
}

// Indicator sets the indicator element.
func (a *Attrs) Indicator(selector string) *Attrs {
	a.indicator = selector
	return a
}

// Sync sets the sync strategy.
func (a *Attrs) Sync(strategy string) *Attrs {
	a.sync = strategy
	return a
}

// Validate enables form validation.
func (a *Attrs) Validate() *Attrs {
	a.validate = true
	return a
}

// Boost enables hx-boost.
func (a *Attrs) Boost() *Attrs {
	b := true
	a.boost = &b
	return a
}

// NoBoost disables hx-boost.
func (a *Attrs) NoBoost() *Attrs {
	b := false
	a.boost = &b
	return a
}

// PushURL enables pushing a URL to history.
func (a *Attrs) PushURL(url string) *Attrs {
	a.pushURL = url
	return a
}

// ReplaceURL enables replacing the URL in history.
func (a *Attrs) ReplaceURL(url string) *Attrs {
	a.replaceURL = url
	return a
}

// Select sets the selector for partial content.
func (a *Attrs) Select(selector string) *Attrs {
	a.select_ = selector
	return a
}

// SelectOOB sets the selector for out-of-band swaps.
func (a *Attrs) SelectOOB(selector string) *Attrs {
	a.selectOOB = selector
	return a
}

// Ext sets HTMX extensions.
func (a *Attrs) Ext(extensions string) *Attrs {
	a.ext = extensions
	return a
}

// Encoding sets the form encoding type.
func (a *Attrs) Encoding(encoding string) *Attrs {
	a.encoding = encoding
	return a
}

// Preserve marks the element as preserved during swap.
func (a *Attrs) Preserve() *Attrs {
	a.preserve = true
	return a
}

// History sets history behavior.
func (a *Attrs) History(value string) *Attrs {
	a.history = value
	return a
}

// WithVal adds a value to the action's hx-vals.
func (a *Attrs) WithVal(key string, value any) *Attrs {
	if a.action == nil {
		action := Action{}
		a.action = &action
	}
	*a.action = a.action.WithVal(key, value)
	return a
}

// WithVals adds multiple values to the action's hx-vals.
func (a *Attrs) WithVals(vals map[string]any) *Attrs {
	if a.action == nil {
		action := Action{}
		a.action = &action
	}
	*a.action = a.action.WithVals(vals)
	return a
}

// Set adds a custom attribute.
func (a *Attrs) Set(key, value string) *Attrs {
	if a.extra == nil {
		a.extra = make(map[string]string)
	}
	a.extra[key] = value
	return a
}

// Map returns all attributes as a map.
func (a *Attrs) Map() map[string]string {
	m := make(map[string]string)

	if a.action != nil {
		m[a.action.Attr()] = a.action.URL()
		if v := a.action.Vals(); v != "" {
			m["hx-vals"] = v
		}
		if len(a.action.Headers()) > 0 {
			h, _ := json.Marshal(a.action.Headers())
			m["hx-headers"] = string(h)
		}
	}

	if a.trigger != nil {
		if s := a.trigger.String(); s != "" {
			m["hx-trigger"] = s
		}
	}

	if a.target != nil {
		if s := a.target.String(); s != "" {
			m["hx-target"] = s
		}
	}

	if a.swap != nil {
		if s := a.swap.String(); s != "" {
			m["hx-swap"] = s
		}
	}

	if a.confirm != "" {
		m["hx-confirm"] = a.confirm
	}

	if a.disable != "" {
		m["hx-disabled-elt"] = a.disable
	}

	if a.indicator != "" {
		m["hx-indicator"] = a.indicator
	}

	if a.sync != "" {
		m["hx-sync"] = a.sync
	}

	if a.validate {
		m["hx-validate"] = "true"
	}

	if a.boost != nil {
		if *a.boost {
			m["hx-boost"] = "true"
		} else {
			m["hx-boost"] = "false"
		}
	}

	if a.pushURL != "" {
		m["hx-push-url"] = a.pushURL
	}

	if a.replaceURL != "" {
		m["hx-replace-url"] = a.replaceURL
	}

	if a.select_ != "" {
		m["hx-select"] = a.select_
	}

	if a.selectOOB != "" {
		m["hx-select-oob"] = a.selectOOB
	}

	if a.ext != "" {
		m["hx-ext"] = a.ext
	}

	if a.encoding != "" {
		m["hx-encoding"] = a.encoding
	}

	if a.preserve {
		m["hx-preserve"] = "true"
	}

	if a.history != "" {
		m["hx-history"] = a.history
	}

	for k, v := range a.extra {
		m[k] = v
	}

	return m
}

// String returns the attributes as a space-separated string.
func (a *Attrs) String() string {
	m := a.Map()
	if len(m) == 0 {
		return ""
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := m[k]
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, template.HTMLEscapeString(v)))
	}

	return strings.Join(parts, " ")
}

// HTML returns the attributes as safe HTML for use in templates.
func (a *Attrs) HTML() template.HTMLAttr {
	return template.HTMLAttr(a.String())
}

// HX is a convenience function to start building attributes.
func HX() *Attrs {
	return NewAttrs()
}
