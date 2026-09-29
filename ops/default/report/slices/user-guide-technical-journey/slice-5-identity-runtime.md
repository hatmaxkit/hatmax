# Slice 5: Identity and Runtime

Status: delivered
Delivery set: user-guide-technical-journey
Plan: [User Guide Technical Journey Plan](../../../plan/user-guide-technical-journey.md)
Tracker: [User Guide Technical Journey Tracker](../../../tracker/user-guide-technical-journey.md)
Branch: `docs/user-guide-identity-runtime`
PR: #52

## Purpose

Explain identity, sessions, authorization, cryptographic support, static
configuration, logging, and mutable settings as distinct but connected
application boundaries.

## Delivered Behavior

The User Guide now follows credentials through the authentication service,
application query adapter, durable session, secure cookie, middleware, and
request context. It separates authenticated identity, route-level roles, and
resource authorization, and places TOTP and cryptographic primitives without
introducing an alternate authentication architecture.

The runtime chapter separates startup configuration from schema-checked
mutable settings. It explains loading precedence, explicit validation,
constructor ownership, safe logging, settings schemas and adapters, fallback
behavior, application workflows, and the lifetime test for choosing the
correct layer.

The previous Sign In and Change Settings at Runtime exercises were removed
after their durable material was incorporated. Inbound journey links now use
the new chapters.

## Contracts Added or Changed

- `auth.Service` owns authentication workflows through an application-supplied
  `auth.Queries` adapter.
- Secure cookies and middleware establish identity; services retain
  resource-specific authorization.
- `crypto` supplies bounded primitives rather than a parallel auth stack.
- Static configuration is loaded and validated once before construction.
- Runtime settings use application-owned schemas, workflows, and storage.
- Logs exclude credentials, tokens, keys, and decrypted personal data.

## Files of Interest

- `docs/tutorials/user-guide/identity-and-sessions.md`
- `docs/tutorials/user-guide/configuration-and-runtime-settings.md`
- `docs/tutorials/user-guide/README.md`

## Validation

- `make docs-check` passed.
- Capability audit passed for `auth`, `crypto`, `config`, `settings`, and
  `log`.
- Authentication, cookie, middleware, configuration, setting fallback, and
  logging claims were checked against current reference contracts.
- Superseded `sign-in.md` and `settings.md` link audit passed.
- `git diff --check` passed.
- Pull request #52 merged into `dev` at
  `d1bf11a5c750dccb6954a54700c5e01bb2634a92`.

## Risks and Follow-ups

Events, background jobs, mail, images, and telemetry are deferred to Slice 6.
This slice names their configuration only as startup-owned input.
