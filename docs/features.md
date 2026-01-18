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
| `middleware` | Request ID, role-based access, middleware stack |
| `render` | Template FuncMap utilities |
| `render/ui` | UI components (Badge, Chip, Price, Stat) |
| `modal` | Modal dialog configuration |
| `pagination` | Generic pagination (`Result[T]`) |

## Auth & Security

| Package | Description |
|---------|-------------|
| `auth` | Authentication service, middleware, context helpers |
| `crypto` | PASETO v4 tokens, AES-256-GCM encryption, Argon2id hashing |

## Data

| Package | Description |
|---------|-------------|
| `db` | PostgreSQL connection (pgx/v5), migrations |
| `model` | Base types (ID, timestamps), password hashing, roles |
| `validation` | Field validation (fluent API), error handling |
| `seed` | Database seeding with tracking |

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
| `pubsub` | Publish/subscribe messaging |
| `testhelper` | Testing utilities |
