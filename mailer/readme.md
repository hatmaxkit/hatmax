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

## Runtime Resolution

```go
// Static only (from config.Config.Mailer)
m := mailer.NewFromConfig(cfg, log)

// Dynamic override (settings before cfg)
m := mailer.NewWithConfig(settingsSvc, cfg, log)
```

`NewWithConfig` resolves provider/mode with dynamic settings overrides on top of static config.

## API

```go
type Mailer interface {
    Send(ctx context.Context, msg *Message) error
}
```
