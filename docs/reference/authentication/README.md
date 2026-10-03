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

`VerifiedProof` has a closed `Method` and `VerifiedAt`. The only supported method
is `PasswordProof`; its production creation follows the core password verifier.
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
required step, not an enrollment/verification endpoint. No pending secret, SQL
table or consume method is supplied. AUTH-05 must deliver actual verification
and atomic factor/challenge consumption before any executable continuation.
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
