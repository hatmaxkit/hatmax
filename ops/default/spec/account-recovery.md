<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Account Lifecycle and Password Recovery

Date: 2026-10-04
Status: Approved
Approved: 2026-10-04
Concern: account-recovery
Parent: [Authentication security foundation](authentication-security.md)
Model: [Account recovery model](account-recovery-model.md)
Plan: [Delivery plan](../plan/account-recovery.md)
Tracker: [Delivery tracker](../tracker/account-recovery.md)
Prerequisites: [Authentication and sessions](authentication-sessions.md), [Authenticators](authenticators.md)
Inspected baseline: `6e02dea9602e954ed9654b3ce2383049c32f808f`
Implementation status: Partial
Execution gate: Open

## Purpose and Ownership

Implement AUTH-06 through verification of the current account mailbox, authenticated
password change and mailbox-authorized password reset. Extend the delivered
credential, session and authenticator engine. Include only the AUTH-07 errors,
attempt admission and resource bounds needed for these operations.

Core owns token generation/verification, password policy/KDF orchestration,
proof evaluation and typed atomic storage contracts. Applications own identity
normalization, account eligibility, schemas/adapters, transport, mail composition
and delivery, operator authorization and domain access policy. No role name or
application-specific entity grants authority inside core.

The existing authenticator concern owns enrollment, removal, replacement and
backup-set regeneration. This concern preserves those contracts. Recovery from
loss of every qualifying factor and operator-assisted identity proofing require
their own threat review; these password operations cannot authorize them.

## Delivered Prerequisites and Current Gaps

The linked prerequisite concerns and their delivery sets are complete. Mailbox
verification is delivered; protected password change is implemented in Slice 2.
The reset and final acceptance slices remain pending.

- `auth.Service` owns the validated password policy and shared bounded verifier.
- `auth.Queries.ReplacePassword` and the Ticked adapter check `CredentialState`,
  replace the complete record, advance `AuthVersion` and revoke sessions. They
  remain trusted storage primitives. `RecoveryService.ChangePassword` adds actual
  actor-derived recent-proof authorization and complete continuation invalidation.
- Session/factor operations serialize through the subject row and recheck actual
  proof, policy, generation, factor bindings and trusted time.
- Current-mailbox verification, purpose-bound tokens and atomic confirmation are
  delivered. One-use token-consumption/password-replacement remains Slice 3.

Extend `auth` and the repository-owned Ticked adapter. Exact Go identifiers are
finalized with the delivery plan; operation names below are logical contracts.

## Mailbox State and Token Authority

A mailbox is the application's current canonical account address. Verification
records control of that exact address at a trusted time. It is distinct from
account activation and authentication assurance. Core does not infer activation
or application access from mailbox verification.

Token purposes are closed: `verify-mailbox` and `reset-password`. Each token binds
an independent record ID, subject, exact canonical mailbox, captured `AuthVersion`,
purpose, trusted policy revision and creation/expiry. Use an independent CSPRNG
32-byte secret and a purpose-separated digest; store no plaintext bearer. The
canonical presentation is a UUIDv4 record ID, a dot and the unpadded base64url
secret. Lookup uses the ID; verification checks the complete binding.

At most one current token exists per subject and purpose. Accepted reissue
replaces that token atomically with a new ID and secret. It neither extends an
old token nor resets shared budgets. Competing issue/consume operations serialize
through the subject; a token replaced before consumption cannot win. Issuance
alone cannot change a password, account activation or session state.

Reject at or after expiry. Another purpose, target, subject, policy revision or
authentication-state version makes the token unavailable. Deactivation, mailbox
change and protected authentication-state changes invalidate outstanding tokens;
later reactivation cannot revive them. Consumers changing a mailbox must clear
verification, advance `AuthVersion` and invalidate affected tokens/sessions
atomically. This concern verifies an existing address; authorization to change
that address needs its own workflow.

## Observable Operations

### Request and Confirm Mailbox Verification

An eligible active account with an unverified current mailbox can request a
verification token. The application resolves the account under its normalization
policy and sends only to the captured current address. Confirmation requires the
actual secret and rechecks eligibility, target, version, policy and time.

Commit token consumption, mailbox verification time, one `AuthVersion` increment
and invalidation of all sessions and pending authentication/enrollment/recovery
records together. Preserve established factors and backup-code state.
Confirmation grants no session, MFA proof or role. A repeated confirmation is
unavailable; an authenticated lookup can separately report current verification.

### Change Password with Recent Authentication

