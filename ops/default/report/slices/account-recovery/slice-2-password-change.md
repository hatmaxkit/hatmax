<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 2: Protected Password Change

Status: delivered
Delivery set: account-recovery
Plan: `ops/default/plan/account-recovery.md`
Tracker: `ops/default/tracker/account-recovery.md`
Branch: `feat/account-password-change`
PR: `#104`

## Purpose

Change the complete password through an actual current session and recent proof,
with shared password policy/KDF and atomic revocation. This is Slice 2 of 4.

## Delivered Behavior

- Ownership comes only from the session bearer. Password proof requires explicit
  policy permission and no established primary factors. With factors, proof
  defaults to phishing-resistant MFA; supported TOTP/backup MFA requires an
  explicit trusted lower-assurance policy.
- Authorization commits one shared completion attempt before candidate checking
  and hashing outside database locks. Checker, cancellation, KDF admission and
  final transaction failures cannot refund that attempt.
- The final subject-first transaction rechecks current actor/generation/version,
  policy, factor bindings and trusted time after lock waits and writes. It replaces
  the full credential, advances version once, deletes every session and restricted
  continuation, revokes mailbox slots and persists a kind-2 notification together.
- Mailbox verification, activation, roles, confirmed factors, replay state and
  backup verifiers/consumption remain. Completion returns safe subject/time only.
  Ticked clears the cookie and requires normal sign-in with current required MFA.
- Protected Ticked forms preserve complete bounded Unicode candidates. Notification
  failure retains committed state and intent for the existing bounded explicit retry.

## Implementation Notes

T2.1: `c0abf13f36e5`; T2.2: `ecf129548459`.

The adapter discovers ownership from the actual bearer, then locks the subject
before actor/factors and other affected records. Primary-factor metadata across
all relying parties is bounded to 20; a 21st row denies. The owned authorization
captures exact actor/factor/policy snapshots. A concurrent mutation cannot overwrite
a newer credential. Final clock checks roll back expiry during slow writes.

Ticked enables the form under its existing explicit trusted recovery-origin and
active-mail boundary. Trusted assembly permits password proof without primary
factors and retains the phishing-resistant default with factors. Each handler
has finite process-local IP, active-call and map admission. No dependency, schema
migration, retry worker or external mail send was added.

## Contracts Added or Changed

`PasswordChangePolicy`, `PasswordChangeAuthorization` and `PasswordChanged`
separate trusted policy, owned adapter state and safe completion. The service
normalizes the default factor proof and caps age to the existing recent-proof
bound. `RecoveryQueries.AuthorizePasswordChange` and `CommitPasswordChange`
extend the typed storage contract; SQL `RecoveryPasswordFactors` and generated
DAL are synchronized. Existing complete password replacement SQL is used within
the protected transaction.

`PasswordChangeHandler` owns GET/POST `/account/password`; POST accepts only
one `password` field, no query, at most 16 KiB body and 4096 candidate bytes.
Same-origin protection, cache/referrer headers and finite ingress apply.
`MailboxDelivery.ChangePassword` commits before kind-2 dispatch; existing notice
retry accepts verification/change intents with unchanged finite bounds.

## Files of Interest

- `auth/password_change.go`: policy, snapshot checks and shared credential orchestration.
- `examples/ticked/internal/feat/auth/password_change.go`: actual admission/final transactions.
- `examples/ticked/internal/web/password_change.go`: protected form and cookie boundary.
- Password-change integration tests in the corresponding adapter/web packages.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#recent-proof-password-change)
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#change-a-password).

## Validation

Focused checks passed with Go 1.27.1. Tagged tests require explicit PostgreSQL
connection settings and isolate schemas on the owned PostgreSQL 18.6 cluster.
Build scratch is bounded; mail is captured without external sending.

- `make source-license-check` — passed, 839 headers and 110 annotations.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test ./auth ./config ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^TestPasswordChangeTransactions$' -count=1 -timeout=120s ./examples/ticked/internal/feat/auth` — expanded authority cases passed in 14.097s.
- `go test -tags=integration -race -run '^(TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 78.652s.
- `go test -tags=integration -race -run '^TestPasswordChangeTransportTransactions$' -count=1 -timeout=90s ./examples/ticked/internal/web` — final production transport passed in 1.396s.
- `git diff --check` — passed.

Evidence establishes actual no-factor, UV WebAuthn and explicitly permitted
TOTP/backup policies; foreign/stale actor denial; post-lock factor/generation/time
checks; slow-write proof expiry; one-winner races; real shared KDF admission;
durable shared budgets; checker/cancellation failures; notice capacity; forced
rollback at every final mutation boundary; complete session/pending/token
invalidation; retained mailbox/MFA/replay/backup state and required-MFA re-entry.
Production transport establishes GET/forbidden-request safety, complete Unicode
input, cookie clearing and retained notification intent after provider failure.

AR-03/AR-04 and password-change portions of AR-06 through AR-10 are supported.
This is focused slice evidence; the one full integrated `make check` remains
after all four merges in the approved delivery plan.

## Risks and Follow-ups

- One-use mailbox password reset remains Slice 3; complete browser journeys and
  acceptance mapping remain Slice 4.
- Captured mail establishes application behavior, not actual external delivery.
  Concurrent/lost-acknowledgment retry may duplicate mail within finite bounds.
- Process-local admission requires application deployment controls. Ambiguous
  commit errors are not rollback proof or authorization for automatic replay.
- All-factor-loss identity proofing and broader AUTH-07 remain separate concerns.
