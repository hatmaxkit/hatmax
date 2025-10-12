# Microservice Integration Patterns

**Draft Type:** Technical Pattern  
**Date:** 2025-10-12  
**Status:** Draft

## Overview

This document explores typical integration patterns for microservices, focusing on how services orchestrate their dependencies including repositories, domain services, pub/sub managers, and external systems.

## Problem Space

Each microservice orchestrates its own dependencies: repositories, domain services, pub/sub managers, etc. The goal is to make this composition explicit while keeping startup logic minimal and declarative. Every dependency reports its own health through checkers or heartbeats, and the microservice only coordinates them.

## Proposed Integration Approach

Structure the main entrypoint so that:
- The `DepManager` manages health and aggregation
- Each dependency (repo, manager, external system) registers itself with the `DepManager`
- The microservice exposes probes, discovery, and health endpoints using options
- Dependencies can link hierarchically (e.g., `UserService -> UserRepo`) for derived health states

## Integration Pattern Example

### Basic Service Structure

```go
func main() {
  ctx := context.Background()

  // Dependency manager with aggregation policy
  dm := NewDepManager(AllCriticalMustPass{})

  // Core microservice configuration
  svc := micro.New(
    ":8080",
    micro.WithPing(),
    micro.WithTimeout(1500*time.Millisecond),
    micro.WithRoutesDiscoveryChi("/.well-known/routes"),
    micro.WithDepManager(dm),
  )

  // Example repository checker
  db := DBChecker{name: "db", Ping: func(ctx context.Context) error { return nil }}
  dm.AddChecker(db, DepConfig{Critical: true, PollInterval: 2*time.Second})

  // Example worker pool heartbeat
  workers := &WorkersHB{name: "workers", ch: make(chan Heartbeat, 4)}
  dm.AddHeartbeat(workers, DepConfig{Critical: false, TTL: 5*time.Second})

  // Service-level aggregation: Service -> Repo
  dm.Link(DepID("UserService"), DepID("db"))

  // Simulate periodic worker heartbeat
  go func() {
    for {
      workers.ch <- Heartbeat{At: time.Now()}
      time.Sleep(time.Second)
    }
  }()

  // Start the microservice
  _ = svc.Start(ctx)
}
```

## Integration Principles

### 1. Isolation
Each dependency encapsulates its own check logic:
- Database connections manage their own health checks
- Message queues handle their own connectivity verification
- External services define their own availability criteria
- No cross-dependency health logic

### 2. Declarativity
Dependencies are registered, not hard-coded:
- Configuration-driven dependency registration
- Clear separation between dependency definition and implementation
- Easy modification of health check parameters without code changes

### 3. Composability
Managers can be replaced or extended:
- Interface-based dependency management
- Pluggable health aggregation policies
- Modular health checking strategies
- Easy testing with mock dependencies

### 4. Transparency
`/healthz` reflects derived and base states clearly:
- Hierarchical health status representation
- Clear indication of derived vs. direct health states
- Detailed error information for failed dependencies
- Traceability of health status derivation

### 5. No Global App Object
Each microservice is self-contained and uses options for configuration:
- Functional options pattern for service configuration
- No shared global state
- Clean dependency injection
- Easy unit testing

## Dependency Management Patterns

### Repository Dependencies

```go
// Database connection health
type DBChecker struct {
    name string
    db   *sql.DB
}

func (d DBChecker) Name() string {
    return d.name
}

func (d DBChecker) Check(ctx context.Context) error {
    return d.db.PingContext(ctx)
}

// Registration
db := DBChecker{name: "postgres", db: dbConn}
dm.AddChecker(db, DepConfig{
    Critical: true, 
    PollInterval: 5*time.Second,
})
```

### Message Queue Dependencies

```go
// Queue heartbeat
type QueueHB struct {
    name string
    ch   chan Heartbeat
}

func (q *QueueHB) Name() string {
    return q.name
}

func (q *QueueHB) Heartbeat() <-chan Heartbeat {
    return q.ch
}

// Registration
queue := &QueueHB{name: "rabbitmq", ch: make(chan Heartbeat, 10)}
dm.AddHeartbeat(queue, DepConfig{
    Critical: true,
    TTL: 30*time.Second,
})
```

### External Service Dependencies

```go
// HTTP service checker
type HTTPChecker struct {
    name string
    url  string
}

func (h HTTPChecker) Name() string {
    return h.name
}

func (h HTTPChecker) Check(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", h.url, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    return nil
}

// Registration
authService := HTTPChecker{name: "auth-service", url: "http://auth:8080/ping"}
dm.AddChecker(authService, DepConfig{
    Critical: true,
    PollInterval: 10*time.Second,
})
```

### Service-Level Health Rollup

```go
// Link service health to its dependencies
dm.Link(DepID("UserService"), DepID("postgres"))
dm.Link(DepID("UserService"), DepID("rabbitmq"))
dm.Link(DepID("OrderService"), DepID("postgres"))
dm.Link(DepID("OrderService"), DepID("auth-service"))
```

## Service Configuration Patterns

### Minimal Configuration

```go
svc := micro.New(
    ":8080",
    micro.WithPing(),
    micro.WithDepManager(dm),
)
```

### Full-Featured Configuration

