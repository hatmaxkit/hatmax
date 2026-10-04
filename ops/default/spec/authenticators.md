<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authenticators

Date: 2026-10-04
Status: Approved
Approved: 2026-10-04
Concern: authenticators
Parent: [Authentication security foundation](authentication-security.md)
Model: [Authenticator model](authenticators-model.md)
Plan: [Delivery proposal](../plan/authenticators.md)
Tracker: [Delivery tracker](../tracker/authenticators.md)
Inspected baseline: `247487f273a54a8e65a8e6b5f86191f38bcb0be8`
Implementation status: Slices 1-4 delivered; Slice 5 active
Execution gate: Open

## Purpose and Ownership

Implement the already approved AUTH-05 mechanisms and complete AUTH-03's actual
factor-verification/session-completion boundary. Provide WebAuthn registration
and assertions, TOTP with replay state and individually salted one-use backup
codes. Extend the delivered session contracts rather than introducing another
authentication engine.

Core owns protocol verification, proof evaluation, challenge orchestration and
required atomic storage contracts. Applications own trusted RP/origin and access
policy, encryption keys, persistence adapters, HTTP/browser presentation and
domain authorization. Core has no application-product or tenant dependency.
Account recovery and operator-assisted all-factor loss retain AUTH-06 ownership;
the mechanisms here cannot silently reset established stronger factors.

The concern/model and five-slice delivery plan were approved on 2026-10-04.
Slices 1-4 are delivered after verified merges. Slice 5 is active under the recorded plan.

## Delivered Prerequisites and Inspected Gaps

The [credential tracker](../tracker/credential-security.md) and
[session tracker](../tracker/authentication-sessions.md) record completed
sets and exact integrated gates. Their checks are not repeated for this proposal.

- `auth/proof.go` accepts only actual password proof; stronger requirements fail
  closed and current pending outcomes contain no continuation or stored record.
- `auth/models.go` still exposes TOTP setup fields; `RequireTOTP` checks enrollment
  rather than verified factor completion. Neither supplies MFA authority.
- `crypto/totp.go` uses `pquerna/otp` v1.5.0, but its Boolean verification has no
  durable accepted step. Backup codes use shared-salt hashing and index matching
  without atomic consumption. Replace those active contracts together.
- Required `auth.Queries` and the Ticked adapter already serialize authentication
  state through the subject row. Session creation/rotation check account version,
  policy, expiry and capacity under locks. Extend that boundary for real factors.
- No WebAuthn dependency, authenticator records or executable pending table exists.

## Proposed Protocol Profile

Use `github.com/go-webauthn/webauthn` v0.18.2 as the proposed reviewed protocol
dependency. Use its parsed-data validation APIs behind the owning core adapter;
core service methods do not accept HTTP requests or expose a caller-supplied
proof-approval callback. Persist bounded ceremony data returned by its begin
operation, including requested options; do not reconstruct it from client input.
Pin the actual module/transitive dependencies and inspect license/security
evidence before the first implementation task changes `go.mod`.

The proposed profile pins WebAuthn Level 3 Recommendation of 2026-08-25, updating
the parent's Level 2 design baseline for verified backup flags and current library
semantics. The required registration/assertion checks remain protocol-owned.
No new cryptographic implementation is selected.

Trusted configuration supplies one RP ID, an exact finite origin allowlist and
policy revision. Require HTTPS except explicitly configured localhost development;
derive neither RP ID nor accepted origin from request headers. Cross-origin and
embedded ceremonies are rejected in this profile. Registration advertises only
ES256 initially, resident keys are required, user verification is required and
attestation preference is `none`. Assertion requires verified presence and user
verification. Accounts use a stable opaque random 32-byte WebAuthn user handle,
never an email address. Initial login is subject-first; discoverable usernameless
login requires a separately reviewed admission/identity-lookup contract.

The proposed general authenticator class accepts both backup-eligible and
non-backup-eligible credentials. Record verified eligibility/state, validate
their relationship and enforce an immutable eligibility bit. A strict counter
profile rejects a non-increasing nonzero counter; zero/zero is supported.
Handle concurrent counter changes under the same credential lock as completion.
Failing this policy reports denied proof; it does not claim a confirmed clone.
Without trusted attestation evidence, a credential is not labeled hardware-bound
or non-exportable. No NIST AAL or complete ASVS level is inferred.

## Proof and Policy

Keep `AccessRequirement` as trusted operation policy. Replace the password-only
proof representation with closed core-produced facts sufficient to distinguish:

| Actual completed evidence | Permitted authority |
| --- | --- |
| Password | Password-only policy |
| Fresh password plus TOTP | MFA where the trusted policy permits TOTP; never phishing resistance |
| Fresh password plus one-use backup code | MFA where the trusted policy explicitly permits that fallback; never phishing resistance |
| WebAuthn assertion with verified presence and user verification | MFA and phishing-resistant MFA within the supported authenticator class |

