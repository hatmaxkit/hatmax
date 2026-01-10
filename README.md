# HatMax

<p align="center">
  <img src="docs/img/hero.png" width="800">
</p>

[![Go Reference](https://pkg.go.dev/badge/github.com/hatmaxkit/hatmax.svg)](https://pkg.go.dev/github.com/hatmaxkit/hatmax)
[![CI](https://github.com/hatmaxkit/hatmax/actions/workflows/ci.yml/badge.svg)](https://github.com/hatmaxkit/hatmax/actions)
[![codecov](https://codecov.io/gh/hatmaxkit/hatmax/branch/main/graph/badge.svg)](https://codecov.io/gh/hatmaxkit/hatmax)

**A lightweight Go library for building single-binary web applications.**

## What is HatMax?

HatMax is a focused library for building web applications that compile to a single executable and use Postgres as their primary database. It provides authentication primitives, lifecycle management, and structured logging as core features, letting you focus on your domain logic instead of reinventing infrastructure.

If you've built a few Go web apps and found yourself copying similar patterns (lifecycle management, auth flows, database setup, middleware chains), HatMax extracts those patterns into small, composable packages. It's designed for applications that benefit from simplicity: one binary, one database.

## Core Principles

- **Single binary deployment** - Build applications that compile to one executable. Simple deployment, straightforward operations.
- **Just use Postgres** - Leverage Postgres for relational data, JSONB, full-text search, pub/sub, and queues. Start simple, add specialized tools only when truly needed.
- **PubSub** - Domain events with no external broker. Default implementation uses Postgres `LISTEN`/`NOTIFY`.
- **HTML + HTMX** - Server-side rendering with HTMX for dynamic interactions. Build modern UX with straightforward patterns.
- **Explicit over magic** - Every dependency visible in constructors, every middleware applied manually, every lifecycle hook opt-in.

## What's Included

- **Lifecycle management** - Component startup, shutdown, and route registration with automatic discovery
- **Authentication** - User signup, signin, sessions, password hashing, and middleware for protected routes
- **Structured logging** - Context-aware logging with multiple output formats
- **Configuration** - Load from YAML, environment variables, and command-line flags
- **Database** - Postgres connection pooling, migrations, transactions, and health checks
- **HTTP middleware** - Request tracking, logging, panic recovery, and CORS
- **Model utilities** - ID generation, timestamp helpers, and common patterns
- **Templates** - HTML rendering with embedded assets and HTMX support
- **Type-safe queries** - Integration with sqlc for compile-time SQL validation
- **Testing patterns** - Comprehensive test coverage with practical examples

## Architecture

HatMax applications follow a straightforward pattern:

```
HTTP Request
    ↓
Handler (chi.Router)
    ↓
Service Layer (business logic)
    ↓
sqlc Queries (type-safe SQL)
    ↓
Postgres
```

### Direct Data Access

Services use `sqlc`-generated code directly. Write SQL queries, generate type-safe Go code, and call those methods from your service layer.

```go
type UserService struct {
    queries *sqlc.Queries
    log     log.Logger
}

func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
    return s.queries.GetUserByID(ctx, id)
}
```

### Clear Dependencies

Constructor signatures reveal what each component needs to function.

```go
func NewUserHandler(
    authSvc *AuthService,
    userSvc *UserService,
    tmpl *TemplateManager,
    cfg *Config,
    log log.Logger,
) *UserHandler
```

### Lifecycle Discovery

Components implement interfaces to declare their capabilities. The `Setup` function discovers these capabilities, and `Start` orchestrates initialization with rollback on failure.

```go
type Startable interface { Start(context.Context) error }
type Stoppable interface { Stop(context.Context) error }
type RouteRegistrar interface { RegisterRoutes(chi.Router) }

// In main.go
database := db.New(assetsFS, "postgres", cfg, logger)
authService := auth.NewService(database, cfg, logger)
userHandler := user.NewHandler(authService, assetsFS, cfg, logger)

deps := []any{database, authService, userHandler}
starts, stops, registrars := app.Setup(ctx, router, deps...)
app.Start(ctx, logger, starts, stops, registrars, router)
```

## Persistence

HatMax follows the "Just Use Postgres" philosophy, leveraging its capabilities to handle most application needs:

- **Relational data** - Standard tables and foreign keys
- **Semi-structured data** - JSONB columns with indexing
- **Full-text search** - `tsvector` and `pg_trgm`
- **Pub/sub** - `LISTEN`/`NOTIFY` for real-time updates
- **Queues** - Advisory locks + queue tables for background jobs
- **Geospatial** - PostGIS extension for location data
- **Time-series** - TimescaleDB extension for metrics

Queries are written in SQL and type-checked with `sqlc`. This keeps the data layer explicit and maintainable. As your application grows, you can integrate specialized tools where they provide clear value.

## Scope

HatMax is focused on single-binary applications with straightforward patterns:

- **Database** - Postgres-first approach with sqlc for type-safe queries
- **Architecture** - Monolithic applications that deploy as one executable
- **Presentation** - HTML templates with HTMX for interactive experiences
- **Data layer** - Direct SQL queries, explicit service coordination
- **Authorization** - Core auth primitives as building blocks
- **Interface** - Standard HTTP handlers and middleware

## Gallery

See the [Gallery](docs/gallery.md) for visual examples of the HatMax design system and the ticked reference implementation.

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage by package
make test-coverage

# Display coverage table by package
make test-coverage-summary

# Generate HTML coverage report
make test-coverage-html

# Run all quality checks (format, vet, test, coverage, lint)
make check
```

## Status

HatMax is under active development. The library API is stabilizing but may change before v1.0. Contributions and feedback are welcome.

**Current focus:**
- Core library packages (app, log, config, db, auth, middleware, model)
- Authentication primitives (signup, signin, sessions, middleware)
- Comprehensive test coverage and examples
- Example application demonstrating all features

**Roadmap:**
- v1.0: Stable library API with full test coverage
- v1.x: Optional code generator for scaffolding new projects
- Future: Advanced features (rate limiting, background jobs, file uploads)

## License

MIT
