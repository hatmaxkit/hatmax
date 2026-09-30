---
id: TKT-20260930211811
title: Apply default mail senders before validation
status: open
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:11Z
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
