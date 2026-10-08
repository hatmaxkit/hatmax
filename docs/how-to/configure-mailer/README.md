<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Configure Mail Delivery

Use runtime mode `dry_run` while validating message construction. Select
`active` only after the provider configuration is complete.

## Configure SMTP

Add this section to the file loaded with `config.Load`. Set `SMTP_PASSWORD`
before loading; environment references are expanded at load time:

```yaml
mailer:
  enabled: true
  mode: dry_run
  provider: smtp
  default_from:
    email: noreply@example.com
    name: Example
  smtp:
    host: smtp.example.com
    port: 587
    username: example
    password: ${SMTP_PASSWORD}
    starttls: true
```

In a function returning an error, with the loaded `*config.Config`, a
`log.Logger` and the caller's `ctx`, create the runtime-selected mailer:

```go
mail := mailer.New(cfg, logger)
```

Send a validated message:

```go
err := mail.Send(ctx, &mailer.Message{
	From: mailer.Address{
		Email: cfg.Mailer.DefaultFrom.Email,
		Name:  cfg.Mailer.DefaultFrom.Name,
	},
	To:      []mailer.Address{{Email: "reader@example.com"}},
	Subject: "Welcome",
	Text:    "Welcome to Example.",
})
if err != nil {
	return err
}
```

Active providers use `DefaultFrom` when `Message.From` is empty. Set `From`
explicitly during `dry_run` because `NoopMailer` validates the message without
applying a provider configuration.

## Activate delivery

Change `mailer.mode` to `active` and construct the mailer again after dry-run
logs show the intended sender, recipients, and subject. Settings and static
configuration are resolved once, not on every send. An unknown mode or provider
resolves to `NoopMailer`; verify the selected values before relying on delivery.

## Verify the result

In `dry_run`, `Send` validates and logs the message without contacting the
provider. To test active SMTP locally, point host and port at an owned capture
server and select its advertised TLS/authentication capabilities; confirm its
captured message. The settings above require STARTTLS. A local capture proves
transport behavior, while inbox receipt requires the actual configured provider.
See
[Mailer Reference](../../reference/mailer/README.md) for provider requirements.
