# Configure Mail Delivery

Use runtime mode `dry_run` while validating message construction. Select
`active` only after the provider configuration is complete.

## Configure SMTP

Add:

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

Create the runtime-selected mailer:

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
```

Active providers use `DefaultFrom` when `Message.From` is empty. Set `From`
explicitly during `dry_run` because `NoopMailer` validates the message without
applying a provider configuration.

## Activate delivery

Change `mailer.mode` to `active` after dry-run logs show the intended sender,
recipients, and subject. An unknown mode or provider resolves to `NoopMailer`;
verify the selected values before relying on delivery.

## Verify the result

In `dry_run`, `Send` validates and logs the message without contacting the
provider. In `active`, verify delivery through the configured provider. See
[Mailer Reference](../../reference/mailer/README.md) for provider requirements.
