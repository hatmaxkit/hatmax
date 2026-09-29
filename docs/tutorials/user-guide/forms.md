# Accept a Form

This chapter exercises same-origin request protection and field validation. By
the end, cross-site input is refused and local input receives useful feedback.

## Before You Begin

Complete [Serve a Page](pages.md) and keep the guide application running.

The router installs `middleware.RequireSameOrigin` before it registers routes.
The `/name` handler trims the submitted value and applies `Required` and
`MinLength(2)` through `validation.Field`.

## Check the request boundary

A state-changing request without `Origin` or `Referer` is refused:

```sh
curl -sS -o /dev/null -w '%{http_code}\n' \
  -d 'name=Ada' http://localhost:8080/name
```

The response status is `403`.

## Check validation

Send same-origin requests:

```sh
curl -fsS -H 'Origin: http://localhost:8080' \
  -d 'name=' http://localhost:8080/name

curl -fsS -H 'Origin: http://localhost:8080' \
  -d 'name=A' http://localhost:8080/name

curl -fsS -H 'Origin: http://localhost:8080' \
  -d 'name=Ada' http://localhost:8080/name
```

The responses report a required value, a minimum length, and `Accepted Ada`.

## Verify the result

Repeat the valid and invalid values in the browser. HTMX replaces only the
result paragraph. This chapter is complete when the same handler preserves
both the request boundary and validation behavior.

See [Middleware](../../reference/middleware/README.md) and
[Validation](../../reference/validation/README.md).

Continue with [Sign In](sign-in.md).

---

[Previous: Serve a Page](pages.md) · [User Guide](README.md) ·
[Next: Sign In](sign-in.md)
