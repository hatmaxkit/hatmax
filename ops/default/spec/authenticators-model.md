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
Implementation status: Slices 1-2 delivered; Slice 3 active

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

## Slice 1 Concrete Binding

`NewAuthenticatorService` requires the existing credential service (and its
shared bounded password verifier), a mandatory `AuthenticatorQueries` adapter
and validated `config.AuthenticatorConfig`. The adapter owns stable opaque RP
handles, enrollment-only pending records, one subject-budget row and atomic
registration confirmation. It exposes typed storage commands; callers cannot
submit an approved proof to the service. No optional storage upgrade is used.

Pending secrets use an `enroll1.` wire prefix and SHA-256 domain
`hatmax/pending/enrollment/v1`; ordinary session parsing rejects that prefix.
Records capture version, RP, random handle, required profile/revision/freshness,
password time and the exact v0.18.2 begin-session encoding. Each successful
reservation increments a durable revision and attempts, with one finite lease.
The final transaction locks subject before pending, checks post-lock time, and
inserts the verified credential while incrementing version and deleting all
sessions and pending enrollment rows. Registration returns safe metadata only.

WebAuthn storage keeps bounded credential ID/public key, counter, verified
presence/UV/backup flags and versioned library credential data. Initial enrollment
is the only executable purpose in this slice. TOTP authenticated-encryption keys
remain application-owned explicit key identities; encrypted envelopes bind
subject, factor identity and material purpose as associated data. No seed or
unused encryption interface is introduced before Slice 3.

Parser evidence uses `FuzzEnrollmentResponse` for 20 seconds without KDF or DB
work. The real PostgreSQL selector is
`Test(EnrollmentTransactions|Credential|Proof|Session|Control)` with `-race`,
`-count=1`, and `-timeout=120s`; missing `DB_HOST` is a failure. Browser evidence
remains Slice 5.

## Slice 2 Concrete Binding

`NewWebAuthnService` requires the credential service and a `WebAuthnQueries`
adapter that includes enrollment storage. The existing enrollment-only
constructor stays valid. Both use the same bounded pinned protocol configuration;
there is no optional storage upgrade or caller-provided proof callback.

Authentication and step-up use distinct `assert1.` and `stepup1.` tokens and
SHA-256 domains. Pending purposes 2 and 3 share the bounded collection and durable
subject attempt budget with enrollment. A stored owned ceremony includes allowed
credential IDs; bounded factor bindings capture security revisions at begin.
Step-up additionally captures the current session ID, digest and generation.
Its entry policy is explicit server-owned password-or-better access under the
same policy revision, with freshness disabled; completion must satisfy the
separately captured stronger requirement. Initial authentication has no password
fact. The nullable enrollment password timestamp is absent for assertions.

A confirmed factor snapshot carries security revision and a distinct replay
revision. Each assertion advances only replay revision, counter and current
backup state, atomically with pending consumption and session completion.
Security revision changes invalidate constituent bindings; routine assertion
counter updates do not invalidate previously issued sessions. Final completion
matches both snapshot revisions and the previous counter/backup state under lock.
Zero/zero counters are permitted; otherwise the new counter must strictly exceed
the old counter. Backup eligibility cannot change. Required UV/presence and
signature verification happen in the library; core independently enforces the
counter policy and finite profile before sending a typed completion command.

`VerifiedProof` has closed password/WebAuthn methods. WebAuthn includes factor ID,
security revision and actual assertion time; password includes no factor fields.
Both satisfy password-or-better policy, and only actual UV WebAuthn satisfies MFA
and phishing-resistant MFA in this slice. Session storage constraints and safe
metadata agree on this shape. Session validation and actor controls recheck the
current factor ownership/revision. Password reauthentication produces only a
fresh password fact, never refreshes WebAuthn constituent time.

Final transactions lock subject, pending and actor sessions before factors.
Post-lock trusted time rechecks pending expiry, lease, policy, live actor generation
and completed proof age before writes and again before commit. Session admission
and cleanup remain bounded. Any completion failure rolls back replay state,
pending deletion and insertion/rotation; durable admission remains charged.

