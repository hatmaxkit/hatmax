<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# mailer

Email delivery with multiple providers.

## Usage

```go
// Create mailer (SMTP, Mailgun, SendGrid, SES, or Noop)
mailer := mailer.NewSMTPMailer(smtpConfig)
mailer := mailer.NewMailgunMailer(mailgunConfig)
mailer := mailer.NewSendGridMailer(sendGridConfig)
mailer, err := mailer.NewSESMailer(ctx, sesConfig)
mailer := mailer.NewNoopMailer(log) // for tests

// Send email
msg := &mailer.Message{
    From:    mailer.Address{Email: "noreply@example.com", Name: "My App"},
    To:      []mailer.Address{{Email: "user@example.com"}},
    Subject: "Welcome",
    HTML:    "<h1>Hello</h1>",
    Text:    "Hello",
}

if err := mailer.Send(ctx, msg); err != nil { ... }
```

SMTP, Mailgun, SendGrid, and SES apply `DefaultFrom` before validation when
`msg.From.Email` is empty. An explicit sender overrides the default. The
complete default address, including its name, is used without changing `msg`.
If neither address supplies an email, delivery fails before transport.
Direct `Message.Validate` and `NoopMailer` still require an explicit sender.

Mailgun supports `Message.Attachments` through multipart file parts. Filenames
and bytes are preserved, including empty files; an omitted MIME type uses
`application/octet-stream`. Missing filenames or invalid MIME types fail before
delivery. Messages without attachments retain URL-encoded delivery.

## Runtime Resolution

```go
// Static only (from config.Config.Mailer)
m := mailer.New(cfg, log)

// Dynamic override (settings take precedence over cfg)
m := mailer.NewWithSettings(settingsSvc, cfg, log)
```

`NewWithSettings` resolves provider/mode with dynamic settings overrides on top of static config.

## API

```go
type Mailer interface {
    Send(ctx context.Context, msg *Message) error
}
```
