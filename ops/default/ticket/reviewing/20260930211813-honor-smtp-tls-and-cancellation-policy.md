---
id: TKT-20260930211813
title: Honor SMTP STARTTLS and cancellation policy
status: reviewing
kind: bug
severity: high
priority: high
scope: api
tags: architecture-review, api, hardening
source: review
reported_at: 2026-09-30T21:18:13Z
ready_at: 2026-10-01T11:20:43Z
started_at: 2026-10-01T11:20:43Z
reviewed_at: 2026-10-01T11:34:15Z
branch: fix/ticket-20260930211813-smtp-policy
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/81
commits: b9937c408f9f20982f4c8a9ba90531ea8681fc43
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

## Implementation

SMTP owns its context-aware connection, absolute deadline, and cleanup. TLS=true requires implicit TLS and takes precedence over StartTLS. Otherwise StartTLS=true requires an advertised, successful upgrade before AUTH, MAIL, or DATA. Both flags false preserve opportunistic upgrades; failed TLS never falls back to plaintext. Both paths verify certificates unless InsecureSkipVerify is explicitly enabled.

The transport transaction has a total 30-second limit, with shorter caller deadlines respected. Cancellation closes the connection; returned errors retain the context cancellation or deadline identity. Public signatures, configuration fields, default sender handling, MIME serialization, and runtime resolution remain unchanged. No automatic retry is introduced because failure after DATA can leave delivery uncertain.

Validation passed: make check with Go 1.26.7 and isolated PostgreSQL 18.6 (81.3% total coverage, 86.0% mailer), make docs-check, 20 race-enabled mailer repetitions, and git diff --check. Local SMTP servers cover encryption policy, certificate rejection, authentication ordering, canceled callers, and interruption of greeting, hello, TLS handshakes, AUTH, DATA, and QUIT.

Delivery: [SMTP transport policy](../../report/20261001113415-smtp-policy.md).
