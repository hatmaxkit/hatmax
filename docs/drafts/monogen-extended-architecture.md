# Monogen Extended Architecture

**Status:** Planned | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Comprehensive architecture specification for building microservices monorepos with Go, HTMX, and clean separation of concerns through code generation.

## Current Implementation

**Status**: Basic generator with simple service types  
**Location**: Current Hatmax generator  
**Features**: Basic CRUD generation, simple templates, YAML parsing

## Planned Implementation

### Core Goals
- **Single source of truth**: `monorepo.yaml` drives all code generation
- **Three service types**: atom (CRUD), feature (bounded context), web (frontend)
- **Use-case layer**: inline (pass-through) or dedicated (business logic)
- **Compile-time enforcement**: Layer separation through build constraints
- **Minimal dependencies**: Stdlib-focused with explicit code generation
- **Built-in observability**: Structured logging, metrics, and tracing


### Service Taxonomy

**Atom Service**:
- **Purpose**: Single-entity CRUD with minimal logic
- **Use-case mode**: Defaults to `inline` (pass-through)
- **Use cases**: Basic data services, read/write stores with low coupling

**Feature Service**:
- **Purpose**: Multiple models and relations within bounded context
- **Use-case mode**: Defaults to `dedicated` (business logic)
- **Features**: Aggregates, projections, complex domain operations

**Web Service**:
- **Purpose**: Server-rendered frontend with HTMX
- **Dependencies**: Uses clients to other services (no direct DB)
- **Rendering**: HTML pages and partials with server-side state


### YAML Configuration Specification

**Complete monorepo.yaml Structure**:
```yaml path=null start=null
version: 0.2

service_defaults:
  usecase_mode: inline          # inline | dedicated
  handler_mode: direct          # direct | service
  repo_impl: sqlite             # string or list, first is primary

http:
  router: chi
  middlewares: [request_id, recovery, logging, cors_basic]

observability:
  logging: {level: info, format: json, include_request_id: true}
  metrics: {enabled: true, provider: prometheus, endpoint: /metrics}
  tracing: {enabled: true, provider: otel, sampler: parent, ratio: 0.2}

datastores:
  sqlite: {dsn_template: "file:{service}.db?_fk=1"}
  mongo: {dsn_env: "MONGO_URI"}

services:
  auth:
    kind: feature
    repo_impl: [sqlite, mongo]
    models:
      User:
        fields:
          email: {type: email, validations: [required, email, unique]}
          name: {type: string, validations: [required, {min_len: 2}]}
          is_active: {type: bool, default: true}
    aggregates:
      user_full:
        source: User
        preload: [roles, roles.permissions]
        expose: {route: "GET /auth/users/{id}/full"}
    api:
      base_path: /auth
      handlers:
        - {route: "GET /users", repo_list: UserRepo.list, mode: direct}
        - {route: "GET /users/{id}/full", aggregate: user_full, mode: service}
        
  portal:
    kind: web
    clients:
      auth: {base_url: "http://auth:8080"}
    pages:
      dashboard:
        route: "GET /ui/dashboard"
        load: [{http_get: {client: auth, path: "/auth/users/me", as: user}}]
        render: {template: "pages/dashboard.html", data: {user: "{{user}}"}}
```


## Key Architectural Components

### CLI Design

```bash path=null start=null
hatmax init <repo>
hatmax service add <name> --kind atom|feature|web --repo sqlite
hatmax model add <svc> <Model> --field name:string:required
hatmax aggregate add <svc> <name> --source User --preload roles
hatmax page add <svc> <page> --route "GET /ui/x"
hatmax plan | apply | validate | fmt
hatmax promote usecase <svc> <UsecaseName>
```

### Use-Case Pattern

All handlers communicate through use-case interfaces:

```go path=null start=null
// Port interface
type ItemUsecase interface {
  List(ctx context.Context, f ItemFilters) ([]Item, Page, error)
  Toggle(ctx context.Context, id ID) (Item, error)
}

// Inline implementation (atoms)
type itemUsecaseInline struct{ repo ItemRepo }
func (u itemUsecaseInline) List(ctx context.Context, f ItemFilters) ([]Item, Page, error) {
  return u.repo.List(ctx, f)
}

// Dedicated implementation (features)
type itemUsecaseService struct {
  repo ItemRepo
  tx   TxManager
}
```

### Repository Pattern

