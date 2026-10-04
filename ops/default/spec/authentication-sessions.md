<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions

Date: 2026-10-03
Status: Approved
Approved: 2026-10-03
Concern: authentication-sessions
Parent: [Authentication security foundation](authentication-security.md)
Model: [Authentication and sessions model](authentication-sessions-model.md)
Plan: [Delivery plan](../plan/authentication-sessions.md)
Tracker: [Delivery tracker](../tracker/authentication-sessions.md)
Inspected baseline: `56b1cb05ea4243fbb87fe41ef3c7bb83b72e4fb5`
Implementation status: Implemented
Completed: 2026-10-04
Execution gate: Closed

## Purpose and Ownership

Provide reusable authentication outcomes and stateful session guarantees.
Password verification must not imply that every operation's required proof has
been met. Session possession must not outlive expiry, revocation, current account
state or the applicable authentication policy.

This concern refines approved AUTH-03/AUTH-04 and the error/resource obligations
needed for those operations. The completed credential-security set supplies
Unicode policy, the shared verifier and conditional authentication-state storage.
Core owns proof evaluation, secret processing and lifecycle orchestration.
Applications supply trusted operation policy, persistence adapters, authorization,
HTTP presentation and security-event destinations. Core does not own roles or
application-domain policy. Dependencies point from applications toward these
contracts; core has no application-package or product-document dependency.

