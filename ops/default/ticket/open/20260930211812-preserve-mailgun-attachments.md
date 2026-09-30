---
id: TKT-20260930211812
title: Do not silently drop Mailgun attachments
status: open
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:12Z
commits:
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

## Observed Behavior

Send serializes a URL-encoded message without reading Message.Attachments and returns success after HTTP 2xx.

Sources: `mailer/mailgun.go:32-119`.

Evidence: TestObservedMailgunAttachment sent to a loopback HTTP server and confirmed that attachment filename and data were absent from the accepted request.

Impact: Switching providers can silently remove files from delivered messages while reporting a successful send.

## Expected Outcome

Implement Mailgun multipart attachments or return an explicit unsupported-content error until attachment support exists. Do not silently report success for omitted content.

## Validation

Verify attachment bytes, filename, MIME type, multiple attachments, and normal messages against a local HTTP test server.

Review: [Architecture Nit Review](../../report/20260930211800-architecture-nit-review.md#f13).
