<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Security Model

Date: 2026-10-02
Status: Approved
Approved: 2026-10-02
Revised: 2026-10-03
Behavior: [Authentication security foundation](authentication-security.md)

## Ownership and Representation

Core defines logical records, validated proof and atomic operation guarantees.
The consuming application supplies its schema, queries and storage adapter.
There is no core organization, workspace, billing or administration-role table.
Field order below should be preserved when comparing the eventual Go contract
with application columns/indexes and provider payloads.

## Credential Record

Proposed logical fields: subject ID, encoded hash, scheme/version, bounded work
parameters and credential-change time. The encoding can carry version/parameters
and random salt without requiring redundant application columns. Define exact
encoding and limits before implementation. Stored data cannot request arbitrary
memory, iteration counts or algorithms.

Credential changes validate current state and serialize their durable writes.
Concurrent changes cannot overwrite a newer credential through a stale request.
Define input processing for the selected format and use it consistently at
creation/change/verification. There is no bcrypt reader or credential migration.

## Authenticator Record

| Logical field | Stored representation obligation |
| --- | --- |
| Subject and authenticator ID | Stable binding; unique authenticator identity under its protocol scope |
| Kind and protocol version | Validated supported type; no assurance inferred from a display name |
| Credential/public key material | WebAuthn credential ID and public key; origin/RP policy belongs to trusted configuration |
| Counter and backup properties | Protocol-defined counter/flags; policy distinguishes syncable and device-bound credentials |
| TOTP secret and accepted step | Authenticated encrypted envelope, external key identity and atomic replay state |
| Backup-code verifier | Versioned KDF verifier with independent random salt for each low-entropy code |
| Created, confirmed and revoked times | Enrollment confirmation and revocation are distinct lifecycle facts |

WebAuthn presence/verification is established by each validated assertion, not
by trusting a stored flag or client claim forever. Hardware/non-exportability
assumptions require supported evidence and policy, not a fabricated Boolean.
Backup-code consumption remains one-time even under concurrent transactions.

Replace the shared-salt backup-code encoding. New records use the selected
per-secret verifier; no reader or migration is required for the old encoding.

## Pending Authentication and Full Session

Pending-record fields: record ID, subject, opaque-secret digest, purpose, required
proof, creation/expiry, captured attempt budget/count, and consumed/revoked time.
Protocol challenges/enrollment data are bounded and purpose-bound. Pending records
are never returned by full-session lookup.

Full-session fields: ID, subject, opaque-secret digest, established proof/method
properties and their verification times, full-authentication time, creation,
last relevant activity, absolute
expiry, inactivity policy and revocation/version state. Exact storage format and
activity-update cadence remain design choices. Consumer navigation/domain state
is separate from proof and cannot raise assurance.

A full session is complete for the selected access policy, not proof sufficient
for every operation. Single-factor ordinary access and phishing-resistant MFA
have distinct verified properties. Activity or password reauthentication cannot
refresh the time of stronger proof. Required proof is evaluated against current
consumer policy; policy changes cannot inherit stronger assurance from an old
session. Finalize policy binding/version and concurrent checks with consumers.

Expired means current time is at or after the applicable boundary. Reactivating
a disabled subject cannot revive sessions revoked during deactivation. Renewal
rotates the secret; an old token cannot remain a parallel valid representation
of the renewed authentication.

## Recovery and Security Events

Recovery-record fields: ID, subject, purpose, intended target, opaque-secret
digest, creation/expiry and consumed/revoked state. Uniqueness/scope includes
subject and purpose where reissue replaces previous authorization. Mail delivery
state and recipient preferences remain application-owned.

Consumption and protected state changes share a transaction. Generic results
distinguish accepted, replayed, expired, denied and operating failure without
embedding a password/token in the result. Event metadata includes record
references, outcome and trusted proof context; never reusable authentication
material. Delivery/recovery evidence cannot be promoted into stronger proof
without performing the stronger method.

## Atomic Storage Operations

- Consume pending proof/factor and create exactly one full session as one commit.
- Consume a backup verifier or accepted TOTP step only once under concurrency.
- Replace password/security state, consume recovery authorization and revoke
  affected sessions/challenges without a partial durable result.
- Rotate session state with current-proof checks and invalidate the prior secret.
- Count failed attempts durably even though the caller receives an auth failure.
- Map duplicate identities and stale credential changes to classified outcomes.

Exact consumer-owned interfaces must make these guarantees implementable without
requiring consumers to duplicate core orchestration or pass an untyped database
handle into domain logic. Review lock order, uniqueness, rollback and failure
translation with real persistence adapters before finalizing the contract.
