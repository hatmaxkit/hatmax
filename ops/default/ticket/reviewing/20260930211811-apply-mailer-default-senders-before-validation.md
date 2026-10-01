---
id: TKT-20260930211811
title: Apply default mail senders before validation
status: reviewing
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:11Z
ready_at: 2026-10-01T10:29:16Z
started_at: 2026-10-01T10:29:16Z
reviewed_at: 2026-10-01T10:36:16Z
branch: fix/ticket-20260930211811-mailer-defaults
pr: pending
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Every active provider validates Message before applying its configured DefaultFrom. Validate rejects an empty From, so the fallback branch cannot supply a missing sender.

Sources: `mailer/message.go:55-58, mailer/smtp.go:34-42, mailer/mailgun.go:32-41, mailer/sendgrid.go, mailer/ses.go`.

Evidence: TestObservedDefaultSender received 'from address is required' despite a configured default. docs/reference/mailer/README.md promises this fallback.

Impact: The documented default sender behavior fails despite a correctly configured provider.

## Expected Outcome

Normalize the sender before sender-dependent validation in each active provider. Avoid unexpected mutation of the caller's message.

## Validation

Test a missing sender with a configured default, an explicit sender overriding the default, and a missing sender without a default for each active provider.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f12).

## Implementation

SMTP, Mailgun, SendGrid, and SES now normalize a local message copy before validation. When From.Email is empty, the complete configured DefaultFrom is applied. Explicit senders remain unchanged. Missing sender emails still fail validation before transport, and the caller's Message is not modified.

The normalized sender is used consistently in the SMTP envelope and MIME header, Mailgun and SendGrid requests, and both SES simple and raw delivery. Direct Message.Validate and NoopMailer keep their explicit-sender requirement. Tests exercise actual local transport boundaries, all default and explicit sender cases, other required fields, and caller immutability.

Delivery: [Mailer default senders](../../report/20261001103616-mailer-default-senders.md).