```go
svc := micro.New(
    ":8080",
    micro.WithPing(),
    micro.WithTimeout(2*time.Second),
    micro.WithLiveness(
        HTTPGET("http://auth:8080/ping", 200),
    ),
    micro.WithReadiness(
        Func("database", func(ctx context.Context) error {
            return db.PingContext(ctx)
        }),
    ),
    micro.WithRoutesDiscoveryChi("/.well-known/routes"),
    micro.WithDepManager(dm),
)
```

### Policy-Based Configuration

```go
// Strict policy: all critical dependencies must pass
dm := NewDepManager(AllCriticalMustPass{})

// Ratio-based policy: 80% of critical deps, 60% overall
dm := NewDepManager(RatioPolicy{
    Critical: 0.8,
    Overall:  0.6,
})
```

## Operational Benefits

### Manifest-Style Main File
The main file reads as a manifest of what the service depends on:
- Clear declaration of all dependencies
- Explicit health check configuration
- Visible service composition
- Easy operational understanding

### Consistent Diagnostics
Diagnostics are consistent across services:
- Standardized health response format
- Common dependency status representation
- Unified error reporting
- Predictable troubleshooting experience

### Evolutionary Architecture
Dependencies can evolve without changing the microservice contract:
- Interface-based dependency definition
- Configurable health policies
- Pluggable dependency implementations
- Non-breaking dependency updates

### Orchestration Integration
Works with orchestration systems expecting standard probes:
- Kubernetes liveness and readiness probes
- Docker health checks
- Nomad service health
- Load balancer health checks

## Configuration Management

### Environment-Based Configuration

```go
func configureDependencies(dm *DepManager) {
    // Database configuration
    if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
        db := connectDB(dbURL)
        checker := DBChecker{name: "database", db: db}
        dm.AddChecker(checker, DepConfig{
            Critical: true,
            PollInterval: 5*time.Second,
        })
    }

    // Queue configuration
    if queueURL := os.Getenv("QUEUE_URL"); queueURL != "" {
        queue := connectQueue(queueURL)
        dm.AddHeartbeat(queue, DepConfig{
            Critical: false,
            TTL: 30*time.Second,
        })
    }
}
```

### Service Discovery Integration

```go
func configureServiceDependencies(dm *DepManager) {
    services := discoverServices()
    for _, svc := range services {
        if svc.Critical {
            checker := HTTPChecker{
                name: svc.Name,
                url:  svc.HealthURL,
            }
            dm.AddChecker(checker, DepConfig{
                Critical: svc.Critical,
                PollInterval: svc.CheckInterval,
            })
        }
    }
}
```

## Testing Patterns

### Mock Dependencies

```go
type MockChecker struct {
    name   string
    err    error
}

func (m MockChecker) Name() string {
    return m.name
}

func (m MockChecker) Check(ctx context.Context) error {
    return m.err
}

// Test setup
func TestHealthEndpoint(t *testing.T) {
    dm := NewDepManager(AllCriticalMustPass{})
    
    // Add failing dependency
    dm.AddChecker(MockChecker{name: "test-db", err: errors.New("connection failed")}, 
        DepConfig{Critical: true, PollInterval: 1*time.Second})
    
    // Test health endpoint returns 503
    // ... test implementation
}
```

### Integration Testing

```go
func TestServiceIntegration(t *testing.T) {
    // Start test dependencies
    testDB := startTestDB()
    defer testDB.Close()
    
    // Configure real dependencies
    dm := NewDepManager(AllCriticalMustPass{})
    dm.AddChecker(DBChecker{name: "test-db", db: testDB}, 
        DepConfig{Critical: true, PollInterval: 1*time.Second})
    
    // Start service
    svc := micro.New(":0", micro.WithDepManager(dm))
    go svc.Start(context.Background())
    
    // Test health endpoints
    // ... integration tests
}
```

## Operational Trade-offs

### Benefits

#### Clear Service Manifest
- The main file shows exactly what the service depends on
- Health check intervals and criticality are explicit
- Easy to understand operational requirements
- Clear documentation of service boundaries

#### Operational Consistency
- Standardized health checking across all services
- Common troubleshooting procedures
- Predictable operational behavior
- Unified monitoring integration

#### Flexibility
- Easy to add or remove dependencies
- Configurable health aggregation policies
- Environment-specific dependency configuration
- Non-disruptive operational changes

### Challenges

#### Configuration Complexity
- Initial setup requires understanding of dependencies
- Health check tuning may require iteration
- Policy selection needs operational experience
- Multiple configuration points to manage

#### Performance Considerations
- Health check overhead scales with dependency count
- Network latency affects check performance
- Concurrent checking requires careful resource management
- Check frequency tuning impacts both accuracy and load

## Expected Outcomes

Each microservice remains self-contained yet introspectable:
- Dependencies report health via heartbeats or polling
- Route discovery is automatic
- All checks aggregate into a single coherent operational view
- Consistent behavior across all services
- Clear operational boundaries and responsibilities

## Next Steps

This integration pattern will be validated through:

1. Implementation across multiple microservice types
2. Load testing of health check performance
3. Operational feedback on configuration complexity
4. Integration testing with orchestration platforms
5. Refinement based on real-world usage patterns

