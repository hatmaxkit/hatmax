# Middleware Configuration and Generation

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

HTTP middlewares handle cross-cutting concerns like logging, recovery, request tracking, and rate limiting. This document defines declarative middleware configuration in YAML and automatic generation of Chi middleware setup code.

## Current Implementation

Currently, middleware setup is handled manually in generated code with basic defaults. No declarative YAML configuration exists yet.

## Planned Implementation

Declarative middleware configuration with sensible defaults, service-specific overrides, and automatic code generation.

## Current Chi Middlewares

### Essential/Safety Middlewares
- **Recoverer** - Catches panics and prevents server crashes
- **RequestID** - Generates/injects request ID into context
- **RealIP** - Adjusts RemoteAddr based on X-Forwarded-For, X-Real-IP headers
- **Logger** - Automatic request/response logging with timing and status

### Performance/Control Middlewares
- **Timeout** - Cancels requests that exceed specified duration
- **Throttle** - Limits concurrent requests (connection-based)
- **StripSlashes** - Removes redundant double slashes from URLs

### Utility Middlewares
- **Heartbeat** - Health check endpoint (e.g., /ping)
- **GetHead** - Allows GET routes to also respond to HEAD requests
- **RouteHeaders** - Conditional routing based on HTTP headers

## Proposed YAML Configuration

### Global Middleware Configuration
```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"
middlewares:
  global:
    # Always enabled by default (safety-first approach)
    defaults: [recoverer, request_id, real_ip, logger]

    # Optional middlewares with configuration
    timeout: 30s
    throttle: 100
    strip_slashes: true
    heartbeat:
      path: /health
      response: "OK"
    get_head: true

  # Service-specific middleware overrides
  service_overrides:
    # Disable certain middlewares for specific services
    disable: []
    # Add service-specific middlewares
    additional: []

services:
  todo:
    kind: atom
    # Service can override global middleware config
    middlewares:
      timeout: 10s  # Override global timeout
      throttle: 50  # Override global throttle
      additional: [route_headers]  # Add service-specific middleware
      disable: [logger]  # Disable logger for this service

    models:
      Item:
        # ... model definition

    api:
      base_path: /todo
      # Handler-specific middleware (if needed)
      middlewares:
        - path: "/items/bulk"
          additional: [timeout: 60s]  # Longer timeout for bulk operations

      handlers:
        - id: todo_items_create
          route: "POST /items"
          source: repo
          model: Item
          op: create
          # Handler-specific middleware overrides
          middlewares:
            throttle: 10  # More restrictive for creates
```

### Alternative Simplified Configuration
```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"

# Simple global defaults
middlewares: [recoverer, request_id, logger, timeout:30s, throttle:100]

services:
  todo:
    middlewares: [heartbeat:/health]  # Add to defaults
    # or
    middlewares:
      exclude: [logger]  # Remove from defaults
      include: [heartbeat:/health]  # Add custom
```

### Minimal Configuration (Defaults Only)
```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"
# If no middlewares section, use sensible defaults

services:
  todo:
    # ... service definition
    # Implicit middlewares: [recoverer, request_id, real_ip, logger]
```

## Default Middleware Strategy

### Tier 1: Always Enabled (Safety)
- **Recoverer** - Never disabled, prevents crashes
- **RequestID** - Essential for tracing and logging correlation
- **RealIP** - Important for accurate client IP in logs/metrics

### Tier 2: Enabled by Default (Recommended)
- **Logger** - Request/response logging (can be disabled for high-throughput services)
- **StripSlashes** - URL normalization (minimal overhead)

### Tier 3: Optional (Configurable)
- **Timeout** - Application-specific (needs duration config)
- **Throttle** - Application-specific (needs limit config)
- **Heartbeat** - Environment-specific (needs path config)
- **GetHead** - API design choice
- **RouteHeaders** - Special use cases only

### Tier 4: Never Default (Explicit Only)
- **RouteHeaders** - Too specific, needs explicit routing rules

## Generation Rules

### Middleware Order
Middlewares are applied in a specific order for optimal functionality:

1. **Recoverer** (outermost - catches everything)
2. **RealIP** (early - needed for accurate logging)
3. **RequestID** (early - needed for logging correlation)
4. **Logger** (after ID/IP setup)
5. **Timeout** (request-level)
6. **Throttle** (connection-level)
7. **StripSlashes** (URL normalization)
8. **GetHead** (route behavior)
9. **Heartbeat** (special routes)
10. **RouteHeaders** (routing logic)
11. **Application handlers** (innermost)

### Code Generation Patterns

