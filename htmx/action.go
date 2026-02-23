package htmx

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Action represents an HTMX HTTP action (GET, POST, PUT, PATCH, DELETE).
type Action struct {
	method  string
	url     string
	params  map[string]string
	headers map[string]string
	vals    map[string]any
}

// Get creates a GET action for the specified URL.
func Get(u string) Action {
	return Action{method: "get", url: u}
}

// Post creates a POST action for the specified URL.
func Post(u string) Action {
	return Action{method: "post", url: u}
}

// Put creates a PUT action for the specified URL.
func Put(u string) Action {
	return Action{method: "put", url: u}
}

// Patch creates a PATCH action for the specified URL.
func Patch(u string) Action {
	return Action{method: "patch", url: u}
}

// Delete creates a DELETE action for the specified URL.
func Delete(u string) Action {
	return Action{method: "delete", url: u}
}

// WithParam adds a URL query parameter to the action.
func (a Action) WithParam(key, value string) Action {
	if a.params == nil {
		a.params = make(map[string]string)
	}

	a.params[key] = value

	return a
}

// WithParams adds multiple URL query parameters to the action.
func (a Action) WithParams(params map[string]string) Action {
	if a.params == nil {
		a.params = make(map[string]string)
	}

	for k, v := range params {
		a.params[k] = v
	}

	return a
}

// WithHeader adds an HTTP header to the action.
func (a Action) WithHeader(key, value string) Action {
	if a.headers == nil {
		a.headers = make(map[string]string)
	}

	a.headers[key] = value

	return a
}

// WithHeaders adds multiple HTTP headers to the action.
func (a Action) WithHeaders(headers map[string]string) Action {
	if a.headers == nil {
		a.headers = make(map[string]string)
	}

	for k, v := range headers {
		a.headers[k] = v
	}

	return a
}

// WithVals adds values to be included in the request body (hx-vals).
func (a Action) WithVals(vals map[string]any) Action {
	if a.vals == nil {
		a.vals = make(map[string]any)
	}

	for k, v := range vals {
		a.vals[k] = v
	}

	return a
}

// WithVal adds a single value to the request body (hx-vals).
func (a Action) WithVal(key string, value any) Action {
	if a.vals == nil {
		a.vals = make(map[string]any)
	}

	a.vals[key] = value

	return a
}

// Method returns the HTTP method (get, post, put, patch, delete).
func (a Action) Method() string {
	return a.method
}

// URL returns the action URL with query parameters.
func (a Action) URL() string {
	if len(a.params) == 0 {
		return a.url
	}

	u, err := url.Parse(a.url)
	if err != nil {
		// If parsing fails, just return the original URL
		return a.url
	}

	q := u.Query()
	for k, v := range a.params {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	return u.String()
}

// Headers returns the HTTP headers as a map.
func (a Action) Headers() map[string]string {
	return a.headers
}

// Vals returns the hx-vals as a JSON string.
func (a Action) Vals() string {
	if len(a.vals) == 0 {
		return ""
	}

	b, err := json.Marshal(a.vals)
	if err != nil {
		return "{}"
	}

	return string(b)
}

// Attr returns the HTMX attribute name for this action (e.g., "hx-get").
func (a Action) Attr() string {
	return fmt.Sprintf("hx-%s", a.method)
}

// String returns a string representation of the action.
func (a Action) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%s=%s", a.Attr(), a.URL()))

	if v := a.Vals(); v != "" {
		parts = append(parts, fmt.Sprintf("hx-vals='%s'", v))
	}

	if len(a.headers) > 0 {
		h, _ := json.Marshal(a.headers)
		parts = append(parts, fmt.Sprintf("hx-headers='%s'", string(h)))
	}

	return strings.Join(parts, " ")
}
