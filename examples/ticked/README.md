# Ticked - Hatmax Reference Example

Single-binary todo list application demonstrating Hatmax framework patterns.

A familiar problem so you can focus on the framework, not the domain.

## Architecture

Single Go binary with embedded assets:

- **Auth** - User authentication (signup, signin, sessions)
- **List** - Todo list management (the actual domain)
- **Admin** - Admin dashboard

All features share a single PostgreSQL database.

## Prerequisites

- Go 1.21+
- PostgreSQL running locally
- Make
- sqlc (for regenerating queries)

## Quick Start

### 1. Setup PostgreSQL

Create database and user:

```bash
createdb ticked
createuser -P dev  # password: dev
```

### 2. Run migrations

```bash
make migrate
```

### 3. Build and run

```bash
make run
```

Access the application at http://localhost:8080

## Available Commands

```bash
make build      # Build the binary
make run        # Build and run
make test       # Run tests
make migrate    # Run database migrations
make sqlc       # Regenerate sqlc queries
make clean      # Remove binary and logs
```

## Configuration

Default configuration in `config.yaml`:

```yaml
server:
  port: "8080"

database:
  host: localhost
  port: 5432
  user: dev
  password: dev
  name: ticked
  sslmode: disable

log:
  level: debug
  format: text
```

Override with environment variables (prefix `TICKED_`):

```bash
TICKED_SERVER_PORT=9000 make run
TICKED_DATABASE_HOST=db.example.com make run
```

## Hatmax Patterns

- Single binary with embedded assets
- Feature-based organization under `internal/feat/`
- YAML config + env vars + flags
- Lifecycle via `app.Setup()`
- Graceful shutdown
