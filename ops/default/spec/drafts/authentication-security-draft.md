<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Authentication Security Foundation

Date: 2026-10-02
Status: Draft for contract and compatibility review
Implementation status: Proposed; existing behavior remains unchanged
Inspected baseline: `dev`, `67023146f984b984509d32f176ea8522ddf9a301`
Model: [Authentication security model](authentication-security-draft-model.md)

## Purpose and Ownership

Provide reusable authentication contracts that let downstream applications meet
strong security requirements without duplicating HatMax's credential/session
service. A consumer's requirement can justify improving core; existing core
behavior is evidence to inspect, not a ceiling on the consumer's security.

Core owns credential policy and verification, authenticator verification,
challenge/session semantics, secure secret generation, generic recovery and
atomic storage contracts. Applications own concrete persistence adapters,
enrollment/recovery presentation, mail delivery, selected authentication policy,
and their domain authorization. Organizations, workspace roles and subscriptions
remain application domains.

This draft proposes contracts and acceptance evidence. Exact Go interfaces,
compatibility changes and delivery units require review before implementation.

## References and Verification Target

- [OWASP ASVS 5.0.0](https://github.com/OWASP/ASVS/tree/v5.0.0/5.0): versioned
  application-control references below, including higher-assurance authentication.
- [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html): password,
  authenticator, recovery and session guidance. Refer to specific sections rather
  than claiming an assurance level from an algorithm name.
- [WebAuthn Level 2](https://www.w3.org/TR/webauthn-2/): registration/assertion
  protocol baseline; choose the supported protocol/library version during design.
- [OWASP password storage guidance](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).
- [UUIDv4 format](https://www.rfc-editor.org/rfc/rfc9562.html#section-5.4).

These references define verifiable requirements, not a certification claim.
Application ASVS level and NIST assurance depend on configuration, authenticators,
deployment, lifecycle and evidence beyond this toolkit. No new dependency or
cryptographic implementation is selected by this document.

Separate mechanism quality from application access policy. Support validated
policies that permit ordinary single-factor access and policies that require
phishing-resistant MFA for every access. Required proof is checked per operation;
role or policy changes cannot silently retain weaker privileged access. The
application selects defaults and classifies sensitive operations. Supporting
either policy does not establish an ASVS level or NIST AAL for the toolkit.

## Inspected Gaps

| Gap | Source evidence | Consequence for the contract |
| --- | --- | --- |
| Password policy | `config/config.go` defaults minimum length to 8; `auth/service.go` counts bytes and has no weak/breached-password check | Explicit Unicode-aware policy and a blocklist boundary are required |
| Password representation | `model/password.go` is bcrypt-only; its 72-byte cap prevents accepting every 64-character Unicode password | Review a versioned storage strategy; do not truncate or restrict the consumer to that legacy limit |
| MFA sign-in | `auth.Service.Signin` creates a full session immediately after password verification; `RequireTOTP` checks enrollment | Distinguish incomplete authentication from an authorized application session |
| Session bearer entropy | Sign-in uses `model.NewID`, which generates UUIDv4 with 122 random bits | Use an independent CSPRNG bearer secret; keep UUIDs for record identity |
| Session lifecycle | Only creation/absolute expiry are modeled; malformed duration falls back to 24 hours; expiry uses `Before` | Validate duration, enforce equality boundary, inactivity/re-authentication, rotation and revocation |
| Backup-code storage | `crypto/totp.go` hashes short codes using a shared literal salt; verification returns an index without consumption | Random salt per stored code and atomic one-use semantics are required |
| Phishing resistance | No WebAuthn registration/assertion contract in inspected auth | Add a reusable phishing-resistant authenticator boundary; TOTP is not equivalent |
| Recovery and uniqueness | No general password-update/all-session-revocation flow; signup uses a lookup before insertion | Transactional recovery and classified concurrent duplicate outcomes are required |
| Attempt controls | IP limiter is process-local and owns an unjoined cleanup loop | Account/challenge budgets and resource lifecycle need explicit boundaries |

Existing Argon2 primitives in `crypto/crypto.go` do not supply an encoded password
record, configurable resource envelope or migration policy. Their presence alone
does not satisfy the new password-storage contract.

## Proposed Core Requirements

The identifiers below name requirements, not slices. All are proposed until
this specification and its model are accepted.

### AUTH-01 — Credential policy

Support a documented password policy suitable for the selected authentication
profile: a 15-character minimum when password-only access is permitted, with a
shorter minimum of at least 8 only where MFA is always enforced. Permit at least
64 Unicode characters and count code points for policy length. Configure finite
character/byte limits to bound work; reject invalid input without truncation.
Use identical processing at registration, change, reset and verification.

Check complete candidate passwords against common, compromised and application-
specific disallowed values. Provide a narrow caller-supplied check boundary,
with deadline/error semantics; failures cannot silently approve the password.
Do not prescribe an external service, character-composition rules or periodic
rotation. Support password managers, paste and clear policy feedback.

### AUTH-02 — Stored credential format and migration

Use a self-describing, salted password record with bounded hashing parameters.
Argon2id is the preferred direction for new records using a standard encoded
format. Finalize algorithm support and resource settings through measurement
and the consumer's cryptographic requirements; FIPS-dependent deployments need
an appropriate validated implementation/profile. No custom password prehash
scheme is selected.

Existing bcrypt records require an explicit verification and upgrade path.
Upgrade a successfully verified legacy record atomically when safe. A failed
opportunistic upgrade must not by itself reject otherwise valid authentication;
concurrent credential/account changes still require fresh state checks. Never
write a partial replacement. Do not
promise a NIST/FIPS profile merely because Argon2id or bcrypt is supported.
Hashing parameters from a stored record are untrusted input and must be bounded
before allocating memory or performing expensive work.

### AUTH-03 — Authentication outcome and required proof

Represent successful full authentication separately from pending enrollment,
pending second factor, denied proof and operating failure. Pending records cannot
be resolved through the ordinary session validator. The caller supplies required
proof for the target operation; a weaker method does not silently satisfy it.

Store full-proof time and verified method properties for step-up decisions.
Recheck account activation and current authenticator state on completion.
Factor verification, one-time consumption and session creation share an atomic
storage operation. Define factor enrollment and factor-change revocation with
the same state boundaries rather than post-login redirects.

### AUTH-04 — Sessions

Generate opaque session secrets using a CSPRNG with at least 128 random bits;
the proposed default is 32 random bytes through the existing secure-token helper.
Record IDs remain separate UUIDs. Store/lookup digests through caller adapters
without requiring plaintext database bearer tokens.

Validate positive absolute/inactivity durations at startup. Reject use at or
after expiry, observe inactivity policy and rotate tokens after authentication
and reauthentication. Expose narrow validated session metadata needed by callers;
the current auth middleware supplies only user/ID context.

Support current/all-session revocation, user-visible session management and
administrative revocation. Disabled accounts cannot regain access by reactivation
of an old still-valid token. Define concurrent-session limits and time updates
without an unbounded session collection or a write on every request by default.

### AUTH-05 — Authenticators

Support WebAuthn registration and assertion verification through a reviewed
standard implementation. Validate challenge, origin, relying-party identity,
credential binding, user presence/verification and supported algorithm policy.
Registration and assertions have one-use, expiring challenges. Stored counters
and backup flags are handled according to supported authenticator semantics.
For a policy requiring phishing-resistant MFA, require verified user verification
or the supported combination with an independent factor. User presence or a
passkey label alone does not establish multi-factor proof.

Distinguish phishing-resistant verified methods, hardware/non-exportable key
properties where demonstrable, and syncable credentials. Do not infer NIST AAL3
from the word passkey or a client-provided assurance flag. The application
selects accepted authenticator classes and routes requiring stronger proof.

Keep TOTP support with an explicit lower-assurance role where policy allows it.
Verification uses trusted server time and defines replay/step consumption.
Backup codes are random, purpose-bound, individually salted and one-time.
Their use cannot satisfy a route that requires phishing-resistant proof.

### AUTH-06 — Lifecycle and recovery

Provide generic mailbox-verification, password-change/reset and authenticator
management contracts through narrow caller-owned storage interfaces. Tokens
bind subject, purpose, target and expiry; reissue/consumption/revocation have
concurrent-safe outcomes. GET is not a state-changing recovery operation.

Password reset does not disable stronger factors or automatically authenticate
the account. Recovery cannot silently restore a higher assurance than its actual
evidence supports. Protect enrollment/removal with the required recent proof;
support safe recovery notifications through application-owned delivery.

Email ownership verification is distinct from authenticating an application
session. Platform administrators can initiate the supported recovery process
without selecting or learning the user's password. Exact all-factor-loss and
operator-assisted recovery contracts require threat and identity-proofing review.

### AUTH-07 — Errors, attempts and resource bounds

Distinguish policy rejection, invalid proof, expiry/replay, exhausted budget,
uniqueness conflict and operating failure without exposing secrets. Map a
concurrent duplicate signup into a stable classified outcome, not raw SQL text.
Let application handlers produce non-enumerating account responses.

Enforce per-subject/challenge attempt budgets atomically and provide bounded
anti-automation integration. Document distributed versus process-local guarantees,
retry/reset semantics and cleanup. Any owned loop has cancellation and shutdown.
Authentication work has explicit size, timeout, memory and concurrency limits.

Security-event hooks carry bounded, redacted metadata. They cannot carry bearer
secrets, passwords, TOTP material or raw recovery links. The application owns
destinations, retention and contextual risk policy.

## Initial Standards Trace

This is an initial mapping, not a complete ASVS assessment. References identify
controls in the pinned 5.0.0 release; descriptions are summaries of this draft's
work, not copied control text.

| Core requirement | ASVS reference | Acceptance evidence to design |
| --- | --- | --- |
| AUTH-01 | v5.0.0-6.2.1, v5.0.0-6.2.4, v5.0.0-6.2.9, v5.0.0-6.2.12 | Unicode lengths, long passwords and disallowed candidate cases |
| AUTH-02 | v5.0.0-11.4.2 | Versioned records, distinct salts, bounded parsing and legacy upgrade |
| AUTH-03/AUTH-05 | v5.0.0-6.3.3 | Incomplete and weak proof cannot enter strong-profile routes |
| AUTH-04 | v5.0.0-7.2.3, v5.0.0-7.2.4, v5.0.0-7.3.1, v5.0.0-7.3.2 | Secret entropy, rotation and absolute/inactivity boundaries |
| AUTH-04/AUTH-06 | v5.0.0-7.4.1, v5.0.0-7.4.2, v5.0.0-7.5.1, v5.0.0-7.5.2 | Revocation, reactivation and recent-proof transitions |
| AUTH-05 | v5.0.0-6.5.1, v5.0.0-6.5.2 | Concurrent replay rejection and per-code salt properties |
| AUTH-06 | v5.0.0-6.4.3, v5.0.0-6.4.4, v5.0.0-6.4.6 | Recovery preserves factor/assurance policy |
| AUTH-07 | v5.0.0-6.1.1, v5.0.0-6.3.1, v5.0.0-6.3.8 | Durable budgets, non-enumerating results and resource cleanup |

Threat review covers credential stuffing/enumeration, phishing, factor/recovery
replay, session theft, record disclosure, malicious hash parameters, resource
exhaustion and concurrent security changes. Include deterministic time tests,
real adapter transactions, race/fuzz tests where material, browser authenticator
journeys and measured KDF cost under bounded concurrency.

## Compatibility and Consumer Integration

Review affected auth/model/crypto/config APIs, examples, public documentation and
generator output together. Existing passwords, sessions and backup codes each
need an explicit migration/invalidation decision; incompatible security formats
must not be accepted indefinitely by an undocumented fallback.

Consumers implement persistence according to the agreed atomicity contract.
Core must not require a private application package or ship domain-specific
tables. Publish the compatible core dependency before validating consumers
independently of local workspace overrides.

## Decisions Before Promotion

Finalize the credential policy/KDF profile and migration behavior, authenticator
classes and validation dependency, generic transaction interface, authentication
result/session metadata, recovery proof and record formats. Set timeout/attempt/
concurrency/retention budgets and identify behavior requiring compatibility release
notes. Do not treat all consumers as using the same application assurance policy.

After acceptance, divide work into bounded credential, session, authenticator and
recovery delivery units with consumer integration evidence. This draft is a core
prerequisite proposal, not an implementation plan or a completed correction.
