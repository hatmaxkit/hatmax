# Service Lifecycle Management

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Defines a simple lifecycle mechanism to initialize and shut down components without wrapping external libraries. Components implement optional interfaces for startup, shutdown, and route registration.

## Current Implementation

Implemented in the `hm` library with `Setup()`, `Start()`, and `Serve()` functions. Generated services use signal-based graceful shutdown and dependency injection via XParams.

## Design Principles

Minimal, opt-in contracts:
- Components implement `Start(ctx)` and `Stop(ctx)` if needed
- Handlers register routes via method calls or interfaces
- `hm.Setup()` discovers capabilities and orchestrates lifecycle
- `hm.Start()` handles startup sequence with rollback
- `hm.Serve()` provides graceful HTTP server management

### Goals
- Zero wrappers around libraries like chi
- Small, explicit interfaces
- Deterministic startup and LIFO shutdown
- Error handling with rollback on partial startup
- Signal-based graceful shutdown

## Current Implementation Example

Generated `main.go` structure:

```go
func main() {
	cfg, err := config.LoadConfig("config.yaml", "APP", os.Args)
	if err != nil {
		log.Fatalf("cannot load config: %v", err)
	}

	logger := hm.NewLogger(cfg.Log.Level)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	xparams := config.XParams{
		Log: logger,
		Cfg: cfg,
	}

	router := chi.NewRouter()

	var deps []any

	ItemRepo := sqlite.NewItemRepo(xparams)
	deps = append(deps, ItemRepo)

	ItemHandler := todo.NewItemHandler(ItemRepo, xparams)
	deps = append(deps, ItemHandler)

	starts, stops := hm.Setup(ctx, router, deps...)

	if err := hm.Start(ctx, starts, stops); err != nil {
		log.Fatal(err)
	}

	hm.Serve(router, hm.ServerOpts{Port: cfg.Server.Port}, logger)
}
```

## Lifecycle Contracts

### Optional Interfaces
```go
type Startable interface{ Start(context.Context) error }
type Stoppable interface{ Stop(context.Context) error }
type RouteRegistrar interface{ RegisterRoutes(chi.Router) }
```

### XParams Pattern
```go
type XParams struct {
	Log hm.Logger
	Cfg *Config
}
```

### Core Functions

**hm.Setup()** - Discovers capabilities and registers routes:
```go
func Setup(ctx context.Context, r chi.Router, comps ...any) (
    starts []func(context.Context) error,
    stops  []func(context.Context) error,
) {
    for _, c := range comps {
        if rr, ok := c.(RouteRegistrar); ok { rr.RegisterRoutes(r) }
        if s,  ok := c.(Startable); ok      { starts = append(starts, s.Start) }
        if st, ok := c.(Stoppable); ok      { stops  = append(stops,  st.Stop) }
    }
    return
}
```

**hm.Start()** - Executes startup sequence with rollback:
```go
func Start(ctx context.Context, starts, stops []func(context.Context) error) error {
    // Executes starts in order, rolls back on failure
}
```

**hm.Serve()** - Provides graceful HTTP server:
```go
func Serve(handler http.Handler, opts ServerOpts, logger Logger) {
    // Starts HTTP server with graceful shutdown
}
```

## Generated Handler Pattern

Generated handlers implement `RouteRegistrar` interface:

```go
type ItemHandler struct {
    svc     ItemService
    xparams config.XParams
}

func (h *ItemHandler) RegisterRoutes(r chi.Router) {
    r.Route("/items", func(r chi.Router) {
        r.Post("/", h.CreateItem)
        r.Get("/", h.ListItems)
        r.Get("/{id}", h.GetItem)
        r.Put("/{id}", h.UpdateItem)
    })
}
```

## Key Features

### Automatic Route Registration
Handlers implementing `RouteRegistrar` are automatically discovered by `hm.Setup()` and their routes are registered.

### Signal-based Shutdown
Generated services use `signal.NotifyContext()` for graceful shutdown on SIGINT/SIGTERM.

### Dependency Injection
Components receive dependencies through the `XParams` pattern containing logger and configuration.

### Error Handling
`hm.Start()` executes startup functions in order and rolls back on failure.

### HTTP Server Management
`hm.Serve()` handles HTTP server lifecycle with graceful shutdown.

---

**Summary**: Simple lifecycle management with optional interfaces. Generated services use `hm.Setup()` for route registration and dependency discovery, `hm.Start()` for controlled startup, and `hm.Serve()` for HTTP server management with graceful shutdown.
