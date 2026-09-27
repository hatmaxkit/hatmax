# Package Map

This map groups the public Hatmax packages by responsibility. Import paths use
the module `hatmax.adrianpk.com`.

## Application assembly

| Package | Responsibility |
| --- | --- |
| `app` | Component discovery, startup, route registration, serving, and shutdown |
| `config` | Static configuration loading, defaults, and validation |
| `settings` | Runtime setting schemas, validation, storage boundary, and typed access |
| `log` | Structured logging abstraction and implementations |

## HTTP and presentation

| Package | Responsibility |
| --- | --- |
| `web` | Form parsing, form errors, templates, and redirects |
| `htmx` | HTMX request headers, response headers, and attribute builders |
| `middleware` | Request identity, locale, cache, roles, rate limits, same-origin checks, and telemetry |
| `render` | Shared template functions and function-map composition |
| `ui` | HTML components, static assets, settings forms, and template helpers |
| `modal` | Modal-dialog configuration |
| `format` | Price and number formatting |
| `pagination` | Page parameters and result metadata |
| `i18n` | YAML translation loading and locale fallback |

## Identity, data, and validation

| Package | Responsibility |
| --- | --- |
| `auth` | Users, sessions, authentication service, cookies, and request middleware |
| `crypto` | Authenticated encryption, lookup hashes, password helpers, PASETO, and TOTP |
| `db` | Postgres connection and embedded SQL migrations |
| `model` | Identifiers, password hashing, roles, nullable UUID values, and time |
| `validation` | Validation errors, scalar rules, field builders, and text normalization |
| `seed` | Ordered one-time seed execution and symbolic references |
| `slug` | Unicode-aware URL slug generation |

## Infrastructure

| Package | Responsibility |
| --- | --- |
| `image` | Image records and storage, repository, and processing boundaries |
| `image/local` | Local filesystem image storage |
| `image/s3` | S3-compatible image storage |
| `image/stdprocessor` | Standard image decoding and resizing |
| `mailer` | Mail messages, providers, runtime selection, and settings |
| `pubsub` | Event envelopes and broker interfaces |
| `pubsub/postgres` | Durable polling pubsub backend |
| `scheduler` | Due-job polling, handlers, schedules, and fakes |
| `scheduler/postgres` | Postgres job-store implementation |
| `telemetry` | Request counters, panic aggregation, and settings schemas |

## Testing

| Package | Responsibility |
| --- | --- |
| `fake` | Mailer and telemetry fakes |
| `testhelper` | Isolated Postgres test databases and test logging |

Use the [Reference index](../index.md) for package contracts and the
[How-to Guides](../../how-to/index.md) for integration procedures.
