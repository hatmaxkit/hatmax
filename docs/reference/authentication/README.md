<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication

`auth` creates users and sessions through a caller-supplied `Queries`
implementation. The implementation note is
[auth/readme.md](../../../auth/readme.md).

`NewService(queries, cfg, checker, logger)` returns a service or a construction
error. It requires a caller-owned checker, validates credential settings and
owns one shared policy/verifier. Signup checks complete normalized candidates;
sign-in verifies PHC Argon2id records and rechecks current persistent state.
Session record IDs use `model.NewID`; bearer secrets use 32 random bytes.

## Password policy

`NewPasswordPolicy(cfg, checker)` constructs an immutable credential policy.
`PasswordChecker.Disallowed(ctx, password)` checks the complete normalized
candidate against caller-owned common, compromised or application-specific
values. A checker is required, must honor cancellation and support concurrent
calls, and must not retain candidates. The caller documents its source coverage
and provenance; supplying a checker alone does not establish breach coverage.

`PasswordPolicyConfig` has these fields:

| Field | Default | Constraint |
| --- | --- | --- |
| `MinLength` | 15; 8 with `MFARequired` | At least the profile minimum and at most `MaxLength` |
| `MaxLength` | 1024 code points | Between 64 and 1024 |
| `MaxBytes` | 4096 UTF-8 bytes | At least four times `MaxLength`; at most 4096 |
| `CheckTimeout` | 2 seconds | Positive; at most 30 seconds |
| `MFARequired` | false | Trusted server policy; only true when password access always requires MFA |

Zero numeric values select defaults. Other invalid settings or a missing
checker return `ErrPasswordPolicy`. The MFA flag records the application's
requirement; authentication must separately enforce the required factors.

`Prepare(ctx, password)` returns the accepted NFC-normalized candidate for
hashing. It bounds raw bytes before normalization, rejects invalid UTF-8, then
checks normalized code-point and byte lengths. It preserves case and whitespace,
and neither truncates nor imposes character-composition rules. Use NFC processing
when verifying the same credential format. Apply candidate policy when creating
or changing credentials; ordinary sign-in verifies the stored credential.

