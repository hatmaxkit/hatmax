<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Controls Model

Date: 2026-10-05
Approved: 2026-10-05
Status: Approved
Behavior: [Authentication attempts, errors and security events](authentication-controls.md)
Parent model: [Authentication security model](authentication-security-model.md)
Implementation status: Partial

## Ownership and Representations

Core supplies closed operation/outcome values, immutable validated settings and
typed admission/observer contracts. Applications own identity normalization,
namespace/key provisioning, SQL tables/queries, trusted time and delivery adapters.
Extend the existing Ticked `internal/feat/auth`, migration and generated-query
layout. The logical contracts below match the settled Go/SQL/config representations
recorded for Slices 1–4. Those runtime slices are merged and delivered; actual
browser/affected acceptance is reviewing in Slice 5. Its maintainer merge and
the exact integrated gate remain pending.

## Credential Admission Record

| Logical field | Storage and constraint |
| --- | --- |
| Namespace | Trusted stable application deployment identifier, 1–64 safe ASCII bytes; never request-selected |
| Purpose | Closed `registration` or `password-proof`; stable across operation/policy revisions |
| Identity key | 32-byte HMAC-SHA-256 over a versioned admission domain, unambiguous namespace and canonical identity; no plaintext identity in the admission table |
| Window start | Trusted database UTC time, read after the row is locked |
| Captured window/cooldown | Validated durations; preserved through the current window |
| Captured limit | Positive finite limit; a stricter current limit applies without reinitialization |
| Charged operations | Integer from zero through the captured limit; a lower current limit denies additional charges without truncating spent counts |
| Cooldown until | Absent before the last admitted operation; fixed trusted boundary after that charge |

Unique key: `(namespace, purpose, identity_key)`. The key identifies the exact
canonical account lookup value; all password-proof entrypoints and actual
session-derived reauthentication converge on it. Registration is independent.
No subject foreign key is required: missing identities use the same budget and
creating a user does not erase or replace it. Actual subject records and sessions
remain in their existing tables with unchanged proof/version authority.

Use a separate application-owned secret of at least 32 random bytes through
standard library HMAC; do not reuse password-verifier or bearer material. Replicas
and restarts share namespace/key identity. Do not include access-policy revisions
or request-specific values in the counter key. Key rotation or normalization
changes must preserve active records; the proposal does not authorize a key
reset/migration. HMAC identifiers are private pseudonyms, not anonymous data or
safe public event fields. Raw identities never enter this table or core events.

Window end is `window_start + captured_window`. The last admitted operation sets
`cooldown_until = trusted_now + captured_cooldown`. Deny until trusted time is at
or after both boundaries. Denial updates no timestamp or counter. New admission
after both boundaries replaces the captured settings and starts at one. A stricter
limit applies as the minimum of captured and current limits; an already larger
spent count remains unchanged and denies the rest of the captured window. Denial
cannot add a cooldown or extend it. The last newly admitted charge sets the
original captured cooldown once. Current duration changes take effect in the
next window rather than renewing a live one.

Index cleanup by namespace and the later of window end/cooldown end. Deletion
rechecks eligibility under lock; an admission that has renewed the row makes it
ineligible. Expired records are disposable resource state, not authentication
history. A missing record is not an unlimited allowance if shared capacity is full.

## Shared Capacity and Atomic Operations

Maintain one bounded capacity record per configured namespace, not a new one per
request. It carries the effective maximum and current number of admission rows
across both purposes. Insertion and deletion update it atomically; an existing
record's charge does not consume a new slot. Refuse new identities when full,
without deleting live records or granting untracked work. Counter drift and
invalid settings are operating errors, not permission to bypass capacity.

Admission and cleanup use one deterministic lock order: namespace capacity, then
credential admission rows in a stable key order. They finish before any account,
session, factor or recovery mutation transaction starts. They never nest under
the subject-row locks used by delivered security mutations. Cryptography and
observer calls occur outside all database locks.

