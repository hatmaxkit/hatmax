# Failure and Security Boundaries

Hatmax packages expose failures at the boundary that can act on them. Startup
returns infrastructure errors before routes are registered. Handlers translate
validation and service errors into HTTP responses. Background runners record
job outcomes through their stores.

Fallbacks are intentionally narrow. Invalid duration strings may use documented
defaults, but authentication failures do not become anonymous success and
invalid encrypted data does not become plaintext. A caller must still decide
which errors are safe to show to a user and which belong only in logs.

## Browser request boundaries

Authentication middleware validates a session before it stores a user in the
request context. Role middleware acts on that confirmed user. Same-origin
middleware rejects unsafe cross-site methods, but it does not replace output
escaping, authorization, or application-specific CSRF decisions.

Secure session cookies require HTTPS. Local command-line checks can inspect
the contract, but a browser login workflow should use TLS or an explicitly
development-only cookie policy owned by the application.

## Secret boundaries

Configuration may contain database passwords, encryption keys, mail-provider
credentials, and other secrets. Documentation and logs should name the fields
without printing live values. The crypto package validates key sizes and
authenticated data; key storage and rotation remain application operations.

See [Authentication](../../reference/authentication/index.md),
[Middleware](../../reference/middleware/index.md), and
[Crypto](../../reference/crypto/index.md).
