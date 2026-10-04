<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authenticator Model

Date: 2026-10-04
Status: Approved
Approved: 2026-10-04
Behavior: [Authenticators](authenticators.md)
Parent model: [Authentication security model](authentication-security-model.md)
Session foundation: [Authentication and sessions model](authentication-sessions-model.md)
Implementation status: Pending

## Representation and Ownership

Core logical records and finite outcomes align with application-owned Go/SQL
bindings. Ticked supplies the real PostgreSQL adapter and initial schema/SQLC
mapping. Review corresponding fields in this order before each affected task;
the proposal is not a final Go signature or a library-struct database schema.
Return owned snapshots. Safe public records omit secrets, digests, encrypted
seeds, pending verifier state and reusable authorization material.

## Subject and Authenticator

Extend the subject's current monotonic `AuthVersion` boundary with a stable opaque
32-byte WebAuthn user handle, unique within RP scope. Handle creation is conditional
and idempotent under the subject lock; it cannot change on an email rename.

| Logical field | Stored representation and constraint |
| --- | --- |
| Authenticator ID, subject ID, RP scope | Stable record binding; subject foreign key; never request-selected authority |
| Kind and record version | Closed WebAuthn/TOTP kinds and supported encoding version |
| Created, confirmed and revoked state | Unconfirmed material cannot authenticate; removed active rows confer no authority |
| Record revision | Positive monotonic revision for conditional commits and snapshot mismatch detection |
| WebAuthn credential ID | Bounded binary ID; unique `(RP scope, credential ID)` even across subjects |
| COSE public key and algorithm | Bounded validated encoding; supported algorithm, separate from display metadata |
| Sign counter, backup eligibility/state, UV initialization | Validated protocol fields; counter and flags updated with completion; no inferred hardware label |
| Bounded library-required credential metadata | Versioned exact fields needed by the pinned verifier, including transports/AAGUID/attestation metadata; validated limits, no private key |
| TOTP encrypted seed and key identity | Authenticated encryption envelope bound to subject/record/kind/version; decryption failure denies without exposing material |
| Last accepted TOTP step | Nullable before confirmation, then monotonic accepted integer; exact window rechecked at commit |

Use kind-specific child records or constrained columns so invalid cross-kind
combinations fail. SQL constraints validate supported enums, bounds, version and
the eligibility/state relation. Enrollment mutation counts retained setup and
active records under the subject lock. Revocation invalidates captured proof by
advancing auth version and removing/revoking the applicable record atomically.
Any retained revocation history needs an independently bounded application owner;
this set introduces no unbounded authenticator tombstone collection.

## Backup Code Set

Fields are subject ID, set ID/revision, code ID, complete PHC Argon2id verifier,
creation and expiry/revocation where applicable. Each verifier has an independent
salt. Input binds purpose, subject and code ID to a random 128-bit secret. The
identifier is public lookup metadata, never authentication by itself. Its
canonical wire form and matching KDF input are finalized before Slice 3.

Unique `(subject, code ID)` permits one bounded lookup and KDF. A subject has one
active set with at most ten rows. Consume/remove a code with pending completion;
two transactions cannot use it successfully. Replace/delete the previous set in
the same authorized regeneration transaction. Plaintext codes are returned once
after commit and are not persisted or logged. A lost success response requires a
fresh authorized regeneration; do not retain plaintext to replay it.

## Pending Record and Verification Lease

| Logical field | Stored representation and constraint |
| --- | --- |
| Record ID, subject ID, secret digest | Separate pending collection; independent namespace and unique digest; no full-session lookup |
| Purpose and protocol | Closed initial authentication, reauthentication or enrollment; enrollment mode is initial or established; intended protocol and target fixed by server |
| Captured auth version, policy revision and requirement | Required owned snapshot; rechecked against trusted current state |
| Already verified facts and their times | Core-derived password proof only until real protocol completion; cannot be client-written or refreshed by activity |
| Actor session ID, digest and generation | Required together for reauthentication/established-factor management; absent for initial sign-in/setup |
| Target factor ID/revision or enrollment mode | Bounds and subject binding; mutable only through an authorized new operation |
| Protocol challenge and library ceremony data | 32-byte fresh challenge, bounded versioned begin output, RP/origin/options binding; not client-reconstructed |
| Created and expiry | Positive finite lifetime; account/security-state mutation invalidates all captured versions |
| Attempt count/limit and lease revision/deadline | Durable bounded reservation before verification; one lease per pending; final commit checks matching lease and exact expiry |

