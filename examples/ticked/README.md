# Ticked - HatMax Reference Example

Single-binary todo list application demonstrating HatMax framework patterns with authentication, event-driven architecture, and Postgres-based pub/sub.

Nobody implements a todo list with this level of infrastructure in the real world. But a todo list is the archetypical example for a reason: it's familiar, simple to understand, and lets us focus on the framework patterns rather than complex business logic.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Single Binary :8080                       │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    Auth     │  │    List     │  │   Admin     │              │
│  │   Handler   │  │   Handler   │  │   Handler   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    Auth     │  │    List     │  │   Admin     │              │
│  │   Service   │  │   Service   │  │   Service   │              │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘              │
│         │                │                │                     │
│         │                │ publish        │                     │
│         │                ▼                │                     │
│         │         ┌─────────────┐         │                     │
│         │         │   PubSub    │─────────┼──────┐              │
│         │         │  (NOTIFY)   │         │      │ subscribe    │
│         │         └──────┬──────┘         │      ▼              │
│         │                │                │ ┌─────────────┐     │
│         │                │                │ │   Audit     │     │
│         │                │                │ │   Service   │     │
│         │                │                │ └──────┬──────┘     │
│         │                │                │        │            │
│         └────────────────┼────────────────┴────────┘            │
│                          ▼                                      │
│                   ┌─────────────┐                               │
│                   │    sqlc     │  Type-safe queries            │
│                   └──────┬──────┘                               │
└──────────────────────────┼──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                        PostgreSQL                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │    auth     │  │   ticked    │  │    audit    │              │
│  │   schema    │  │   schema    │  │   schema    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

### Patterns Demonstrated

- **Single Binary**: All features compile to one executable with embedded assets
- **HTML + HTMX**: Server-side rendering with dynamic interactions
- **Postgres PubSub**: Domain events via LISTEN/NOTIFY (no external broker)
- **Store Pattern**: Type-safe queries with sqlc
- **Lifecycle Management**: Automatic component discovery via `app.Setup()`
- **Feature-based Organization**: Code organized under `internal/feat/`

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

### Application
```bash
make build      # Build the binary
make run        # Build and run
make clean      # Remove binary and logs
```

### Database
```bash
make migrate    # Run database migrations
make sqlc       # Regenerate sqlc queries
```

### Development
```bash
make test       # Run tests
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

## API Endpoints

### Auth
- `GET /signup` - Signup page
- `POST /signup` - Register user
- `GET /signin` - Login page
- `POST /signin` - Authenticate user
- `POST /signout` - Logout

### List
- `GET /list` - Todo list view
- `POST /list/items` - Add item
- `POST /list/items/{itemID}/toggle` - Toggle item
- `DELETE /list/items/{itemID}` - Delete item

### Admin
- `GET /admin` - Dashboard
- `GET /admin/users` - Users list
- `GET /admin/users/{userID}` - User details
- `POST /admin/users/{userID}/toggle` - Toggle user active status
- `POST /admin/users/{userID}/roles` - Update user roles
- `GET /admin/events` - Audit events

> Use `GET /debug/routes` to list all registered endpoints.
