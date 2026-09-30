---
id: TKT-20260930211813
title: Honor SMTP STARTTLS and cancellation policy
status: open
kind: bug
severity: high
priority: high
scope: api
tags: architecture-review, api, hardening
source: review
reported_at: 2026-09-30T21:18:13Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

SMTP Send ignores ctx and never consults StartTLS. With TLS false, smtp.SendMail upgrades opportunistically but does not require STARTTLS. StartTLS true therefore permits plaintext delivery to a server that does not advertise the extension.

Sources: `mailer/smtp.go:34-85, mailer/mailer.go:22-32`.

Evidence: TestObservedSMTPTransport delivered to a plaintext loopback server with StartTLS true, including with an already canceled context. The unused StartTLS field is also documented as a current limitation.

Impact: A configuration that appears to require transport encryption does not enforce it. A canceled or stalled send can continue indefinitely.

## Expected Outcome

Own SMTP connection establishment with a context-aware dialer and deadlines. Define and enforce implicit TLS and required STARTTLS behavior, failing before sending message data when required encryption is unavailable.

## Validation

Cover required STARTTLS unavailable, successful upgrade, certificate validation, implicit TLS, canceled context, and stalled handshake or DATA phases.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f14).
