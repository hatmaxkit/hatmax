<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# mailer

Email delivery with multiple providers. Start with this complete local dry run:

## Usage

```go
package main

import (
	"context"
	"log"

	hatmaxlog "hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
)

func main() {
	mail := mailer.NewNoopMailer(hatmaxlog.NewTestLogger("info"))
	msg := &mailer.Message{
		From:    mailer.Address{Email: "noreply@example.com", Name: "Example"},
		To:      []mailer.Address{{Email: "user@example.com"}},
		Subject: "Welcome",
		HTML:    "<h1>Hello</h1>",
		Text:    "Hello",
	}
	if err := mail.Send(context.Background(), msg); err != nil {
		log.Fatal(err)
	}
}
```

No-op validates and logs the sender, recipients and subject without delivering
mail. Active constructors are `NewSMTPMailer(SMTPConfig)`,
`NewMailgunMailer(MailgunConfig)`, `NewSendGridMailer(SendGridConfig)` and
`NewSESMailer(ctx, SESConfig)`. SES construction also returns an error. Provider
configuration and transport acceptance do not establish inbox receipt.

SMTP, Mailgun, SendGrid, and SES apply `DefaultFrom` before validation when
`msg.From.Email` is empty. An explicit sender overrides the default. The
complete default address, including its name, is used without changing `msg`.
If neither address supplies an email, delivery fails before transport.
Direct `Message.Validate` and `NoopMailer` still require an explicit sender.

Custom `Message.Headers` must have non-empty printable ASCII names without
spaces or colons and values without CR or LF. All providers reject unsafe
headers before transport; SMTP and SES raw MIME also check before serialization.
Pass unfolded values, not pre-folded lines. Valid values remain unchanged.
See the [message reference](../docs/reference/mailer/README.md#message).

SMTP honors required STARTTLS (`StartTLS=true`) and implicit TLS (`TLS=true`).
Implicit TLS takes precedence when both are enabled. With neither enabled,
SMTP upgrades opportunistically when the server advertises STARTTLS, otherwise
allows plaintext. Failed TLS negotiation never falls back to plaintext.
Certificates are verified unless `InsecureSkipVerify` is explicitly enabled.
The entire SMTP transaction is limited to 30 seconds or a shorter caller
deadline; cancellation closes the connection. See the
[transport reference](../docs/reference/mailer/README.md#smtp-transport).

Mailgun supports `Message.Attachments` through multipart file parts. Filenames
and bytes are preserved, including empty files; an omitted MIME type uses
`application/octet-stream`. Missing filenames or invalid MIME types fail before
delivery. Messages without attachments retain URL-encoded delivery.

## Runtime Resolution

With a loaded `*config.Config`, a `log.Logger` and optional
`mailer.SettingsProvider`, choose one constructor in the composition root:

```go
// Static configuration.
staticMail := mailer.New(cfg, logger)

// Alternative: apply settings overrides at construction.
settingsMail := mailer.NewWithSettings(settingsService, cfg, logger)
```

Settings are read once during construction, using a background context;
changing them later requires constructing a new mailer. A successful settings
read replaces its static value; a failed read retains it. Disabled mail,
`disabled`/`dry_run` modes, unknown modes/providers, missing Mailgun or SendGrid
credentials, and SES initialization failures resolve to no-op. SMTP errors
surface on `Send`. Empty mode means active and empty provider means SMTP when
enabled. See [Mailer Reference](../docs/reference/mailer/README.md).

## API

```go
type Mailer interface {
    Send(ctx context.Context, msg *Message) error
}
```