Stronger supported combinations also satisfy ordinary password-permitted access;
that profile is a minimum required proof, not a demand to repeat a password when
an actual stronger authenticator has completed.

All methods have actual verification times and current factor bindings. A
composite proof's freshness starts at its oldest required constituent. Password
activity or password reauthentication cannot refresh an earlier stronger fact.
Stored metadata is safe to return but cannot be submitted as a completion token.
Current policy can require factor completion for an enrolled account even when
its general ordinary-user profile permits unenrolled password access. Enrollment
presence never proves that requirement was satisfied.

## Restricted Continuations and Completion

After real password verification, unmet supported policy can issue a bounded
continuation with its own independent 32-byte secret and purpose-separated digest.
Purposes are finite: initial authentication, reauthentication and enrollment.
Enrollment explicitly distinguishes initial setup from established-factor changes.
An enrollment continuation is permission for that restricted setup
operation only. No ordinary session validator or middleware resolves it.
Unsupported policy/class returns no executable continuation.

Bind pending state to subject, captured `AuthVersion`, trusted policy revision,
required proof, already verified facts, intended protocol and exact target.
Reauthentication and established-factor enrollment also bind the current session
ID, digest and generation.
Allowed methods/options are fixed by the server before returning a challenge;
request fields cannot select a weaker proof or another subject.

Before protocol/KDF work, reserve a durable attempt against both pending and
subject factor budgets. Only one bounded verification lease is active per pending
record. Attempts are never refunded after verification or operating failure;
lease expiry permits retry within remaining budget. A typed internal operation
reservation binds the loaded record/revision to the final operation, without a
public approval API. Malformed bodies fail before parsing; valid continuation
submissions with invalid protocol proof spend an attempt.

After actual core verification, storage locks and rechecks current state and
trusted time. Consume pending state and factor replay/counter state together with
exactly one admitted session insertion or generation rotation. No raw secret is
returned before commit. Replay, expiry equality, password/account/policy change,
factor revocation or competing completion cannot issue another session.
Storage failure rolls back consumption and session change; the prior attempt
reservation remains spent. A new attempt performs verification again.

## Enrollment and Authenticator Changes

Initial enrollment requires fresh actual password proof, an active current
account and no established factor. Its restricted continuation cannot access
ordinary application routes. Confirm WebAuthn registration through the protocol
verifier or TOTP setup through a valid code before activating the record.
Confirmation consumes setup, advances `AuthVersion` and invalidates other
sessions/pending state atomically. Registration alone does not issue a strong
session; the account performs an actual assertion/sign-in afterward.

An account with established factors must present recent proof satisfying trusted
management policy for additional enrollment, replacement, removal or backup-set
regeneration. The proposed default is phishing-resistant MFA; a caller may
explicitly select permitted MFA for a lower-assurance profile, including upgrading
a TOTP-only account. Password-only management of established factors is rejected.
Application-provided roles and client assurance claims cannot authorize these
changes. Such mutations advance version and revoke other sessions/pending state;
when retaining the actor, rotate its bearer atomically and bind its still-current
verified proof to the new state. It must not retain proof of a removed factor.
Reject removing the last usable authenticator or a proof constituent needed by
current policy. Loss of every qualifying factor requires the AUTH-06 recovery
contract, rather than a password-only management bypass.

TOTP seeds remain encrypted through authenticated encryption with externally
supplied keys and explicit key identity. No secret enters user/session snapshots.
Use existing `pquerna/otp` with SHA-1, six digits, 30-second steps and skew at most
one. Verification identifies the exact matched step from trusted time; commit
rechecks that it is still in-window and strictly above the accepted step. Setup
consumes its first accepted step, preventing immediate replay as authentication.

Generate eight backup codes by default. Each code has a non-secret record ID and
an independent random 128-bit secret, displayed once after a committed set change.
The input has one canonical bounded ASCII representation. Use the existing
bounded Argon2id verifier with an independent salt per record and an input binding
code purpose, subject and record ID. Direct subject/ID lookup avoids trying a KDF
against every code. Consume the code under the factor lock with completion;
regeneration invalidates the old set atomically. It cannot weaken stronger policy.

## Proposed Defaults and Bounds

All are proposed toolkit bounds, not standards-mandated assurance profiles.
Validate construction and reject unknown enums, malformed state and silent fallback.