Derive the subject from an actual session bearer; request fields cannot select
the account. Require current recent proof satisfying trusted operation policy.
Password-only proof is permitted only without established factors under a policy
allowing it. With established factors require at least recent MFA; the proposed
default is phishing-resistant MFA. A lower-assurance policy may explicitly permit
supported MFA, consistent with the delivered factor-management contract.

Validate/hash the new candidate through the existing policy and shared verifier.
Before commit, recheck actor expiry, generation, account version, policy, factor
bindings and proof age. Atomically replace the complete credential, increment
`AuthVersion` and invalidate every session, including the actor, and all pending
authentication/enrollment/recovery records. Preserve mailbox verification and
established factors. Return completion without a replacement session; the next
sign-in must satisfy current authentication policy.

### Request and Complete Password Reset

An eligible active account needs a previously verified current mailbox before
a reset token can be issued. An unverified account uses mailbox verification
first. A request cannot choose an alternate destination or establish verification.
Applications expose the same acknowledgment for existing, missing, inactive,
unverified and throttled account requests; internal outcomes remain classified.

Completion proves the reset token, validates/hashes the new password and rechecks
current token binding and eligibility inside the commit. Consume the token,
replace the complete credential, increment `AuthVersion` and invalidate every
session and pending authentication/enrollment/recovery record atomically.
Concurrent reset/change requests cannot overwrite a newer credential snapshot.

Reset preserves mailbox verification, WebAuthn/TOTP factors, accepted TOTP steps
and backup-code verifiers/consumption. It cannot activate an inactive account,
clear MFA policy, replace a factor or issue an authentication cookie. The user
signs in through the ordinary current-policy flow afterward. Loss of all required
factors remains unresolved even after a successful password reset.

An application-authorized operator may initiate this same reset request. Its
operator-facing response contains no bearer or chosen/generated password. Core
supplies no administrative password-replacement or MFA-bypass endpoint.

## Atomicity, Attempts and Failure Behavior

Use narrow typed storage operations for issue, reserve, confirm, password change,
reset and cleanup. Load owned snapshots, perform cryptographic work outside
database locks, then lock subject first and recheck records and trusted time in
the committing transaction. Separately committing password replacement and token
consumption does not satisfy this contract.

Before reset KDF work, durably charge a token attempt and the subject's completion
budget, and reserve one finite verification lease. Check the digest before password
checking/hashing. A wrong secret for a current ID spends an attempt; an unknown ID
does no KDF work and allocates no durable subject record. Oversized/malformed input
fails before storage. Policy rejection, verification failure, cancellation or
operating failure after reservation do not refund it. Lease expiry permits retry
within the remaining budget; it grants no authority. Password change uses the
same finite admission and recent-proof checks. Reissue and credential mutation
cannot reset attempt-window counters.

Failed final storage changes roll back credential, consumption, verification,
version, revocation and notification intent together. Charged attempts remain
spent. An ambiguous commit result returns an operating failure without claiming
rollback or automatically retrying a mutation. Replay after commit cannot produce
another completion or secret.

Internal outcomes distinguish accepted, unavailable token/state, denied proof,
password-policy rejection, exhausted budget, stale state/capacity and operating
failure. Public adapters group missing/wrong/expired/replaced/replayed tokens into
one unavailable response and redact storage errors. Results/events never contain
passwords, verifiers, token digests or raw recovery links.

Public handlers require finite ingress limits before account lookup and must
review acknowledgment timing as well as message/status differences. Subject/token
budgets are shared through durable adapter state. The broader distributed
IP/identity limiter and event-hook framework remain AUTH-07 work; process-local
ingress limits cannot be advertised as deployment-wide protection.

## Delivery and Transport Boundary

Return an issued bearer only to the trusted application mail-dispatch path after
commit; omit it from JSON, logs and operator responses. Mail failure cannot roll
back a committed token or revive its predecessor. Retry uses the still-valid token
only if retained securely by the application; otherwise explicit bounded reissue
replaces it. No unrestricted resend loop.

Successful verification, password change and reset persist one redacted
notification intent with their mutation through the application adapter. The
application owns durable dispatch, destinations, retry/retention limits and
operational failure reporting. Delivery failure cannot revert the security
mutation or require token replay. This is an adapter obligation, not a new core
mail worker or general outbox framework.

GET can render a form; it cannot consume a token or mutate security state.
Changes use explicit protected submissions. Applications use trusted HTTPS link
origins, bounded bodies, CSRF/origin protection, `no-store` and `no-referrer` on
token pages, and remove tokens from telemetry/navigation. No token or restricted
continuation resolves through ordinary session lookup.

## Selected Defaults and Bounds

