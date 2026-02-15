# htmx

Type-safe HTMX abstractions for Go. Provides primitives for triggers, actions, targets, swaps, and response headers.

## Usage

### Attribute Builder

```go
// Fluent API for building hx-* attributes
attrs := htmx.HX().
    Post("/items").
    TargetID("list").
    SwapOuter().
    Confirm("Are you sure?")

// Use in templates
attrs.HTML()  // template.HTMLAttr
attrs.Map()   // map[string]string
```

### Primitives

```go
// Triggers
htmx.OnClick()
htmx.OnSubmit()
htmx.OnLoad()
htmx.OnRevealed()
htmx.OnChange().Delay(500 * time.Millisecond)
htmx.OnKeyup().Throttle(300 * time.Millisecond)
htmx.Every(5 * time.Second)

// Actions
htmx.Get("/items")
htmx.Post("/items").WithVal("id", "123")
htmx.Delete("/items/1").WithParam("force", "true")

// Targets
htmx.TargetID("my-element")      // #my-element
htmx.TargetSelf()                // this
htmx.TargetClosest("tr")         // closest tr
htmx.TargetFind(".content")      // find .content
htmx.ItemTarget("item", 123)     // #item-123

// Swaps
htmx.SwapOuter()
htmx.SwapInner()
htmx.SwapBeforeEnd()
htmx.SwapDelete()
htmx.SwapOuter().After(500 * time.Millisecond).ScrollTop()
```

### Response Headers

```go
func handler(w http.ResponseWriter, r *http.Request) {
    // Check if HTMX request
    if htmx.IsHTMXRequest(r) {
        htmx.Retarget(w, "#notifications")
        htmx.Reswap(w, htmx.SwapBeforeEnd())
    }

    // Redirects
    htmx.Redirect(w, "/dashboard")
    htmx.Refresh(w)

    // URL management
    htmx.PushURL(w, "/new-url")
    htmx.ReplaceURL(w, "/replaced")

    // Trigger events
    htmx.TriggerEvent(w, "itemAdded")
    htmx.TriggerEventWithData(w, "notify", map[string]string{"message": "Done"})

    // Stop polling
    htmx.StopPolling(w)  // returns 286
}
```

### Out-of-Band Updates

```go
// In templates
wrapper := htmx.OOBWrap("counter").Swap(htmx.OOBInner())
wrapper.Open()   // <div id="counter" hx-swap-oob="innerHTML">
wrapper.Close()  // </div>

// OOB strategies
htmx.OOBSwap()      // true (outerHTML)
htmx.OOBInner()     // innerHTML
htmx.OOBBeforeEnd() // beforeend
htmx.OOBDelete()    // delete
```

### Template FuncMap

```go
// Add HTMX helpers to templates
tmpl := template.New("").Funcs(htmx.FuncMap())

// Or combine with render helpers
funcs := render.FuncMapWithHTMX()
```

In templates:

```html
<!-- Actions -->
<button {{hxPost "/toggle" | .TargetID "item" | .SwapOuter | .HTML}}>Toggle</button>

<!-- Quick helpers -->
<button {{hxPostAttrs "/delete" "item-1"}}>Delete</button>

<!-- Triggers -->
<input {{hxGet "/search" | .Trigger (hxOnKeyup.Delay (ms 300)) | .TargetID "results" | .HTML}}>

<!-- Targets -->
<div hx-target="{{hxTargetClosest "tr"}}">...</div>
```

## Request Helpers

```go
htmx.IsHTMXRequest(r)      // true if HX-Request header
htmx.IsBoosted(r)          // true if HX-Boosted header
htmx.IsHistoryRestore(r)   // true if history restore
htmx.GetCurrentURL(r)      // HX-Current-URL value
htmx.GetPromptResponse(r)  // HX-Prompt value
htmx.GetRequestTarget(r)   // HX-Target value
htmx.GetRequestTrigger(r)  // HX-Trigger value (element ID)
```
