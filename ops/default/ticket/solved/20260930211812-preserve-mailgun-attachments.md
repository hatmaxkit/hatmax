---
id: TKT-20260930211812
title: Do not silently drop Mailgun attachments
status: solved
resolution: fixed
kind: bug
severity: medium
priority: normal
scope: domain
tags: architecture-review, domain, correctness
source: review
reported_at: 2026-09-30T21:18:12Z
ready_at: 2026-10-01T10:59:14Z
started_at: 2026-10-01T10:59:14Z
reviewed_at: 2026-10-01T11:06:45Z
closed_at: 2026-10-01T11:17:17Z
branch: fix/ticket-20260930211812-mailgun-attachments
pr: https://forge.adrianpk.com/hatmax/hatmax/pulls/80
commits: 55389df1de84c39bd800bbd7fac3274343e20d39, 30dbde7b87634a23a1665df6a3b4a2da4d58e06e
---
<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
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

## Implementation

Mailgun switches to multipart/form-data when Message.Attachments contains entries, sending one attachment part per file with its filename, bytes, and MIME type. Empty MIME types use application/octet-stream; zero-byte files remain attachments. Missing filenames and malformed MIME types fail before transport, including when a later attachment is invalid.

Messages without attachments retain URL-encoded delivery. Both paths preserve all existing form fields, including repeated CC/BCC entries, reply-to, and custom headers. Default sender behavior remains unchanged, caller data is not modified, and provider failures and cancellation remain errors.

Delivery: [Mailgun attachments](../../report/20261001110645-mailgun-attachments.md).

Merged into `dev` through PR #80 at `30dbde7b87634a23a1665df6a3b4a2da4d58e06e`.
