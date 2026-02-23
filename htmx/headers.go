package htmx

import (
	"encoding/json"
	"net/http"
)

// HTMX Response Headers
const (
	HeaderLocation           = "HX-Location"
	HeaderPushURL            = "HX-Push-Url"
	HeaderRedirect           = "HX-Redirect"
	HeaderRefresh            = "HX-Refresh"
	HeaderReplaceURL         = "HX-Replace-Url"
	HeaderReswap             = "HX-Reswap"
	HeaderRetarget           = "HX-Retarget"
	HeaderReselect           = "HX-Reselect"
	HeaderTrigger            = "HX-Trigger"
	HeaderTriggerAfterSettle = "HX-Trigger-After-Settle"
	HeaderTriggerAfterSwap   = "HX-Trigger-After-Swap"
)

// HTMX Request Headers
const (
	HeaderRequest        = "HX-Request"
	HeaderBoosted        = "HX-Boosted"
	HeaderCurrentURL     = "HX-Current-URL"
	HeaderHistoryRestore = "HX-History-Restore-Request"
	HeaderPrompt         = "HX-Prompt"
	HeaderRequestTarget  = "HX-Target"
	HeaderTriggerName    = "HX-Trigger-Name"
	HeaderRequestTrigger = "HX-Trigger"
)

// IsHTMXRequest returns true if the request was made by HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get(HeaderRequest) == "true"
}

// IsBoosted returns true if the request is from a boosted element.
func IsBoosted(r *http.Request) bool {
	return r.Header.Get(HeaderBoosted) == "true"
}

// IsHistoryRestore returns true if the request is a history restoration request.
func IsHistoryRestore(r *http.Request) bool {
	return r.Header.Get(HeaderHistoryRestore) == "true"
}

// GetCurrentURL returns the current URL from the request headers.
func GetCurrentURL(r *http.Request) string {
	return r.Header.Get(HeaderCurrentURL)
}

// GetPromptResponse returns the user's response to an hx-prompt.
func GetPromptResponse(r *http.Request) string {
	return r.Header.Get(HeaderPrompt)
}

// GetRequestTarget returns the id of the target element.
func GetRequestTarget(r *http.Request) string {
	return r.Header.Get(HeaderRequestTarget)
}

// GetRequestTrigger returns the id of the element that triggered the request.
func GetRequestTrigger(r *http.Request) string {
	return r.Header.Get(HeaderRequestTrigger)
}

// GetTriggerName returns the name of the element that triggered the request.
func GetTriggerName(r *http.Request) string {
	return r.Header.Get(HeaderTriggerName)
}

// Redirect tells HTMX to do a client-side redirect.
func Redirect(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderRedirect, url)
}

// Refresh tells HTMX to do a full page refresh.
func Refresh(w http.ResponseWriter) {
	w.Header().Set(HeaderRefresh, "true")
}

// Retarget changes the target of the response.
func Retarget(w http.ResponseWriter, selector string) {
	w.Header().Set(HeaderRetarget, selector)
}

// Reswap changes the swap strategy of the response.
func Reswap(w http.ResponseWriter, swap Swap) {
	w.Header().Set(HeaderReswap, swap.String())
}

// ReswapStr changes the swap strategy using a string value.
func ReswapStr(w http.ResponseWriter, strategy string) {
	w.Header().Set(HeaderReswap, strategy)
}

// Reselect changes which part of the response is swapped.
func Reselect(w http.ResponseWriter, selector string) {
	w.Header().Set(HeaderReselect, selector)
}

// PushURL pushes a new URL into the browser's history.
func PushURL(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderPushURL, url)
}

// PreventPushURL prevents the URL from being pushed to the history.
func PreventPushURL(w http.ResponseWriter) {
	w.Header().Set(HeaderPushURL, "false")
}

// ReplaceURL replaces the current URL in the browser's history.
func ReplaceURL(w http.ResponseWriter, url string) {
	w.Header().Set(HeaderReplaceURL, url)
}

// PreventReplaceURL prevents the URL from being replaced in the history.
func PreventReplaceURL(w http.ResponseWriter) {
	w.Header().Set(HeaderReplaceURL, "false")
}

// TriggerEvent triggers a client-side event.
func TriggerEvent(w http.ResponseWriter, event string) {
	w.Header().Set(HeaderTrigger, event)
}

// TriggerEventAfterSettle triggers a client-side event after the settle step.
func TriggerEventAfterSettle(w http.ResponseWriter, event string) {
	w.Header().Set(HeaderTriggerAfterSettle, event)
}

// TriggerEventAfterSwap triggers a client-side event after the swap step.
func TriggerEventAfterSwap(w http.ResponseWriter, event string) {
	w.Header().Set(HeaderTriggerAfterSwap, event)
}

// TriggerEventWithData triggers a client-side event with JSON data.
func TriggerEventWithData(w http.ResponseWriter, event string, data any) {
	eventData := map[string]any{event: data}

	b, err := json.Marshal(eventData)
	if err != nil {
		w.Header().Set(HeaderTrigger, event)

		return
	}

	w.Header().Set(HeaderTrigger, string(b))
}

// TriggerEvents triggers multiple client-side events.
func TriggerEvents(w http.ResponseWriter, events map[string]any) {
	b, err := json.Marshal(events)
	if err != nil {
		return
	}

	w.Header().Set(HeaderTrigger, string(b))
}

// Location does a client-side redirect with additional options.
type LocationConfig struct {
	Path    string            `json:"path"`
	Source  string            `json:"source,omitempty"`
	Event   string            `json:"event,omitempty"`
	Handler string            `json:"handler,omitempty"`
	Target  string            `json:"target,omitempty"`
	Swap    string            `json:"swap,omitempty"`
	Values  map[string]string `json:"values,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Select  string            `json:"select,omitempty"`
}

// Location performs a client-side redirect with configurable options.
func Location(w http.ResponseWriter, config LocationConfig) {
	if config.Target == "" && config.Swap == "" && len(config.Values) == 0 &&
		len(config.Headers) == 0 && config.Select == "" && config.Source == "" &&
		config.Event == "" && config.Handler == "" {
		w.Header().Set(HeaderLocation, config.Path)

		return
	}

	b, err := json.Marshal(config)
	if err != nil {
		w.Header().Set(HeaderLocation, config.Path)

		return
	}

	w.Header().Set(HeaderLocation, string(b))
}

// LocationSimple performs a simple client-side redirect to a path with a target.
func LocationSimple(w http.ResponseWriter, path, target string) {
	if target == "" {
		w.Header().Set(HeaderLocation, path)

		return
	}

	Location(w, LocationConfig{Path: path, Target: target})
}

// StopPolling returns status code 286 to tell HTMX to stop polling.
func StopPolling(w http.ResponseWriter) {
	w.WriteHeader(286)
}

// RespondEmpty sends an empty response (useful with hx-swap="none").
func RespondEmpty(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// RespondDelete sends an empty response for delete operations.
// This is useful when the target element should be removed.
func RespondDelete(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}