**Router Setup:**
```go
func (h *TodoHandler) Routes() chi.Router {
    r := chi.NewRouter()

    // Global middlewares (generated based on config)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Timeout(30 * time.Second))
    r.Use(middleware.Throttle(100))

    // Service-specific middlewares
    r.Use(middleware.Heartbeat("/health"))

    // Route definitions
    r.Post("/items", h.CreateItem)
    r.Get("/items", h.ListItems)

    return r
}
```

**Handler-Specific Middlewares:**
```go
// For handlers that need different middleware config
r.Route("/items", func(r chi.Router) {
    r.Use(middleware.Timeout(60 * time.Second)) // Override for this group
    r.Post("/bulk", h.BulkCreateItems)
})
```

## Configuration Schema

### Middleware Definition
```go
type MiddlewareConfig struct {
    Global    GlobalMiddlewares    `yaml:"global"`
    Overrides ServiceOverrides    `yaml:"service_overrides,omitempty"`
}

type GlobalMiddlewares struct {
    Defaults       []string          `yaml:"defaults"`
    Timeout        string            `yaml:"timeout,omitempty"`        // "30s", "2m"
    Throttle       int               `yaml:"throttle,omitempty"`       // concurrent connections
    StripSlashes   bool              `yaml:"strip_slashes,omitempty"`
    Heartbeat      *HeartbeatConfig  `yaml:"heartbeat,omitempty"`
    GetHead        bool              `yaml:"get_head,omitempty"`
}

type HeartbeatConfig struct {
    Path     string `yaml:"path"`                    // "/health", "/ping"
    Response string `yaml:"response,omitempty"`      // "OK", custom response
}

type ServiceMiddlewares struct {
    Timeout    string   `yaml:"timeout,omitempty"`
    Throttle   int      `yaml:"throttle,omitempty"`
    Additional []string `yaml:"additional,omitempty"`
    Disable    []string `yaml:"disable,omitempty"`
}
```

### Validation Rules
1. **Timeout format** - Must be valid Go duration (e.g., "30s", "2m30s")
2. **Throttle limits** - Must be positive integer
3. **Middleware names** - Must match supported chi middlewares
4. **Path format** - Heartbeat path must start with "/"
5. **Conflict detection** - Cannot disable and add same middleware

## Future YAML Scaffolding Tool

The complexity of middleware configuration suggests value in a scaffolding tool:

### Interactive Setup
```bash
hatmax init --interactive

? Service type: [atom, aggregate, gateway]
? Middlewares:
  ✓ Recoverer (safety - always enabled)
  ✓ RequestID (tracing - recommended)
  ✓ RealIP (logging - recommended)
  ✓ Logger (debugging - recommended)
  ? Timeout: [none, 30s, 1m, 2m, custom]
  ? Rate limiting: [none, 100 req/s, 1000 req/s, custom]
  ? Health check: [none, /health, /ping, custom]

Generated hatmax.yaml with recommended middleware configuration.
```

### Template-Based Generation
```bash
hatmax init --template api-gateway
# Generates with: timeout, throttle, heartbeat, logger, rate limiting

hatmax init --template microservice
# Generates with: recoverer, request_id, logger, basic timeout

hatmax init --template high-performance
# Generates with: minimal middlewares, custom throttling
```

## Next Steps

### Immediate
1. **Basic YAML support** - Add middleware config to YAML schema
2. **Default middlewares** - Generate recoverer, request_id, logger setup
3. **Template updates** - Modify router generation for middleware chains

### Medium Term
4. **Service overrides** - Per-service middleware configuration
5. **Advanced middlewares** - Timeout, throttle, heartbeat support
6. **Handler-specific** - Per-route middleware overrides

### Long Term
7. **Interactive tool** - `hatmax init --interactive` for middleware setup
8. **Custom middlewares** - User-defined middleware support

## Benefits

### Developer Experience
- **Sensible defaults** - Projects work securely out of the box
- **Declarative configuration** - Infrastructure as code
- **No boilerplate** - Middleware setup automatically generated
- **Flexibility** - Override defaults when needed

### Operational Benefits
- **Consistent logging** - All services have request tracing
- **Crash protection** - Recoverer prevents service downtime
- **Performance controls** - Timeout and throttling built-in
- **Health monitoring** - Standardized health check endpoints

### Maintenance
- **Centralized config** - Middleware changes in one place
- **Version control** - Middleware configuration tracked with code
- **Easy updates** - Change YAML, regenerate, deploy

---

**Summary**: Declarative middleware configuration in YAML with sensible defaults (recoverer, request_id, logger), configurable timeouts/throttling, service-specific overrides, and proper middleware ordering for Chi-based services.
