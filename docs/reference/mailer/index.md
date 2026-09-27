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
