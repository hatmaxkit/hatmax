<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Slice 4: Recovery Acceptance

Status: drafting
Delivery set: account-recovery
Plan: `ops/default/plan/account-recovery.md`
Tracker: `ops/default/tracker/account-recovery.md`
Branch: `test/account-recovery-acceptance`
PR: pending

## Purpose

Establish actual browser recovery journeys and finite final boundary regressions,
then map the approved criteria to exact evidence. This is Slice 4 of 4 in the
[plan](../../../plan/account-recovery.md) and [tracker](../../../tracker/account-recovery.md).
The full integrated gate remains after this slice's verified merge.

## Delivered Behavior

- Chromium executes production verification, password-change and reset forms
  against actual PostgreSQL/services, captured mail and real navigator WebAuthn.
  GET and opening a fragment link cannot consume a token. Explicit POST removes
  the fragment, preserves source validation and clears a cookie only after commit.
- Mailbox proof and reset tokens cannot serve as session cookies. Weak password
  or backup proof cannot authorize phishing-resistant password/operator actions.
  Actual recent WebAuthn permits change and operator initiation; the operator
  receives only the neutral mailbox acknowledgment.
- Change/reset preserve actual factors and backup consumption. Password sign-in
  remains insufficient at a strong route; fresh navigator proof succeeds. A spent
  backup code remains denied while an unused retained code supplies supported MFA.
- Failed dispatch cannot revive an older link; failed notification preserves its
  committed intent for explicit retry. An actual dropped committed reset response
  leaves an invalid old cookie; replay cannot repeat the mutation and normal
  sign-in with the new password still requires current MFA.
- Earlier caller deadlines bound actual subject-lock waits. A stale reservation
  owner cannot release a newer lease; releasing the current lease cannot refund
  attempts. Bounded retained-token cleanup honors both purposes and live leases
  without changing current sessions or admission.

## Implementation Notes

T4.1: `b599c7722b43`; T4.2: `52096c68eef6`.

The existing finite authenticator browser harness is shared by the new
`TestAccountRecoveryBrowser` selector. It still verifies original registration,
assertion/step-up, management, TOTP/backup and rejected browser proof before the
recovery extension. The composite fixture explicitly selects a bounded 40-attempt
authenticator subject window; recovery budgets and ingress defaults remain
unchanged. It waits the real ingress window before final lost-response/replay
assertions. The child deadline is 240 seconds under a 300-second Go timeout.

The browser found native no-referrer form navigation carrying an opaque origin.
Recovery pages now submit a same-origin fetch with the exact supported form media
type, ten-second client deadline and no automatic replay. Existing middleware
continues to reject opaque/foreign sources; a real foreign-origin browser POST
cannot consume the valid verification token. Provider/transport controls are
fixture-only and never manufacture authentication, tokens or factor records.

No public credential API, schema migration, dependency, retry worker or external
mail send was added. Supported core/configuration/example/User Guide references
now match implemented reset, browser and application obligations.

## Contracts Added or Changed

`mailboxPage` attaches shared explicit submission behavior to recovery forms.
The server still requires protected POST, finite field/body/candidate bounds,
`no-store` and `no-referrer`; cookie and atomic storage contracts remain intact.
Client response loss is an unknown outcome, never a trigger for automatic replay.

`TestAccountRecoveryBrowser`, its captured-mail/owned transport fixture and
`testdata/recovery.mjs` extend actual integration evidence. Mandatory Chromium,
Node native WebSocket and explicit owned database prerequisites fail when absent.
`TestRecoveryBoundaries` closes caller-deadline, owned-lease release and finite
cross-purpose retention cases at the production adapter.

## Files of Interest

- `examples/ticked/internal/web/account_recovery_browser_test.go`: production recovery assembly and owned response-loss fixture.
- `examples/ticked/internal/web/testdata/recovery.mjs`: actual forms, navigator proof, dispatch/retry and re-entry.
- `examples/ticked/internal/web/mailbox.go`: origin-preserving bounded form submission.
- `examples/ticked/internal/feat/auth/recovery_boundaries_integration_test.go`: actual deadline, lease and cleanup boundaries.
- [Authentication reference](../../../../../docs/reference/authentication/README.md#recovery-browser-acceptance),
  [configuration](../../../../../docs/reference/configuration/README.md#recovery-limits),
  and [User Guide](../../../../../docs/tutorials/user-guide/identity-and-sessions.md#reset-a-forgotten-password).

## Validation

Focused checks passed with Go 1.27.1, Chromium 151.0.7922.173, Node v26.8.1 and
owned PostgreSQL 18.6. Database tests isolate/drop schemas and use explicit
connection settings. Browser profiles and socket scratch are owned and cleaned;
mail is captured without external sending. The owned database was stopped after
all tests and a zero-other-client check.

- `make source-license-check` — passed: 852 headers and 111 annotations.
- `make vet` — passed.
- `make lint-strict` — passed: zero issues.
- `make docs-check` — passed, including example compilation and local links.
- `go test -race ./auth ./config ./model ./middleware ./examples/ticked/internal/feat/auth ./examples/ticked/internal/web` — passed.
- `go test -tags=browser -race -run '^TestAccountRecoveryBrowser$' -count=1 -timeout=300s ./examples/ticked/internal/web` — passed in 75.007s.
- `go test -tags=integration -race -run '^(TestRecoveryBoundaries|TestPasswordResetTransactions|TestPasswordChangeTransactions|TestMailboxTransactions|TestMailboxDeliveryTransactions|TestCredentialTransactions|TestSessionTransactions|TestWebAuthnTransactions|TestFallbackTransactions|TestFactorAuthority)$' -count=1 -timeout=180s ./examples/ticked/internal/feat/auth` — passed in 95.502s.
- `go test -tags=integration -race -run '^(TestMailboxTransportTransactions|TestPasswordChangeTransportTransactions|TestPasswordResetTransportTransactions)$' -count=1 -timeout=180s ./examples/ticked/internal/web` — passed in 62.509s.
- `go test -run '^$' -fuzz '^FuzzRecoveryToken$' -fuzztime=20s -parallel=2 -timeout=60s ./auth` — passed: 405428 executions, no KDF/database work.
- `git diff --check` — passed.

The [acceptance map](../../../tracker/account-recovery.md#acceptance-evidence-map)
connects AR-01 through AR-10 to exact selectors and remaining boundaries. Delivered
Slices 1–3 supply actual token/proof/version/expiry/budget races, full fault rollback,
all-session/pending invalidation and retained WebAuthn/TOTP/replay/backup state.
This slice reruns those affected production adapters and transports and adds
actual browser recovery, lost-response and final finite boundary evidence.
AR-10's focused documentation/consumer compilation checks pass; its one exact
immutable integrated dev `make check` remains pending the fourth merge.

## Risks and Follow-ups

- After verified merge, deliver this report, finalize the immutable integrated
  candidate and run one full `make check`. Do not mark the set or bounded AUTH-06
  coverage closed before that result. Repository corrections use the approved
  `fix/account-recovery-validation` branch and merge procedure only if needed.
- Applications own trusted origins/RP/policy, active mail provider delivery,
  bounded cleanup/notice retry scheduling, trusted proxies and deployment-wide
  anti-automation. Neutral timing is not a production side-channel audit.
- Notification retries may duplicate mail within finite bounds. Lost/ambiguous
  commit responses require operating review or normal sign-in, never assumed
  rollback or automatic mutation replay.
- Virtual devices prove browser/library/adapter integration, not physical
  hardware, attestation, non-exportability or assurance certification. All-factor-loss
  identity proofing, broader account lifecycle and AUTH-07 remain separate concerns.
