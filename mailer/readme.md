# mailer

Email delivery with multiple providers.

## Usage

```go
// Create mailer (SMTP, SendGrid, SES, or Noop)
mailer := mailer.NewSMTP(smtpConfig)
mailer := mailer.NewSendGrid(apiKey, from)
mailer := mailer.NewSES(awsConfig, from)
mailer := mailer.NewNoop()  // for tests

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

## API

```go
type Mailer interface {
    Send(ctx context.Context, msg *Message) error
}
```
