---
id: TKT-20260930211820
title: Reject SMTP header line injection
status: open
kind: bug
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:20Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Custom Message.Headers are written as raw key/value lines without CR/LF validation. A header value containing CRLF creates an additional header in the delivered MIME message.

Sources: `mailer/smtp.go:323-328, mailer/message.go:55-85`.

Evidence: TestObservedSMTPTransport captured X-Review-Injected as a separate header from a CRLF-containing X-Review value.

Impact: Applications passing user-derived header metadata can introduce unintended headers or terminate the header section. Subject encoding is a separate path and was not identified as vulnerable by this probe.

## Expected Outcome

Validate custom header names and values before SMTP serialization. Reject line delimiters and invalid field names while preserving legitimate MIME values.

## Validation

Cover CR and LF in header names and values, header/body boundary injection, valid custom headers, and normal non-ASCII subjects.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f21).
