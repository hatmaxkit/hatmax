# Serve a Page

This chapter follows one full page and one HTMX partial through the companion
application. By the end, a form replaces one element without replacing the
complete document.

## Before You Begin

Complete [Add Postgres](postgres.md), then run the guide application with
database features enabled.

## Inspect the templates

`examples/guide/assets/templates/home/page.html` is the complete page.
`status.html` is the partial response. `web.NewTemplateManager` parses both
files during startup from the embedded filesystem.

The name form declares:

```html
<form hx-post="/name" hx-target="#name-result" hx-swap="innerHTML">
```

The browser posts the form to `/name` and replaces only `#name-result` with the
returned partial.

## Follow route registration

`pages.RegisterRoutes` installs the handlers after all startup functions
succeed. `pages.home` renders `home/page`; `pages.checkName` renders
`home/status`.

Change the initial text `Waiting` in `page.html` to `Enter a name`, restart the
process, and reload the page. The new text comes from the embedded template.
Restore `Waiting` before continuing.

## Check the result

Open `http://localhost:8080/`, enter `Ada`, and choose **Check name**. The
result changes to `Accepted Ada` without replacing the page.

You can inspect the partial directly:

```sh
curl -fsS \
  -H 'Origin: http://localhost:8080' \
  -H 'HX-Request: true' \
  -d 'name=Ada' \
  http://localhost:8080/name
```

The response is `Accepted Ada`.

## Recover from template problems

A missing template is a startup error when the filesystem walk cannot read it,
or a render-time `500` when a handler asks for an unknown template name. Match
the namespace and filename used by the handler.

See [HTTP](../../reference/http/README.md) and
[HTMX](../../reference/htmx/README.md) for exact behavior.

Continue with [Accept a Form](forms.md).

---

[Previous: Add Postgres](postgres.md) · [User Guide](README.md) ·
[Next: Accept a Form](forms.md)
