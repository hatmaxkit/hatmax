<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# SendGrid Custom Headers

Status: delivered
Ticket: [TKT-20261001103301](../ticket/solved/20261001103301-initialize-sendgrid-custom-headers.md)
Branch: `fix/ticket-20261001103301-sendgrid-headers`
PR: #132
Integrated into dev: `ed6de3b677b2c6eb477dfb8c92cc21e1c58c6b76`
Implementation: `2893ef9f61f641ecb7685ffd288e8ea7ddf4a827`
Validated candidate: `931220c74a8338142c1409d40eb2e2d823c2ef6b`

## Changes

SendGrid uses the SDK header setter to initialize its outgoing header map.
Absent, empty and populated caller maps remain unchanged, and custom header
values reach the provider HTTP payload unchanged. Other providers and header
validation retain their existing behavior.

## Validation

The new local HTTP regression reproduced the nil-map panic before the fix.
The SDK sends to a local endpoint; no external email is sent.

- `GOWORK=off go test ./mailer`: passed.
- `GOWORK=off go vet ./mailer`: passed.
- `GOWORK=off make lint-strict`: passed.
- `GOWORK=off make docs-check`: passed.
- `git diff --check`: passed.

## Integrated Validation

The validated candidate includes PRs #132, #133 and #134. The affected-package
gate passed against that exact integrated commit with an isolated native
PostgreSQL instance. The scaffold acceptance excludes global SQLC and verifies
the application module's own tool, build, readonly tests and reopened commands.

- `GOWORK=off go test ./mailer ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off go test -tags=acceptance ./generator/execute -run '^Test(ApplicationScaffoldAcceptance|ScaffoldOwnedTools|ScaffoldBaseline|PrepareApplicationCoversExactPlannedScaffold)$' -count=1 -timeout 10m`: passed.
- `GOWORK=off go vet ./mailer ./generator/... ./internal/hatmaxcli`: passed.
- `GOWORK=off make lint-strict`: passed.

Post-merge closure: `GOWORK=off make docs-check` and `git diff --check` passed.
