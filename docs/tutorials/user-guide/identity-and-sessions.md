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

## Bound Password Entry Across Replicas

Supply a shared durable `CredentialAdmission` when constructing authentication.
Registration has its own window. Sign-in, password reauthentication, initial
WebAuthn enrollment and password-based fallback share a proof window for the
canonical account identity. Session reauthentication derives that identity from
the live actor. Core does not fold email case or invent aliases; application
lookup and admission must receive the same canonical value.

Successful and failed admitted work consumes one charge. Default proof admission
permits ten operations in ten minutes with a ten-minute cooldown; registration
permits three in one hour with a one-hour cooldown. Denial does not extend the
cooldown. Restart, session issuance and credential changes do not reset it.
Invalid encoding/byte bounds perform no password work. Storage failure is a
failure to admit work; do not retry an ambiguous charge automatically.

Ticked requires `TICKED_CREDENTIAL_NAMESPACE` and `TICKED_CREDENTIAL_KEY` at
startup. Provision a stable namespace and canonical base64 key containing 32–64
random bytes through application-owned private configuration. All replicas share
that material and consistent limits. Keep it separate from TOTP encryption keys
and never commit it or expose it in forms/logs. Schedule bounded admission cleanup
explicitly. See [configuration](../../reference/configuration/README.md#credential-admission-limits)
for validated limits and [authentication](../../reference/authentication/README.md#standalone-credential-admission)
for the storage contract. Public response/timing policy remains application-owned.

## Use Neutral Registration and Password Responses

Ticked registration processes one bounded form, then directs you to normal
sign-in. New and duplicate addresses receive the same acknowledgment; registration
does not sign you in or set a session cookie. Acknowledgment does not confirm
that an account was created. Password-policy feedback can ask you to choose a
longer or different candidate without disclosing account state.

Public sign-in, initial enrollment and fallback password failures use one message
and acknowledgment target. Missing or inactive accounts, wrong passwords and
exhausted proof budgets do not receive account-specific retry information. The
common ingress defaults to twelve requests per peer per minute and 32 active
requests; waits occupy that capacity and canceled requests release it. Syntax,
origin and overall admission errors can be rejected before account lookup.
See [HTTP ingress configuration](../../reference/configuration/README.md#authentication-http-ingress)
for finite limits and trusted-proxy requirements.

Use the [browser acceptance command](../../reference/authentication/README.md#authentication-controls-browser-acceptance)
when checking an application composition against real PostgreSQL and Chromium.
It verifies neutral registration and password replies, active-request capacity,
redacted observations and retained MFA/recovery without granting access through
fixture controls. Local browser evidence does not establish deployment assurance.

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

## Initial Authenticator Enrollment

Construct `auth.NewAuthenticatorService` with the existing credential service,
required atomic `AuthenticatorQueries` and explicit RP/origin configuration.
Start enrollment by verifying the account password again. Treat the returned
setup bearer as permission for this ceremony only; it must not enter application
cookies, URLs or the session middleware. Pass the browser registration response
to `FinishWebAuthnEnrollment` using the same trusted access requirement.

A confirmed key advances account version and invalidates previous sessions and
pending setup. Do not set a new authentication cookie from the safe factor result:
registration is not a completed strong sign-in. Initial setup rejects accounts
with established factors. Assertion completion and authorized factor changes are
not available in this delivery yet. Ticked demonstrates the JSON begin/finish
endpoints; see the [enrollment reference](../../reference/authentication/README.md#restricted-webauthn-enrollment)
for the browser call and finite storage/input requirements.


## Sign In or Step Up with WebAuthn

After registration, construct `auth.NewWebAuthnService` using the required store
and explicit RP settings. In Ticked, post JSON `email` to
`/authenticators/authentication/begin`, decode its browser options and pass them
to `navigator.credentials.get`. Send the credential JSON to
`/authenticators/authentication/finish` with `X-Assertion-Token` from begin.
The handler sets an application cookie only after actual assertion verification
and the final database commit. The challenge token grants no application access.

To upgrade a live password session, post `{}` and its existing cookie to
`/authenticators/step-up/begin`. Complete the returned ceremony through
`/authenticators/step-up/finish`. Successful completion replaces the actor cookie
and invalidates its previous bearer. `/authenticators/proof` requires current
recent phishing-resistant proof; the ordinary password session cannot enter it.
Errors issue no cookie and preserve an existing session cookie.

Choose current requirements in server code. UV WebAuthn meets password-or-better,
MFA and phishing-resistant MFA in the supported profile. Activity cannot renew
its verification time; password reauthentication gives password proof only.
Routine counter updates preserve other valid sessions, while removal/security
revision changes invalidate bound proof. Migration 005 adds the required session
and assertion representation after migration 004. See the
[WebAuthn reference](../../reference/authentication/README.md#webauthn-authentication-and-step-up)
for the exact contracts and response bounds. Browser acceptance
and established-factor management are separate remaining delivery boundaries.

## Sign In with TOTP or a Backup Code

Supply an application-managed encryption key identity and 32-byte key to
`auth.NewFallbackService`. Ticked reads these from `TICKED_TOTP_KEY_ID` and
`TICKED_TOTP_KEY` (standard Base64); supply both before startup. Keep keys outside
configuration tracked in Git. Apply migration 006 after the WebAuthn migrations.

For a new account with no established factors, post email/password to
`/authenticators/totp/setup/begin`. Import the returned provisioning URL into your
OTP application. Post its current six-digit code to
`/authenticators/totp/setup/finish`, using the returned `X-Fallback-Token`.
Confirmation consumes that step and signs out prior sessions. It grants no access.
Wait for the next OTP step before signing in; reusing setup's code fails.

Post email/password to `/authenticators/totp/authentication/begin`, then send the
actual OTP code to `/authenticators/fallback/authentication/finish` with the
returned token header. A successful committed completion sets the session cookie.
`/authenticators/mfa` accepts this recent password-plus-factor proof;
`/authenticators/proof` requires phishing-resistant proof and rejects it.
Step-up uses the matching `/totp/step-up/begin` route with the actor cookie and
password, then `/fallback/step-up/finish`.

Create a backup set through `/authenticators/backup/issue` with `{}` and a recent
authorized cookie. Ticked requires phishing-resistant MFA here. Applications
allowing TOTP-only management can explicitly wire recent `RequireMFA` in trusted
server code; requests cannot select or weaken this policy. Store returned codes
once in a safe place. Issue replaces previous codes, rotates the current cookie,
and signs out other sessions. It preserves the actor's original proof times.

Use `/authenticators/backup/authentication/begin` with email/password, then submit
one exact backup code to the shared finish route. Do not alter spaces, case or
encoding. The code is consumed only with successful access completion. Backup
proof satisfies MFA, and cannot regenerate its own set. See the
[fallback reference](../../reference/authentication/README.md#totp-and-backup-proof)
for bounds, freshness and storage requirements. The management and browser
sections below cover additional-factor/removal journeys.

After actual recent stronger sign-in or step-up, Ticked's
`/authenticators/manage` page can add, replace or remove your own authenticators.
The example requires phishing-resistant management proof. It protects the last
usable primary factor and signs you out when you remove or replace the factor
used by the current session. A retained session receives a new cookie with the
same proof age and expiry; other sessions and unfinished ceremonies are revoked.
For TOTP-only profiles, trusted application code may explicitly allow recent MFA
management to enroll a passkey. Request fields cannot change that policy.

## Verify Browser Integration

After wiring migrations, trusted RP/origin policy and encryption keys, run the
[finite browser acceptance check](../../reference/authentication/README.md#browser-acceptance)
with an owned PostgreSQL fixture, Chromium and Node. The tagged check performs
actual passkey registration, sign-in, step-up and the management page's add-passkey
control. It also completes TOTP and backup-code MFA and rejects their use at the
phishing-resistant route. Missing prerequisites fail the check.

Keep each begin token separate from your session cookie. Begin and failed finish
preserve any existing cookie; initial registration/setup confirmation clears it.
Only committed authentication/step-up or an eligible management rotation issues
a new bearer. Treat successful registration as a prompt to sign in, and successful
step-up as replacement of the previous session bearer.

Use another authenticator when adding a second resident passkey for the same
account. The acceptance fixture uses two virtual devices. Its successful result
establishes browser integration; production hardware and deployment assurances
remain your application's responsibility.

## Verify a mailbox

Mailbox ownership is separate from sign-in proof. To enable Ticked verification,
configure an active mail provider and set `TICKED_RECOVERY_ORIGIN` to the trusted
HTTPS origin of your application. Localhost HTTP requires explicit development
configuration. These routes stay disabled without active delivery.

Open `/account/mailbox`, enter your current account email and request a message.
The response does not reveal whether that account exists or can receive a token.
Keep JavaScript enabled for the protected forms. Open the link from the message
and submit the verification form. Opening the
link alone cannot verify the address. The secret stays in the URL fragment until
the form removes it and submits it in a protected POST.

Successful verification signs out existing sessions and cancels unfinished
security ceremonies. Sign in again using your existing password and any required
MFA. Verification does not change your address, activate your account or replace
lost authenticators. An expired or replaced link requires a new bounded request.
See the [mailbox reference](../../reference/authentication/README.md#mailbox-verification)
for timing, retry, storage and deployment boundaries.


## Change a password

Ticked enables `/account/password` with its active-mail and trusted recovery-origin
configuration. Sign in recently before opening the form. Without an established
MFA factor, Ticked permits recent password authentication. With established
factors, use actual phishing-resistant authentication or step-up; password alone
cannot authorize the change. An application that permits supported TOTP/backup
MFA for this operation must choose that policy in trusted server code.

Enter your complete new password and submit the form. Existing password rules
still apply. Success signs out every session and cancels unfinished authentication,
enrollment and mailbox links. Sign in again with the new password and any currently
required MFA. Current mailbox verification and established authenticators stay
intact; spent backup codes remain spent. Notification delivery failure cannot undo
a successful change. See the
[password change reference](../../reference/authentication/README.md#recent-proof-password-change)
for policy, retry and resource limits. Losing a required factor needs a separate
identity-recovery procedure; changing the password cannot replace it.


## Reset a forgotten password

With Ticked's active-mail and trusted recovery-origin configuration enabled,
open `/account/password/reset` and enter your current account email. Reset requires
that address to have been verified earlier. The acknowledgment does not reveal
whether the account is eligible. Open the mailed link and explicitly submit the
complete new password; opening the page alone cannot reset it. Request a new link
when the old one has expired or been replaced.

Success signs out every session and cancels unfinished security ceremonies.
Sign in normally with the new password and any currently required MFA. Confirmed
authenticators remain required and spent backup codes remain spent. Losing all
required authenticators needs a separate identity-recovery procedure.

An authorized Ticked operator can request this same mailed flow at
`/admin/password-reset` after recent phishing-resistant authentication. The
operator cannot choose your new password, receive your link or sign you in.
A failed notification does not undo a successful reset. If the browser loses the
completion response, check normal sign-in with the new password before requesting
another link; replaying a consumed link cannot repeat the reset. See the
[reset reference](../../reference/authentication/README.md#mailbox-password-reset)
for retry, timing and deployment boundaries.


After wiring recovery routes, run the
[recovery browser acceptance check](../../reference/authentication/README.md#recovery-browser-acceptance)
with the owned database, Chromium, Node and captured mail fixture. It exercises
the forms, required-MFA re-entry, notifications and a lost committed response.
Your application remains responsible for actual provider delivery, trusted proxy
and deployment-wide anti-automation controls, bounded cleanup/retry scheduling
and a separate procedure for all-factor loss.

## Observe Authentication Outcomes

Create one `SecurityObservations` instance with an application-owned, cooperative
`SecurityObserver`, then pass it to authentication construction. Child services
share its callback budget. Ticked records checked events through its logger.
Use the [observation settings](../../reference/configuration/README.md#security-observation)
to choose a timeout and concurrency bound, and configure a sink that returns
within that deadline. Keep destination access and retention under application
policy.

Events distinguish pending work, completed authentication, committed changes and
classified failures without credentials or raw account identities. Observer
errors, saturation and late callbacks only update delivery counters; they cannot
change a session or retry the mutation. Lookups, lists and GET do not add mutation
events. An explicit discard observer is available for applications that choose
no delivery. A timely log callback is best-effort evidence; durable mandatory
audit needs its own reviewed persistence contract.