The typed admission operation receives canonical identity, purpose and validated
settings, derives its private key inside the trusted adapter, checks shared
capacity, uses trusted time and commits one charge or a classified denial. It
returns bounded retry timing only for trusted internal handling; it returns no
bearer, identity digest, credential or authorization proof. Cancellation/ambiguous
commit yields operating failure and no subsequent expensive work or automatic
retry. There is no release/refund endpoint.

Cleanup accepts a validated finite batch and context deadline. It locks/rechecks
only expired candidates, deletes up to that batch and updates capacity together.
No credential, session, challenge, factor or recovery row is changed. Stable
ordering/indexes and real PostgreSQL concurrency evidence are required before
delivery. Use bounded counts rather than a full-table recount on each request.

## Process-Local Ingress State

| Logical field | Representation and rule |
| --- | --- |
| Peer key | Canonical trusted `middleware.ClientIP` result; IPv4/IPv6 normalization follows existing proxy rules |
| Window start/count | One bounded counter per peer; no timestamp slice or stored request body |
| Peer capacity | Validated finite map size; a live entry is not evicted to admit another peer |
| Active count | Finite simultaneous work, including acknowledgment waits; no waiting queue |
| Cleanup cursor/batch | Bounded progress over expired entries; no full unbounded scan |

One guard instance is shared across the actual authentication route assembly.
Middleware construction starts no worker. Admission/cleanup are race-safe, reject
invalid peers/configuration and observe exact expiry equality. Scope and lifecycle
are process-local; replicas do not share this state. Application shutdown owns
any caller-created scheduler and waits for its cancellation before infrastructure
release. Preserve narrower existing recovery limits during assembly.

## Outcome and Event Values

Expected internal codes cover accepted/completed, pending/incomplete proof,
missing/inactive identity, invalid proof, expired/replayed state, candidate-policy
rejection, attempt exhaustion, capacity, stale state and operating/unknown result.
Preserve existing `errors.Is` identities; specifically, a real email uniqueness
constraint remains `ErrEmailTaken`, while an unrelated SQL constraint is operating
failure. Error-to-public mapping is application-owned and exhaustive.

| Event field | Bound and provenance |
| --- | --- |
| Event ID | Independent record identity; not derived from a bearer or admission key |
| Occurrence time | Trusted UTC time of the observed outcome |
| Operation | Closed registration, sign-in, reauthentication, session, pending/factor and recovery operation values |
| Outcome | Closed classification; ambiguous commit remains unknown/operating |
| Subject/record reference | Optional bounded trusted identifier, at most 128 safe ASCII bytes each; omit unavailable or invalid references |
| Proof method | Optional actual verified method from the result; no stronger proof inferred from policy or request fields |

Serialized payload is at most 1024 bytes. It has no arbitrary map, free-form error,
identity/admission digest, raw identity/IP, credentials, cookie, token or link.
There is no new event table or core outbox. Applications may map the validated
event into an existing publisher/logger, but transport-specific payloads cannot
flow back into the core event shape.

Emit at most one terminal event per public core entrypoint; helpers do not emit a
second event. The actual mutation result determines completion. No observer runs
inside an admission or subject transaction. Pending challenge issuance and
partial password proof remain distinct from completed MFA/session authentication.

## Observation Failure and Lifecycle

The observer is caller-owned, context-aware and concurrency-safe. Core uses
non-waiting bounded callback admission; saturation skips invocation and records
a safe bounded failure classification. A callback gets at most the configured
deadline or the caller's remaining time. No detached callback worker or retry is
created. Callback re-entry cannot wait indefinitely for observer capacity.

Delivery failure classification is limited to saturation, rejection, canceled,
deadline or operating failure, with bounded counters/enum diagnostics and no
driver/callback error text or identity labels. It cannot replace a committed
authentication result. A callback returning success after cancellation is late
delivery, not confirmed delivery. An observer that ignores context violates the
cooperative contract and is not forcibly terminated by another goroutine.

