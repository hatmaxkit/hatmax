---
id: TKT-20261001103301
title: Initialize SendGrid custom headers before assignment
status: open
kind: bug
severity: medium
priority: normal
scope: domain
tags: mailer, correctness
source: implementation
reported_at: 2026-10-01T10:33:01Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

## Observed Behavior

SendGridMailer.Send assigns custom headers into sgMsg.Headers after NewSingleEmail constructs a message whose Headers map is nil. A valid message with Headers containing one entry panics with "assignment to entry in nil map" before reaching transport.

Source: `mailer/sendgrid.go`, custom-header loop. Observed while testing F12 with a valid explicit sender and `Headers: map[string]string{"X-Notice": "invoice"}`. This is independent of default-sender normalization.

## Expected Outcome

Initialize or populate SendGrid headers through the SDK's supported API without changing the caller's message. Preserve header values in the outgoing request.

## Validation

Exercise absent, empty, and populated Headers using a local SendGrid HTTP endpoint. Verify no panic, exact outgoing custom-header values, and unchanged caller data.