Parser fuzz: `go test ./auth -run '^$' -fuzz '^FuzzAssertionResponse$'
-fuzztime=20s -parallel=2 -timeout=60s`, with no KDF/database work.
Real PostgreSQL: `go test -race -tags=integration
./examples/ticked/internal/feat/auth
-run '^Test(WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$'
-count=1 -timeout=120s`; missing `DB_HOST` fails. Browser evidence remains Slice 5.

## Slice 3 Concrete Binding

`NewFallbackService` requires the credential service, a mandatory `FallbackQueries`
adapter, finite `config.FallbackConfig`, and an owned seed key ring (one active
identity, at most eight 32-byte keys). AES-256-GCM envelopes contain a random
12-byte nonce and bind purpose/version, subject, factor and key identity as AAD.
No key is generated implicitly. Ticked enables fallback routes only when both
`TICKED_TOTP_KEY_ID` and `TICKED_TOTP_KEY` are supplied; partial/invalid settings
fail initialization. The latter is canonical standard Base64 of 32 key bytes.

TOTP has one active record per subject in a separate typed table, counted together
with WebAuthn under the shared subject lock. It uses six digits, SHA-1, 30-second
steps and skew zero or one (default one). Verification returns the actual matched
step, with current step preferred over adjacent steps. Completion checks that
step against post-lock time and strictly advances accepted step/replay revision.
Initial setup consumes its accepted step, advances account version, and revokes
all sessions and pending records without issuing a session.

Fallback purposes 4 (setup), 5 (sign-in), and 6 (step-up) use distinct `totpset1.`,
`fallback1.` and `fallstep1.` prefixes/domains. They share `auth_pending`, its
finite cap, expiry cleanup and durable subject budget. Their bounded ceremony
payload contains captured method/skew/factor bindings and encrypted setup material;
RP columns are null/empty for these purposes. Password time is always captured
from actual password verification. Step-up captures current actor digest,
identity and generation. Reservation commits attempts/lease before OTP or code
verification; final completion rechecks the exact snapshot and current factor.

Closed session methods 3 and 4 represent password plus TOTP or backup code.
`VerifiedAt` is the actual password time, the older constituent; `FactorAt` is
actual factor verification/completion time. MFA accepts these methods, while
phishing-resistant MFA accepts only UV WebAuthn. Routine accepted-step/code-use
changes do not change security revision. Backup session proof binds the active
set, so consuming its own code does not invalidate the newly completed session.

Backup wire format is `backup1.<UUID identifier>.<22-character Base64url secret>`:
128 random secret bits, exact canonical ASCII parsing, no case/space rewriting.
Subject plus identifier selects one stored verifier before one KDF. Each verifier
is an independently salted PHC Argon2id record using the credential service's
bounded engine and input domain `hatmax/backup/v1` plus subject, ID and secret.
Issue defaults to eight codes and admits one through ten. Recent management proof
is server-owned MFA or phishing-resistant MFA (default), with at most configured
recent-proof age. Password-only and backup-code actors cannot regenerate a set.
Reservation charges the durable shared budget before hashing. Replacing the set,
advancing account version, revoking sessions/pending and rotating the retained
actor commit together. Plaintexts return once, only after that commit.

Parser fuzz selector: `go test ./crypto -run '^$' -fuzz '^FuzzBackupCode$'
-fuzztime=20s -parallel=2 -timeout=60s`, without KDF/database work.
Real PostgreSQL selector: `go test -race -tags=integration
./examples/ticked/internal/feat/auth
-run '^Test(FallbackTransactions|WebAuthnTransactions|EnrollmentTransactions|CredentialTransactions|ProofTransactions|SessionTransactions|ControlTransactions)$'
-count=1 -timeout=180s`; missing `DB_HOST` fails. Browser evidence remains Slice 5.
