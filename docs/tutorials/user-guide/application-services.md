<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Application Services

Mail, images, and telemetry support product features without becoming hidden
global utilities. Features depend on the smallest interface they consume;
`main` selects concrete adapters and makes their lifecycle visible.

## Deliver Mail Through One Boundary

Depend on `mailer.Mailer` and build a validated `mailer.Message` inside the
application workflow that owns the communication. Hatmax adapters support
SMTP, Mailgun, SendGrid, SES, and a no-op implementation.

Static configuration and optional runtime settings choose mode, provider, and
sender. Disabled, dry-run, invalid, or unavailable provider configurations can
resolve to the no-op mailer. That behavior is useful for development but must
be operationally visible: successful no-op validation is not remote delivery.
Resolution reads settings once; construct a new mailer to apply later changes.
An active SMTP adapter reports invalid connection configuration on send, rather
than falling back to no-op. Provider acceptance still does not prove inbox
receipt, and a lost transport response can leave delivery uncertain.

Do not send mail directly from an HTTP handler. The service decides when the
message is warranted and whether delivery failure fails the workflow, is
retried through a job, or is recorded for later recovery.

## Separate Image Records, Storage, and Processing

The `image` package defines three boundaries:

- `Repository` persists image and variant metadata;
- `Store` puts, gets, deletes, and addresses bytes;
- `Processor` detects content and creates bounded variants.

The application supplies the repository. It chooses local or S3 storage and
the standard processor in `main`, then passes those interfaces to its feature
service. A complete upload workflow validates input, stores bytes, persists
metadata, creates requested variants, and compensates for partial failure.

`image/local` provides one-process filesystem storage, `image/s3` provides
S3-compatible object storage, and `image/stdprocessor` provides standard
decoding and resizing. None of those adapters grants authorization to a URL
or decides which user may read an image.
The standard processor bounds encoded input and decoded dimensions and checks
cancellation between phases. Local storage copies input without those limits
or context checks. Callers close input and `Get` readers, bound concurrent
processing, choose an extension matching the resulting content type, and clean
up partial objects when storage or metadata persistence fails.

## Observe Without Owning Product Behavior

Telemetry request middleware increments a counter and recovery middleware
records grouped panic information. The collector truncates messages at 200
bytes and limits stack function names; it does not redact secrets or cap the
number of distinct groups. Read-and-reset operations hand accumulated data to the
application-owned exporter or report.

Runtime telemetry settings describe mode and instance identity. They do not
authorize logging secrets or unbounded request data. Metrics and crash records
describe operation; durable business audit events remain application data.
The exporter owns mode lookup and transmission. Registering telemetry schemas
does not make collectors consult those settings or start an export loop.

## Assemble Replaceable Adapters Explicitly

Concrete selection belongs in the composition root:

```text
configuration and settings
  -> mail provider
  -> image store and processor
  -> telemetry collectors
  -> feature services that consume their interfaces
```

Most constructors retain dependencies. Mail runtime selection reads settings
and SES construction loads SDK configuration, so treat those steps as explicit
composition work rather than claiming every adapter constructor is free of I/O.
Components that open resources or start
work implement lifecycle interfaces and appear in dependency order in
`app.Setup`; plain adapters and services are passed directly to consumers.

Use no-op and fake implementations intentionally in tests. A fake should
record the narrow contract being asserted, while integration tests cover
provider encoding, storage, cleanup, and failure translation.

For exact contracts, see [Mailer](../../reference/mailer/README.md),
[Image](../../reference/image/README.md), and
[Telemetry](../../reference/telemetry/README.md).

---

[Previous: Events and Background Work](events-and-background-work.md) ·
[User Guide](README.md) ·
[Next: Testing and Evolution](testing-and-evolution.md)
