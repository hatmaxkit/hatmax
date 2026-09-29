# Pages and Partials

Hatmax uses server-rendered HTML as the canonical browser response. An
ordinary navigation receives a complete page. An HTMX interaction can receive
only the fragment that changed. Both responses use the same handler, service,
template language, and application state.

```text
ordinary request -> handler -> full-page template -> complete document
HTMX request     -> handler -> partial template   -> targeted replacement
```

HTMX changes response granularity; it does not create a second client-side
application model.

## Own Templates with the Application

Embed application templates and construct one `web.TemplateManager` in the
composition root:

```go
//go:embed assets
var assetsFS embed.FS

templates := web.NewTemplateManager(
	assetsFS,
	logger,
	web.WithFuncMap(ui.FuncMap()),
)
```

The manager is a lifecycle component. During startup it walks the embedded
filesystem and parses every `.html` template. A read or parse failure stops
startup before routes are registered.

Organize templates by feature namespace:

```text
assets/templates/invoices/page.html
assets/templates/invoices/row.html
assets/templates/invoices/form.html
```

The handler selects a namespace and template without knowing the embedded
filesystem path:

```go
h.templates.Render(w, "invoices", "page", view)
h.templates.RenderPartial(w, "invoices", "row", invoice)
```

`Render` identifies a complete HTML response and sets its content type.
`RenderPartial` identifies a fragment response. Template execution failures
are logged and become `500` responses.

## Treat a Page as a View of Application State

A page handler gathers the state needed for one browser view, then passes a
view model to the template:

```go
type pageView struct {
	Invoices []invoiceView
	Errors   web.FormErrors
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.service.List(r.Context())
	if err != nil {
		h.log.Errorf("list invoices: %v", err)
		http.Error(w, "Cannot load invoices", http.StatusInternalServerError)

		return
	}

	h.templates.Render(w, "invoices", "page", pageView{Invoices: invoices})
}
```

The view model is application-owned presentation data. It prevents templates
from reaching through service or store objects and gives the handler one
explicit response contract.

## Make Partials Stable Replacement Units

A useful partial corresponds to a stable element in the page. A create form
might ask HTMX to append the returned invoice row:

```html
<form hx-post="/invoices"
      hx-target="#invoice-list"
      hx-swap="beforeend">
  <!-- fields -->
</form>

<div id="invoice-list">
  {{range .Invoices}}
    {{template "assets/templates/invoices/row.html" .}}
  {{end}}
</div>
```

After the service creates the invoice, the handler renders `row.html`. The
same row template is also used while rendering the full list, so ordinary and
HTMX responses do not drift into separate representations.

The `htmx` package can build attributes through `ui` components or template
functions. It also reads HTMX request headers and writes response headers for
redirects, retargeting, swaps, URL history, refreshes, and browser events. Use
those headers when the server needs to change the client action; keep normal
HTML fragments as the default response.

## Preserve Ordinary HTTP Behavior

An interaction should have an ordinary HTTP meaning whenever practical.
Forms retain an `action` and `method`; links retain an `href`. HTMX attributes
enhance the interaction rather than becoming its only definition.

When a successful action must navigate, use
`web.RedirectOrHXRedirect(w, r, url)`. It writes `HX-Redirect` for an HTMX
request and a `303 See Other` redirect for an ordinary request. This keeps one
handler correct for both clients.

For a response that deliberately differs by request type, make the decision
explicit:

```go
if htmx.IsHTMXRequest(r) {
	h.templates.RenderPartial(w, "invoices", "table", view)

	return
}

h.templates.Render(w, "invoices", "page", view)
```

Do not infer partial rendering from an arbitrary query flag when the actual
contract is an HTMX request.

## Keep Trust Boundaries Visible

Go templates escape ordinary strings. Hatmax UI renderers and some HTMX
helpers return trusted `template.HTML` or `template.HTMLAttr` values. Build
those values from known components and escaped inputs; do not cast untrusted
application or request data merely to bypass escaping.

For exact response headers, template functions, and rendering behavior, see
[HTMX](../../reference/htmx/README.md),
[HTTP](../../reference/http/README.md), and
[Rendering](../../reference/rendering/README.md). The architectural rationale
is in [Server-Rendered HTML with HTMX](../../explanation/server-rendered-htmx/README.md).

---

[Previous: The Request Boundary](request-boundary.md) ·
[User Guide](README.md) ·
[Next: Forms and Validation](forms-and-validation.md)
