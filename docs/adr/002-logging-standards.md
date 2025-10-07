# ADR-002: Logging Standards and Conventions

**Status:** Accepted
**Date:** 2025-10-03
**Deciders:** Adrian

## Context

Heterogeneous logging patterns across the codebase created cognitive fatigue and subtle bugs. Different ways of passing/naming loggers and other cross-cutting dependencies led to inconsistency. We need a clear, easy-to-apply logging policy that's verified by CI.

## Decision

### Logger Interface Design

**Ergonomic Logger interface** with slog compatibility:
```go
type Logger interface {
    Debug(v ...any)
    Debugf(format string, a ...any)
    Info(v ...any)
    Infof(format string, a ...any)
    Error(v ...any)
    Errorf(format string, a ...any)
    SetLogLevel(level LogLevel)
    With(args ...any) Logger
}
```

**Key principles:**
- **No `Warn` level** - use `Info` or `Error` based on criticality
- **Printf-style convenience** - `Infof`, `Debugf`, `Errorf` for formatted output
- **slog backing** - Implementation uses Go's standard `log/slog` package
- **Level control** - Explicit log level management
- **Structured logging** - `With()` for key-value pairs when needed
- **Ergonomic defaults** - Simple `Info("message")` for common cases

### Infrastructure Integration

**No embedding** for logging functionality. Instead:
- **Private field**: `log Logger` (exactly this name, no variations)
- **Public method**: `Log() Logger` for accessing the logger
- **Scope**: Only infrastructure types (handlers/services/repos/clients)

**Domain packages remain clean** - no logger imports or exposure.

### Constructor Pattern

**Unified dependency injection** via single `Deps` struct:
```go
type ServiceDeps struct {
    Log Logger
    Cfg Config
    // other required dependencies
}

func NewService(deps ServiceDeps) *Service {
    if deps.Log == nil {
        deps.Log = NewNopLogger()
    }
    return &Service{
        log: deps.Log,
        cfg: deps.Cfg,
    }
}
```

**Functional options** only for truly optional parameters, not core dependencies.

### Default Behavior

**NopLogger as fallback** - never panic on nil logger:
```go
type NopLogger struct{}
func (NopLogger) Debug(v ...any)                  {}
func (NopLogger) Debugf(format string, a ...any)  {}
func (NopLogger) Info(v ...any)                   {}
func (NopLogger) Infof(format string, a ...any)   {}
func (NopLogger) Error(v ...any)                  {}
func (NopLogger) Errorf(format string, a ...any)  {}
func (NopLogger) SetLogLevel(level LogLevel)      {}
func (NopLogger) With(args ...any) Logger         { return NopLogger{} }
```

All constructors default to `NopLogger` when `nil` is passed.

### Request-Scoped Logging

**Handler-level logger scoping** - no context pollution:
- Handlers create request-scoped loggers at the beginning of each request
- Pre-configured with `request_id`, `method`, `path` from middleware
- Passed explicitly to services when request context is needed
- Clean, explicit dependencies without context magic

```go
func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
    reqLogger := h.logForRequest(r)

    reqLogger.Info("creating item")

    item, err := h.service.CreateItem(r.Context(), req, reqLogger)
}

func (h *ItemHandler) logForRequest(r *http.Request) Logger {
    return h.log.With(
        "request_id", middleware.GetReqID(r.Context()),
        "method", r.Method,
        "path", r.URL.Path,
    )
}
```

### Output Format

**Environment-based formatting**:
- `LOG_FORMAT=text` - Human-readable format for development (TTY)
- `LOG_FORMAT=json` - Structured JSON for production/cloud

**Standard fields** across all log entries:
- `ts` - Timestamp
- `level` - Log level (info/error/debug)
- `service` - Service identifier
- `msg` - Log message
- `request_id` - Request correlation ID (when available)
- `route` - HTTP route (for web requests)
- `err` - Error details (when present)

### Audit Trail (Future Enhancement)

**Standard audit fields** injected by generator:
- `created_at`, `updated_at` - Timestamps
- `created_by`, `updated_by` - Actor identification
- `version` - Optimistic locking
- `deleted_at` - Soft delete (optional)

**AuditRepo decorator pattern**:
```go
type AuditRepo struct {
    inner UserRepo
    events EventStore
}
```

Central audit logic in `core/{audit,idgen,clock}` packages - generated repos delegate, don't reimplement.

## Enforcement Rules

### Verified by Custom Linter

**Infrastructure requirements**:
- All types in `services/**/(handlers|services|repos|clients)/` must have:
  - Field named exactly `log Logger` (not `logger`, `Logger`, `logr`)
  - Method `Log() Logger`

**Domain restrictions**:
- Types in `domain/**` cannot:
  - Import logging packages
  - Expose `Log()` methods
  - Have logger-related fields

### Naming Consistency

- Constructors: `NewXxx` (not `CreateXxx`)
- Repositories: `*Repo` suffix with canonical methods (`GetByID`, not `Find`)
- Logger field: exactly `log` (enforced by linter)

## Consequences

### Positive
- Homogeneous developer experience - always `X.Log()`
- Clean domain layer without infrastructure concerns
- Early detection of violations via CI
- Reduced cognitive load and debugging time
- Consistent structured logging across services

### Negative
- Requires custom analyzer implementation
- Additional boilerplate in infrastructure types
- Learning curve for team members

### Mitigation
- Comprehensive examples and documentation
- IDE snippets for common patterns
- Generator templates include correct patterns
- Clear error messages from custom linter

## Examples

### Service Implementation
```go
type UserService struct {
    log  Logger
    repo UserRepo
}

func NewUserService(deps UserServiceDeps) *UserService {
    if deps.Log == nil {
        deps.Log = NewNopLogger()
    }
    return &UserService{
        log:  deps.Log,
        repo: deps.Repo,
    }
}

func (s *UserService) Log() Logger {
    return s.log
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) error {
    logger := logctx.From(ctx).With("operation", "create_user", "email", req.Email)

    logger.Info("creating user")

    if err := s.repo.Create(ctx, user); err != nil {
        logger.Error("failed to create user", "error", err)
        return err
    }

    logger.Info("user created successfully", "user_id", user.ID)
    return nil
}
```

### Context Helper Usage
```go
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    logger := logctx.From(r.Context())
    logger.Info("handling request", "method", r.Method, "path", r.URL.Path)
}
```

---

**TL;DR**: Unified Logger interface, `log` field + `Log()` method in infrastructure, clean domain, NopLogger defaults, request-scoped logging via context, and CI-enforced conventions.
