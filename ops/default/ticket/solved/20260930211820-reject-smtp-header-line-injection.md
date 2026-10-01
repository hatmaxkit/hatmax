---
id: TKT-20260930211820
title: Reject SMTP header line injection
status: solved
resolution: fixed
closed_at: 2026-10-01T15:41:24Z
kind: bug
severity: medium
priority: normal
scope: api
tags: architecture-review, api, correctness
source: review
reported_at: 2026-09-30T21:18:20Z
ready_at: 2026-10-01T15:30:00Z
started_at: 2026-10-01T15:30:00Z
reviewed_at: 2026-10-01T15:37:29Z
branch: fix/ticket-20260930211820-smtp-headers
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/88
commits: bbbda2650a33c2b1be7e93ebefd76e7b2520fddc, 6e3fea1326a3713fc362e535d17d6fbbadc95d7a
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

## Selected Contract

Message validation owns custom-header safety for every provider. Names must contain at least one printable ASCII character, excluding spaces and colons, as defined by RFC 5322 field-name syntax. Values must not contain CR or LF, including pre-folded values. Invalid metadata returns an error before transport without echoing header contents or changing the caller's message. The shared SMTP/SES raw MIME builder repeats the header check before writing any bytes; values that pass remain unchanged. Address, subject, attachment, reserved-header, and provider delivery policies are outside this finding.

## Delivery

Custom headers now pass a common validation check before provider delivery and a defensive check before SMTP/SES raw serialization. Invalid input returns an error without message bytes or caller mutation. Valid fields remain unchanged; SMTP DATA and SES raw API captures retain custom headers and decoded non-ASCII subjects across text, HTML, alternative, and attachment paths.

`make check`, `make docs-check`, `go test -race ./mailer -count=1`, and `go test ./mailer -run '^TestHeader' -count=20` passed. Total coverage is 81.6%; mailer coverage is 86.5%.

Report: [Custom Header Safety](../../report/20261001153729-custom-header-safety.md).

PR #88 was verified merged into dev at 6e3fea1326a3713fc362e535d17d6fbbadc95d7a. F21 is resolved.