Pending states are live, leased, expired, canceled and consumed. Consumed/canceled
records are removed. Expired rows are reclaimed before admission and by explicit
batch cleanup. A failed or expired lease never refunds its spent attempt. A new
lease increments revision; a stale verifier cannot complete through it.
Enrollment additionally stores encrypted unconfirmed material in a bounded
setup record; neither the seed nor ceremony state appears in ordinary results.

One per-subject factor-budget row stores window start, attempt count and cooldown
boundary. Reservation locks this row with the subject and pending, charges both
budgets, commits before expensive work and returns an internal owned snapshot.
Failure is a classified outcome with durable charged attempts, not a rolled-back
authentication error. Reset is determined by trusted locked time; new pending
records cannot reset the row. A non-admitted busy request need not spend an attempt.

## Completed Proof and Session

Replace the password-only fact with a closed representation of actual constituent
methods, factor IDs/revisions and verification times. A UV WebAuthn assertion
has protocol-verified presence/verification and supported credential-class facts;
password/TOTP or password/backup has both actual constituent times. Stronger
authority and effective freshness are computed by core; clients cannot declare
either. Record data is bounded by the finite supported combinations.

Persist proof constituent bindings alongside the session in a constrained shape.
Validation rechecks current account version, trusted policy, lifecycle and
proof sufficiency. Factor security mutations advance `AuthVersion`; a removed
constituent cannot remain part of a rotated actor's proof. Relevant activity
changes neither constituent times nor factor class. Existing independent bearer,
digest, generation, admission, expiry and safe-context contracts remain in force.

## Required Atomic Operations

| Operation | Locked checks and one-commit guarantee |
| --- | --- |
| Begin pending/setup | Active subject/version and explicit restricted authority; bounded expired-row reclamation/cap; insert digest and exact begin state |
| Reserve verification | Current subject/pending/policy, time, attempts and unleased state; charge subject/pending budgets and assign one finite lease before work |
| Finish registration/setup | Matching live lease, target/subject/version and current enrollment authority; consume setup, activate verified factor, advance auth version and revoke other sessions/pending together |
| Complete initial authentication | Matching live lease and actual verification bound to current factor revision/state; recheck time/counter/step/code; consume factor replay state and pending, enforce session cap and insert exactly one session |
| Complete step-up | Recheck original live digest/generation and all above state; consume pending/replay state with exactly one generation rotation; old bearer invalid after commit |
| Change established factors/backup set | Recheck actor recent proof satisfying trusted management policy/current constituents and retained-count/last-factor constraints; advance auth version, revoke affected rows and rotate retained actor coherently |
| Cancel/cleanup | Actor-owned cancel or expired-only finite batch; no later acquisition of subject locks from cleanup |

Acquire subject first, then pending/session records in stable ID order, then
authenticator/backup records in stable ID order. Final expiry/proof/step checks
use trusted time sampled after waits. Budget reservation releases its transaction
before cryptographic work. No untyped transaction handle or external proof callback
crosses into core. Required typed adapters may carry bounded core-produced commit
commands; exact signatures must preserve these checks without reimplementing
core policy in every caller.

Cryptographic work may use a current owned snapshot outside the final transaction.
Commit conditionally matches the subject/factor/pending revisions and validates
candidate step/counter/proof times against current locked state; a stale valid
cryptographic result is insufficient. Operating failure returns no issued secret,
no successful count and no partial durable completion. Attempt reservations are
separate already-committed admission state and remain spent.

## Invariants and Mapping Review

- The same pending digest never resolves as a session, recovery or other purpose.
- No session is completed from setup flags, method labels or client UV claims.
- Valid protocol proof plus a stale account, factor, generation or lease is denied.
- Counter/OTP/code consumption and session completion share the final commit.
- Setup/removal/regeneration cannot retain invalid factor proof in the actor.
- Subject/RP/credential uniqueness and finite collection counts hold under races.
- Go validation, SQL constraints, wire parsing and generated mappings agree on
  types, enum values, field bounds and microsecond expiry equality.

Before implementation review the library credential/session encoding, encrypted
envelope/key dependency, internal reservation representation and proof/SQL
constituent mapping together. These are concrete design bindings of the proposed
contract, not permission to add placeholder interfaces or an unimplemented method.
