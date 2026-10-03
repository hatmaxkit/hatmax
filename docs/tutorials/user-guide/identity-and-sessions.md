<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Identity and Sessions

Identity crosses persistence, application, HTTP, and cryptographic boundaries:

```text
credentials -> auth service -> user and session store -> secure cookie
secure cookie -> auth middleware -> validated user -> request context
```

Hatmax supplies the authentication workflow and browser boundary. The
application supplies the `auth.Queries` adapter, chooses which routes require
identity, and enforces resource-specific authorization in its services.

## Persist Users and Sessions Through the Contract

`auth.Service` depends on the caller-supplied `auth.Queries` interface. Its
operations create and load users, create and delete sessions, and remove
expired sessions. A Postgres application implements that contract with its
own migrations, SQLC queries, and adapter wiring.

The service normalizes the workflow around stable failures. Signup validates
email and password requirements, rejects an existing address, hashes the
password, and creates the user. Sign-in verifies the stored hash and creates a
session with configured absolute and inactivity lifetimes. Session validation
checks current account version and expiry under locks before returning owned
user and safe session metadata. Password, activation and role changes invalidate
old sessions.

Handlers translate those results into safe form feedback. They do not expose
whether an address exists, a password hash failed, or a database operation
failed when that distinction would leak account information.

## Carry the Session in a Secure Cookie

Sign-in requires current server policy and returns an authentication outcome.
Only `result.CompletedSession()` permits setting its token with
`auth.SetSessionCookie`. Pending enrollment/proof and unavailable-method denial
contain no session or continuation and must not set a cookie. Password proof
cannot satisfy MFA or phishing resistance, including for an enrolled user. Hatmax uses
the `session` name and applies `HttpOnly`, `Secure`, `SameSite=Lax`, path `/`,
and the supplied maximum age. Sign-out deletes the durable session and clears
the browser cookie.

Because the cookie is secure, normal browser authentication requires HTTPS.
A command-line localhost check can send it explicitly, but weakening the
production cookie contract to simplify local testing creates a different
security model.

`auth.RequireAuth` reads the cookie, validates the session, and stores the user,
user ID and safe session metadata in the request context. Missing or invalid
identity redirects to
`/signin`. `auth.OptionalAuth` continues anonymously when validation fails.

Install authentication on the narrowest route group that needs it:

```go
required := auth.AccessRequirement{Proof: auth.RequirePassword, Revision: "password-v1"}
router.Group(func(router chi.Router) {
	router.Use(auth.RequireAuth(authService, required, auth.RelevantActivity))
	router.Get("/invoices", invoiceHandler.page)
})
```

Handlers retrieve the established identity with `auth.GetUser` or
`auth.GetUserID`; `auth.GetSession` exposes safe lifecycle metadata. They do not
parse session tokens again. Select `auth.NoActivity` for background polling;
trusted relevant activity refreshes inactivity only at the persistence cadence.
It never extends absolute expiry or the authentication time. Operation policy
can require recent proof with `MaxAge`; exact age equality rejects it. Session
policy revisions must match current trusted policy, and activity cannot refresh
password verification time.

## Reauthenticate and Manage Sessions

Call `Reauthenticate(ctx, token, password, required)` for a live session when
proof is no longer recent. It verifies the password again and rotates the bearer
atomically. Update the cookie only after completed issuance. Failed work keeps
the old valid session; an expired session needs a new sign-in. Password
reauthentication cannot meet a stronger current policy.

`ListSessions` and `RevokeSessions` require recent proof and derive ownership
from the actor bearer. Choose current, selected, others or all for revocation.
Lists contain safe metadata with a bounded next cursor. All/current revocation
invalidates the actor; clear its browser cookie. Session admission rejects at
capacity instead of evicting another device. Expired rows are reclaimed under the
same subject lock and through explicit bounded cleanup.

Ticked's **Sessions** link opens its own session list and reauthentication form.
Reauthenticate before managing sessions when the screen requires recent proof.
These self-service operations accept no target subject or administrator claim;
applications define any cross-subject administrative workflow separately.

## Separate Authentication from Authorization

Authentication answers who is making the request. Route middleware can enforce
coarse role requirements with `middleware.RequireRole`, `RequireAnyRole`, or
`RequireRoles`. Application services still decide whether that identity may
act on a particular invoice, account, or tenant.

Do not treat a hidden button or a successful session as resource
authorization. Pass the verified identity into the service workflow and query
data within that ownership boundary.

TOTP policy follows the same division. `auth.RequireTOTP` can redirect a
signed-in user who must finish setup, while the application chooses the route,
grace policy, recovery flow, and protected operations.

## Choose the Correct Cryptographic Primitive

The `auth` service uses Hatmax model password hashing and opaque stored session
token digests. Raw 32-byte random bearer secrets appear only on issuance and in
the secure cookie. The separate `crypto` package supports other application needs:

- AES-256-GCM for authenticated encryption of strings;
- HMAC lookup hashes for searchable protected values;
- Argon2id helpers for application-owned password-like secrets;
- signed PASETO tokens for explicit token-based protocols;
- TOTP secrets, codes, QR images, and backup codes.

These are primitives, not an alternate authentication stack. Choose them only
for a defined application boundary. Keep encryption and lookup keys in static
secret configuration, use the same associated data for encryption and
decryption, and plan key rotation before storing protected values.

Never log credentials, session tokens, encryption keys, TOTP secrets, backup
codes, or decrypted personal data. Logs may identify the operation and stable
record IDs needed for diagnosis.

## Test the Identity Boundary

Service tests cover signup, password rejection, expiry, inactive users, and
query failures with a small `auth.Queries` fake. Handler and middleware tests
cover cookies, redirects, authenticated context, roles, and safe error
messages. Postgres integration tests cover the application query adapter and
session cleanup.

For exact contracts, see
[Authentication](../../reference/authentication/README.md),
[Crypto](../../reference/crypto/README.md), and
[Middleware](../../reference/middleware/README.md). The focused
[authentication how-to](../../how-to/add-authentication/README.md) covers the
integration procedure.

---

[Previous: Models and Data Flow](models-and-data-flow.md) ·
[User Guide](README.md) ·
[Next: Configuration and Runtime Settings](configuration-and-runtime-settings.md)
