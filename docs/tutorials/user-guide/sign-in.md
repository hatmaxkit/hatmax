# Sign In

This chapter uses the Ticked reference application to create an account, store
a session, and reach an authenticated route.

## Before You Begin

Read [Forms and Validation](forms-and-validation.md). Stop the guide
application so port `8080` is available. Ensure Postgres accepts the defaults in
`examples/ticked/config.yaml`.

## Prepare Ticked

From the repository root:

```sh
cd examples/ticked
make db-init
make run-fg
```

The database, migrations, templates, broker, stores, services, and handlers
start in the order declared by `examples/ticked/main.go`.

## Create a session

In another terminal, store cookies in a temporary jar:

```sh
curl -sS -c /tmp/ticked-guide.cookies -b /tmp/ticked-guide.cookies \
  -d 'email=ada@example.com&password=password1&confirm_password=password1' \
  -D - -o /dev/null \
  http://localhost:8080/signup
```

The response is `200`, includes `HX-Redirect: /list-items`, and sets the
`session` cookie. The first account in a new Ticked database receives the
administrative role used in later chapters.

Open the protected page with that cookie:

```sh
curl -fsS -b /tmp/ticked-guide.cookies \
  http://localhost:8080/list-items | grep -F 'ada@example.com'
```

Without the cookie, the same path redirects to `/signin`.

## Understand the boundary

`auth.Service` validates credentials and sessions through the
application-owned `auth.Queries` implementation. `auth.SetSessionCookie`
creates a secure cookie. Ticked's route middleware validates the token before
it stores the user in the request context.

The development check uses curl against localhost. For browser authentication,
serve the application over HTTPS so the browser sends the secure cookie.

## Recover from sign-in problems

- `Invalid email or password` means the user lookup or password comparison
  failed.
- A redirect back to `/signin` means the cookie is absent, expired, or not
  accepted by the browser transport.
- A database error during signup is logged by Ticked and does not create a
  usable session.

## Verify the result

This chapter is complete when the cookie-authenticated request renders the
account email and the same request without the cookie redirects.

See [Authentication](../../reference/authentication/README.md) and the focused
[authentication how-to](../../how-to/add-authentication/README.md).

Continue with [Models and Data Flow](models-and-data-flow.md).

---

[Previous: Forms and Validation](forms-and-validation.md) · [User Guide](README.md) ·
[Next: Models and Data Flow](models-and-data-flow.md)