These are approved toolkit bounds, not standards-mandated assurance claims.
Validate configuration at construction; reject unknown purposes, invalid ranges
and silent fallback. Trusted time is UTC with existing microsecond precision.

| Setting or input | Selected default and bound |
| --- | --- |
| Verification / reset lifetime | 24h / 1h; configurable 1m–24h / 1m–1h |
| Token size and syntax | Exactly 80 ASCII bytes; UUIDv4 plus 32-byte canonical secret |
| Canonical mailbox | At most 254 bytes; caller validates its identity policy |
| Current token rows | At most 2 per subject, one per purpose, including terminal rows |
| Token attempts | 5; configurable 1–10, durable before password work |
| Subject issuance budget | 3 per purpose per fixed 15m window; configurable 1–10 |
| Subject completion/change budget | 10 per fixed 15m window; configurable 1–20 |
| Recent password-change proof | At most 5m; trusted policy may tighten it |
| Lease / total operation timeout | 5s / 5s; configurable 1s–30s, lease must cover the operation timeout; earlier caller deadline wins |
| Work admission | Existing shared KDF concurrency bound; no additional unbounded queue |
| Completion body | At most 16 KiB before decoding; password uses existing policy limits |
| Policy revision | Nonempty trusted identity, at most 128 bytes |
| Terminal token retention / cleanup | 24h after consumption, revocation or expiry; explicit batches of at most 1000 rows, no owned loop |

Retained token capacity is fixed by subject/purpose slots; reissue replaces a slot.
Attempt state is at most three rows per subject: two issuance purposes and one
shared completion/change window. Rollover uses trusted time and atomic updates;
deletion/reissue cannot grant another budget in the same window. Unknown identities
use application ingress admission, not unbounded core rows.

## Acceptance Criteria

| ID | Observable evidence |
| --- | --- |
| AR-01 | Independent secrets/digests and exact subject/purpose/target/version/policy binding reject substitution; stored/public metadata has no bearer |
| AR-02 | Mailbox confirmation verifies only the current address, grants no authentication, and atomically consumes/increments/invalidates with factors preserved |
| AR-03 | Password change requires actual current recent proof; actor substitution, stale generation, lost factor binding and weaker management proof fail |
| AR-04 | Change/reset share signup's password policy and bounded verifier; policy/checker/hash failures change no credential |
| AR-05 | Reset preserves MFA/replay state, never auto-authenticates or activates, and cannot replace a required lost factor |
| AR-06 | Competing issue/consume/change/reset and forced failures establish one winner, complete revocation and rollback at the real PostgreSQL adapter |
| AR-07 | Exact expiry/lease/window boundaries, durable attempts, non-resetting budgets, finite admission and cleanup hold under concurrent/hostile requests |
| AR-08 | Production Ticked handlers establish GET safety, protected POST, non-enumerating initiation, unavailable-token grouping, cookie clearing and redacted failures |
| AR-09 | Mutations persist one notification intent; dispatch failures cannot restore tokens/sessions; operator initiation exposes no password/bearer |
| AR-10 | Consumers compile, documentation matches actual guarantees and the exact integrated delivery candidate passes the full repository gate |

Use deterministic time and rollback/fault tests, real adapter concurrency and
production HTTP/browser journeys. Fake storage or captured mail does not establish
deployment-wide anti-automation, actual provider delivery or identity proofing.

## References and Review Decisions

Sources inspected on 2026-10-04:

- [ASVS 5.0.0 V6](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x15-V6-Authentication.md): 6.4.3 covers reset without MFA bypass; 6.4.6 covers operator initiation without choosing a password. The all-factor-loss evidence in 6.4.4 is outside this bounded implementation.
- [NIST SP 800-63B-4, account recovery](https://pages.nist.gov/800-63-4/sp800-63b.html#account-recovery): recovery evidence, authenticator lifecycle and notification depend on required assurance. Mailbox possession does not establish phishing-resistant authentication or full recovery at a claimed AAL.
- [OWASP forgot-password guidance](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html): supports one-use expiring random tokens, consistent initiation responses, ordinary sign-in after reset and safe transport/delivery boundaries.

The concern/model and implementation scope were approved on 2026-10-04,
including the verified-mailbox prerequisite, verification-time session
invalidation, recent-proof change policy, no-session-retention result, token
representation, finite budgets and durable notification obligation. The internal
four-slice plan records exact branches/tasks/reports and activates Slice 1.
Approval authorizes this bounded set; it does not close broader AUTH-07 or
all-factor-loss recovery. Implementation status remains Pending until delivery.
