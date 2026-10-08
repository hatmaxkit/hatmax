<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# The Request Boundary

An HTTP request enters a Hatmax application through an explicit chain:

```text
browser -> router middleware -> route middleware -> handler -> service
```

The chain is also a sequence of ownership boundaries. Middleware establishes
facts that apply across requests. A handler translates HTTP input into an
application call. A service decides what the application is allowed to do.
Keeping those roles distinct makes the behavior of a route visible without
moving product rules into transport code.

The Go blocks below are contextual fragments, not standalone programs. The
[guide companion](../../../examples/guide/main.go) is the complete runnable
composition for lifecycle, routes, pages and forms. Invoice-specific fields,
handlers and services are application-owned illustrative names; they require
that feature's implementation. Use the
[bootstrap procedure](../../how-to/bootstrap-application/README.md) for a
complete signal-aware process with a health endpoint.

## Build the Router Once

The composition root constructs one router and installs application-wide
middleware before any routes become reachable:

```go
router := app.NewRouter(
	logger,
	app.WithMiddleware(
		middleware.DefaultStack()...,
	),
	app.WithPing(),
)
```

`DefaultStack` adds request IDs, connection-peer identity, request logging,
and panic recovery. Forwarded client IPs require explicitly approved proxy
networks; the zero-argument call ignores those headers. See the
[trusted proxy contract](../../reference/middleware/README.md#client-ip-and-trusted-proxies)
when deploying behind a proxy. Applications add only the middleware required
by their contract, such as `middleware.RequireSameOrigin`, locale selection,
telemetry, or a rate
limit. Middleware order is behavior: each entry wraps the entries and handler
that follow it.

The router is constructed before components start, but feature routes are
registered only after startup succeeds. A handler participates in that phase
by implementing `RegisterRoutes(chi.Router)`. This preserves the lifecycle
rule from the previous chapter: a request cannot reach a feature whose
dependencies failed to start.

## Place Policy at the Narrowest Boundary

Application-wide middleware belongs on the root router. Policy for a group of
routes belongs on a router group:

```go
func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/invoices", h.list)

	router.Group(func(router chi.Router) {
		router.Use(h.requireAccount)
		router.Post("/invoices", h.create)
	})
}
```

Route grouping says where the policy applies without repeating checks inside
every handler. Authentication and role middleware can put a verified user in
the request context; handlers consume that fact. Product authorization still
belongs in the application service when it depends on the resource or
operation rather than only on the route.

## Keep Handlers at the Translation Edge

A handler owns HTTP-specific work:

- read path, query, form, header, and context values;
- reject malformed transport input;
- invoke a service with typed application values;
- translate a known result or error into an HTTP response;
- select a full page, partial, redirect, or status response.

It should not own persistence queries or repeat domain decisions. Conversely,
a service should not inspect `http.Request`, select template names, or write
response headers. That split allows the same application behavior to be
called from a web route, a scheduled task, or a test without recreating HTTP.

Use `web.ParseIDParam` for UUID route parameters and `web.ParseForm` for form
input. Return `400` when the request cannot be parsed, `401` or `403` when the
established identity is insufficient, `404` for a known missing resource, and
`500` for an unexpected internal failure. Log internal context, but send only
safe, stable messages to the browser.

## Protect State-Changing Requests

`middleware.RequireSameOrigin` allows safe methods and checks the source of
other methods. It rejects a cross-site source, a malformed `Origin` or
`Referer`, and a source whose scheme or host does not match the request.

Install it before state-changing routes are registered:

```go
router := app.NewRouter(
	logger,
	app.WithMiddleware(middleware.RequireSameOrigin),
)
```

Same-origin enforcement is one browser boundary. It does not replace
authentication, resource authorization, output escaping, or any additional
CSRF policy the application requires.

## Follow One Request

For a request to create an invoice, the normal path is:

```text
POST /invoices
  -> request identity and recovery middleware
  -> same-origin and account policy
  -> handler parses the form
  -> service validates and creates the invoice
  -> handler selects the browser response
```

The route is visible, the policy is visible, and each failure is handled at
the boundary that understands it. The next chapters expand the two web-specific
parts of this path: selecting HTML responses and accepting form input.

For exact router lifecycle and middleware behavior, see
[Application Lifecycle](../../reference/application-lifecycle/README.md) and
[Middleware](../../reference/middleware/README.md).

---

[Previous: Lifecycle and Wiring](lifecycle-and-wiring.md) ·
[User Guide](README.md) ·
[Next: Pages and Partials](pages-and-partials.md)
