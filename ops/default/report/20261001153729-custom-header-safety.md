<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Custom Header Safety

Status: reviewing
Ticket: [TKT-20260930211820](../ticket/reviewing/20260930211820-reject-smtp-header-line-injection.md)
Branch: `fix/ticket-20260930211820-smtp-headers`
PR: pending
Implementation: pending

## Delivered Behavior

- Message.Validate rejects empty or invalid custom header names and CR/LF in values before provider transport. Names use the RFC 5322 field-name byte ranges, not the narrower HTTP token grammar.
- SMTP and SES raw MIME also validate headers before writing any message bytes. An invalid field returns nil bytes and an error, rather than stripping characters or silently altering metadata.
- Valid values remain unchanged, including empty values, tabs, punctuation, encoded words, and Unicode. SMTP DATA and SES raw API captures retain valid custom fields and ordinary encoded subjects across supported body shapes.
- Source documentation, the mailer reference, package guidance, and Unreleased describe the unfolded-value contract and pre-transport rejection.

## Contracts and Ownership

Message owns the shared custom-header contract. Names contain one or more ASCII bytes 33-57 or 59-126, following [RFC 5322 section 3.6.8](https://www.rfc-editor.org/rfc/rfc5322#section-3.6.8). Values must not contain CR or LF, including folded continuations. Callers must supply unfolded values. The implementation does not encode custom values or validate field-specific semantics.

All providers already call Message.Validate before transport; the new rule therefore applies consistently to SMTP, Mailgun, SendGrid, SES simple/raw, and Noop. Errors identify an invalid name or line break without copying untrusted header contents. Default-sender handling and all caller data remain unchanged. The raw builder repeats only custom-header validation at its serialization boundary.

## Validation

All checks used Go 1.26.7. Full repository database tests used an isolated native PostgreSQL 18.6 cluster, stopped after validation. SMTP and SES acceptance use local TCP and HTTP endpoints, not external services.

- Before the fix, `go test ./mailer -run '^TestHeader(Validation|Rejection)$' -count=1` failed: public validation accepted unsafe names and values, and SMTP/Mailgun delivered CRLF-containing metadata. The run also reached the already-ticketed SendGrid nil-header-map panic; that independent issue is unchanged.
- `make check`: passed licensing, formatting, vet, all tests, coverage, and strict lint. Total coverage: 81.6%; mailer: 86.5%.
- `make docs-check`: passed.
- `go test -race ./mailer -count=1`: passed.
- `go test ./mailer -run '^TestHeader' -count=20`: passed all 20 repetitions.
- `git diff --check`: passed.

Named cases cover empty names, whitespace, colon, CR/LF, control and high bytes, Unicode names, added-header and header/body-boundary attacks, folded values, safe metadata, provider rejection, and caller immutability. Captured MIME is parsed to check actual header and subject preservation; existing transport encryption, cancellation, attachment, and sender tests continue to pass.

## Boundary

Only F21 is corrected. There are no new dependencies or changes to address validation, subject encoding, attachments, reserved-field policy, SMTP encryption, retries, or the scaffold baseline. The existing SendGrid custom-header initialization ticket remains separate. Diataxis keeps the exact public contract in the existing mailer reference without documentation restructuring.
