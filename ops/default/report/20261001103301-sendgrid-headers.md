<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# SendGrid Custom Headers

Status: reviewing
Ticket: [TKT-20261001103301](../ticket/reviewing/20261001103301-initialize-sendgrid-custom-headers.md)
Branch: `fix/ticket-20261001103301-sendgrid-headers`
PR: pending

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