| Setting or input | Default and bound |
| --- | --- |
| Pending lifetime | 5m; configurable 1m through 10m; exact equality expires |
| Recent enrollment/management proof | 5m; operation policy may tighten existing session bounds |
| Pending retained rows per subject | 5; configurable 1 through 10; reject capacity without eviction |
| Active authenticators per subject | 10; configurable 1 through 20, including unconfirmed retained setup |
| Pending attempts | 5; configurable 1 through 10; durable before verification |
| Subject factor attempts | 10 per 15m window and 15m cooldown; finite server policy, reissue cannot reset it |
| Verification lease and total operation timeout | 5s, at most 30s; earlier caller deadline wins |
| Protocol work admission | 2 active verifications per service; no waiting queue; credential KDF retains its existing shared limit |
| RP origins | At most 8 exact trusted origins |
| Ceremony request body | At most 64 KiB before JSON/CBOR parsing |
| Client data / attestation object / COSE key / credential ID | 8 KiB / 32 KiB / 4 KiB / 1024 bytes; structural/depth bounds before allocation |
| Stored ceremony data | At most 16 KiB; exact versioned library representation |
| Backup codes per set | 8; configurable 1 through 10; code input at most 128 ASCII bytes |
| Safe factor list | At most 20 records; no secret/public proof-assertion authority |
| Pending cleanup | Existing explicit batch contract, at most 1000 rows; no owned loop |

Reclaim bounded expired subject pending/setup rows before admission; count all
retained rows under the subject lock. Cleanup removes expired records without
waiting for a subject lock afterward. The subject-budget row is one bounded row
per account and expires logically; reissuing pending state does not reset it.
These budgets protect the actual endpoints delivered here. Broader distributed
password/IP anti-automation and generic event hooks retain AUTH-07 ownership.

## Acceptance Criteria

| ID | Observable evidence |
| --- | --- |
| AU-01 | Real library registration/assertion validates challenge, RP, origin, signature, subject binding, algorithm and UP/UV; invalid variants issue nothing |
| AU-02 | Verified password, password/TOTP, password/backup and UV WebAuthn have the documented proof/freshness properties; weaker facts cannot satisfy stronger policy |
| AU-03 | Distinct pending namespaces/purposes never authorize application access; captured account/policy/session changes reject completion |
| AU-04 | Competing completions have one winner; factor counter/step/code and pending consumption share insertion/rotation commit; forced failures roll back durable completion |
| AU-05 | Encrypted TOTP storage, exact step/time/skew checks and concurrent replay rejection hold at the real adapter boundary |
| AU-06 | Backup records have independent salts and bounded KDF work; one-use/regeneration and denial under stronger policy hold under concurrency |
| AU-07 | Initial setup is restricted; established-factor changes require current recent strong proof, advance version and invalidate/rotate atomically |
| AU-08 | Retained-row, attempt, admission, body/parser, timeout and cleanup bounds hold under hostile/concurrent requests; reissue cannot reset subject budget |
| AU-09 | Real Ticked browser journeys and PostgreSQL transactions establish enrollment, MFA completion and step-up; pending/failure never gets an application cookie |
| AU-10 | Current session/credential consumers compile; documentation matches actual supported guarantees; exact integrated full gate passes |

Browser evidence must use the production begin/finish handlers with a real browser
and virtual authenticator driving actual library verification. Signed protocol
fixtures demonstrate verifier rejection and transaction behavior; they do not
replace browser evidence or establish hardware certification. TOTP/backup paths
use real code/KDF verification, not caller-asserted proof fixtures.

## References and Review Decisions

Sources inspected on 2026-10-04:

- [WebAuthn Level 3 Recommendation, 2026-08-25](https://www.w3.org/TR/2026/REC-webauthn-3-20260825/): registration/assertion and backup-state/counter semantics. Proposed protocol pin replaces the parent's earlier baseline for this concern.
- [go-webauthn v0.18.2](https://github.com/go-webauthn/webauthn/releases/tag/v0.18.2) and [validation APIs](https://github.com/go-webauthn/webauthn/blob/v0.18.2/webauthn/login.go): concrete candidate, not a certification claim.
- [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html): OTP and look-up-secret replay resistance, authenticator binding and assurance boundaries.
- [RFC 6238](https://www.rfc-editor.org/rfc/rfc6238.html) and [pquerna/otp v1.5.0](https://github.com/pquerna/otp/blob/v1.5.0/totp/totp.go): existing TOTP implementation and trusted time-step evaluation.
- [ASVS 5.0.0 V6](https://github.com/OWASP/ASVS/blob/v5.0.0/5.0/en/0x15-V6-Authentication.md): parent AUTH-03/AUTH-05 verification target; a bounded set is not a complete assurance assessment.

Review the concrete dependency/protocol pin, UV-required subject-first profile,
accepted backup eligibility/counter policy, initial-versus-established enrollment
authority and management policy, backup fallback and proposed finite budgets
together with the model.
The behavior list comes from AUTH-05; review resolves its implementation choices.
Approval on 2026-10-04 promotes this concern/model together and activates
Slice 1 of the five-slice delivery plan. The selected profile and finite bounds
are implementation contracts; material changes require review.
