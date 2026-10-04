<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# auth

Password, UV WebAuthn and password/TOTP or backup authentication, bound step-up and restricted enrollment with explicit policy and finite bounds.

## Password policy

`NewPasswordPolicy` provides Unicode-aware credential policy with bounded input
and a caller-owned `PasswordChecker`. `Prepare` returns the accepted
NFC-normalized password for hashing. See the
[authentication reference](../docs/reference/authentication/README.md#password-policy)
for limits, errors and cancellation behavior. The service requires a checker,
applies policy at signup, and uses the versioned model verifier.

## Usage

```go
// Create service with your Queries implementation
svc, err := auth.NewService(queries, cfg, checker, log)
if err != nil { return err }

// Signup
user, err := svc.Signup(ctx, "user@example.com", "a distinct password phrase")

// Trusted operation policy, selected by server code.
required := auth.AccessRequirement{Proof: auth.RequirePassword, Revision: "password-v1"}
result, err := svc.Signin(ctx, "user@example.com", "a distinct password phrase", required)
if err != nil { return err }
session, completed := result.CompletedSession()
if !completed { return auth.ErrSessionProof }

// Validate session (e.g., in middleware)
validated, err := svc.ValidateSession(ctx, sessionToken, required, auth.RelevantActivity)

// Signout
svc.Signout(ctx, sessionToken)
```

## Sessions

`Signin` returns an `AuthenticationResult`. Only its completed outcome carries
`IssuedSession`: safe lifecycle metadata plus a canonical padded
URL Base64 token made from 32 random bytes. Send that token only in the issuance
response/cookie. `SessionRecord` stores its purpose-separated SHA-256 digest;
`Session` and `ValidatedSession` contain no bearer or digest. Validation returns
owned current user and session snapshots.

Absolute expiry defaults to 24 hours and inactivity to 30 minutes. Equality at
either expiry is expired. Storage validates active state, captured auth version
and current time after acquiring account/session locks. Password, role or
activation changes invalidate old sessions. Relevant activity writes at most
once per configured interval (one minute by default); it never renews absolute
expiry or authentication time. Coalescing may expire up to one interval early.
Choose `NoActivity` for polling/background traffic. The HTTP method alone does
not establish relevant activity.

`SessionSettings` validates lifetimes, cadence, cleanup batch and operation
timeout at construction; malformed settings fail instead of falling back.
See the [session reference](../docs/reference/authentication/README.md#session-lifecycle).

## Required proof

`AccessRequirement` carries trusted `Proof`, `Revision` and optional `MaxAge`.
Every sign-in/validation and auth middleware call requires it. Revisions are
nonempty printable ASCII of at most 128 bytes; applications own their lifecycle
and keep secrets/personal data out of them. `MaxAge` zero disables freshness;
otherwise it is 1 second through absolute lifetime in whole microseconds.
Exact freshness equality rejects access. Activity cannot renew proof time.

The password verifier produces `PasswordProof`; actual signed UV WebAuthn
assertions produce `WebAuthnProof` with factor ID/security revision and actual
verification time. Session metadata carries those facts plus
`PolicyRevision`. A policy mismatch or unmet stronger/recent proof fails before
touching activity. Strong profiles never become password-only access.

The password-only sign-in path returns pending enrollment for unmet MFA or proof according to the account's setup
state, with `AuthenticationMethodUnavailable`. Phishing-resistant MFA returns
denied/unavailable. Each has no session, secret, pending row or executable
continuation. Setup flags choose explanatory state only; they prove no factor.
Use `CompletedSession()` before setting a cookie. Metadata/result DTOs are not
proof receipts accepted by a completion API. See the
[proof reference](../docs/reference/authentication/README.md#required-proof-and-outcomes).

## Middleware

```go
// Require authentication
r.Use(auth.RequireAuth(svc, required, auth.RelevantActivity))

// Optional authentication (adds user to context if present)
r.Use(auth.OptionalAuth(svc, required, auth.NoActivity))

// Redirect for TOTP setup (does not verify factor proof)
r.Use(auth.RequireTOTP(auth.TOTPEnforcement{
    Enabled:   func() bool { return settingsSvc.GetBool(ctx, "security.require_2fa") },
    GraceDays: func() int { return settingsSvc.GetInt(ctx, "security.2fa_grace_period_days") },
    SetupURL:  "/settings/2fa",
}))
```

## User Model

```go
type User struct {
    ID             string
    Email          string
    PasswordHash   string
    AuthVersion    int64
    Roles          []string
    Active         bool
    TOTPSecret     string     // TOTP secret key
    TOTPEnabled    bool       // Whether 2FA is active
    TOTPVerifiedAt *time.Time // When 2FA was verified
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// Helper methods
user.HasRole("admin")
user.HasAnyRole("admin", "moderator")
user.NeedsTOTPSetup()
user.InTOTPGracePeriod(7)
```

## Context Helpers

```go
// In handlers (after middleware)
user, ok := auth.GetUser(r.Context())
userID, ok := auth.GetUserID(r.Context())
session, ok := auth.GetSession(r.Context())
```

## Queries Interface

```go
type Queries interface {
    CreateUser(ctx context.Context, id, email, passwordHash string, createdAt, updatedAt time.Time) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    GetUserByID(ctx context.Context, id string) (*User, error)
    CreateSession(ctx context.Context, state CredentialState, session SessionRecord, requirement AccessRequirement, limit int) (*Session, error)
    RotateSession(ctx context.Context, state CredentialState, current SessionDigest, generation int64, replacement SessionRecord, requirement AccessRequirement) (*Session, error)
    ListSessions(ctx context.Context, actor SessionDigest, requirement AccessRequirement, limit int, cursor string) (*SessionPage, error)
    RevokeSessions(ctx context.Context, actor SessionDigest, requirement AccessRequirement, selection SessionSelection) (int64, error)
    ReplacePassword(ctx context.Context, state CredentialState, passwordHash string, changedAt time.Time) (*User, error)
    ValidateSession(ctx context.Context, digest SessionDigest, requirement AccessRequirement, activity SessionActivity, interval time.Duration) (*ValidatedSession, error)
    DeleteSession(ctx context.Context, digest SessionDigest) error
    DeleteExpiredSessions(ctx context.Context, limit int) (int64, error)
}
```

Storage returns owned snapshots with a positive `AuthVersion`. Every credential,
activation and security-state mutation advances that version. `CreateUser`
enforces email uniqueness and returns `ErrEmailTaken` for duplicates.
`CreateSession` checks active state, version, policy and proof at locked current
time before insertion. It reclaims bounded expired subject rows and enforces the
retained-row cap under the same subject lock.
`ReplacePassword` atomically checks state, increments the version, replaces the
complete credential and revokes sessions. Stale, inactive or missing state returns
`ErrCredentialChanged`. See the [storage reference](../docs/reference/authentication/README.md#queries).

The caller owns authorization and candidate preparation for `ReplacePassword`;
this storage contract does not supply a password-reset or recovery workflow.

## TOTP and Backup Authentication

Construct `NewFallbackService(credentials, fallbackQueries, config.FallbackConfig,
SeedKeys)` with explicit application-owned encryption keys. Use `BeginTOTPSetup`
and `ConfirmTOTPSetup` only for initial setup; confirmation consumes the step and
revokes sessions without issuing access. Begin password/TOTP or password/code
with `BeginFallbackAuthentication`, or bind a current actor with
`BeginFallbackStepUp`. `FinishFallback` verifies actual proof and consumes the
step/code with session completion in one transaction.

These methods satisfy MFA, never phishing-resistant MFA. `IssueBackupCodes`
requires actual recent management proof and returns codes once plus a rotated
actor only after commit. See the [fallback reference](../docs/reference/authentication/README.md#totp-and-backup-proof).

Low-level `crypto.MatchTOTPCode(secret, code, trustedTime, skew)` returns a matched
integer step. The caller must atomically consume that step; primitive verification
alone grants no session. `crypto.NewBackupSecret` and `ParseBackupCode` provide
canonical secrets/identifiers; hashing and durable one-use completion belong to
the bounded service. The old boolean TOTP and shared-salt/index backup APIs are
removed.

## Reauthentication and Session Management

`Reauthenticate(ctx, token, password, required)` verifies the password again for
a live current session. Proof need not already be recent; revision, supported
method, current account and both expiries must still hold. Rotation preserves ID
and creation time, replaces the bearer and advances generation atomically.
Set the replacement cookie only after `result.CompletedSession()` succeeds.
Failed verification or transaction work returns no secret and preserves the old
session. An expired session requires sign-in; password cannot satisfy MFA.

`ListSessions(ctx, token, required, cursor)` returns a `SessionPage` of safe
metadata with a current ID and opaque next cursor. `RevokeSessions(ctx, token,
required, SessionSelection{Scope: SessionOthers})` revokes other sessions.
Scopes are current, selected (with ID), others and all. Both operations derive
the subject from the bearer and recheck recent proof under locks. They accept
no target subject or administrative authority. Applications own cross-subject
administrative authorization and its integration.

Management defaults to a 5-minute proof age, tightened by any shorter operation
requirement. Capacity is 10 retained sessions per subject; page size is 50.
Configuration bounds and required adapter transactions are in the
[authentication reference](../docs/reference/authentication/README.md).

## Restricted WebAuthn Enrollment

`NewAuthenticatorService(credentials, authenticatorQueries, cfg.Authenticator)`
requires the existing credential service, mandatory atomic enrollment storage and
explicit trusted RP configuration. It shares the credential verifier's KDF budget.
`BeginWebAuthnEnrollment` verifies a fresh actual password and admits initial setup
only while the active account has no established factor. Its `enroll1.` token is
independent of ordinary session tokens and cannot access application routes.
`FinishWebAuthnEnrollment` verifies a bounded ES256 registration with required
presence/UV through go-webauthn v0.18.2, then atomically activates the key, advances
account version and deletes earlier sessions/pending setup. It returns safe factor
metadata, never an issued session. Assertion sign-in is not yet implemented.

See the [enrollment reference](../docs/reference/authentication/README.md#restricted-webauthn-enrollment)
for configuration, required atomic operations, parser bounds and Ticked endpoints.


## WebAuthn Assertion Completion

`NewWebAuthnService` requires the base credential service, `WebAuthnQueries`
(including enrollment storage) and validated RP configuration. Begin subject-first
sign-in with `BeginWebAuthnAuthentication`; begin bound actor rotation with
`BeginWebAuthnStepUp`. Neither restricted challenge grants application access.
`FinishWebAuthn` validates the real signature/UV and atomically consumes replay
state and pending with insertion/rotation. Set the cookie only from a committed
`IssuedSession`. Password reauthentication produces password proof only.

See the [WebAuthn reference](../docs/reference/authentication/README.md#webauthn-authentication-and-step-up)
for required storage, bounds, counter/backup semantics and Ticked endpoints.

Established-factor management uses `FactorService` with mandatory `FactorQueries`
and a trusted `FactorPolicy`. Safe finite listings, additional enrollment,
replacement and removal recheck recent proof, exact ownership/revision and current
usable-factor policy under the subject lock. `change1.` continuation tokens are
separate from initial setup and sessions. Actual confirmation commits activation,
version advancement, all-session/pending revocation and optional bearer rotation.
The retained proof keeps its original times; removing its constituent revokes the
actor. TOTP changes require an explicitly supplied fallback service/key ring.

## Mailbox verification

`NewRecoveryService(base, recoveryQueries, cfg.Recovery, "mailbox-v1")` composes
with the existing credential service. `RequestMailboxVerification` returns a
transient `MailboxIssue` for trusted mail dispatch after storage commit. Its
`Token.Bearer()` must not appear in public JSON, logs or operator responses.
`ConfirmMailboxVerification` verifies the current address and revokes prior
sessions/continuations without authenticating. `CleanupMailboxTokens` performs
one bounded retained-token batch and starts no worker.

The adapter must serialize through the subject before token/budget locks,
durably charge failed attempts, recheck current identity/version/policy/time,
and commit verification, consumption, version advancement, all-session and
continuation invalidation plus notification intent together. Existing MFA and
replay records remain unchanged. See the
[mailbox contract](../docs/reference/authentication/README.md#mailbox-verification).
Password change and reset are not exposed by this service yet.
