<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Mailer Default Senders

Status: reviewing
Ticket: [TKT-20260930211811](../ticket/reviewing/20260930211811-apply-mailer-default-senders-before-validation.md)
Branch: `fix/ticket-20260930211811-mailer-defaults`
PR: pending

## Delivered Behavior

- SMTP, Mailgun, SendGrid, and SES apply the complete DefaultFrom address before validation when Message.From.Email is empty. A name-only From does not prevent the configured default from supplying both email and name.
- An explicit sender overrides the default. If neither address supplies an email, validation fails before transport. Recipient, subject, and body requirements remain enforced.
- Each active provider normalizes its own local message copy. Caller data stays unchanged on successful delivery and validation failure.
- The effective sender reaches the SMTP envelope and MIME header, Mailgun form, SendGrid JSON, and SES source. SES raw delivery uses that same sender in its MIME header rather than the original empty From.
- Direct Message.Validate and NoopMailer still require an explicit sender. Public signatures and provider configuration are unchanged.

## Contracts and Ownership

An unexported Message helper owns sender normalization. Providers invoke it before existing validation and continue with their local copy. Only the From value is replaced; nested slices, maps, attachments, and reply-to data are not written by normalization. No deep-copy API, dependency, transport rewrite, or runtime-resolution change is introduced.

The configuration Godoc, mailer reference, package usage, and Unreleased describe default precedence and caller ownership. Noop and direct validation behavior remain distinct from active-provider delivery.

## Validation

All Go checks used `GOTOOLCHAIN=go1.26.7`. Database-related repository tests used an isolated native PostgreSQL 18.6 cluster on loopback, stopped after verification. Provider tests used local HTTP and plaintext SMTP servers with synthetic credentials; no external email was sent. This does not establish live-provider acceptance or SMTP TLS policy.

- Before the fix, `go test ./mailer -run '^TestSender' -count=1` failed for missing senders with valid defaults across SMTP, Mailgun, SendGrid, SES simple, and SES raw paths.
- `go test -race ./mailer -count=20`: passed all 20 repetitions after the final changes.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.0%; mailer: 78.7%.
- `make docs-check`: passed.
- `git diff --check`: passed.

Named cases cover default and explicit senders, name-only inputs and defaults, absent defaults, required recipients, empty recipient emails, subject and body validation, MIME sender identity, and unchanged caller data. SMTP fixture cleanup joins its own bounded server goroutine.

## Boundary and Follow-ups

Only F12 behavior is corrected. SMTP TLS/cancellation, Mailgun attachments, and SMTP header safety remain owned by their existing tickets.

A valid SendGrid message with custom headers exposed a preexisting nil-map panic during the first regression setup. It is captured separately in [TKT-20261001103301](../ticket/open/20261001103301-initialize-sendgrid-custom-headers.md) and is not fixed here. The F12 regression matrix does not send custom headers.