Length and normalization follow
[NIST SP 800-63B-4 password guidance](https://pages.nist.gov/800-63-4/sp800-63b.html#passwordver).
The policy is available independently and is enforced by `Service.Signup`.
The current service permits password-only access and always requires at least 15
code points. Its constructor does not expose an always-MFA minimum override.

| Error | Condition |
| --- | --- |
| `ErrPasswordPolicy` | Invalid configuration or an uninitialized policy |
| `ErrPasswordEncoding` | Invalid UTF-8 |
| `ErrPasswordTooShort` | Too few normalized code points |
| `ErrPasswordTooLong` | Character or byte limit exceeded |
| `ErrPasswordDisallowed` | Checker rejects the complete candidate |
| `ErrPasswordCheckFailed` | Checker fails; inspect the wrapped cause with `errors.Is` or `errors.As` |
| `context.Canceled` / `context.DeadlineExceeded` | Request canceled or checker deadline reached |

All preparation failures return an empty candidate. Checker error text is
redacted; the wrapped cause remains diagnostic material and must not be exposed
to users or logged without review. The checker receives the earlier of the
configured deadline and the caller's deadline. Cancellation is cooperative:
no detached goroutine is created to abandon an uncooperative checker. A checker
that returns success after cancellation/deadline cannot approve the password.

## Versioned password verifier

`model.NewPasswordVerifier(cfg)` constructs a shared, concurrency-safe verifier.
`Hash(ctx, password)` returns an encoded record; `Verify(ctx, record, password)`
returns an error or nil for a match. This primitive has no persistence operations
and is available independently of the current auth service.

The supported PHC representation is:

```text
$argon2id$v=19$m=<KiB>,t=<iterations>,p=<lanes>$<salt>$<output>
```

Version 19 is Argon2's algorithm version. This format's input contract is NFC;
it does not add a custom prehash or an application-specific version field.
Parameters appear exactly in `m,t,p` order with unsigned canonical decimal
values. Salt and output use canonical standard Base64 without padding: 16 random
salt bytes per creation and 32 output bytes. Records are at most 128 bytes.
Unknown algorithms/versions, bcrypt, extra fields, alternate encodings and
parameters outside the configured bounds return `model.ErrPasswordRecord`
before admission or KDF execution. No legacy reader or rehash path is provided.

`PasswordVerifierConfig` contains:

| Field | Zero-value default | Constraint |
| --- | --- | --- |
| `MemoryKiB` | 65536 (64 MiB) | 19456 through 262144 KiB |
| `Iterations` | 3 | 1 through 10; at least 2 below 64 MiB |
| `Parallelism` | 4 | 1 through 8 Argon2 lanes |
| `MaxMemoryKiB` | Creation memory | At least creation memory, at most 262144 KiB |
| `MaxIterations` | Creation iterations | At least creation iterations, at most 10 |
| `MaxParallelism` | Creation lanes | At least creation lanes, at most 8 |
| `MaxConcurrent` | 2 | 1 through 16; maximum memory times concurrency at most 512 MiB |

Creation parameters and verification ceilings are immutable after construction.
The defaults use the second recommended Argon2id profile in
[RFC 9106 section 4](https://www.rfc-editor.org/rfc/rfc9106.html#section-4).
The lower supported floor follows
[OWASP's Argon2id guidance](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id).
Benchmark on the deployment hardware before selecting a different resource
profile. These parameters do not establish a NIST/FIPS or application compliance
level.

Hash and verification both reject invalid UTF-8 and raw input above 4096 bytes
before normalization, then NFC-normalize and enforce 1024 code points and 4096
bytes. They preserve whitespace and case and do not truncate. Creation must
first apply the candidate policy; the model does not enforce a minimum or run a
blocklist. Verification uses the stored cost rather than current creation cost.

Share one instance across all credential operations in the resource budget.
Hash and verification share its admission slots; full admission returns
`model.ErrPasswordVerifierBusy` immediately without a waiting queue. The default
bounds active KDF memory to 128 MiB and active lanes to eight. Memory figures
cover Argon2 work buffers, not total process memory or buffers retained by Go's
garbage collector. Separate instances and processes have separate budgets.
Applications own request limits, attempt controls and timeout configuration.

Contexts are checked before and after derivation. The existing Argon2 primitive
cannot be interrupted: an admitted call runs synchronously and retains its slot
until the KDF returns. Cancellation or deadline expiry discards the result; no
background goroutine continues abandoned work. Owned temporary input/output
byte slices are cleared; Go strings and the primitive's internal buffers are
outside that clearing guarantee.

| Outcome | Meaning |
| --- | --- |
| nil | Supported record matches |
| `model.ErrPasswordMismatch` | Supported, valid record does not match |
| `model.ErrPasswordRecord` | Malformed, unsupported or out-of-budget record |
| `model.ErrPasswordInput` | Invalid encoding or input size |
| `model.ErrPasswordVerifierConfig` | Invalid settings or uninitialized verifier |
| `model.ErrPasswordVerifierBusy` | All admission slots occupied |
| `context.Canceled` / `context.DeadlineExceeded` | Context failure; result discarded |

Hash failures return an empty record. Errors contain no password or encoded
record material. The auth service owns a shared verifier, and the former model
bcrypt helpers and `auth.bcrypt_cost` setting have been removed.

## User and session

`User` fields are `ID`, `Email`, `PasswordHash`, `AuthVersion`, `Roles`, `Active`,
`TOTPSecret`, `TOTPEnabled`, `TOTPVerifiedAt`, `CreatedAt`, and `UpdatedAt`.

`HasRole` reports an exact role match. `HasAnyRole` reports whether any
supplied role matches. `NeedsTOTPSetup` is true when `TOTPSecret` is empty.
`InTOTPGracePeriod(days)` is false when `days` is not positive. Otherwise it
is true while the current time is before `CreatedAt` plus that many days.

`Session` contains `ID`, `UserID`, `AuthVersion`, `PolicyRevision`, `Generation`, `Proof`, `AuthenticatedAt`,
`CreatedAt`, `LastActivityAt`, `ExpiresAt`, and `InactivityTTL`. It contains no
bearer or lookup digest. `SessionRecord` adds a fixed 32-byte `SessionDigest` for
storage. `IssuedSession` adds the transient raw `Token` for immediate issuance.
`ValidatedSession` returns an owned current `User` and safe `Session` snapshot.

## Session lifecycle

Secrets contain 32 random bytes from the cryptographic source. Their sole
accepted encoding is exactly 44 characters of canonical padded URL Base64.
`ParseSessionToken` checks size before decoding, rejects alternate alphabets,
padding bits, line breaks and unpadded input, then derives SHA-256 over
`hatmax/session/v1` followed by a zero byte and the decoded secret. Source failures
return no secret/digest and cannot persist a session. Raw tokens must not be
stored or logged. A session ID is never a bearer.

`Session.Check(now)` checks shape, positive version/generation and ordered times.
Absolute expiry is measured from completed authentication; inactivity from stored
relevant activity. Equality at either deadline is expired. Activity never changes
absolute expiry, creation or authentication time. Future activity/proof times and
invalid duration snapshots fail closed.

The required validation storage operation acquires the subject lock followed by
the session lock, checks active/current `AuthVersion`, then evaluates current
trusted time. A timestamp captured before waiting on a lock does not satisfy the
contract. Missing sessions return `ErrSessionNotFound`, stale account state
`ErrCredentialChanged`, corrupt records `ErrSessionRecord`, and expired state
`ErrSessionExpired`. Revoked/expired sessions cannot be renewed.

`ValidateSession(ctx, token, requirement, activity)` requires trusted `NoActivity` or
`RelevantActivity`. Polling/background routes use `NoActivity`. Relevant activity
updates only when the configured persistence interval has elapsed, in the same
transaction as validation. Coalescing can conservatively expire up to one
interval early; it cannot extend expiry. No activity decision comes from client
flags. Concurrent touches serialize and return owned snapshots only after commit.
A failed write/commit returns no authorized result and leaves activity unchanged.

The Ticked schema contains only `token_digest`, with an exact 32-byte constraint,
positive auth version/generation and bounded time/duration constraints. Its
adapter uses `clock_timestamp()` after row locks. Custom adapters must implement
the same required atomic contracts; a getter plus independent touch is insufficient.
The initial schema replaces the old shape; no legacy-token reader or historical
schema upgrade is supplied.

| Configuration | Default | Bound |
| --- | --- | --- |
| `auth.session_ttl` | 24h | 1 minute through 30 days |
| `auth.session_inactivity_ttl` | 30m | 1 minute through absolute lifetime |
| `auth.session_activity_interval` | 1m | 1 second through 5 minutes; at most one quarter of inactivity |
| `auth.session_timeout` | 5s | Positive; at most 30 seconds; earlier caller deadline wins |
| `auth.session_recent_proof_age` | 5m | 1 second through absolute lifetime; whole microseconds |
| `auth.session_max_per_subject` | 10 | 1 through 100 retained rows; zero selects default |
| `auth.session_page_size` | 50 | 1 through 100; zero selects default |
| `auth.session_cleanup_batch` | 1000 | 1 through 1000; zero selects default |

Absent duration strings select defaults; malformed, zero/negative and
inconsistent configured durations fail construction. Stored durations use whole
microseconds to match PostgreSQL precision. Configuration is snapshotted, so
later mutation cannot weaken a live service. Storage must honor context and bound
work; late validation/issuance success is discarded. Cleanup deletes one bounded
batch of absolute or inactivity-expired rows and returns its count. The example
uses `SKIP LOCKED` to avoid waiting on busy rows.

This service path verifies passwords only. Enrollment fields and the TOTP setup
middleware do not establish verified MFA or phishing-resistant proof. Stronger
proof policies are enforced explicitly as described below. Reauthentication and
session control have their own contracts.

## Required proof and outcomes

`AccessRequirement` is trusted server policy with `Proof`, `Revision` and
`MaxAge`. Required profiles are `RequirePassword`, `RequireMFA` and
`RequirePhishingResistantMFA`. Revision must be 1 through 128 printable ASCII
bytes and must contain no secret/personal data. It identifies the current
application policy; a mismatch with stored `PolicyRevision` rejects the session.
There is no default or client-selectable policy revision.

`MaxAge` zero omits a recent-proof requirement. Otherwise it must be 1 second
through absolute lifetime in whole microseconds. A proof age equal to or greater
than that bound is expired. Required policy is copied into the operation;
application code owns current policy selection. Middleware captures its supplied
value: replace that wiring when policy changes, or call service validation with
current application policy per request. A validated context is not authority for
later domain writes without their required current-state checks.

`VerifiedProof` has closed password/WebAuthn methods and `VerifiedAt`.
Password proof follows the core password verifier. Actual UV WebAuthn proof
also binds a confirmed factor ID/security revision and satisfies stronger policy.
Its time is nonzero, no earlier than creation and no later than completed
authentication/current time. Unknown methods or malformed/future facts fail
closed. Roles, enrollment and `TOTPVerifiedAt` do not supply method facts.
Current revision, method properties and age are evaluated at the adapter's locked
time before activity writes. No public proof-assertion or completion API accepts
these metadata DTOs to produce MFA.

`Signin(ctx, email, password, requirement)` returns `AuthenticationResult`:

| Outcome | Current behavior |
| --- | --- |
| `AuthenticationCompleted` | Verified password under password policy; `Issued` contains safe metadata and the transient secret |
| `AuthenticationPendingEnrollment` | Unmet MFA without setup; method unavailable, no issued value or continuation |
| `AuthenticationPendingProof` | Unmet MFA with setup; method unavailable, no issued value or continuation |
| `AuthenticationDenied` | Phishing-resistant MFA verifier unavailable; no issued value or continuation |

Completed results use `AuthenticationSatisfied`; unmet results use
`AuthenticationMethodUnavailable`. The pending distinction describes the next
required step, not an enrollment/verification endpoint. This password-only path supplies no pending secret or executable continuation.
Use the separate WebAuthn begin/finish APIs for actual factor verification and
atomic challenge/counter consumption.
`CompletedSession()` returns a session only for a completed, satisfied result
with a nonnil issued value. Transports must use that check, rather than infer
access from a nonnil pointer or setup flag.

Invalid credentials retain their classified errors with no result. Invalid
requirements return `ErrAccessRequirement` before credential/storage work;
operating/verifier failures return an error with no successful result. During
validation, `ErrSessionPolicy` means revision mismatch, `ErrSessionProof` means
unmet properties, and `ErrSessionProofExpired` means freshness expired. These are
separate from account-state, lifecycle and malformed-record failures.

The supported password path can be tested end to end; rejection/property tests
for other profiles do not demonstrate actual MFA or phishing resistance.

## Queries

`Queries` is the persistence boundary:

| Method | Arguments |
| --- | --- |
| `CreateUser` | id, email, password hash, created at, updated at |
| `GetUserByEmail` | email |
| `GetUserByID` | id |
| `CreateSession` | expected `CredentialState`, complete `SessionRecord`, trusted requirement, admission cap |
| `RotateSession` | expected account state, current digest/generation, replacement record, requirement |
| `ListSessions` | actor digest, recent requirement, page size, opaque cursor |
| `RevokeSessions` | actor digest, recent requirement, finite selection |
| `ReplacePassword` | expected `CredentialState`, encoded password, change time |
| `ValidateSession` | digest, trusted requirement/activity, persistence interval |
| `DeleteSession` | digest |
| `DeleteExpiredSessions` | bounded batch limit; returns count |

Reads return caller-owned snapshots. A missing read row is `sql.ErrNoRows`.
New users have positive `AuthVersion`; every credential or activation change
increments it, including disable/re-enable cycles. The example also increments
it for role changes. `CredentialState` contains `UserID` and expected `Version`.

`CreateUser` enforces persistent email uniqueness and reports `ErrEmailTaken`
for a concurrent duplicate. Signup does not depend on a prior lookup.
`CreateSession` atomically locks/checks active state and expected version before
reclaiming at most 100 expired subject rows, counting all retained rows and
inserting a session bound to that subject. Capacity rejects with
`ErrSessionCapacity`; it never silently evicts another session. `ReplacePassword` locks/checks the
same state, replaces the complete encoded record, increments its version and
revokes existing sessions in one transaction. Stale, inactive or missing state
returns `ErrCredentialChanged`; operating failures roll back the entire write.
The replacement caller owns policy preparation and required authorization.

Implement these required methods directly. An unlocked read followed by insertion,
a compare/write without a transaction, or an optional upgrade interface does not
satisfy the contract. The example adapter uses Postgres user-row locks with all
security writes taking the same lock order. Recovery and password-change API
workflows are separate from this storage primitive.

## Service

`NewService` snapshots validated credential settings and constructs its policy
and shared verifier. Missing dependencies/checker or invalid settings fail at
construction. Later configuration mutation cannot weaken credential settings.

| Error | When |
| --- | --- |
| `ErrInvalidEmail` | `Signup` receives an empty email. |
| `ErrPasswordTooShort` | The candidate has fewer normalized code points than the configured minimum. |
| `ErrEmailTaken` | Persistent creation finds a duplicate email. |
| `ErrUserNotFound` | Sign-in or `GetUserByID` finds no row. |
| `ErrInvalidPassword` | A supported stored credential does not match. |
| `ErrCredentialChanged` | Persistent state is stale, inactive or missing during creation/validation. |
| `ErrSessionNotFound` | Validation or sign-out finds no session. |
| `ErrSessionExpired` | Absolute or inactivity deadline has been reached. |
| `ErrSessionCapacity` | Retained subject rows exhaust admission capacity. |
| `ErrSessionGeneration` | Rotation generation is stale or exhausted. |
| `ErrSessionSelection` | Scope/selected ID is invalid. |
| `ErrSessionCursor` | Cursor is oversized, malformed or noncanonical. |

An inactive user on sign-in returns the error text
`user is not active`. That value is not one of the sentinels above.

`Signup` returns the created user after candidate checking, salted Argon2id
hashing and a uniqueness-enforcing write. Policy and verifier errors remain
classified; no partial user or credential is created on preparation failure.

`Signin` reads an owned user snapshot and verifies its stored record without
rerunning new-password policy. A mismatch becomes `ErrInvalidPassword`;
invalid/unsupported records, invalid input, busy admission and context failures
retain their verifier classification. Session insertion checks the captured
auth version and active state atomically, so verification against stale
credentials cannot issue a session.

Both operations use the earlier caller deadline or `auth.password_timeout`
(default five seconds), including the persistent write. Checking also has its
own two-second default bound. Argon2 work remains synchronous and cannot be
interrupted; a late result is discarded and the slot remains held until completion.

`Signin` returns a completed result with `IssuedSession` only after the
current-state/policy/proof insert commits. Unmet results create no session.
`Signout` parses and deletes only the presented digest. `GetUserByID` can return
an inactive user. `CleanupExpiredSessions` performs one bounded batch. Operating
errors retain their wrapped cause; do not expose diagnostic storage errors to
users.

## Reauthentication and Control

`Reauthenticate(ctx, token, password, requirement)` validates the old live
session with current policy/revision but without requiring already-recent proof,
then performs actual password verification. It captures proof time before entropy
work and submits a replacement to `RotateSession`. Stronger current requirements
reject password-only state. No asserted proof or continuation API is accepted.

Rotation locks the current subject then session and rechecks account version,
old digest/generation, current policy and expiry with one trusted post-lock clock.
It checks fresh replacement proof against that clock. ID and creation time stay
fixed; generation advances by one; digest, proof/authentication/activity times,
absolute expiry and inactivity snapshot change together. Failed proof, entropy,
stale state or transaction work returns no new secret and leaves the old valid
session unchanged. After commit the old bearer cannot validate, sign out the
replacement or rotate again. Expired sessions require a new sign-in.

`ListSessions(ctx, token, requirement, cursor)` and `RevokeSessions(ctx, token,
requirement, selection)` derive subject ownership from a revalidated bearer.
The recent-proof age is the tighter of the operation's nonzero `MaxAge` and
`session_recent_proof_age`; zero operation age uses the configured value.
Database checks run after locks and before the result or deletion. Roles, a target
subject ID or a context snapshot cannot grant management authority. Applications
own any separate cross-subject administrative authorization; these self-service
APIs accept no target subject.

`SessionSelection` supports `SessionCurrent`, `SessionSelected` (ID required),
`SessionOthers` and `SessionAll`. IDs are 1 through 128 printable ASCII bytes and
are permitted only for selected scope. Selected foreign/missing IDs report
`ErrSessionNotFound`, with no cross-subject deletion. Multi-record revocation
locks subject sessions in stable ID order before evaluating current actor proof.
Deleting all/current includes the actor and invalidates that bearer.

`SessionPage` contains only `[]Session`, `CurrentID` and `NextCursor`, never a
digest or bearer. Stable ascending record-ID keyset queries stay scoped to the
actor subject and fetch at most configured page size plus one. Cursors are
canonical raw URL Base64 of a bounded ID, at most 128 bytes; they confer no
permission. Pages reflect each transaction's current state, not a historical
snapshot across concurrent inserts/deletes. Retained expired/stale rows may be
listed for termination, while they can never authorize operations.

Admission serializes through the subject lock and counts all retained session
rows, including stale/expired state. It reclaims a batch of at most 100 expired
subject rows first. Revocation removes rows; no history/tombstone collection is
created. Explicit cleanup deletes at most its configured batch and honors context.
`Signout` retains current-bearer withdrawal without requiring recent proof; it
cannot select another session or subject. Failed storage withdrawal does not
report success or clear the browser cookie. Management operations require recent
proof independently.

Ticked exposes `/sessions`, `/reauthenticate` and `/sessions/revoke`. Its form
repeats password verification and updates the cookie only after committed
rotation; failed work preserves the existing cookie. The screen shows safe
metadata and self-service revocation, with reauthentication required for stale
proof. These handlers choose policy on the server and do not count pre-validation
as relevant activity before a failed reauthentication.

## Context

A nil context makes `GetUserID`, `GetUser`, and `GetSession` return the zero
value and false. `WithUserID`, `WithUser`, and `WithSession` store those
values. `RequireAuth` and `OptionalAuth` store the user, user ID and safe session
metadata after validation.

## Middleware

`SessionCookieName` is `session`.

`RequireAuth(svc, requirement, activity)` redirects to `/signin` with status `303` when the cookie is
missing or `ValidateSession` returns an error. Success stores the user and
calls the next handler. It does not clear the cookie.

`OptionalAuth(svc, requirement, activity)` calls the next handler without a user when the cookie is
missing or validation fails.

`SetSessionCookie` sets `session` on path `/`, with the supplied `MaxAge`,
`HttpOnly`, `Secure`, and `SameSite=Lax`. `ClearSessionCookie` sets the same
cookie with an empty value and `MaxAge` of `-1`.

`RequireTOTP` calls the next handler when `Enabled` is nil or returns false,
when no user is in the context, when `TOTPEnabled` is true, or when
`InTOTPGracePeriod` accepts `GraceDays`. A nil `GraceDays` uses zero days.
Otherwise it redirects to `SetupURL`, or `/totp-setup` when that field is
empty, with status `303`.

## Restricted WebAuthn Enrollment

`NewAuthenticatorService(credentials *Service, queries AuthenticatorQueries,
cfg config.AuthenticatorConfig)` validates an owned configuration snapshot. Its
password verifier shares the existing credential service's finite KDF admission.
RP ID, exact origin allowlist and access requirement come from trusted server
code. HTTP headers cannot select the RP or weaken required policy. HTTPS is
required, except explicit `localhost_development` for `http://localhost`.

| API | Behavior |
| --- | --- |
| `BeginWebAuthnEnrollment(ctx, email, password, requirement)` | Verify the actual password; reject inactive/current-state conflicts and established factors; return restricted bearer and exact browser creation options |
| `FinishWebAuthnEnrollment(ctx, token, response, requirement)` | Reserve a durable attempt/lease, parse bounded browser data, verify registration and commit activation/revocation; return safe metadata |
| `CleanupEnrollments(ctx)` | Delete a configured finite expired-row batch with a bounded caller operation; no owned loop |

The registration profile requests resident keys, ES256 only, required user
verification and no attestation. The closed parser accepts `none` attestation;
both presence and UV must verify. Cross-origin/embedded ceremonies, foreign RP,
origin, challenge, subject handle, ID or algorithm fail. The library COSE parser also validates
the ES256 curve and point before persisting attestation-none credentials. Backup eligibility/state
are recorded from protocol output; state without eligibility fails. No hardware
or non-exportability certification follows from these flags. The RP handle is a
stable random 32-byte value independent of email.

`EnrollmentChallenge.Token` is a transient `enroll1.` bearer with independent
32-byte entropy and a distinct digest domain/collection. Keep it out of URLs,
logs, ordinary cookies and user/session snapshots. Stored setup binds account
version, exact policy/freshness, actual password time, RP configuration fingerprint,
opaque handle and the exact v0.18.2 `SessionData`. Reissue does not reset the
per-subject factor budget. Stored public metadata is not a proof receipt.

`AuthenticatorQueries` is a required typed storage contract. Subject-first
transactions check active version, policy and trusted time after waits. Begin
reclaims at most ten expired subject rows before checking retained capacity;
it never evicts a live pending row. Reserve charges pending and subject attempts
and commits one finite revision-bound lease before protocol work. Admitted invalid
JSON/proof and operating failure spend an attempt. Busy/non-admitted work does
not. Release only clears its own lease revision, with no refund. If caller
cancellation prevents release, its finite lease expires; no detached cleanup runs.

Confirmation rechecks live lease, exact captured state and recent password,
inserts the unique RP/credential record, increments `AuthVersion`, and removes
all sessions and other pending enrollment in one commit. A failed final commit
returns no success and rolls back activation/revocation; the prior admission
remains spent. Retrying must perform verification again. Exact expiry or proof
freshness equality rejects. Password/account mutation and replay cannot activate
a stale key. Explicit cleanup uses at most 1000 expired rows and never obtains a
subject lock after pending locks.

| Bound | Default / permitted range |
| --- | --- |
| Pending TTL | 5m / 1m–10m |
| Recent password | 5m / 1s–10m; trusted requirement may tighten |
| Retained pending / authenticators | 5 / 1–10; 10 / 1–20 |
| Pending / subject attempts | 5 / 1–10; 10 / 1–100 |
| Subject window / cooldown | 15m each / 1m–1h each |
| Timeout / lease | 5s each / 1ms–30s, lease at least timeout; earlier caller deadline wins |
| Protocol admission | 2 / 1–2 active verifications, no waiting queue |
| Origins / cleanup | 8 maximum / 1000 default and maximum |
| Body / client data / attestation | 64 KiB before parsing / 8 KiB / 32 KiB |
| COSE key / credential ID / stored ceremony | 4 KiB / 1024 bytes / 16 KiB |
| JSON / CBOR nesting | 16 levels; CBOR maps/arrays at most 64 entries, no indefinite lengths/tags |

Ticked wires the production service and PostgreSQL adapter explicitly. Migration
`004-authenticators.sql` adds the enrollment tables to existing databases and is
also applied after the user/session schema in focused integration fixtures:


- `POST /authenticators/enrollment/begin` accepts only JSON `email` and `password`.
- `POST /authenticators/enrollment/finish` accepts the browser credential JSON
  and the restricted bearer in `X-Enrollment-Token`.
- Both require matching Origin/Referer, `Content-Type: application/json` and finite
  bodies. Responses are not cached; failures expose generic errors, and exhausted
  capacity/admission/budgets return HTTP 429.
- Begin never issues a cookie. Successful finish clears a previous session cookie
  because confirmation invalidated those sessions; failure does not change it.

In a browser capable of WebAuthn JSON conversion, the response returned by begin
can be used as follows on the configured RP origin:

```javascript
const publicKey = PublicKeyCredential.parseCreationOptionsFromJSON(begin.options.publicKey);
const credential = await navigator.credentials.create({ publicKey });
const response = await fetch("/authenticators/enrollment/finish", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-Enrollment-Token": begin.token },
  body: JSON.stringify(credential.toJSON()),
});
```

This delivery exposes initial-registration APIs. Registration alone supplies no
strong sign-in. The separate assertion APIs below complete authentication;
additional-factor management, browser acceptance remain pending. Existing password-only
routes keep their explicit password policy; enrolling a key does not make those
routes enforce MFA automatically.

## WebAuthn Authentication and Step-up

Construct `auth.NewWebAuthnService(credentials, queries, cfg.Authenticator)` with
mandatory `auth.WebAuthnQueries`. It includes enrollment storage and adds typed
subject snapshots, assertion admission/reservation, and atomic completion.
Enrollment-only applications can keep `NewAuthenticatorService`.

`BeginWebAuthnAuthentication(ctx, email, requirement)` identifies the subject and
returns an `AssertionChallenge` for its confirmed credentials. It requests UV
and binds the stable RP handle, exact RP configuration, allowed credential IDs,
security revisions, current account version and trusted policy. No password fact
or application access is produced. `BeginWebAuthnStepUp(ctx, sessionToken,
requirement)` instead binds a live current session ID, digest and generation.
The entry requirement is password-or-better with the same trusted policy revision;
completion must meet the supplied stronger requirement. Older proof can enter
step-up while its session remains live.

Pass the serialized `navigator.credentials.get` result to
`FinishWebAuthn(ctx, pendingToken, browserJSON, requirement)`. The pinned library
validates the actual ES256 signature, challenge, RP, exact owned origin,
credential/subject binding, UP and UV. This profile rejects cross/top origins,
attested data and extensions in assertion authenticator data. Response bounds
are 64 KiB total, 8 KiB client data, 1 KiB credential ID, exactly 37 bytes of
assertion authenticator data, at most 80 signature bytes and an optional exact
32-byte user handle. JSON nesting is at most 16.

`assert1.` and `stepup1.` secrets have separate digest domains and purposes;
ordinary session and enrollment lookup reject them. All purposes share the
finite pending cap and per-subject factor attempt budget. Durable reservation
spends an attempt before parsing/signature work and grants one finite lease.
Failure never refunds admission or creates a session/cookie.

Completion locks subject, pending/actor and factor in order, rechecks current
policy/version, factor security/replay revisions, counter, immutable backup
eligibility, current backup state, live actor generation, expiry and lease.
Zero/zero counters are accepted; all other successful assertions strictly
increase the counter. Counter updates advance replay revision without changing
the factor's security revision. Counter/backup state, pending consumption and
one admitted session insertion or generation rotation commit together. Failed
commit rolls back all completion writes while admission remains spent.

`VerifiedProof{Method: WebAuthnProof}` stores the actual verification time,
`FactorID` and `FactorRevision`. UV WebAuthn satisfies password-or-better, MFA
and phishing-resistant MFA within this supported profile. Session validation
and actor controls recheck the current factor binding. Activity does not refresh
proof time. Password reauthentication produces password proof only; callers
requiring strong proof must use actual WebAuthn step-up.

Apply `005-webauthn-completion.sql` after migration 004. Ticked wires these
same-origin JSON endpoints under server-owned phishing-resistant policy:

| Endpoint | Input / result |
| --- | --- |
| `POST /authenticators/authentication/begin` | JSON `email`; restricted challenge, no session cookie |
| `POST /authenticators/step-up/begin` | Empty JSON `{}` and current session cookie; restricted bound challenge |
| `POST /authenticators/authentication/finish` | Browser assertion JSON and `X-Assertion-Token: assert1.…`; sets cookie after commit |
| `POST /authenticators/step-up/finish` | Browser assertion JSON and `X-Assertion-Token: stepup1.…`; replaces cookie after commit |
| `GET /authenticators/proof` | Current session cookie; returns safe metadata only when current recent strong proof satisfies policy |

Responses are `no-store`. Errors preserve an existing cookie and issue no bearer;
begin and finish purposes cannot cross endpoints. Ticked's ordinary routes keep
their explicit password-or-better minimum. Enrolling a factor alone does not
change application policy. Established factor management and
browser acceptance remain later delivery work. Synced/backup flags do not prove
hardware storage, non-exportability or an assurance certification.

## TOTP and Backup Proof

`NewFallbackService(credentials, fallbackQueries, fallbackConfig, seedKeys)`
requires mandatory typed storage and explicit seed keys. No optional store upgrade
or caller proof callback exists. `FallbackConfig` requires printable issuer
(1..128 ASCII bytes), uses shared `AuthenticatorConfig` admission bounds without
requiring an RP, defaults to skew one and eight backups. `StrictStep` selects
zero skew; `BackupCodes` accepts 1..10. `SeedKeys` supplies one active identity and
1..8 identities of exactly 32 bytes, copied at construction. Identities are
printable ASCII, 1..64 bytes. Retain decryption keys while their records remain.

Seeds are AES-256-GCM encrypted at rest in a versioned nonce/ciphertext envelope.
Associated data binds purpose/version, subject, factor and key identity. Changing
any binding or missing the referenced key fails closed. Seed material never
enters user/session snapshots. `BeginTOTPSetup(ctx, email, password, requirement)`
returns an immediate provisioning URL and restricted token only after actual
password verification and initial-factor admission. Confirmation with
`ConfirmTOTPSetup` checks an actual code, inserts the factor with the matched step
already consumed, advances `AuthVersion`, and deletes all sessions and pending
records atomically. It issues no access. Existing-factor changes use separate
management flows.

`BeginFallbackAuthentication(ctx, email, password, method, requirement)` captures
actual password time, current subject version and one owned TOTP factor or backup
set. `BeginFallbackStepUp(ctx, actorToken, password, method, requirement)` also
captures actor ID/digest/generation. `FallbackTOTP` and `FallbackBackup` are trusted
server selections. A phishing-resistant requirement fails before password work.
`FinishFallback(ctx, pendingToken, code, requirement)` durably reserves attempts
and lease before cryptography. The final transaction rechecks current policy,
subject, factor/replay state, pending/actor snapshot and post-lock time, then
consumes step/code and pending with session insertion or rotation. Equality at
expiry, proof age or lease is expired. Operating failure returns no bearer and
rolls back completion; admitted attempts remain spent.

`totpset1.`, `fallback1.` and `fallstep1.` have separate purposes and digest domains.
`CleanupPending` removes at most the configured expired-only cleanup batch.
All use the same bounded pending collection and durable subject factor budget as
WebAuthn. Reissuing or canceling a pending operation cannot reset that budget.
TOTP uses SHA-1, six digits, 30-second steps and at most one adjacent step. Accepted
steps strictly increase; routine replay updates preserve security revision.
Backup format is exactly 67 ASCII bytes: `backup1.<UUID>.<22-character Base64url>`.
Whitespace, case changes, padding, alternate encodings and malformed IDs reject
before KDF. Subject/ID lookup selects one verifier; it does not scan a set.
Each code has an independent 16-byte salt and full bounded PHC Argon2id verifier,
bound to purpose/subject/ID/secret, using the credential service's shared engine.

Closed `PasswordTOTPProof` and `PasswordBackupProof` store oldest constituent time
in `VerifiedAt` (actual password) and actual factor time in `FactorAt`, plus
factor/set ID and security revision. They satisfy password-or-better and MFA,
never phishing-resistant MFA. Activity and password-only reauthentication cannot
renew factor authority. Backup proof binds the active set, so consuming a code
does not invalidate its own completed session; replacing the set does.

`IssueBackupCodes(ctx, actorToken, managementRequirement)` requires actual recent
MFA or phishing-resistant MFA, with nonzero age at most configured recent age.
Use phishing-resistant management by default; MFA is an explicit application
policy for allowed TOTP-only profiles. Password-only and backup-code actors cannot
replace a set. Reservation durably charges the shared budget before hashing.
Replacing the finite set, advancing account version, revoking all sessions/pending,
and retaining one generation-rotated actor commit together. Retained proof times
and session expiry remain unchanged. Codes return once after commit; no replay
cache or stored plaintext exists.

Apply `006-fallback-proof.sql` after migration 005. Ticked enables routes only
with both `TICKED_TOTP_KEY_ID` and canonical standard-Base64 `TICKED_TOTP_KEY`
encoding 32 externally supplied key bytes. Both absent disables these routes;
partial/invalid configuration fails startup. Never commit keys. Ticked explicitly
allows MFA fallback access under its ordinary policy revision; backup management
keeps recent phishing-resistant policy.

| Endpoint | Input / result |
| --- | --- |
| `POST /authenticators/totp/setup/begin` | JSON email/password; token and immediate provisioning URL, no cookie |
| `POST /authenticators/totp/setup/finish` | JSON code, `X-Fallback-Token: totpset1.…`; confirmed metadata, clears cookie |
| `POST /authenticators/{totp,backup}/authentication/begin` | JSON email/password; restricted token only |
| `POST /authenticators/{totp,backup}/step-up/begin` | JSON password and actor cookie; restricted bound token |
| `POST /authenticators/fallback/authentication/finish` | JSON code, `X-Fallback-Token: fallback1.…`; sets committed session cookie |
| `POST /authenticators/fallback/step-up/finish` | JSON code, `X-Fallback-Token: fallstep1.…`; replaces committed cookie |
| `POST /authenticators/backup/issue` | Empty JSON and recent authorized cookie; codes once and rotated cookie |
| `GET /authenticators/mfa` | Current cookie; safe metadata only when recent MFA satisfies policy |

JSON bodies are at most 64 KiB, password at most 4096 bytes, email at most 254,
and code at most 128; unknown fields, non-object bodies and trailing JSON reject. Every response is
`no-store`; pending purposes cannot cross finish routes. Failed completion preserves
cookies. Ordinary session parsing rejects every pending/code wire format.