The bounded concern is implemented in PR #95/#96/#97. Its
[completed tracker](../tracker/authentication-sessions.md#requirement-evidence-and-handoff)
records AS-01 through AS-09 and that set's exact integrated full gate. This set
established the password/session portion of AUTH-03. The subsequent
[authenticator concern](authenticators.md) delivers actual stronger-factor
verification, atomic consumption/completion and its own integrated evidence.

## Verified Baseline

- `auth.Service.Signin` issues a session immediately after password verification.
- `Session` stores a plaintext UUID bearer with only creation and absolute expiry.
- `ValidateSession` uses `ExpiresAt.Before`, accepts equality at expiry and checks
  account activation without binding the session to its captured `AuthVersion`.
- Invalid session duration silently falls back to 24 hours. There is no inactivity,
  rotation, proof freshness, concurrent-session cap or bounded management API.
- Middleware attaches user metadata, although session context helpers exist.
  `RequireTOTP` checks setup rather than completed factor proof.
- The Ticked adapter already serializes credential/session writes through the user
  row. Password changes revoke sessions and role/activation changes advance version.
- `crypto.GenerateSecureToken(32)` already uses CSPRNG bytes and padded URL Base64.
  Its reference currently describes raw Base64; update that description with the
  affected implementation documentation rather than creating a second encoding.

## Behavior

### Required proof and outcomes

Trusted server code supplies an immutable access requirement and policy revision
for each operation. Requirements distinguish password-only, MFA and
phishing-resistant MFA; recent proof additionally has a finite maximum age.
Changing a role, route or setting cannot manufacture verified method properties.
A policy revision mismatch invalidates the old authorization snapshot.

The result distinguishes `completed`, `pending_enrollment`, `pending_proof` and
`denied`; operating failure is an error with no successful result. Only completed
results carry a full-session secret. In this set, pending outcomes describe an
unmet requirement; they grant neither access nor a continuation secret. Later
executable pending state must be purpose-bound, expiring and stored separately;
its secret must never resolve through session validation.

This set implements production password verification and the outcome/session
boundary. Stronger requirements never fall back to password-only completion.
Existing enrollment flags are not factor proof. If no supported verifier can
satisfy a requirement, return a classified unavailable-method outcome without an
issuable completion path. Pending enrollment/proof may explain the required next
step, but does not provide access or complete enrollment.

The subsequent [authenticator concern](authenticators.md) supplies AUTH-05's
production verification and atomic proof consumption. This session foundation
adds no public completion API accepting a method name, client Boolean, asserted
timestamp or caller-fabricated proof receipt. Actual completion verifies through
core's reviewed mechanism and consumes proof with session creation/rotation in
one transaction. It cannot reinterpret an earlier password result as MFA.

### Secrets and validation

Generate 32 independent random bytes per full or pending secret; UUIDs remain
record identifiers. Use canonical padded URL Base64 (44 bytes for 32 random bytes),
matching the existing helper. Reject noncanonical, malformed or oversized input
before storage lookup. Store only purpose-separated SHA-256 digests of decoded
high-entropy secrets. Do not apply a password KDF to session bearers.

Plaintext secrets appear only in the immediate issue/rotation response and the
chosen transport. Stored records, ordinary lookups, context metadata, session
lists, logs and security events contain no reusable secret. Pending and full
secrets use distinct lookup namespaces; pending purposes cannot substitute for
one another. No UUID/plaintext compatibility reader or historical migration is
needed. Update active adapters, initial schemas and callers together.

Validation rechecks subject activation, captured auth-state version, policy
revision, session generation/revocation and absolute/inactivity expiry. Equality
at either expiry is expired. The final checks evaluate trusted current time after
lock waits; a timestamp sampled before a blocked transaction is insufficient.

Relevant authenticated activity can advance the stored activity time at a bounded
cadence, with a conditional update. It never moves absolute expiry or method-proof
time. Coalescing may expire a session conservatively early by at most one cadence;
it cannot prolong access beyond actual activity. Background polling is not user
activity. Validate/touch cannot revive expired, revoked or superseded state.

These guarantees hold at the validation/transaction linearization point. Protected
domain mutations still require application authorization and any needed current-
state recheck in their own transaction; a context snapshot is not an indefinite
permission grant.

### Reauthentication, rotation and control

Successful reauthentication repeats the proof required by current policy,
atomically rotates the secret and advances session generation. It resets the
applicable lifetime/activity clocks only for a completed fresh authentication.
Password proof or activity cannot refresh stronger proof times. Failed proof or
transaction failure leaves the old valid session unchanged and returns no new
secret. An expired session requires a new sign-in.

A rotation racing with another rotation, sign-out, revocation, password change
or account disable has one serialized outcome. After rotation commits, the prior
secret cannot validate or rotate again. Subject disable/re-enable never revives
an old session, because its captured auth-state version is stale.

Provide current-session revocation, subject-wide revocation, other-session
revocation and bounded subject-session listing. Viewing/terminating sessions uses
a validated actor and recent proof according to current policy. The application
owns administrative authorization for a different target subject; receiving a
subject ID from a request does not authorize that operation.

Enforce a per-subject session limit under the same lock as insertion. At capacity,
reject new admission with a classified outcome; do not silently evict another
session. Listing returns safe metadata and a bounded cursor, never token digests
or reusable secrets. Remove revoked records and reclaim expired subject records
before admission so retained rows cannot grow without a per-subject bound.
Cleanup is explicit, cancellable and batch-bounded; no owned background loop is
introduced. Future executable pending admission needs a separately bounded
per-subject collection, designed with its actual authenticator endpoint.

## Defaults and Bounds

These approved toolkit defaults are not mandated assurance profiles.
All settings validate at construction; malformed values fail without fallback.
Applications choose tighter policy where required.

| Setting | Default | Supported bounds or relation |
| --- | --- | --- |
| Absolute session lifetime | 24 hours | 1 minute through 30 days |
| Inactivity lifetime | 30 minutes | At least 1 minute; at most absolute lifetime |
| Activity persistence cadence | 1 minute | At least 1 second; at most 5 minutes and one quarter of inactivity lifetime |
| Recent-proof maximum age | 5 minutes | At least 1 second; at most absolute lifetime; operation policy may tighten |
| Sessions per subject | 10 | 1 through 100 retained session rows |
| Management page size | 50 | 1 through 100; opaque cursor at most 128 bytes |
| Cleanup batch | 1000 | 1 through 1000 rows per call |
| Session storage-operation timeout | 5 seconds | Positive, at most 30 seconds; earlier caller deadline wins |
| Policy revision | Explicit caller-owned value | Nonempty printable ASCII, at most 128 bytes; no token or personal data |

The credential verifier keeps its existing independent size/KDF/admission limits.
Pending records currently have no factor-submission endpoint; durable failed-factor
budgets and pending retention/admission limits must be delivered with that
endpoint, not inferred from session limits or outcome enums.

## Failure and Atomicity Contract

Classify invalid proof, insufficient/recent-proof requirements, expired/replayed
state, stale account/policy/session state, unavailable method, capacity exhaustion,
invalid configuration and operating failure. Never expose credentials or database
text through ordinary public outcomes. Handlers own non-enumerating responses.
A storage timeout must not be reported as successful revocation or rotation.

Required storage operations enforce the [model's transaction invariants](authentication-sessions-model.md#atomic-operations).
No optional upgrade interface, untyped database handle or callback that approves
arbitrary claimed proof is introduced. Store owned snapshots. Operations requiring
both locks acquire subject before session/pending records, then record IDs in
stable order. Failure rolls back the whole transition. Validation reads and
conditional activity writes must observe one coherent current state.

## Acceptance Criteria

| ID | Observable acceptance evidence |
| --- | --- |
| AS-01 | Unique 32-byte CSPRNG secrets; canonical parser; storage/events/context contain no reusable secret; pending secrets fail full-session lookup |
| AS-02 | Absolute and inactivity equality reject access; malformed durations fail startup; relevant activity cannot change absolute expiry or proof times |
| AS-03 | Password-only policy completes; stronger policy cannot complete from password or enrollment flags; unavailable verification fails closed |
| AS-04 | Current required proof, freshness and policy revision are evaluated on protected operations; old weak sessions cannot acquire stronger authority |
| AS-05 | Rotation is atomic, prior secret invalid after commit; competing rotations and rollback have exactly the documented outcomes |
| AS-06 | Password/account-state mutations and disable/re-enable invalidate old sessions; racing validation/touch cannot resurrect them |
| AS-07 | Session admission caps hold under concurrent insertions; safe management pages and cleanup obey their finite bounds |
| AS-08 | Recent-proof session management enforces subject ownership; trusted administrative authorization remains application-owned |
| AS-09 | Updated Ticked/Postgres adapter demonstrates persistence and transaction claims; middleware exposes safe validated metadata; active callers compile |

Use deterministic time tests, token-parser fuzzing without database work, race
checks and real PostgreSQL lock/rollback/concurrency tests. Test the supported
password path end to end. Future-method test fixtures prove rejection/property
matching only; they cannot establish production MFA or phishing resistance.

## References and Evidence Boundary

The parent spec's pinned standards remain the verification target. Relevant
sources checked on 2026-10-03:

- [ASVS 5.0.0 V7](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x16-V7-Session-Management.md): v5.0.0-7.1.1/7.1.2, 7.2.3/7.2.4, 7.3.1/7.3.2, 7.4.1/7.4.2 and 7.5.1/7.5.2 guide lifetime, capacity, secret, rotation, revocation and recent-proof evidence.
- [ASVS 5.0.0 V6](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x15-V6-Authentication.md): v5.0.0-6.3.3 remains a real-authenticator obligation, not an outcome-enum claim.
- [NIST SP 800-63B-4, Session Management](https://pages.nist.gov/800-63-4/sp800-63b.html#session): session proof inheritance and independently enforced timeout/reauthentication constraints.

A passing toolkit set does not establish complete ASVS coverage, a NIST AAL or
FIPS validation. Protocol verification/enrollment, recovery and broader durable
attempt controls retain their owners in AUTH-05/AUTH-06/AUTH-07. This concern
supplies session contracts they must integrate, not substitute flows.

## Review Decisions

Approval on 2026-10-03 accepts the defaults/capacity behavior, conservative activity
coalescing, policy-version invalidation, three-slice boundary and the explicit
partial AUTH-03 delivery. Exact Go method/field signatures and SQL bindings are
settled before the corresponding slice changes callers; their required semantic
operations and stored representations are defined in the model. Material changes
to this boundary or authenticator completion require another reviewed scope.