Durable mandatory audit delivery, destination retention and event retry are
consumer obligations. Delivered recovery notification intents remain atomic with
their mutations and independent of this best-effort observer.

## Slice 1 Settled Representations

`config.CredentialAdmissionConfig` / `CredentialAdmissionSettings` capture
`password_*`, `registration_*`, `max_identities`, `timeout` and `cleanup_batch`
under `credential_admission`. `CredentialAdmissionPurpose` closes
`CredentialRegistration` and `CredentialPasswordProof`.
`NewCredentialAdmission`, `Admit` and `Cleanup` orchestrate validated identity,
caller deadlines and the typed `CredentialAdmissionQueries` boundary.

Ticked's `CredentialAdmissionStore` captures an immutable namespace and a copied
32–64-byte application key. `credential_admission_capacity` stores namespace,
private HMAC key binding, finite row count and configured maximum. A different
key for an existing namespace is operating failure, never a new budget.
`credential_admissions` stores namespace/purpose/private identity key, captured
window start/end, cooldown microseconds, captured attempt limit/count and optional
cooldown end. Primary key is namespace/purpose/key; retirement index uses
`GREATEST(window_end, COALESCE(cooldown_until, window_end))`.
Migration `009-credential-admission.sql` and matching sqlc queries own storage.
Capacity precedes row locks; trusted time is read after locks. Existing records
may continue at a lowered capacity; new identities wait until occupancy permits
insertion. Cleanup locks/rechecks retired rows and decrements capacity atomically.
No subject/session/factor/recovery table is touched. Slice 2 connects these
implemented admission methods to every actual password entrypoint.

## Slice 2 Settled Representations

At Slice 2, `NewService(queries, cfg, checker, admission, logger)` requires a non-nil
`*CredentialAdmission`; optional construction and process-local fallback are
not supported. Enrollment and fallback reuse this credentials service and its
same admission instance. Slice 4 adds the required observation parameter to
this constructor, as recorded below. A package-local structural helper checks bounded UTF-8
password input before committing the selected purpose, without calling the
new-password checker. Public entrypoints charge before account lookup; session
reauthentication charges the validated subject email before actual verification.
Fallback helper calls charge once. Admission is nested under the existing whole
operation deadline; fallback proof operations use the lesser credential/factor
timeout. Recovery new-password policy and backup proof budgets remain distinct.

Ticked requires `TICKED_CREDENTIAL_NAMESPACE` and a canonical base64
`TICKED_CREDENTIAL_KEY` decoding to 32–64 application-owned bytes. Its adapter
copies the key; assembly clears its temporary decoded bytes. Missing/invalid
material fails construction without logging it. No schema/query change is needed
beyond delivered migration 009. Unit acceptance fakes cover orchestration only;
real adapter fixtures share explicit fixture keys across constructors.

## Slice 3 Settled Representations

`middleware.RateLimitConfig` supplies finite limit/window, maximum peers and
cleanup batch. `NewRateLimiter(cfg)` returns a validated limiter or error and
starts no worker. A map indexes one list node per canonical peer; the bounded
list rotates at most one cleanup batch per admission or explicit `Cleanup`.
Nodes contain only peer, window start and count. No live eviction or timestamp
slices are used; exact expiry equality retires a window.

`config.AuthenticationIngressConfig` supplies peer requests/window, maximum
peers/active requests, cleanup batch and acknowledgment. Settings derive the
maximum actual credential/authenticator/recovery deadline, with a five-second
floor for existing recovery initiation. Acknowledgment must fit work plus
100ms observer budget and 100ms margin, up to 31s; default is 6s.

