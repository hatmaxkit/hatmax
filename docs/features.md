# HatMax Features

## Core

| Package | Description |
|---------|-------------|
| `app` | Application lifecycle management (Startable, Stoppable interfaces) |
| `config` | Environment-based configuration loading |
| `log` | Structured logging with slog |

## Web

| Package | Description |
|---------|-------------|
| `web` | Chi router helpers, response utilities, flash messages |
| `htmx` | HTMX primitives (triggers, actions, targets, swaps), response headers, template helpers |
| `middleware` | Request ID, role-based access, rate limiting, locale detection, static cache |
| `render` | Template FuncMap utilities, i18n integration |
| `render/ui` | UI components (Badge, Chip, Price, Stat, Button, Link, Form, Input, Alert, Toast) |
| `modal` | Modal dialog configuration |
| `pagination` | Generic pagination (`Result[T]`) |
| `i18n` | Internationalization with YAML translation files |
| `settings` | Runtime key-value configuration with schema validation |

## Auth & Security

| Package | Description |
|---------|-------------|
| `auth` | Authentication service, middleware, context helpers |
| `crypto` | PASETO v4 tokens, AES-256-GCM encryption, Argon2id hashing, TOTP/MFA with QR codes and backup codes |

## Data

| Package | Description |
|---------|-------------|
| `db` | PostgreSQL connection (pgx/v5), migrations |
| `model` | Base types (ID, timestamps), password hashing, roles, nullable UUID utilities |
| `validation` | Field validation (fluent API), error handling |
| `seed` | Database seeding with tracking, symbolic references (Ref/RefMap) |
| `slug` | URL-friendly slug generation with Unicode support |

## Media

| Package | Description |
|---------|-------------|
| `image` | Image management, variants (original, large, medium, thumbnail) |
| `image/local` | Local filesystem storage |
| `image/s3` | AWS S3 storage |
| `image/stdprocessor` | Image resizing with stdlib |

## Infrastructure

| Package | Description |
|---------|-------------|
| `mailer` | Email delivery (SMTP, SendGrid, AWS SES) |
| `pubsub` | Publish/subscribe messaging |
| `testhelper` | Testing utilities |
| `fake` | Test doubles (mailer) |

## Migrations

### `web/htmx` → `htmx`

Two HTMX packages coexist during transition:

| Package | Content |
|---------|---------|
| `web/htmx` | Legacy, minimal: `RespondDelete(w, err, log)` with error handling |
| `htmx` | Full HTMX support: triggers, actions, targets, swaps, headers, OOB, template helpers |

For new code, use `htmx`. See `htmx/readme.md` for usage.
