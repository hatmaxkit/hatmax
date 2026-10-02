<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Mailer

`mailer` sends a `Message` through `Mailer.Send`. The implementation note is
[mailer/readme.md](../../../mailer/readme.md).

## Message

`Address` is `Email` and `Name`. `String` returns the email when the name is
empty, and otherwise an RFC 5322 address.

`Message` has `From`, `To`, `CC`, `BCC`, `ReplyTo`, `Subject`, `Text`,
`HTML`, `Attachments`, and `Headers`. `Validate` requires a from email, at
least one `To` address with an email, a subject, and text or HTML. It does
not check `CC` or `BCC`. `IsMultipart` is true when both text and HTML are
set. `AllRecipients` returns `To`, then `CC`, then `BCC`.

`Attachment` has `Filename`, `ContentType`, and byte `Data`.

`Headers` contains unfolded custom fields. Names must be non-empty and use
ASCII bytes 33-126 except colon, following
[RFC 5322 field-name syntax](https://www.rfc-editor.org/rfc/rfc5322#section-3.6.8).
Spaces, tabs, control characters, and non-ASCII names are invalid. Values must
not contain CR or LF, including pre-folded lines. Empty values, horizontal
whitespace, punctuation, MIME encoded words, and non-ASCII values are preserved.
This check does not encode custom values or validate field-specific semantics.

`Validate` and every provider reject invalid custom fields before transport.
Errors report an invalid name or a line break without echoing header contents.
SMTP and SES raw MIME also check headers before serialization and return no
message bytes on rejection. Header validation does not mutate the message.

## Interface and constructors

`Mailer` has `Send(context.Context, *Message) error`.

| Constructor | Configuration | Result |
| --- | --- | --- |
| `NewSMTPMailer` | `SMTPConfig` | `*SMTPMailer` |
| `NewMailgunMailer` | `MailgunConfig` | `*MailgunMailer` |
| `NewSendGridMailer` | `SendGridConfig` | `*SendGridMailer` |
| `NewSESMailer` | context and `SESConfig` | `*SESMailer`, or an AWS configuration error |
| `NewNoopMailer` | logger | `*NoopMailer` |

Every provider validates the message before delivery. SMTP, Mailgun, SendGrid,
and SES first apply their configured `DefaultFrom` to a local message copy when
`Message.From.Email` is empty. The complete default address replaces `From`,
including its display name. An explicit sender takes precedence. A missing
sender and a default without an email still fail validation before transport.
`Send` leaves the caller's message unchanged; SMTP and SES raw MIME use the
same effective sender as their envelope or API source.

`Message.Validate` still requires an explicit sender when called directly.
`NoopMailer` does not apply a default sender.

Mailgun sends messages with attachments as `multipart/form-data`, with one
`attachment` file part per entry, as required by the
[Mailgun message API](https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/messages/post-v3--domain-name--messages).
Each part carries the filename, raw bytes, and supplied MIME type; an empty
MIME type defaults to `application/octet-stream`. Empty files are valid.
An absent filename or malformed MIME type returns an error before the HTTP
request. Messages without attachments keep URL-encoded form delivery.
Both encodings preserve recipients, body parts, reply-to, and custom headers.

| Configuration | Fields |
| --- | --- |
| `Config` | `DefaultFrom`, `Provider` |
| `SMTPConfig` | `Host`, `Port`, `Username`, `Password`, `TLS`, `StartTLS`, `InsecureSkipVerify` |
| `MailgunConfig` | `APIKey`, `Domain`, `BaseURL` |
| `SendGridConfig` | `APIKey` |
| `SESConfig` | `Region`, `AccessKeyID`, `SecretAccessKey`, `ConfigurationSetName` |

## SMTP transport

`TLS=true` establishes implicit TLS before reading the SMTP greeting. It takes
precedence when both `TLS` and `StartTLS` are true; no second upgrade is needed.
With `TLS=false`, `StartTLS=true` requires the server to advertise STARTTLS and
complete the upgrade before authentication or message delivery. A server without
the extension causes an error instead of plaintext delivery.

With both flags false, SMTP retains opportunistic STARTTLS: it upgrades when
advertised, otherwise permits plaintext. An advertised upgrade that fails never
falls back to plaintext. Both TLS paths verify the server certificate against
the configured `Host` and system trust roots by default. `InsecureSkipVerify`
disables certificate verification only when explicitly enabled.

The complete transport transaction, including dial, greeting, TLS handshake,
authentication, DATA, and QUIT, has a total 30-second limit. A shorter caller
deadline takes precedence. Cancellation closes the connection and interrupts
blocking SMTP operations; returned errors support `errors.Is` with
`context.Canceled` or `context.DeadlineExceeded`. Cancellation or an error after
DATA can leave delivery uncertain; the mailer does not automatically retry.

## Runtime

`New(cfg, log)` and `NewWithSettings(settings, cfg, log)` resolve a mailer.
Settings override the static config when a setting read succeeds.

Disabled mail, mode `disabled`, mode `dry_run`, an unknown mode, and an
unknown provider return `NoopMailer`. An empty mode is treated as `active`.
An empty provider is `smtp`. Mailgun without an API key or domain, SendGrid
without an API key, and an SES client that fails to initialize also return
`NoopMailer`.

`NoopMailer.Send` validates the message, logs the recipients, subject, and
from address, and returns nil.

Active providers are `smtp`, `mailgun`, `sendgrid`, and `ses`.
`RegisterSchemas` registers the `mailer.*` settings, including `mailer.mode`
values `disabled`, `dry_run`, and `active`.

`SettingsProvider` supplies `GetString`, `GetInt`, and `GetBool`. The runtime
setting keys cover enablement, mode, provider, sender, SMTP, Mailgun, SendGrid,
and SES fields. `Schemas` contains the definitions registered by
`RegisterSchemas`.

`dry_run` resolves to `NoopMailer`; it validates and logs metadata but does not
contact a provider. An active provider may still return a delivery error from
its network or remote API.
