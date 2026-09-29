# app

Lifecycle management for application components.

## Usage

```go
// Components implement optional interfaces
type MyService struct{}

func (s *MyService) Start(ctx context.Context) error { return nil }
func (s *MyService) Stop(ctx context.Context) error { return nil }
func (s *MyService) RegisterRoutes(r chi.Router) {
    r.Get("/health", s.Health)
}

// Setup discovers capabilities automatically
starts, stops, registrars := app.Setup(ctx, router,
    dbPool,
    &myService,
    &anotherService,
)

// Start executes in order and uses the aligned stop slice on failure.
if err := app.Start(ctx, log, starts, stops, registrars, router); err != nil {
    log.Fatal(err)
}

// Shutdown in reverse order (LIFO)
app.Shutdown(srv, log, stops)
```

## API

```go
type Startable interface {
    Start(context.Context) error
}

type Stoppable interface {
    Stop(context.Context) error
}

type RouteRegistrar interface {
    RegisterRoutes(chi.Router)
}
```

Components implement the interfaces they need. Setup inspects and groups them.

`Setup` collects start and stop functions into independent slices. If an
application relies on startup rollback, every ordered startup component must
also implement `Stoppable` so those slices stay positionally aligned. See the
[Application Lifecycle Reference](../docs/reference/application-lifecycle/README.md)
for the exact failure behavior.
