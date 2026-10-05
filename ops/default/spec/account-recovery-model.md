<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Account Recovery Model

Date: 2026-10-04
Status: Approved
Approved: 2026-10-04
Behavior: [Account lifecycle and password recovery](account-recovery.md)
Parent model: [Authentication security model](authentication-security-model.md)
Prerequisite models: [Sessions](authentication-sessions-model.md), [Authenticators](authenticators-model.md)
Implementation status: Pending

## Ownership and Representation

Core defines logical values and typed operation guarantees. Applications own
tables, queries, canonical mailbox policy and mail/notification persistence.
Use the existing subject, `CredentialState`, `AuthVersion`, session generations,
factor bindings and versioned password record. Names below are logical contracts;
they do not assert that Go types or columns already exist.

## Current Mailbox and Credential State

| Logical field | Stored representation and rule |
| --- | --- |
| Subject ID | Existing account identity; foreign keys bind related records |
| Canonical mailbox | Existing identity address, at most 254 bytes; normalized/unique under application policy |
| Mailbox verified at | Nullable trusted UTC time, bound to the current exact address; initially absent |
| Authentication-state version | Existing positive monotonic `AuthVersion`; reject overflow |
| Password record | Existing complete bounded PHC record; no new algorithm or legacy reader |
| Credential changed at | Trusted committed replacement time |
| Active state | Current activation, independent of mailbox verification |

Mailbox confirmation advances version and invalidates sessions/pending records.
Password replacement preserves verification of the unchanged address. Changing
the address clears verification and invalidates prior security state atomically.
Reversing a change cannot revive a token or an earlier verification timestamp.

## Mailbox Token Slot

| Logical field | Stored representation and rule |
| --- | --- |
| Record ID | Independent UUIDv4; unique direct lookup; new ID on accepted reissue |
| Subject ID | Account foreign key; immutable within an issued token |
| Purpose | Closed `verify-mailbox` / `reset-password` enum |
| Target mailbox | Captured canonical address; exact comparison at reservation/commit |
| Authentication-state version | Captured positive version; current active subject must match |
| Policy revision | Captured trusted revision, nonempty and at most 128 bytes |
| Secret digest | SHA-256 over versioned domain separation, purpose, subject, record ID and random secret; fixed 32 bytes |
| Created / expires at | Trusted UTC microseconds; positive configured lifetime; equality expires |
| Attempt count / limit | Captured finite budget; durably increment before reset KDF work |
| Record revision | Positive monotonic mutation revision for reserve/complete comparison |
| Lease until | Nullable trusted expiry for one active verification reservation |
| Consumed / revoked at | Mutually exclusive terminal timestamps; cannot return to live state |

The transient token contains only record ID and its 32-byte random secret. Its
presentation is exactly 80 ASCII bytes. Reject alternate encodings, noncanonical
UUIDs, padding and malformed lengths/bytes before lookup. Compare digests in
constant time; digest serialization has explicit version and field boundaries.

Unique constraints cover record ID and `(subject, purpose)`. Reissue replaces the
slot, including terminal state, with new ID/digest and zero token attempts; shared
budgets remain untouched. At most two slots exist per subject. Replaced IDs have
no current lookup. No historical bearer archive or unbounded list is needed.
Missing/replaced rows map to unavailable.

Eligible means active, matching current target/version/policy, nonterminal,
before expiry and within attempt/lease bounds. Reset also requires current
mailbox verification. The last admitted attempt may complete its own reservation;
the exhausted count denies later reservations. Completion rechecks exact record
revision, lease and trusted commit time.

## Durable Attempt Windows

| Logical field | Stored representation and rule |
| --- | --- |
| Subject ID | Existing account foreign key; unknown identities allocate no row |
| Budget kind | Closed verification-issuance, reset-issuance or shared completion/change enum |
| Window start | UTC fixed 15m boundary from trusted storage time |
| Count / configured limit | Atomic admission counter; no overflow/wrap |

Unique `(subject, budget kind)` permits at most three rows per subject. Reuse the
row at rollover under the subject lock, rather than creating one per request.
Within a window, reissue, cleanup and credential changes cannot delete/reset its
counter. Token/shared completion charges commit together; there is no partial
admission. Exhaustion changes no activation, credential, factor or session state.

Wrong secrets for current IDs commit attempts without acquiring a useful lease;
returning a denial cannot roll back those counters. Unknown IDs do no password
work. A correct secret obtains at most one lease and binds its exact revision to
completion. Release affects only the matching reservation; timeout/failure never
refunds counters. An active lease denies another reservation without stealing it.

## Recent-Proof Password Change

Input binds the actual actor session digest, generation, current subject version
and trusted `AccessRequirement`. Core derives proof from session validation;
a caller-provided Boolean or `VerifiedProof` cannot grant authority. Current
established factors constrain the minimum requirement under the behavior spec.
Capture exact bindings and recheck them under the subject lock.

After authorization/admission, core produces a policy-approved encoded password
outside database locks using the shared verifier. The typed commit request binds
it to the authorized actor/version/requirement; it is not an arbitrary-subject
replacement endpoint. Failure cannot issue a session or expose the password record.

## Atomic Operations and Lock Order

Use existing subject-first lock order. Lock actor/token records after subject
ownership is established; factor checks follow existing factor contracts.
Trusted storage time governs eligibility, leases, windows and final checks.
Request-supplied time, mailbox eligibility and proof classification grant nothing.

