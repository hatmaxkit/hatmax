<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 3: Mailbox Password Reset

Status: reviewing
Delivery set: account-recovery
Plan: `ops/default/plan/account-recovery.md`
Tracker: `ops/default/tracker/account-recovery.md`
Branch: `feat/account-password-reset`
PR: `#105`

## Purpose

Reset the complete password through a one-use token mailed to the previously
verified current address, with atomic revocation and retained MFA. This is Slice
3 of 4 in the approved [plan](../../../plan/account-recovery.md) and
[tracker](../../../tracker/account-recovery.md).

## Delivered Behavior

- Only an active account with a previously verified current mailbox can receive
  a reset token. Exact purpose, subject, address, version, revision and expiry
  bind the record; reissue invalidates its predecessor without refunding budgets.
- Actual secret/purpose reservation commits finite attempts before shared
  password policy/KDF work outside database locks. Rejected candidates, checker
  failures, cancellation and final transaction failures do not refund admission.
- One subject-first transaction rechecks current state, the complete owned lease
  and database time after waits and writes. It replaces the full credential,
  consumes the token, advances version once, deletes every session/restricted
  continuation, revokes other mailbox slots and inserts a kind-3 notice together.
- Activation, roles, address verification, confirmed WebAuthn/TOTP factors,
  accepted replay state and backup verifiers/consumption remain intact. Completion
  returns safe subject/time only; normal current-policy sign-in is still required.
- Production Ticked initiation is neutral and bounded. Completion requires
  explicit protected POST and clears the cookie without granting a new session.
  An operator with current application role and actual recent MFA can initiate
  the same mail-only flow, without choosing a password or receiving the bearer.
- Failed notification delivery retains successful security state and durable
  intent for the existing explicit bounded retry, without a worker.

## Implementation Notes

T3.1: `cd3fbcb6f893`; T3.2: `8c3f9cb54565`.

Reset uses purpose 2 and the configured reset TTL in existing migration 008;
kind-3 completion admission is shared with verification and password change.
Token admission precedes candidate checking, and final completion uses the exact
captured revision/lease rather than caller authority. Competing reset/change and
reissue operations cannot overwrite a newer credential snapshot. Post-write
expiry and injected write failures roll back every security effect; attempts
remain committed. Ambiguous commit failures are not proof of rollback.

Ticked uses its existing explicit trusted recovery-origin and active-mail
configuration. Operator roles remain application-owned; assembly selects actual
phishing-resistant MFA no older than five minutes. The trusted handler constructor
can explicitly permit supported MFA, never password-only operator proof. No new
migration, dependency, external mail send or retry worker was added.

## Contracts Added or Changed

`RecoveryQueries.IssuePasswordReset`, `ReservePasswordReset` and
`CompletePasswordReset` extend the typed adapter contract. `RecoveryService`
exposes `RequestPasswordReset` for trusted transient mail dispatch and
`ResetPassword` for one-use completion. `PasswordReset` carries safe subject/time
only. Shared mailbox record checks now select the correct purpose TTL.

`PasswordResetHandler` owns GET/POST `/account/password/reset`,
`/account/password/reset/confirm` and `/admin/password-reset`. Initiation accepts
only `email`; completion accepts only `token` and `password`, no query fields,
16 KiB body, 80-byte token and 4096 complete candidate bytes. Same-origin,
cache/referrer and finite admission protections apply before account lookup.
Neutral initiation targets six seconds with a five-second work deadline.

`MailboxDelivery` dispatches trusted reset fragments and kind-3 notices. Explicit
notice retry retains its 1–20 batch, five-attempt, seven-day retention and
five-second deadline bounds. Notifications contain no password or bearer.

## Files of Interest

- `auth/password_reset.go`: shared credential orchestration and safe completion.
- `examples/ticked/internal/feat/auth/password_reset.go`: actual admission/final transactions.
- `examples/ticked/internal/web/password_reset.go`: public/operator form and cookie boundaries.
- Password-reset integration tests in the corresponding adapter/web packages.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#mailbox-password-reset)
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#reset-a-forgotten-password).

## Validation

Focused checks passed with Go 1.27.1 and the owned PostgreSQL 18.6 fixture.
Tagged tests require explicit database connection settings and isolate schemas.
Build scratch is bounded; mail is captured without external sending.

- `make source-license-check` — passed: 849 headers, 110 annotations.
- `make vet` — passed.
- `make lint-strict` — passed: zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test ./auth ./config ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^(TestPasswordResetTransactions|TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 94.370s.
- `go test -tags=integration -race -run '^TestPasswordResetTransportTransactions$' -count=1 -timeout=120s ./examples/ticked/internal/web` — passed in 43.693s.
- `git diff --check` — passed.

Evidence establishes wrong-purpose/malformed/unknown/stale/expired/replaced/replayed
rejection before candidate work; inactive/unverified/future-verified exclusion;
durable token/issuance/shared completion bounds; reset/change and double-completion
one-winner races; post-lock state/lease checks; slow-write expiry; complete fault
rollback; all-session/pending invalidation; retained real WebAuthn counters, TOTP
accepted steps and consumed/unused backup codes; actual required-MFA re-entry.
HTTP tests use production handlers and actual sessions/roles/TOTP proof. They
establish six-second neutral initiation across eligible/ineligible/throttled/provider
failure outcomes, GET/forbidden-request safety, Unicode completion, cookie clearing,
retained notice intent and denial of weaker or version-invalid operator sessions.

AR-01/AR-04/AR-05 and reset portions of AR-06 through AR-10 are supported. This
is focused evidence; complete browser journeys/acceptance mapping remain Slice 4
and the one full integrated `make check` remains after all four merges.

## Risks and Follow-ups

- Captured mail establishes application behavior, not actual external delivery.
  Finite retries may duplicate notification mail after concurrent/lost acknowledgment.
- Process-local admission and neutral timing require deployment-wide controls
  and separate production side-channel assessment.
- Ambiguous commit errors require operating review; do not assume rollback or
  authorize automatic replay. Cancellation can leave a finite lease until expiry.
- Browser acceptance and complete acceptance mapping remain Slice 4. All-factor-loss
  identity proofing and broader AUTH-07 remain separate concerns.
