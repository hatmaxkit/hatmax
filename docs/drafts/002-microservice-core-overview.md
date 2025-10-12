# Microservice Core Design Overview

**Draft Type:** Architectural Overview  
**Date:** 2025-10-12  
**Status:** Draft

## Overview

This document provides the overarching architectural vision for a microservice core library that unifies probes, route discovery, and dependency-based health management. This is a high-level exploration that ties together detailed technical approaches covered in companion drafts.

## Architectural Vision

Basic healthchecks often mislead: the process responds while critical dependencies are stalled. Each **microservice** composes domain **services**, **repos**, and **managers**. If a repo is down, the corresponding service is effectively down. 

The core library vision addresses:
- Standard probes suitable for **orchestration services** (e.g., Kubernetes, Nomad)
- Route discovery for self-description
- Health model based on dependencies with heartbeats, polling, TTL, criticality, and aggregation policies
- No "God objects": small interfaces and focused managers

## Core Library Architecture

The proposed architecture centers around a small Go base library that provides six integrated capabilities:

### 1. Optional Probes via Functional Options

Standardized health endpoints:
- `/ping` → 200 if the process responds
- `/livez` → 200 if liveness checks pass, 500 otherwise
- `/readyz` → 200 if ready, 503 otherwise
- `/healthz` → JSON aggregated from dependency states: `up | degraded | down` plus per-dep detail

*→ Detailed semantics covered in: [`004-health-endpoint-semantics.md`](004-health-endpoint-semantics.md)*

### 2. Checker Abstraction

Simple, composable health check foundation:
```go
type Checker func(ctx context.Context) error
```

*→ Complete interface design covered in: [`003-microservice-interfaces-contracts.md`](003-microservice-interfaces-contracts.md)*

### 3. Dependency Manager (DepManager) for Real Health

Two-pronged approach to dependency monitoring:
- **Heartbeater** (push): periodic `Heartbeat{At, Err}` with **TTL**
- **Checker** (poll): periodic `Check(ctx)` with `PollInterval` and timeout

Pluggable aggregation policies enable flexible health definitions.

*→ Full interface specifications in: [`003-microservice-interfaces-contracts.md`](003-microservice-interfaces-contracts.md)*

### 4. Parent → Child Health Rollup

Hierarchical health modeling:
- Model `Service -> Repo(s)` relationships
- Parent status derives from critical children
- Mark `Derived=true` for explicit rollup indication

### 5. Route Discoverability

Self-documenting API surface:
- **net/http**: manual route registration with rich metadata
- **Chi**: automatic discovery via `chi.Walk`
- Support for "virtual routes" (planned/internal endpoints)

*→ Complete implementation approach in: [`005-route-discoverability-approach.md`](005-route-discoverability-approach.md)*

### 6. Minimal Integration

Composable service configuration through functional options that enables clean dependency injection without framework lock-in.

*→ Complete integration patterns and examples in: [`006-microservice-integration-patterns.md`](006-microservice-integration-patterns.md)*

## Architectural Benefits

### Cohesive Health Model
- Unified approach to health and readiness
- Dependency-based health rather than simplistic pings
- Clear attribution of failures

### Self-Describing Services
- Automatic route documentation
- Standard probe semantics across all services
- Consistent operational interface

### Composable Design
- Small, focused interfaces
- Functional options for configuration
- No global state or God objects

## Key Architectural Decisions

1. **Dependency-Based Health Model**: Health reflects actual operational state rather than simplistic process checks

2. **Decentralized Architecture**: Each service manages its own dependencies without central coordination

3. **Composable Design**: Small, focused interfaces with functional options for flexible configuration

4. **Self-Describing Services**: Integrated route discovery and standardized probe semantics across all services

## Next Steps

This overview serves as the unifying architectural vision for the microservice core design. For detailed implementation aspects, see the component-specific drafts:

- [`003-microservice-interfaces-contracts.md`](003-microservice-interfaces-contracts.md) - Core interfaces and data contracts
- [`005-route-discoverability-approach.md`](005-route-discoverability-approach.md) - API discovery implementation
- [`004-health-endpoint-semantics.md`](004-health-endpoint-semantics.md) - Health endpoint standards
- [`006-microservice-integration-patterns.md`](006-microservice-integration-patterns.md) - Integration patterns and examples

These detailed drafts will inform the creation of formal ADRs once:
1. The service architecture path is better defined
2. We have validated the approach with initial implementations
3. We understand the specific requirements for Hatmax's microservice ecosystem

