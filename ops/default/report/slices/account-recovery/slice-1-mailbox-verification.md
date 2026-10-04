<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 1: Mailbox Verification

Status: delivered
Delivery set: account-recovery
Plan: `ops/default/plan/account-recovery.md`
Tracker: `ops/default/tracker/account-recovery.md`
Branch: `feat/account-mailbox-verification`
PR: `#103`

## Purpose

Verify the current active account mailbox through a bounded one-use mail token,
without creating authentication or bypassing required MFA. This is Slice 1 of 4.

## Delivered Behavior

- Eligible unverified accounts can request/reissue a purpose-bound token through
  protected Ticked forms and trusted active mail dispatch. Public initiation
  groups account, storage and provider outcomes under one neutral acknowledgment.
- Confirmation records current mailbox ownership, consumes the token, advances
  `AuthVersion`, invalidates every prior session and restricted continuation,
  and persists a notification intent atomically. It clears the browser cookie
  and requires normal sign-in again.
- Reissue invalidates its predecessor without refunding shared budgets. Failed
  secret attempts survive transaction failure. Expired/stale/replayed state is
  rejected; existing password, account activation and confirmed MFA remain.
- Notification failures cannot undo verification. Explicit bounded application
  retry charges attempts before sending and retains pending intent for seven days.

## Implementation Notes

T1.1: `334046be21b4`; T1.2: `77395af0a3c1`.

PostgreSQL locks the subject before token and budget rows. A committed
revision-bound reservation separates durable attempt charging from final effects.
Confirmation rechecks eligibility and trusted time after lock waits and before
commit. SQL constraints enforce token shape and finite per-row bounds; the
adapter bounds purpose slots, subject budgets and notification capacity.

Ticked enables routes only with an explicit trusted link origin and an active
mailer. Tokens travel in mail-link fragments, are removed from navigation history,
and enter the server only through explicit protected POST. No provider message
body is propagated to public errors. No new dependency, worker or outbox framework
was added.

## Contracts Added or Changed

`Config.Recovery` / `RecoverySettings` validate finite TTLs, attempts, operation
and lease bounds. `RecoveryService` / `RecoveryQueries` own issue, reservation,
confirmation, exact release and retained-token cleanup. `MailboxIssue` is a
transient trusted-dispatch result; ordinary formatting/JSON omit its secret.
`MailboxRecord` omits the digest from JSON and redacts normal formatting.
`User.MailboxVerifiedAt` records current verified address metadata.

Migration `008-account-recovery.sql` and regenerated DAL/callers add mailbox
slots, fixed-window budgets and notification intents together. Existing
integration/browser migration fixtures include the schema.

Ticked limits ingress before lookup to 12 requests per socket IP per minute,
32 active calls and 1024 current IP entries. Form bodies are at most 16 KiB.
Initiation targets six seconds with five seconds of bounded work. Notification
retry takes batches of 1–20, at most five attempts per intent and a five-second
call deadline. Core cleanup owns no loop and removes at most 1000 retained rows.

## Files of Interest

- `auth/recovery.go`, `auth/recovery_token.go`: typed lifecycle and secret boundary.
- `config/recovery.go`: validated settings.
- `examples/ticked/internal/feat/auth/recovery.go`: real mutation transactions.
- `examples/ticked/internal/feat/auth/mailbox_delivery.go`: trusted dispatch/retry.
- `examples/ticked/internal/web/mailbox.go`: protected production forms/admission.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#mailbox-verification)
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#verify-a-mailbox).

## Validation

All commands below passed with Go 1.27.1. Database runs used explicit connection
settings and an isolated schema per test on the owned PostgreSQL 18.6 cluster.
Build scratch storage was bounded; mail was captured without external sending.

- `make source-license-check` — passed.
- `make vet` — passed.
- `make lint-strict` — passed, zero issues.
- `make docs-check` — passed, including all example compilation.
- `go test ./auth ./config ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=integration -race -run '^(TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestEnrollmentTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorChanges|TestFactorAuthority|TestSessionTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 68.026s.
- `go test -tags=integration -race -run '^(TestMailboxTransactions|TestMailboxDeliveryTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — final expanded cases passed in 9.033s.
- `go test -tags=integration -race -run '^TestMailboxTransportTransactions$' -count=1 -timeout=90s ./examples/ticked/internal/web` — passed in 19.259s.
- `go test -run '^$' -fuzz '^FuzzRecoveryToken$' -fuzztime=20s -parallel=2 -timeout=60s ./auth` — passed, 439917 executions.
- `git diff --check` — passed.

Evidence includes real one-use/reissue races, expiry after a held subject lock,
durable token/shared budgets, cancellation, capacity denial, cleanup, early and
late transaction fault injection, all-session/pending deletion, actual WebAuthn
state/replay preservation and ordinary required-MFA re-entry. Production handlers
with real storage prove equal public acknowledgment text/deadline, protected POST,
GET safety, cookie clearing and committed-state preservation across mail failure.

AR-01/AR-02 and the mailbox portions of AR-06 through AR-09 are supported. This
is not complete AR-01 through AR-10 or AUTH-06 delivery. The full integrated
`make check` remains after all four merges, as recorded in the approved plan.

## Risks and Follow-ups

- Password change/reset remain Slices 2/3; complete browser acceptance remains
  Slice 4. Existing browser fixtures compile but no new browser result is claimed.
- Captured mail establishes the application boundary, not actual provider
  delivery or a production timing/side-channel audit.
- Application owners invoke bounded cleanup/retry; no worker starts implicitly.
  Concurrent/lost-acknowledgment retries may duplicate mail within finite limits.
- Process-local admission does not establish deployment-wide anti-automation.
  All-factor-loss identity proofing and broader AUTH-07 controls remain separate.
