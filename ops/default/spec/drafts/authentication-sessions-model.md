<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication and Sessions Model

Date: 2026-10-03
Status: Proposed
Behavior: [Authentication and sessions](authentication-sessions.md)
Parent model: [Authentication security model](../authentication-security-model.md)

## Required Proof and Verified Facts

A trusted requirement contains required method properties, maximum proof age
when applicable and current policy revision. It comes from server configuration
or authorized application policy, never request-form fields. Validation rejects
unsupported or inconsistent requirements.

Verified facts bind subject, method and verification time to actual core
verification. Password verification establishes only password proof. MFA needs
actual independent factors or a verified multi-factor authenticator; phishing
resistance additionally needs the corresponding reviewed protocol properties.
Enrollment, a role string, a stored display label or a caller Boolean supplies
none of those facts. Facts can be matched as pure metadata, but their production
creation is restricted to the corresponding verifier path. Verification times
are not in the future relative to the trusted clock. Reject corrupt/out-of-range
time facts; do not silently refresh or clamp them. Method facts use a closed
validated representation; future verifier additions must select a finite field
and collection bound before accepting their protocol output.

A completed result contains safe session metadata and a transient issue secret.
Pending results contain purpose-bound pending metadata and, only if a supported
continuation exists, its transient secret. Denied/unavailable outcomes contain
no session or continuation secret. Errors return no successful result.

## Full Session Representation

The logical field order below is the order to compare in Go and SQL during
implementation. Public/context metadata omits the digest.

| Logical field | Stored representation and invariant |
| --- | --- |
| ID | UUID record identity; primary key; never used as a bearer |
| Subject ID | Required reference to current user; indexed for bounded management |
| Secret digest | Exactly 32 bytes; unique within the full-session namespace; SHA-256 over a fixed full-session domain and decoded 32-byte secret |
| Auth-state version | Positive BIGINT snapshot matching the subject's `AuthVersion` |
| Policy revision | Required bounded revision from trusted policy; current revision must match |
| Session generation | Positive BIGINT; advances on rotation; stale writers cannot replace current state |
| Verified method facts | Bounded supported representation with actual method verification times; this set writes password facts only |
| Full-authentication time | Trusted completed authentication time; does not advance on activity |
| Created time | Immutable record creation time |
| Relevant activity time | Monotonic trusted accepted activity; coalesced updates never pass actual accepted activity |
| Absolute expiry | Computed from completed authentication; equality is expired |
| Inactivity lifetime | Validated finite snapshot; effective idle expiry is activity time plus duration |

No plaintext token column or redundant client assurance flag exists. Secret
encoding is canonical padded URL Base64; digest input is decoded random bytes,
with distinct fixed namespaces for full sessions and each pending purpose.
Session lookup accepts a digest, never a raw bearer. A separate issue value
carries the raw bearer once, outside the stored record.

Session states are live, expired, revoked and superseded. Expired/revoked records
cannot transition back to live. Revocation removes the stored row; rotation
replaces the digest/generation atomically. Expired rows are reclaimed before
subject admission and by explicit bounded cleanup. No session history/tombstone
collection is introduced; application-owned audit destinations can retain safe
security-event metadata under their own bounds.

## Future Pending Representation and Current Boundary

Executable pending records belong to the later AUTH-05 flow and are stored
separately from sessions. Its logical fields are ID, subject,
secret digest, purpose, captured auth-state version, policy revision, required
proof, already verified facts, creation and expiry. Reauthentication pending
state additionally binds the original session ID/generation. Purpose is one of
initial authentication or reauthentication; enrollment need is an outcome, not
permission to mutate an authenticator.

The later endpoint must select and validate a finite per-subject retained-row
cap and expiry. Count all retained pending rows, reclaim expired rows before
admission and remove canceled/consumed rows. No completed pending record
can be reused. Unknown, malformed or cross-purpose secret lookup fails without
creating a session. Unsupported-method results do not allocate a secret/record
with no executable continuation. In this set, the password path either completes
under its current requirement or returns a non-authorizing requirement outcome.
No pending SQL table or issue/consume endpoint is added solely to reserve future
functionality; outcome metadata has no reusable secret.

The representation reserves the parent invariant for later AUTH-05 integration;
this set must not expose a public pending-completion API without a real verifier
and atomic consumption. AUTH-05 defines protocol challenge fields, factor replay
state and durable attempt budgets alongside its actual submission endpoint.
It may refine the pending representation through the official parent concern.

## Atomic Operations

| Operation | Required inputs | Transaction invariant |
| --- | --- | --- |
| Issue session | Verified subject/version, trusted current policy, newly generated digest and completed proof | Lock/recheck active subject and version, evaluate time after lock waits, reclaim bounded expired rows, enforce subject cap and insert one complete session |
| Validate and touch | Digest, trusted current policy and clock, relevant-activity intent | Recheck active/version/policy/generation and both expiries coherently; conditionally advance activity only while still live; return owned safe snapshot |
| Reauthenticate and rotate | Current digest/generation, current requirement and newly verified proof, replacement digest | Recheck subject/session/policy, repeat all required proof, replace digest and generation and clocks as one transition; old token invalid after commit |
| Revoke current/selected/others/all for subject | Validated actor and target scope with current required recent proof, or explicitly authorized trusted administrative call | Recheck relevant current state; delete only authorized target sessions; no retained token can regain access |
| List subject sessions | Validated actor, authorized subject, bounded page/cursor | Return safe metadata only; stable bounded ordering; never cross target scope |
| Future create/cancel pending | Real supported continuation, subject/version/policy/purpose and digest, or matching owned pending state | AUTH-05 defines retained-row cap, expiry and purpose; no ordinary session lookup; not an interface required by this set |
| Cleanup | Trusted clock, finite batch and context | Remove only expired records, at most configured rows; no owned loop or unbounded scan/result |
| Future complete pending | Actual verifier result and matching live pending/current state | Consume pending/factor replay state and create/rotate exactly one session together; unavailable until AUTH-05 delivers the real path |

Only the supported session rows above are required operations of this set.
Future pending rows state the parent invariants for AUTH-05 review, not placeholder
methods to ship now.

Adapters return owned snapshots and classify state/capacity conflicts rather
than database diagnostics. Required methods implement these contracts directly;
no optional interface or public proof-approval callback is introduced. Final Go
signatures must carry enough state for these checks without a raw database handle.

Operations needing multiple lock kinds acquire subject first, then session or
pending records in stable ID order. Cleanup may lock/delete only expired records
without subsequently acquiring a subject lock. Failed transactions leave no
partial secret replacement, session issuance or pending consumption.

Security time comes from a trusted clock evaluated at the final locked check;
passing only an earlier service timestamp cannot prove current expiry. Adapter
SQL/time precision must agree with equality semantics and deterministic tests.
Authorization holds at that transaction's linearization point; consumers recheck
state for protected domain writes when their concurrency contract requires it.

## Proof and State Transitions

```text
password verification + password-only requirement -> completed session
password verification + unmet stronger requirement -> pending or unavailable; no session
live session + sufficient fresh reauthentication -> atomic rotation
live session + insufficient proof -> unchanged session; denied operation
expired/revoked/stale session -> denied; no activity or rotation revival
later real factor completion + current pending -> atomic one-use completion
```

Activity can change only relevant activity time. Full authentication time and
per-method times advance only through the actual corresponding completed proof.
Policy changes require current evaluation; an old revision does not silently
inherit stronger authority. Account mutation/version increments continue to
serialize through the credential-security subject boundary.
