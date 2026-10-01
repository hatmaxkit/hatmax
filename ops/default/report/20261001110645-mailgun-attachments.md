<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Mailgun Attachments

Status: reviewing
Ticket: [TKT-20260930211812](../ticket/reviewing/20260930211812-preserve-mailgun-attachments.md)
Branch: `fix/ticket-20260930211812-mailgun-attachments`
PR: pending

## Delivered Behavior

- Mailgun sends one multipart attachment part per Message.Attachments entry, retaining file order, filenames, binary bytes, and valid MIME types. Duplicate filenames remain separate files. An empty MIME type uses application/octet-stream; empty and nil data produce zero-byte attachments.
- Missing filenames and malformed MIME types fail before transport rather than allowing success with omitted content. An invalid later file prevents the entire request from being sent.
- Messages without files retain URL-encoded delivery. Multipart preserves the same sender, recipients, repeated CC/BCC fields, subject, text, HTML, reply-to, and custom headers.
- Existing default sender normalization, authentication, endpoint selection, response handling, and request-context cancellation remain in place. Sending does not change caller data.

## Contracts and Ownership

An unexported Mailgun body builder owns HTTP serialization, using standard-library multipart and MIME formatting. No provider SDK, dependency, public signature, or Message validation change is introduced. Metadata is formatted rather than interpolated into multipart headers.

The implementation follows the repeated attachment fields and multipart encoding in the [Mailgun message API](https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/messages/post-v3--domain-name--messages). The mailer reference, package usage, and Unreleased document support and metadata errors.

The multipart request is assembled in memory from caller-owned byte slices. Additional memory is proportional to the serialized message and attachment bytes. No filesystem access, background producer, retry loop, or new size-limit API is added. Remote message-size restrictions remain provider errors.

## Validation

All Go checks used `GOTOOLCHAIN=go1.26.7`. Database-related repository tests used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after verification. Mailgun cases used local HTTP servers and synthetic credentials; no external email was sent. This does not claim live-provider acceptance or CI validation against PostgreSQL 16.

- Before the fix, `go test ./mailer -run '^TestMailgunAttachment' -count=1` failed: attachment-bearing messages still used URL encoding and omitted the files.
- `go test -race ./mailer -count=20`: passed all 20 repetitions after the final changes.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.1%; mailer: 80.2%.
- `make docs-check`: passed.
- `git diff --check`: passed.

Named cases cover no files, an empty attachment slice, text, binary data, default and parameterized MIME, escaped and Unicode filenames, empty files, multiple files with duplicate names, metadata errors, preservation of all message fields, caller immutability, remote rejection, and cancellation. Existing default-sender transport tests remain green.

## Boundary

Only F13 is corrected. SMTP TLS/cancellation, header safety, other providers, runtime resolution, inline files, streaming uploads, and the separate SendGrid custom-header ticket remain unchanged.
