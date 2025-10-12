# Microservice Interfaces and Contracts

**Draft Type:** Technical Design  
**Date:** 2025-10-12  
**Status:** Draft

## Overview

This document explores the design of minimal interfaces and explicit contracts for microservice components, enabling flexible composition without introducing "God objects" or framework dependencies.

## Problem Space

The microservice base must remain composable and not depend on global coordinators or large frameworks. Each feature — probes, dependency checks, route discovery — should be attached via small interfaces and options. This approach supports functional composition and isolation of responsibilities.

## Design Approach

Design minimal interfaces and explicit contracts for each subsystem, allowing flexible composition without introducing "God objects."

## Interface Specifications

### Probes and Functional Options

#### Core Abstraction

```go
type Checker func(ctx context.Context) error
```

This simple function signature serves as the foundation for all health and readiness checks.

#### Standard Options

```go
func WithPing() Option
func WithTimeout(d time.Duration) Option
func WithLiveness(checkers ...Checker) Option
func WithReadiness(checkers ...Checker) Option
func WithLivenessNamed(name string, c Checker) Option
func WithReadinessNamed(name string, c Checker) Option
func WithHealthNamed(name string, c Checker) Option
```

These options are composable and non-intrusive: adding `WithPing()` or `WithLivenessNamed()` only extends the service, never mutates shared global state.

#### Helper Functions

```go
func HTTPGET(url string, want int) Checker
func Func(name string, f func(ctx context.Context) error) (string, Checker)
```

These helpers provide common patterns for creating checkers from HTTP endpoints and arbitrary functions.

### Dependency Health Interfaces

#### Status Model

```go
type Status string
const (
  Up Status = "up"
  Degraded Status = "degraded"
  Down Status = "down"
)
```

A simple three-state model that captures the essential health states without over-complexity.

#### Core Data Structures

```go
type DepSnapshot struct {
  Name     string    `json:"name"`
  Status   Status    `json:"status"`
  LastSeen time.Time `json:"last_seen"`
  Error    string    `json:"error,omitempty"`
  Critical bool      `json:"critical"`
  Derived  bool      `json:"derived"`
}

type Heartbeat struct {
  Err error
  At  time.Time
}
```

The `DepSnapshot` serves as an immutable view of dependency state, while `Heartbeat` provides a push-based health reporting mechanism.

#### Interface Contracts

```go
type CheckerIF interface {
  Name() string
  Check(ctx context.Context) error
}

type Heartbeater interface {
  Name() string
  Heartbeat() <-chan Heartbeat
}

type DepConfig struct {
  Critical     bool
  TTL          time.Duration
  PollInterval time.Duration
}

type Policy interface {
  Decide(all []DepSnapshot) Status
}
```

Each dependency (repo, queue, external service) is modeled independently through these interfaces. The dependency manager aggregates their state through configurable policies.

### Aggregation Policies

```go
type AllCriticalMustPass struct{}
type RatioPolicy struct { Critical float64; Overall float64 }
```

Policies provide strategy-level control over what "healthy" means for a service. The interface allows for custom policies while providing sensible defaults.

### Dependency Manager Contract

```go
type DepID string

type DepManager struct {
  // Stores configs, snapshots, and parent→child relationships
}

func NewDepManager(p Policy) *DepManager
func (m *DepManager) AddHeartbeat(h Heartbeater, c DepConfig)
func (m *DepManager) AddChecker(ch CheckerIF, c DepConfig)
func (m *DepManager) Link(parent, child DepID)
func (m *DepManager) Snapshot() (overall Status, items []DepSnapshot)
```

The manager owns its state and concurrency; it exposes a snapshot model safe for external reads (e.g., for `/healthz`).

### Service Integration

```go
func WithDepManager(m *DepManager) Option
```

Attaches a dependency manager to the service, wiring `/healthz` and optionally `/health/stream` endpoints. This follows the same composable pattern as other service options.

## Design Principles

### Composability

- Small contracts, high composability
- Each manager handles one concern
- Easy to test or replace individual managers
- No hidden dependencies or global registries

### Isolation

- Each feature is independently testable
- No coupling between different concerns (probes, health, discovery)
- Clear ownership boundaries

### Flexibility

- Functional options pattern allows selective feature enablement
- Policy interface enables custom aggregation strategies
- Interface-based design supports testing and mocking

## Technical Benefits

### Clear Boundaries
- Predictable composition
- Flexibility to extend without rewriting existing logic
- Well-defined responsibilities for each component

### Testing Support
- Each interface can be easily mocked
- Components can be tested in isolation
- Integration testing is simplified through composition

### Extensibility
- New policies can be added without changing existing code
- Additional checker types can be implemented
- Service options can be extended independently

## Implementation Considerations

### Interface Design
- Keep interfaces minimal and focused
- Avoid leaking implementation details
- Prefer composition over inheritance
- Use functional options for optional features

### State Management
- Immutable snapshots for external consumption
- Internal state protected by appropriate synchronization
- Clear separation between configuration and runtime state

### Error Handling
- Consistent error reporting across all interfaces
- Context propagation for cancellation and timeouts
- Graceful degradation when dependencies fail

## Expected Outcomes

A flexible, testable foundation for microservice infrastructure that:
- Supports incremental adoption of features
- Maintains clear separation of concerns
- Enables easy testing and replacement of components
- Provides consistent interfaces across all services

## Next Steps

This interface design will serve as the foundation for implementing the microservice core library. Specific implementations will be validated through:

1. Prototype implementations of key interfaces
2. Integration testing with real dependencies
3. Performance validation under load
4. Refinement based on actual usage patterns