| Logical operation | One committing transaction |
| --- | --- |
| Issue token | Recheck active target/version/policy and reset verification prerequisite; charge issuance window; replace purpose slot |
| Reserve token | Resolve current ID, lock subject then token; recheck binding; charge token/shared budget; verify digest and acquire lease if admitted |
| Confirm mailbox | Recheck actual secret/reservation/current binding; consume, mark verification, increment version, invalidate all sessions/continuations, persist notice |
| Authorize password change | Recheck actual actor/current factors/recent policy; charge completion/change window; return owned snapshot for bounded KDF work |
| Commit password change | Recheck actor/version/proof/policy/time; replace record, increment version, invalidate all sessions/continuations, persist notice |
| Complete reset | Recheck actual reserved token/current verified target/version/policy/time; consume, replace record, increment version, invalidate all sessions/continuations, persist notice |

Confirm uses the same token/shared budget and reservation even without password
KDF. Invalidation covers sessions, initial-login/reauthentication pending records,
initial/established-factor enrollment and other mailbox tokens. The consumed
token retains consumed state until cleanup/reissue; it is not also revoked.
Confirmed factors, accepted replay state, encrypted TOTP and backup verifiers
remain intact.

Credential replacement and consumption never commit separately. Forced failures
roll back final effects; independently committed attempts remain spent. Competing
completions have one winner. Lost responses cannot grant another authorization.

## Notification Intent and Transient Mail Delivery

The adapter persists a notification intent with the security mutation. Logical
fields are unique event ID, subject, closed event kind, captured safe destination
reference and committed UTC time. Kinds distinguish verified mailbox, password
change and reset. Uniqueness prevents duplicate intent for one mutation. The
intent contains no password, verifier, digest, bearer or recovery URL.

Application-owned outbox/provider records may add bounded dispatch state,
attempts and retention under their delivery contract. Dispatch failure cannot
recreate or roll back the security operation. Destination references are private
delivery metadata, not public event payloads.

Initial token mail is separate: the raw bearer exists only in a trusted dispatch
result after commit and is omitted from serialization/logging. Retaining it for
retry requires protection and finite expiry/retention. Otherwise reissue replaces
the token under the same budgets.

## Cleanup and External Mapping

Explicit cleanup removes at most 1000 terminal/expired slots whose 24h retention
has elapsed. Use indexed bounded selection; never hold a token lock while waiting
for its subject lock. Do not delete a current active lease; logical expiry still
denies completion. Cleanup cannot reset an unexpired attempt window. Subject
deletion follows application lifecycle.

Mail secrets never appear in public issuance/operator JSON. Completion results
expose outcome/safe references and issue no session. Adapters clear cookies after
successful password change/reset; durable revocation also invalidates clients
that missed the response. Confirmation cannot preserve an authenticated cookie.

Acceptance compares these fields, constraints, operations and lock order with
the concrete adapter schema/queries, failure evidence and production handling.
The model and behavior were approved together on 2026-10-04.

## Password Change Implementation Names

Slice 2 settles `PasswordChangePolicy` (trusted `Requirement` and explicit
`AllowPassword`), `PasswordChangeAuthorization` (owned actor digest/session,
current bounded primary-factor bindings and normalized policy), and
`PasswordChanged` (safe subject/time only). `RecoveryQueries.AuthorizePasswordChange`
charges kind 3 before candidate work; `CommitPasswordChange` receives that owned
snapshot and the encoded record. Factor proof defaults to phishing-resistant MFA;
password proof is allowed only with no established primary factors and explicit
policy permission. Proof age defaults/clamps to the credential service's recent
proof bound. No caller supplies a subject, proof fact or clock to the public service.

`RecoveryPasswordFactors` reads at most 21 owned primary-factor metadata rows
across all relying parties; more than 20 denies. Existing `ReplacePassword` SQL
is used inside the protected final transaction. No schema migration is needed.
Ticked `PasswordChangeHandler` accepts exactly one bounded URL-encoded `password`
field at `/account/password`; the actor comes only from the session cookie.
`MailboxDelivery.ChangePassword` commits first and then attempts kind-2 notices.
Ticked enables this route under its existing explicitly configured recovery/mail
boundary. It explicitly permits password proof without factors and retains the
phishing-resistant default when any primary factor is established.

## Password Reset Implementation Names

Slice 3 settles `RecoveryService.RequestPasswordReset`, `ResetPassword` and
safe `PasswordReset` subject/time completion. Existing redacted `MailboxIssue`
and owned `MailboxRecord` represent both closed token purposes with distinct TTLs.
`RecoveryQueries.IssuePasswordReset`, `ReservePasswordReset` and
`CompletePasswordReset` preserve explicit purpose authority. The adapter shares
subject-first issuance/reservation mechanics while each entrypoint rejects the
other purpose before admission. No additional schema or token wire format is needed.

Reset requires an active subject and a current mailbox verification timestamp
at or before trusted time, at both issuance and final use. The reservation charges
kind 3 plus the token attempt before shared candidate/checker/KDF work. The exact
revision/digest/lease snapshot binds final consumption, complete credential change,
all-session/pending invalidation, other mailbox-slot revocation and kind-3 notice.
No access session, factor change or activation is produced.

`PasswordResetHandler` owns GET/POST `/account/password/reset` and
`/account/password/reset/confirm`; completion accepts exactly `token` and `password`,
with existing token/password/body bounds and no query fields. Links use fragments
and explicit POST. Public initiation has the same six-second neutral response for
all eligibility/storage/provider outcomes. Ticked's `/admin/password-reset` uses
actual recent qualified session proof and existing admin/superadmin role policy
before invoking the same mail-only request. Core defines no operator role or bypass.
