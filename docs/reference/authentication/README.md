<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication

`auth` creates users and sessions through a caller-supplied `Queries`
implementation. The implementation note is
[auth/readme.md](../../../auth/readme.md).

`NewService(queries, cfg, checker, admission, observations, logger)` returns a service or a construction
error. It requires a caller-owned checker, shared durable admission and initialized security observations, validates
credential settings and
owns one shared policy/verifier. Signup checks complete normalized candidates;
sign-in verifies PHC Argon2id records and rechecks current persistent state.
Session record IDs use `model.NewID`; bearer secrets use 32 random bytes.

Core checks that signup email is nonempty and that admission identity is bounded
valid text; it does not parse or canonicalize a mailbox. The application owns
lookup normalization, aliases and public form validation. Use the same canonical
identity for durable admission and account lookup.

## Contract Map

| Concern | Public boundary | Contract |
| --- | --- | --- |
| Password and ordinary access | `Service` | [Policy](#password-policy), [service](#service), [shared admission](#standalone-credential-admission) |
| Session lifetime and withdrawal | `Service`, `Queries` | [Lifecycle](#session-lifecycle), [reauthentication and control](#reauthentication-and-control) |
| Initial passkey setup | `AuthenticatorService`, `AuthenticatorQueries` | [Restricted enrollment](#restricted-webauthn-enrollment) |
| Passkey sign-in and actor step-up | `WebAuthnService`, `WebAuthnQueries` | [Assertion completion](#webauthn-authentication-and-step-up) |
| Password plus app or backup proof | `FallbackService`, `FallbackQueries`, `SeedKeys` | [Fallback proof](#totp-and-backup-proof) |
| Established factor changes | `FactorService`, `FactorQueries`, `FactorPolicy` | [Addition, replacement and removal](#established-authenticator-changes) |
| Mailbox and password recovery | `RecoveryService`, `RecoveryQueries` | [Mailbox](#mailbox-verification), [change](#recent-proof-password-change), [reset](#mailbox-password-reset) |
| Redacted outcomes | `SecurityObservations`, `SecurityObserver` | [Observation contract](#security-observations) |

For a learning path, read [Identity and Sessions](../../tutorials/user-guide/identity-and-sessions.md).
For a specific task, use [Manage Authenticators](../../how-to/manage-authenticators/README.md)
or [Verify a Mailbox and Recover a Password](../../how-to/recover-account/README.md).
For rationale, read [Authentication Proof and Recovery](../../explanation/authentication-proof-and-recovery/README.md).

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

The ordinary `Service.Signin` path verifies passwords only. Enrollment fields and the TOTP setup
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
| `ErrUserInactive` | Password entry finds an inactive account before proof. |
| `ErrInvalidPassword` | A supported stored credential does not match. |
| `ErrCredentialChanged` | Persistent state is stale, inactive or missing during creation/validation. |
| `ErrSessionNotFound` | Validation or sign-out finds no session. |
| `ErrSessionExpired` | Absolute or inactivity deadline has been reached. |
| `ErrSessionCapacity` | Retained subject rows exhaust admission capacity. |
| `ErrSessionGeneration` | Rotation generation is stale or exhausted. |
| `ErrSessionSelection` | Scope/selected ID is invalid. |
| `ErrSessionCursor` | Cursor is oversized, malformed or noncanonical. |

Initial enrollment and fallback retain their existing operation classification
while joining `ErrUserInactive`; unknown fallback identities join `ErrUserNotFound`.
Operating lookup failures remain distinct internally. Public handlers must not
expose these classifications or raw database errors.

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

Install middleware inside the protected route group, after shared router
middleware. Required policy is captured by the middleware's construction;
changing an application variable later does not update an existing route.

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
additional-factor management and browser acceptance are described below. Existing
password-only routes keep their explicit password policy; enrolling a key does not make those
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
change application policy. Established factor management and browser acceptance
are described below. Synced/backup flags do not prove
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

### Established authenticator changes

`NewFactorService(credentials, queries, enrollment, fallback)` requires typed
`FactorQueries` and the existing actual registration verifier. A nil fallback
explicitly disables TOTP additions/replacements. `FactorPolicy` fixes recent
management proof and current access requirements in trusted application code;
use phishing-resistant MFA by default. An explicit MFA management profile can
upgrade a TOTP-only account. Password-only proof never manages established factors.

`List` returns at most twenty owned primary factors for the configured RP with kind, ID, security
revision, creation time and verified backup flags. `FactorSelection` is lookup
metadata; it supplies neither subject identity nor authorization. Zero selection
means addition in `BeginWebAuthnChange`/`BeginTOTPChange`; a nonzero exact owned
selection means replacement. `Remove` requires a nonzero current selection.

Finish requires both the actor bearer and the restricted `change1.` token under
the same server policy. This token never resolves as a session or initial setup.
Actual registration/TOTP confirmation, replacement/removal, version advancement,
all-session/pending revocation and optional actor rotation share one transaction.
A retained actor keeps all actual proof times and expiry boundaries unchanged.
Removing/replacing its proof constituent signs it out; no invalid proof survives.
At least one primary authenticator must remain, and phishing-resistant access
requires a remaining WebAuthn authenticator. Backup codes do not satisfy that
primary-factor constraint. Other RP credentials and TOTP records whose key identity
is unavailable cannot count as usable remaining factors. Core captures actual key
identities from its supplied fallback service; request policy cannot override them.
Recovery after loss belongs to a separate workflow.

Ticked exposes `/authenticators/manage`, safe `/authenticators/factors` listing,
and same-origin JSON `/authenticators/factors/remove`, `/webauthn/begin`,
`/webauthn/finish`, `/totp/begin`, `/totp/finish` beneath the factors path.
Commands accept only `target` (kind/ID/revision); TOTP finish accepts `code`.
WebAuthn finish accepts the bounded real browser registration response. Finish
uses `X-Factor-Change-Token`; only committed retained actors get a new cookie.
TOTP routes require the explicitly configured key ring. Backup regeneration
continues through `/authenticators/backup/issue` and displays codes once.
Apply migration `007-factor-control.sql` together with the current typed adapter.

### Browser acceptance

Run the optional `browser` build-tag test explicitly. It fails when prerequisites
are absent; default package tests do not claim browser coverage. Supply Go 1.27.1,
a Chromium executable with CDP virtual-authenticator support, Node with its built-in
`WebSocket` implementation, and an owned PostgreSQL database user that can create
and drop schemas. The verified local versions are Chromium 151.0.7922.173,
Node v26.8.1 and PostgreSQL 18.6.

```sh
export CHROMIUM_BIN=/usr/sbin/chromium NODE_BIN=/usr/sbin/node
export DB_HOST=/path/to/owned/postgresql/socket DB_PORT=5432
export DB_USER=postgres DB_NAME=postgres DB_PASSWORD=''
go test -race -tags=browser ./examples/ticked/internal/web \
  -run '^TestAuthenticatorBrowser$' -count=1 -timeout=240s -v
```

Replace database settings with the owned fixture's actual values. The test creates
a random schema, applies migrations 001 and 004 through 009, and removes that
schema on cleanup. It starts production Ticked handlers/core services on an
explicit localhost development origin. Password-only, recent MFA and recent
phishing-resistant routes share the application's trusted policy revision;
management requires recent phishing-resistant proof. Factor access is deliberately
phishing-resistant in this fixture to exercise the last-passkey constraint.

The finite Node harness uses the
[Chrome DevTools WebAuthn protocol](https://chromedevtools.github.io/devtools-protocol/tot/WebAuthn/)
with CTAP2/RK/UV, an internal device and a second USB device for another passkey.
It uses an isolated profile, loopback debugging and owned short socket scratch
under `/tmp/hatmax-browser-*`. It closes that browser and removes fixture scratch
after completion. No npm module, externally stored credential or signed-response
substitute supplies the successful browser journeys.

The test-only `/__test/otp` route generates actual codes through the existing OTP
library; it never verifies proof or changes persistence. This route exists only
inside the tagged test server. Production setup/finish verifies and consumes each
code. A previous-step setup code lets the next actual current-step sign-in complete
without waiting thirty seconds; setup generation avoids the final three seconds
of a step. Replay submits the identical consumed code.

Acceptance checks `navigator.credentials.create/get`, production management-page
registration, registration with no session, actual sign-in/step-up, retired actor
denial, TOTP and one-use backup MFA, last-factor protection and stronger-policy
denial. Tampered browser UP/UV/signature responses, pending tokens as cookies,
wrong-purpose finish, unknown/trailing/non-object/oversized bodies and failures
must issue no stronger cookie. Cookie metadata is Secure/HttpOnly/SameSite=Lax;
bearers remain hidden from page scripts and failure diagnostics.

Virtual devices establish browser/library/adapter integration. Hardware identity,
attestation trust, non-exportability, deployment HTTPS/key custody, application
authorization and any AAL/compliance assertion require separate consumer evidence.

## Mailbox verification

`RecoveryService` extends an existing `Service` and a typed `RecoveryQueries`
store. Its trusted policy revision is fixed at construction. Mailbox verification
uses the exact active account address, not a submitted replacement. Existing
accounts remain unverified until they complete the flow.

| Operation | Result and authority |
| --- | --- |
| `RequestMailboxVerification(ctx, mailbox)` | A committed transient `MailboxIssue`, for trusted dispatch only |
| `ConfirmMailboxVerification(ctx, bearer)` | Committed `MailboxVerification` metadata; no session or factor-management authority |
| `CleanupMailboxTokens(ctx)` | At most the configured batch of terminal tokens retained for 24h |

The bearer is exactly 80 ASCII bytes: lowercase canonical UUIDv4, a dot, and the
canonical unpadded URL Base64 encoding of 32 random bytes. Storage keeps only its
SHA-256 digest, separated by purpose, subject and record identity. Formatting
and JSON serialization redact the transient secret; snapshots omit the digest
from JSON. Only `Token.Bearer()` deliberately exposes it to trusted dispatch.
Verification and reset purposes never resolve as session tokens. See
[mailbox password reset](#mailbox-password-reset) for the separate completion contract.

Eligibility binds the subject, purpose, exact target, `AuthVersion`, trusted
policy revision and strictly unexpired time. Equality at expiry is unavailable.
Reissue replaces the purpose slot and invalidates its previous bearer; it does
not reset shared budgets. At most two slots exist per account. Current mailbox
changes must clear verification and advance `AuthVersion`; later reactivation
cannot revive an old token. UTC timestamps use microsecond precision.

PostgreSQL locks the subject before token and budget rows, charges failed-secret
attempts in an independent committed reservation, and uses a revision-bound
finite lease. Final confirmation rechecks time after row waits. It atomically
consumes the token, records mailbox verification, advances the version once,
deletes every subject session and restricted continuation, invalidates other
mailbox tokens and stores one notification intent. Password, account activation,
roles, confirmed factors, accepted TOTP steps and backup-code state are preserved.
A failure rolls back every final effect; previously charged attempts remain.

Internal errors distinguish `ErrRecoveryUnavailable`, `ErrRecoveryAttempts`,
`ErrRecoveryBusy`, `ErrRecoveryCapacity` and operating failure. Ticked groups
missing, wrong, expired, replaced, replayed and stale confirmations into one
redacted unavailable response. Initiation acknowledges all eligible/ineligible,
store and provider outcomes alike, with a six-second target response deadline
and a five-second work deadline. Earlier request cancellation wins; this timing
policy is not a production side-channel audit.

### Ticked transport and dispatch

Set the application-owned `TICKED_RECOVERY_ORIGIN` and configure `mailer.enabled`
and `mailer.mode: active` to enable `/account/mailbox` and
`/account/mailbox/confirm`. The origin requires HTTPS; explicitly enabled local
development permits HTTP only for loopback/localhost. Do not derive links from
request hosts or forwarded headers. A disabled/dry-run or unresolved no-op
mailer cannot enable these routes.

GET renders a form only. Mail carries the bearer in a URL fragment; the form
removes the fragment from navigation history and submits the token only through
an explicit POST. POST requires same-origin protection, exactly one URL-encoded
field (`email` or `token`), no query fields, and a body of at most 16 KiB.
Responses use `no-store` and `no-referrer`. Recovery forms require JavaScript:
explicit submission uses a same-origin fetch with the supported form media type.
This preserves source validation when native no-referrer navigation would send
an opaque origin. Middleware continues to reject foreign/opaque sources. The
request has a ten-second client deadline and never retries automatically.
Successful verification clears the
current session cookie and requires ordinary sign-in again.

Ingress runs before account lookup: at most 12 requests per socket IP per minute,
32 active requests and 1024 current IP entries. Entries expire without a worker.
Forwarded IP headers cannot select the limiter key. These process-local bounds
require separate deployment-wide anti-automation and trusted-proxy controls.

A committed issue is delivered outside database locks. Delivery failure does
not restore its predecessor; explicit bounded reissue is available. Notification
intents contain no bearer. The application attempts dispatch after confirmation;
a failure logs a generic error and cannot change the successful verification.
`MailboxDelivery.DispatchMailboxNotices(ctx, subject, limit)` is the explicit
application retry entrypoint: batches of 1–20, five attempts per intent,
seven-day retention and a five-second call deadline. It starts no worker.
Attempts are committed before sending. Concurrent retries or a lost provider
acknowledgment can duplicate mail within that finite attempt limit. Actual mail
provider delivery, deployment-wide protection and all-factor-loss identity
proofing are separate boundaries.


## Recent-proof password change

Use `RecoveryService.ChangePassword(ctx, actorBearer, candidate, policy)` with
trusted application policy. No account ID, asserted proof fact, client time or
caller-selected notification destination is accepted. The actual session identifies
the subject. `PasswordChangePolicy` contains an `AccessRequirement` and explicit
`AllowPassword`. Its factor requirement defaults to phishing-resistant MFA;
`RequireMFA` explicitly permits supported password/TOTP or backup-code proof.
`RequirePassword` is not a valid factor policy. Only a profile without established
primary factors may use password proof when `AllowPassword` is true. Primary
factors across all relying parties constrain this minimum; the adapter reads at
most 21 rows and denies a profile larger than 20.

The trusted revision must match the current session. Proof age defaults/clamps
to the credential service's recent-proof limit. Sign in or use the appropriate
actual step-up flow when proof is stale; password reauthentication cannot create
MFA proof. Current factor bindings are verified at authorization and commit.

```go
policy := auth.PasswordChangePolicy{
    Requirement: auth.AccessRequirement{Revision: "password-v1", MaxAge: 5 * time.Minute},
    AllowPassword: true, // Permits password proof only without primary factors.
}
changed, err := recovery.ChangePassword(ctx, sessionBearer, newPassword, policy)
```

`RecoveryQueries.AuthorizePasswordChange` locks subject before actor/factors,
rechecks active version/proof/policy/time and commits one kind-3 shared completion
attempt. It returns an owned `PasswordChangeAuthorization`, not caller authority.
The existing checker and shared bounded password verifier prepare/hash the complete
candidate outside database locks. Password policy, checker, KDF admission and
cancellation failures cannot refund the attempt. Oversized input or a malformed
bearer is rejected before storage. The existing fifteen-minute completion window
is shared with mailbox confirmation and reset completion.

`CommitPasswordChange` locks subject first and rechecks the actual actor digest,
generation, expiry, proof time, policy and complete current factor bindings after
lock waits. It samples trusted database time after writes too. One transaction
replaces the full encoded password, advances `AuthVersion` once, deletes every
session including the actor, deletes all restricted authentication/enrollment
continuations, revokes outstanding mailbox slots and inserts a kind-2 notice.
Notice capacity or write failure rolls all these effects back while admission
remains spent. Concurrent completions cannot overwrite a newer credential state.
An ambiguous commit error is an operating failure, not proof of rollback or an
automatic retry instruction.

`PasswordChanged` carries safe subject/time only. Activation, roles, current
mailbox verification, confirmed WebAuthn/TOTP, accepted replay state and backup
verifiers/consumption remain. The caller clears its cookie after successful
completion and requires normal current-policy sign-in; no replacement bearer is
issued. Password replacement cannot recover a lost required authenticator.

Ticked provides GET/POST `/account/password` when the existing explicit trusted
recovery origin and active-mail configuration are enabled. Trusted assembly
permits password proof without factors and uses phishing-resistant proof with
factors. The POST takes exactly one URL-encoded `password` field, no query fields,
at most 16 KiB body and 4096 bytes of complete candidate text. Identity comes only
from the session cookie. Same-origin protection, `no-store`, `no-referrer` and
finite ingress admission apply: 12 requests per socket IP per minute, 32 active
calls and 1024 current IP entries for this handler. Failure text omits candidate
and provider diagnostics; success clears the cookie and issues no replacement.

`MailboxDelivery.ChangePassword` commits first and then attempts the existing
bounded notification dispatch. Kind-2 notices contain no password, verifier,
bearer or recovery link. A provider failure leaves the mutation successful and
retains notification intent for explicit `DispatchMailboxNotices` retry, under
the same five-attempt, seven-day and finite-batch bounds as verification notices.
No retry worker starts. Captured mail tests establish this application boundary;
they do not establish external delivery or deployment-wide anti-automation.


## Mailbox password reset

`RecoveryService.RequestPasswordReset(ctx, mailbox)` issues a purpose-2 token only
for the active account's previously verified current address. Verification time
must be nonzero and not in the future. The exact address, subject, authentication
version and trusted recovery revision bind the issued record. It uses the
configured reset TTL, a separate issuance budget and the same two-slot bounded
storage as verification. Reissue invalidates the preceding reset link; it cannot
refund issuance, token attempts or shared completion admission. Core returns a
transient `MailboxIssue` only to trusted dispatch. The application keeps bearer
values out of public/operator responses and diagnostic output.

`ResetPassword(ctx, bearer, candidate)` rejects oversized candidates and malformed
bearers before storage. `RecoveryQueries.ReservePasswordReset` locks the subject
before its actual token, verifies the secret and purpose, commits finite token
attempts and kind-3 subject admission, then returns an owned revision/lease.
Known wrong secrets spend token attempts without running the password checker.
Foreign-purpose, terminal and stale records cannot enter password work. The
existing password policy and shared bounded verifier prepare/hash the complete
candidate outside database locks. Candidate rejection, checker failure, KDF
admission failure and cancellation cannot refund committed attempts. Canceled
work may leave a finite lease until expiry; no background cleanup worker starts.

`CompletePasswordReset` locks the subject first, rechecks the exact reservation,
current eligibility/version/policy and database time after lock waits, then commits
all effects together: complete encoded credential replacement, one version
advance, token consumption, deletion of every account session and restricted
continuation, revocation of other outstanding mailbox slots and a kind-3 notice.
Trusted time is checked after writes too; expiry or notice capacity/write failure
rolls back every security effect while attempts remain spent. Reset/change races
permit one winner against the same captured version. An ambiguous commit error
is an operating failure, not evidence of rollback or permission to retry blindly.

`PasswordReset` contains safe subject/time only. The operation never activates an
account, changes roles/address verification, issues a session or weakens factor
requirements. Confirmed WebAuthn/TOTP records, accepted counters/time steps and
backup verifiers/consumption remain intact. Ordinary sign-in with the new password
must satisfy the current application policy; all-factor loss requires separate
identity proofing. A reset link is not an authentication, enrollment or step-up
continuation.

### Ticked reset forms and operator initiation

The same explicit trusted recovery origin and active-mail configuration enables
GET/POST `/account/password/reset` and `/account/password/reset/confirm`. GET only
renders forms. Initiation accepts exactly one `email` field and acknowledges
eligible, missing, inactive, unverified, throttled, storage-failed and provider-failed
accounts alike with a six-second target and a five-second work deadline. This
bounded response policy is not a production side-channel audit. Admission occurs
before account lookup: 12 requests per socket IP per minute, 32 active calls and
1024 current IP entries, shared across this handler's public and operator routes.
Forwarded headers cannot choose the limiter key; deployment-wide protection is
application responsibility.

Mail carries the bearer in a fragment at `/account/password/reset/confirm#token=…`.
The form removes the fragment from history and completes only through explicit
POST. Completion accepts exactly `token` and `password`, no query fields, at most
16 KiB of form body, an 80-byte token and 4096 bytes of complete password text.
Same-origin protection, `no-store` and `no-referrer` apply. Success clears the
session cookie without replacement; denial text omits secrets/provider errors.

GET/POST `/admin/password-reset` requires a current actual session, the application's
`admin` or `superadmin` role and trusted recent MFA policy. Ticked selects
phishing-resistant proof with a five-minute age bound; trusted assembly may
explicitly permit supported MFA, never password-only proof. Admission precedes
session/account lookup. The operator submits only an email and receives the same
neutral acknowledgment. It cannot choose a password or notification destination,
obtain the bearer, bypass mailbox verification or grant authentication. Operator
roles belong to the application rather than the reusable core service.

`MailboxDelivery.ResetPassword` commits first and attempts the existing bounded
notification dispatch. Kind-3 notices contain no password, verifier, bearer or
recovery link. Provider failure leaves reset successful and retains notification
intent for explicit `DispatchMailboxNotices` retry (1–20 intents, five attempts,
seven-day retention, five-second call deadline). It starts no retry worker.
Captured-mail PostgreSQL/HTTP tests establish this boundary. The bounded
[recovery browser check](#recovery-browser-acceptance) establishes actual form and
navigator integration; external provider delivery and deployment protection
require separate evidence.

### Recovery browser acceptance

The `browser` build tag also provides `TestAccountRecoveryBrowser`. It extends
the authenticator harness through production mailbox/change/reset forms with
actual PostgreSQL, captured mail, virtual WebAuthn and existing TOTP/backup
ceremonies. Supply the same mandatory executable/database settings as the
[browser check](#browser-acceptance), then run:

```sh
go test -tags=browser -race -run '^TestAccountRecoveryBrowser$' \
  -count=1 -timeout=360s ./examples/ticked/internal/web
```

The recovery browser child has a 300-second deadline. The composite fixture explicitly
sets a finite 40-attempt authenticator subject window because its combined
positive/negative ceremonies exceed the default ten attempts; production defaults
remain unchanged. The journey respects the real one-minute ingress window before
its lost-response/replay checks; ingress limits are not bypassed. The recovery token/issuance/completion budgets use their defaults.

Acceptance covers neutral initiation, mailed fragments removed before explicit
POST, GET safety, a real foreign-origin submission, token/session isolation,
recent actual MFA for password change, Unicode replacement and retained factors,
spent/unused backup-code behavior, current role/MFA operator initiation, dispatch
and notification failure, explicit retry and a dropped committed reset response.
The lost-response fixture closes only its owned client connection after the
production handler commits. Replay cannot repeat the mutation; ordinary sign-in
with the new password still requires actual MFA at the strong route.

Test-only endpoints expose captured mail or control provider/transport failure
within the owned fixture. They never approve proof, mint tokens or mutate factors.
This proves browser/library/adapter integration, not external mail delivery,
hardware/attestation assurance, distributed anti-automation or all-factor-loss
identity recovery.

### Authentication controls browser acceptance

Both browser tests first exercise registration, password admission, shared ingress
and the actual Ticked `SecurityLogger`. Run the finite combined selector with the
same mandatory executable/database settings:

```sh
go test -tags=browser -race -v \
  -run '^(TestAuthenticatorBrowser|TestAccountRecoveryBrowser)$' \
  -count=1 -timeout=600s ./examples/ticked/internal/web
```

Two concurrent registration requests for one address receive equal sign-in
navigation and no session; observations distinguish the committed write from the
classified uniqueness refusal. Candidate-only password feedback and malformed,
ambiguous or oversized forms do not disclose submitted identities. Missing,
incorrect and exhausted password requests share the six-second fixture response
and cannot issue cookies. A final allowed wrong-password operation spends the
remaining durable charge; the following correct password is refused.

The fixture sets two active requests and a finite 1000-request peer window to
exercise common capacity separately from narrower recovery limits. Two admitted
requests retain slots through acknowledgment; a third receives `429` before core
work or a core event. Completed replies release both slots. Fixture getters read
only active count or redacted events and grant no authority.

The schema-local fixture uses 20 password-proof attempts and ten registrations;
a dedicated account's nearly spent window is prepared through actual failed
password operations. The authenticator child has a 210-second deadline; recovery
has 300 seconds. Server header/read/write/idle deadlines, CDP calls, owned profiles
and log capture are finite. Production defaults are unchanged.

The shared observer remains installed through actual WebAuthn, TOTP, backup,
factor and recovery journeys. Acceptance checks unique valid event identifiers,
committed mutation/proof provenance, all four actual proof methods, redacted
application logs and complete timely delivery within the fixture. This confirms
the consumer boundary exercised here; durable audit retention and deployment
side-channel or anti-automation assurance require separate evidence.

## Standalone credential admission

`NewCredentialAdmission(queries, cfg)` validates
`config.CredentialAdmissionConfig` and returns an immutable shared admission
service. `Admit(ctx, canonicalIdentity, purpose)` commits one operation before
lookup/checker/cryptography; `Cleanup(ctx)` removes at most its configured batch.
Both honor earlier caller deadlines. Admission grants no proof or session and
has no refund or retry operation. Identity must be valid UTF-8, 1–254 bytes without
control characters; applications own canonicalization and alias convergence.

Closed purposes are `CredentialRegistration` and `CredentialPasswordProof`.
Known, inactive and unknown identities share durable accounting; no subject is
created by admission. All operations, including later failed or canceled work,
spend their accepted charge. Defaults are ten proof operations per ten-minute
window with ten-minute cooldown, and three registrations per hour with one-hour
cooldown. The final allowed charge sets cooldown; denial does not extend it.
A new window starts only after window and cooldown end, including exact equality.
Tightening does not erase counts; relaxing/restarting does not replenish a window.

The caller implements `CredentialAdmissionQueries`. Ticked supplies
`NewCredentialAdmissionStore(queries, namespace, key)` with typed PostgreSQL
transactions and migration `009-credential-admission.sql`. Supply a stable
1–64-byte namespace using letters, digits, dot, underscore or hyphen, and a
separate application-owned 32–64-byte key; all
replicas share them. The adapter copies the key and stores only private HMAC
identity/binding values. A different key for an existing namespace is operating
failure. Do not put the key in source, public configuration examples or logs.
Private HMAC keys/digests are not authentication bearers or public event fields.

Storage caps both purposes together at 10000 records by default (100–100000).
Full capacity rejects new identities without live eviction; existing identities
retain their counts. Cleanup is explicit, bounded and worker-free; applications
own scheduling/cancellation. `ErrCredentialAdmissionAttempts` carries internal
retry time through `CredentialAdmissionDenial`, while capacity and invalid state
remain separate classifications. Do not disclose identity-specific retry time in
unauthenticated responses or reinterpret storage failure as admitted work.

Pass this required admission service to `NewService`. Signup charges registration
before its checker/hash; sign-in, initial enrollment and password-based fallback
charge proof before lookup/verification. Reauthentication charges the validated
subject identity before verification. Helpers never charge a single check twice.
Raw password byte/encoding bounds precede admission; new-password policy
failures still spend admitted registration work. The existing whole-operation
deadline includes admission. Current factor/recovery budgets, shared KDF limits
and proof authority remain unchanged.
Fixed-window admission does not implement cumulative authenticator disabling
or establish an authentication assurance level.

## Neutral public password transport

Ticked requires common authentication HTTP ingress before public credential work.
Sign-in, initial enrollment, TOTP setup and password-based fallback starts use
its captured whole-work deadline. Missing/inactive accounts, wrong passwords,
exhausted durable proof admission and operating failures share `403` with
`Authentication unavailable`, no cookie or identity-specific retry header, and
the configured acknowledgment target measured from HTTP admission. Successful
restricted challenges retain their dedicated payload and grant no session.

Signup accepts one bounded URL-encoded email/password/confirmation command.
Valid new, duplicate and throttled commands receive the same
neutral `303` navigation to `/signin` after acknowledgment. The response makes
no account-creation claim; normal sign-in establishes whether authentication
succeeded. Uniform operating failure receives `503` with `Registration unavailable`
at the same target, without a cookie, backend details or account references. Signup never calls sign-in or creates a cookie/session. Feedback for
an unsuitable password depends only on the submitted candidate. Syntax/body
and same-origin rejection occurs before account work; raw SQL errors and request
identities are never rendered or logged by these public transports.
Authentication URLs are omitted from ordinary request logging, including GET
queries; unrelated request logging remains enabled.

Applications composing these handlers must install the common boundary after
trusted-proxy resolution, retain active capacity through acknowledgment, use
finite server read/write deadlines and call `Close()` before stopping dependent
services. See [configuration](../configuration/README.md#authentication-http-ingress).

## Security observations

`NewSecurityObservations(observer, cfg.SecurityObservation)` constructs finite
synchronous best-effort delivery. Supply one initialized instance to `NewService`;
all its enrollment, WebAuthn, fallback, factor and recovery services inherit it.
The required `SecurityObserver.Observe(ctx, event)` must honor cancellation and
support concurrent calls. `DiscardSecurityObserver` explicitly discards events
and establishes no audit assurance.

Each public mutation/proof entrypoint attempts one terminal event after its
private operation returns and all database transactions, reservation releases
and verifier slots have finished. Password admission is already committed.
Internal helpers, session lookup/listing, cleanup and GET produce no additional
core events. Transport refusal before a core call produces no core event.

`SecurityEvent` has an independent event ID, UTC time, closed operation/outcome,
optional subject/record IDs and actual verified `ProofMethod`. IDs come from
trusted loaded/reserved/committed state; invalid IDs are omitted. Allowed reference
bytes are ASCII letters, digits, underscore and hyphen, at most 128 bytes each.
`Check()` validates the closed shape and its maximum 1024-byte JSON envelope.
There are no identities, IPs, user agents, credentials, bearer/digest material,
TOTP/backup values, links, arbitrary maps or error strings. Events supply no
authentication authority.

Outcomes distinguish committed mutation, completed session authentication,
pending issuance, pending enrollment/proof and classified denial. Only actual
verifier results or checked trusted actor proof supply methods; requested policy
and enrollment do not become completed MFA. Missing/inactive, invalid proof,
expired/replayed state, policy, exhausted attempts, capacity and stale state are
closed internal denials. Ambiguous storage outcomes are `operating_unknown`;
there is no rollback or success assertion without a confirmed result.

Callbacks acquire a non-waiting bounded slot and receive at most the configured
1–100ms timeout (default 100ms), strictly before a caller deadline. Concurrency
is 1–16 (default 2). No detached worker, queue or retry is created. Re-entry may
use remaining slots but cannot wait for capacity. `Diagnostics()` returns fixed
atomic delivery counters: delivered, saturated, rejected, canceled, deadline
and operating. `ErrSecurityObservationRejected` classifies consumer rejection;
other callback errors and panics produce redacted operating counters. Late nil
return after cancellation/deadline cannot count as delivered.

Cancellation is cooperative. A callback or logging sink that blocks beyond its
context violates the adapter contract; core cannot forcibly terminate it.
Delivery failure never replaces the authentication result, rolls back a mutation
or retries it. Timely callback success is not durable audit evidence. Consumer
retention, personal-data policy, delivery destinations and mandatory audit
persistence remain application obligations. Existing durable recovery
notification intents retain their independent atomic mutation contract. Ticked
uses its application-owned logger for checked events; configure a cooperative
sink and an appropriate retention policy.
