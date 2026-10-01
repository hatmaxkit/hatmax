<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
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

| Configuration | Fields |
| --- | --- |
| `Config` | `DefaultFrom`, `Provider` |
| `SMTPConfig` | `Host`, `Port`, `Username`, `Password`, `TLS`, `StartTLS`, `InsecureSkipVerify` |
| `MailgunConfig` | `APIKey`, `Domain`, `BaseURL` |
| `SendGridConfig` | `APIKey` |
| `SESConfig` | `Region`, `AccessKeyID`, `SecretAccessKey`, `ConfigurationSetName` |

SMTP uses implicit TLS when `TLS` is true. `StartTLS` is accepted by the
configuration type but is not used by the current SMTP send path.

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
