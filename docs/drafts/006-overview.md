# HatMax: A Microservices Generator for Go

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## The Idea

**HatMax** is a code generator that takes a YAML file and generates complete microservices in Go. The idea is simple: you define what you want in YAML, and the generator creates all the code needed to make it work.

## Why This Exists

Microservices are powerful but creating each one from scratch is tedious. Every service needs:
- Data models
- Repository for persistence
- HTTP handlers
- Validations
- Logging
- Authentication
- Tests

HatMax automates all of this. You write a YAML, run the generator, and you have a complete, functional microservice.

## The Philosophy

### Less Framework, More Generated Code
Instead of using heavy frameworks that hide complexity, HatMax generates explicit, clean Go code. You can read, understand, and modify everything it generates.

### Single Source of Truth
Everything is defined in `hatmax.yaml`. No configuration scattered across multiple files. Change the YAML, regenerate, and you have a new version of the service.

### Natural Evolution
You start with simple services (`atom`) and evolve to complex services (`domain`, `composite`, `web`) as we need more functionality.

### Idiomatic Go
The generated code follows Go conventions. No magic, no unnecessary reflection, just clean, maintainable Go code.

## Service Types

### `atom` - The Building Block
A service that handles a single entity with CRUD operations and DDD aggregate support.

```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"
services:
  todo:
    kind: atom
    repo_impl: [sqlite, mongo]
    models:
      Item:
        fields:
          text: {type: text, validations: [required]}
          done: {type: bool, default: false}
    aggregates:
      List:
        fields:
          name: {type: string, validations: [required]}
        children:
          items: {of: Item}
```

Generates: models, aggregates, repositories, REST handlers, validations.

### `domain` - The Bounded Context
A service with multiple entities, relations, and complex business logic.

```yaml
version: 0.1
name: "auth-system"
package: "github.com/company/auth-system"
services:
  auth:
    kind: domain
    models:
      User: # ...
      Role: # ...
      Permission: # ...
    relations:
      - {from: User, to: Role, kind: many_to_many}
```

Generates: related models, aggregates, use cases, events.

### `web` - The Interface
A service that renders HTML with HTMX, consuming other services.

```yaml
version: 0.1
name: "portal-system"
package: "github.com/company/portal-system"
services:
  portal:
    kind: web
    pages:
      dashboard:
        route: "GET /dashboard"
        load:
          - http_get: {client: auth, path: "/users/me"}
        render: {template: "dashboard.html"}
```

Generates: pages, HTMX fragments, forms, templates.

## The DSL: `hatmax.yaml`

Everything is defined here. It's declarative and expressive:

```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"

services:
  todo:
    kind: atom
    repo_impl: [sqlite, mongo]  # Multiple backends
    
    models:
      Item:
        options:
          audit: true  # Automatic audit fields
        fields:
          text: {type: text, validations: [required]}
          done: {type: bool, default: false}
    
    api:
      base_path: /todo
      handlers:
        - {route: "GET /items", source: repo, model: Item, op: list}
        - {route: "POST /items", source: repo, model: Item, op: create}
        # ... more handlers
```

## What It Generates

### File Structure
```
examples/todo-app/
├── hatmax.yml            # Config replicated at monorepo root
└── services/todo/
    ├── go.mod
    ├── config.yaml
    ├── main.go
    ├── Makefile
    └── internal/
        ├── config/
        ├── todo/
        │   ├── item.go           # Domain model
        │   ├── list.go           # Aggregate root
        │   ├── itemrepo.go       # Repository interface
        │   ├── itemhandler.go    # HTTP handlers
        │   └── itemvalidator.go  # Validation functions
        ├── sqlite/
        │   ├── itemrepo.go       # SQLite implementation
        │   └── item_queries.go   # SQL queries
        └── mongo/
            └── itemrepo.go       # MongoDB implementation
```

### Clean and Consistent Code
- **Handlers** that follow the JSON envelope pattern
- **Validations** as pure functions (no tags)
- **Structured logging** with `slog`
- **Clean repository** interfaces
- **Automatically generated** tests

## Design Decisions

### Functional Validation
We don't use validation tags. Instead, we generate functions:

```go
func ValidateCreateItem(ctx context.Context, req CreateItemRequest) []ValidationError {
    var errors []ValidationError
    
    if req.Text == "" {
        errors = append(errors, ValidationError{
            Field: "text", Code: "required", Message: "Text is required",
        })
    }
    
    return errors
}
```

### Unified Logging
All services use the same logging interface:

```go
type Logger interface {
    Info(v ...any)
    Error(v ...any)
    With(args ...any) Logger
}
```

### Consistent API Responses
All responses follow the same format:

```json
// Success
{"data": {...}, "meta": {...}}

// Error
{"error": {"code": "...", "message": "...", "details": [...]}}
```

## The Workflow

1. **Define** your service in `hatmax.yaml`
2. **Run** `hatmax generate`
3. **Get** complete, functional Go code
4. **Modify** the YAML as your needs evolve
5. **Regenerate** the code

## Current Status

### Working
- Complete `atom` services
- DDD aggregates with children
- Model, repository, handler generation
- Functional validation
- Structured logging
- REST API with JSON envelope
- Multiple backends (SQLite, MongoDB)
- Monorepo structure with config replication

### In Development
- `domain` services with relations
- Authentication/authorization system
- `composite` and `web` services
- Configurable middleware
- Observability (metrics, tracing)

### Future
- Multi-service orchestration
- Events and messaging
- Automatic integration tests
- Automated deployment

## Why Go?

Go is perfect for microservices:
- **Simplicity**: Code that's easy to read and maintain
- **Performance**: Fast and efficient
- **Concurrency**: Natural handling of multiple requests
- **Ecosystem**: Excellent libraries for HTTP, databases, etc.
- **Deployment**: Static binaries, easy distribution

## Why YAML?

YAML is:
- **Readable**: Easy for humans to understand
- **Expressive**: Can represent complex structures
- **Versionable**: Can be tracked in Git
- **Declarative**: You describe what you want, not how to do it

## The Value

HatMax lets you:
- **Prototype fast**: From idea to working microservice in minutes
- **Maintain consistency**: All services follow the same patterns
- **Evolve gradually**: From simple to complex as you need
- **Focus on business**: Don't waste time on boilerplate
- **Stay in control**: The generated code is yours, you can modify it

## The Vision

HatMax helps you quickly prototype microservices in Go. You describe them declaratively, and it generates clear, modifiable code you can evolve as your needs grow. It’s not a framework, just a starting point that leaves the choices in your hands.
v

---

**TL;DR**: HatMax generates complete microservices in Go from a YAML file. Define what you want, get clean, functional code.
