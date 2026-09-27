# Server-Rendered HTML with HTMX

Hatmax treats HTML rendered by Go templates as the canonical web response.
HTMX adds targeted requests and replacement behavior without requiring a
second client-side application model.

The `web` package owns template loading and response execution. The `render`
and `ui` packages provide template functions and HTML components. The `htmx`
package constructs attributes and response headers. None of those packages
owns application routes or domain state.

An ordinary request can receive a complete page. An HTMX request can receive
a partial template and select where it is applied. The handler remains the
place that validates input, invokes application behavior, and selects the
response.

This approach favors explicit HTTP behavior and progressive enhancement. It
does not require every interaction to use HTMX, and it does not make generated
HTML safe by default: callers must still distinguish escaped text from trusted
`template.HTML` values.

See the [HTTP](../../reference/http/index.md),
[HTMX](../../reference/htmx/index.md), and [UI](../../reference/ui/index.md)
references for exact contracts.
