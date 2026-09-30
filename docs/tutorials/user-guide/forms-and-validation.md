<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Forms and Validation

A form crosses several boundaries before an application accepts its values:

```text
browser input -> HTTP parsing -> normalization -> validation -> service
              -> persistence constraints -> form response
```

Each boundary has a different job. Browser constraints improve interaction,
but the server remains authoritative. Parsing establishes types. Validation
produces application-safe values and structured errors. A service enforces
rules that depend on application state. Postgres preserves final data
integrity.

## Parse Transport Values in the Handler

Use `web.ParseForm` at the request boundary:

```go
form, err := web.ParseForm(r)
if err != nil {
	http.Error(w, "Invalid form", http.StatusBadRequest)

	return
}

number := form.String("number")
notes := form.String("notes")
```

`FormValues` also reads checkbox values, defaults, and UUID fields. A malformed
UUID is a transport error; a well-formed UUID for a missing or inaccessible
record is an application result handled after parsing.

Normalize text before validating it. `validation.NormalizeText` removes
control characters, trims the result, and collapses whitespace.
`NormalizeOptionalText` additionally turns empty optional text into `nil`.
Normalization should be deterministic and must not silently change the
meaning of domain data.

## Collect Structured Field Errors

Hatmax validation rules accumulate `validation.ValidationErrors` rather than
forcing the handler to stop at the first invalid field:

```go
number := validation.NormalizeText(form.String("number"))
notes := validation.NormalizeText(form.String("notes"))

errors := validation.ValidateAll(
	validation.Field("number", number).
		Required().
		MaxLength(40),
	validation.Field("notes", notes).
		MaxLength(2_000).
		NoHTML(),
)
if errors.HasErrors() {
	h.renderForm(w, invoiceForm{
		Number: number,
		Notes:  notes,
		Errors: web.FormErrorsFrom(errors, "Review the highlighted fields."),
	})

	return
}
```

The exact request type and helper functions are application choices; the
important contract is that field names remain consistent across HTML inputs,
validation errors, and the form view model.

`web.FormErrorsFrom` extracts safe field messages from a wrapped
`validation.ValidationErrors` value. It uses the supplied general message and
does not expose an arbitrary source error to the browser. Templates can use
`First`, `For`, `Has`, and `Any` to place feedback beside a field and at the
form summary.

## Keep Validation in Its Owning Layer

Validation is not one large handler block. Place a rule where its required
knowledge lives:

| Rule | Owner |
| --- | --- |
| Request body can be parsed | handler |
| Text normalization and scalar shape | validation or model constructor |
| Required field and length constraints | model or service input validation |
| Current account may perform the action | service |
| Invoice number must be unique | service and store query |
| Column cannot be null or duplicated | Postgres constraint |

The handler can invoke validation, but domain-valid input should be defined by
the feature rather than by one route. A second caller of the same service must
receive the same application rule. Database constraints remain necessary for
concurrent writes even when the service checks first.

Use validation errors for expected user-correctable input. Preserve other
errors for logging and stable HTTP translation. Do not return raw database or
internal error strings as form feedback.

## Render Invalid and Valid Outcomes

An invalid submission should preserve the submitted values and render the
same form with structured feedback. With HTMX, return the form partial and
target the form container. Without HTMX, render the complete page containing
that form. The status code and swap policy are application decisions, but the
server-owned validation result must be identical.

On success, select the response that represents the completed action:

- render the created or updated partial when the page can change in place;
- return an empty successful response for a deliberate deletion;
- use `web.RedirectOrHXRedirect` when the browser should navigate;
- trigger a named HTMX event when another stable page region owns the refresh.

HTML attributes such as `required`, `maxlength`, and an input type provide
immediate browser feedback. HTMX's `Validate` attribute can ask the browser to
run its constraint validation for an HTMX request. These are interaction aids,
not substitutes for the same Hatmax validation on the server.

## Preserve the Browser Security Boundary

State-changing forms must pass the request policy established in
[The Request Boundary](request-boundary.md). `middleware.RequireSameOrigin`
rejects unsafe cross-site methods before the handler parses their values. A
UI form can also carry an application-provided CSRF token through
`ui.WithCSRFFunc`; token generation and verification remain an explicit
application security decision.

For the exact field rules and error structures, see
[Validation](../../reference/validation/README.md). For form parsing and
presentation behavior, see [HTTP](../../reference/http/README.md) and
[UI](../../reference/ui/README.md).

---

[Previous: Pages and Partials](pages-and-partials.md) ·
[User Guide](README.md) ·
[Next: Presentation Primitives](presentation-primitives.md)