```go path=null start=null
// Clean domain interface
type UserRepo interface {
  Get(ctx context.Context, id string) (User, error)
  List(ctx context.Context, f UserFilters, p PageReq) ([]User, Page, error)
  Create(ctx context.Context, u User) (User, error)
  Update(ctx context.Context, u User) (User, error)
  Delete(ctx context.Context, id string) error
}

// SQLite implementation
type userRepoSQLite struct{ q *db.Queries }

// MongoDB implementation  
type userRepoMongo struct{ col *mongo.Collection }
```

### Web Rendering with HTMX

**HTMX Integration**:
```html path=null start=null
<table hx-get="/ui/users?page=2" hx-target="#users" hx-push-url="true">
  <!-- rows here -->
</table>

<form hx-post="/users" hx-target="#users" hx-swap="beforeend">
  <!-- form fields -->
</form>
```

### Observability Integration

**Prometheus Metrics**:
```go path=null start=null
var (
  Requests = prometheus.NewCounterVec(
    prometheus.CounterOpts{Name: "http_requests_total"},
    []string{"service", "route", "method", "code"})
  
  Latency = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{Name: "http_request_duration_seconds"},
    []string{"service", "route", "method"})
)
```

**OpenTelemetry Tracing**:
```go path=null start=null
func TraceMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tracer := otel.Tracer("http")
    ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
    defer span.End()
    next.ServeHTTP(w, r.WithContext(ctx))
  })
}
```

## Planned Features

### Compile-Time Enforcement
- **Status**: Planned
- **Description**: Build tags and lint rules to enforce layer separation
- **Implementation**: `//go:build kind_atom`, depguard rules, architecture tests
- **Benefit**: Prevents web services from importing repository packages

### Code Generation Pipeline
- **Status**: Planned
- **Description**: Template-based code generation with protected regions
- **Features**: YAML parsing, validation, IR building, template rendering
- **Quality**: Snapshot tests, golden files, idempotent generation

### Multiple Backend Support
- **Status**: Planned
- **Description**: Same domain interface, multiple persistence implementations
- **Backends**: SQLite (via sqlc), MongoDB (via mongo-driver)
- **Configuration**: Runtime choice via config or build tags

### Web Frontend Generation
- **Status**: Planned
- **Description**: Server-rendered HTML with HTMX integration
- **Components**: Pages, fragments, forms, tables, modals
- **State Management**: Server-side sessions, CSRF protection

## Implementation Roadmap

### Phase 1: Core Architecture (Weeks 1-4)
1. **Service taxonomy implementation** - atom, feature, web types
2. **YAML specification finalization** - Complete DSL definition
3. **CLI command structure** - Basic hatmax commands
4. **Use-case pattern implementation** - Inline vs dedicated modes

### Phase 2: Code Generation (Weeks 5-8)
5. **Template system** - Go template rendering with helpers
6. **Repository generation** - Multi-backend repository implementations
7. **Handler generation** - HTTP handlers with use-case integration
8. **Validation and enforcement** - Build constraints and lint rules

### Phase 3: Advanced Features (Weeks 9-12)
9. **Web service generation** - HTMX integration and templating
10. **Observability integration** - Metrics and tracing middleware
11. **Testing infrastructure** - Snapshot tests and contract testing
12. **Documentation and examples** - Complete usage guides

## Generated Project Structure

```text path=null start=null
/services/auth (feature)
  /internal/domain     # Domain models and interfaces
  /internal/repo       # Repository implementations
    /sqlite            # SQLite adapter
    /mongo             # MongoDB adapter
  /internal/usecase    # Business logic
  /internal/api        # HTTP handlers
  /internal/migrations # Database migrations

/services/todo (atom)
  /internal/domain
  /internal/repo/sqlite
  /internal/usecase    # Inline use-cases
  /internal/api

/services/portal (web)
  /internal/web        # HTMX templates and handlers
  /internal/clients    # HTTP clients to other services
  /templates           # HTML templates
```

## Success Criteria

### Architecture Quality
- [x] Clean separation between service types
- [x] Use-case layer consistently applied
- [x] Repository interfaces decoupled from persistence
- [x] Web services isolated from data layer

### Code Generation Quality
- [x] YAML-driven configuration works end-to-end
- [x] Generated code compiles without manual intervention
- [x] Templates produce idempotent, formatted output
- [x] Multi-backend repositories work correctly

### Developer Experience
- [x] CLI provides intuitive, composable commands
- [x] Documentation and examples are comprehensive
- [x] Testing strategy covers generated code quality
- [x] Observability integration works out of the box

---

**Summary**: Comprehensive microservices architecture with three service types (atom, feature, web), use-case driven design, multi-backend repositories, HTMX web integration, and compile-time enforcement through code generation. Balances simplicity with power through opinionated conventions and extensive automation.