Ticked's `AuthenticationIngress` is one application composition instance. Its
root middleware protects POST sign-in/signup/sign-out/reauthentication, session
revocation, authenticator and account recovery paths before body/identity work.
Accepted request context carries fixed work/ack deadlines; public credential
handlers require it before service calls. Active capacity is retained through
response waits and released on every exit. Explicit `Close` stops new admissions
before application context cancellation and infrastructure shutdown. Narrower
existing recovery guards remain inside this shared boundary.

Credential forms are bounded to 16KiB, exact single fields and supported media
type with no query input. Protected POST uses existing same-origin middleware.
The actual HTTP connection receives a read deadline at the captured whole-work
deadline without clearing it before the server drains the body. The server owns finite
header/read timeouts and a write timeout one second longer than acknowledgment;
net/http resets those deadlines between requests. Slow body ingress cannot
retain active capacity indefinitely. The server propagates shutdown context.

Ticked's `AuthenticationLog` omits URI logging on authentication paths, including
safe methods whose query could contain credential material. Ordinary request
logging remains enabled; typed terminal security observation belongs to Slice 4.

Candidate-only policy feedback is explicit; all valid new/duplicate registration
outcomes navigate to normal sign-in without a cookie. Public password denial
uses one fixed response policy without raw identities/errors or identity retry
time. Registration operating failures use a uniform non-authorizing unavailable
response at the same target; shutdown cancellation also returns unavailable. `ErrUserInactive` supplies the missing stable core classification.

## Slice 4 Settled Representations

`config.SecurityObservationConfig` and `SecurityObservationSettings` resolve
`security_observation.timeout` (default 100ms, 1–100ms) and `concurrency`
(default 2, 1–16). `NewSecurityObservations(observer, cfg)` requires a caller-owned
`SecurityObserver`; `DiscardSecurityObserver` is an explicit no-audit adapter.
`NewService(queries, cfg, checker, admission, observations, logger)` requires one
initialized observation instance, inherited by enrollment, WebAuthn, fallback,
factor and recovery services. Ticked supplies one application logger adapter.

`SecurityEvent` has closed string operation/outcome types, an independent UUID,
UTC occurrence time, optional subject/record IDs and optional `ProofMethod`.
Event-only IDs accept ASCII letters, digits, underscore and hyphen up to 128
bytes; invalid trusted references are omitted. Checked JSON is at most 1024
bytes. Outcomes separate committed mutation, completed session authentication,
pending issuance/enrollment/proof, denial reasons and operating/unknown results.
Existing neutral error identities remain intact; private operation facts preserve
more precise classifications when a public error intentionally merges causes.
No reference is copied from a bearer or request-selected target.

Public entrypoints wrap private synchronous implementations. Package-private
attempt facts capture only trusted loaded/reserved state and actual verifier
results. The wrapper observes after all operation defers, leases and callback
slots are released, using the original caller context. Internal helpers never
observe. Session lookup, listing, cleanup, page GET and transport rejections
before a core entrypoint do not generate core mutation events.

`SecurityObservations` owns a finite non-waiting channel and fixed atomic counters
for delivered, saturated, rejected, canceled, deadline and operating delivery.
It invokes the observer synchronously with the lesser configured timeout or a
deadline strictly before the caller's deadline. No worker, queue or retry is
started. Success is checked again against cancellation/deadline after callback
return; panic and arbitrary adapter errors become bounded operating diagnostics.
Callback re-entry can consume remaining slots but cannot wait for a slot.
Consumer cancellation cooperation remains mandatory. Recovery notification
transactions and authentication return values remain independent of delivery.

## Slice 5 Acceptance Representations

The browser fixtures share the actual Ticked logger observer across the existing
service graph and isolated PostgreSQL schema. Finite test-only getters expose
active request count and closed redacted event values, never proof or secret
material. A bounded 512-line/2048-byte capture validates unique event IDs, actual
methods and timely delivery; fixture setup observations are accounted separately.
No product contract, table, migration, configuration default or authority changes
are introduced by acceptance. The exact integrated gate remains post-merge.
