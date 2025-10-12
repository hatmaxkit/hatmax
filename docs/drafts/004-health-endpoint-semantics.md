# Health Endpoint Semantics

**Draft Type:** Technical Standard  
**Date:** 2025-10-12  
**Status:** Draft

## Overview

This document explores the standardization of health endpoint semantics across all microservices to ensure consistent interpretation by orchestration systems, monitoring agents, and operational teams.

## Problem Space

Health endpoints must follow consistent semantics so orchestration systems, monitoring agents, and humans can interpret service state correctly. Simple "is alive" endpoints are insufficient for real observability. Each endpoint should reflect a distinct layer of readiness and responsibility.

## Proposed Standard

Standardize endpoint semantics across all microservices using consistent conventions for different types of health and readiness checks.

## Endpoint Specifications

### Standard Health Endpoints

| Endpoint     | Purpose | Typical HTTP Codes |
|---------------|----------|---------------------|
| `/ping`       | Verifies that the process responds. Does not imply readiness or health. | `200 OK` |
| `/livez`      | Indicates whether the service is alive and should not be restarted. Failing → restart. | `200` / `500` |
| `/readyz`     | Indicates readiness to receive traffic. Service may be alive but still initializing. | `200` / `503` |
| `/healthz`    | Provides full dependency-based health information (aggregated by DepManager). | `200` / `503` |

### Semantic Distinctions

#### `/ping` - Process Response
- **Purpose**: Basic connectivity test
- **Scope**: Process-level responsiveness only
- **Use Case**: Load balancer basic checks, network connectivity verification
- **Implementation**: Simple HTTP response, minimal logic

#### `/livez` - Liveness Check  
- **Purpose**: Determines if the service should be restarted
- **Scope**: Core service functionality
- **Use Case**: Container orchestration restart policies
- **Failure Response**: Service should be restarted

#### `/readyz` - Readiness Check
- **Purpose**: Determines if service can accept traffic
- **Scope**: Service initialization and startup dependencies
- **Use Case**: Load balancer traffic routing decisions
- **Failure Response**: Remove from traffic rotation, do not restart

#### `/healthz` - Comprehensive Health
- **Purpose**: Full operational health based on all dependencies
- **Scope**: Complete dependency tree and system state
- **Use Case**: Monitoring, alerting, operational dashboards
- **Implementation**: Requires full dependency aggregation

## Health Endpoint Implementation

### `/healthz` Detailed Behavior

The `/healthz` endpoint is the most sophisticated and requires full dependency management:

- Evaluated periodically via registered `Checker` or `Heartbeater` interfaces
- Aggregated through a `Policy` that defines acceptable ratios or criticality rules
- Output structured as JSON for easy consumption by monitoring systems

#### Response Structure

```json
{
  "status": "down",
  "items": [
    {"name": "db", "status": "down", "critical": true, "error": "timeout"},
    {"name": "UserService", "status": "down", "critical": true, "derived": true, "error": "child_critical_down"}
  ]
}
```

#### Status Mapping

| Status | Meaning | HTTP Code |
|---------|----------|-----------|
| `up` | All dependencies operational | `200 OK` |
| `degraded` | Non-critical dependencies failing | `503 Service Unavailable` |
| `down` | One or more critical dependencies failing | `503 Service Unavailable` |

### Response Format Standards

#### Success Responses
- **HTTP 200**: Service is fully operational
- **Content-Type**: `application/json` for structured endpoints
- **Response Time**: Should be fast (< 100ms) for basic checks

#### Failure Responses  
- **HTTP 500**: Internal service failure (liveness)
- **HTTP 503**: Service unavailable (readiness/health)
- **Error Details**: Structured error information when possible

## Advanced Features

### Streamed Health (`/health/stream`)

Optionally, `/health/stream` may be exposed for continuous health updates via **Server-Sent Events (SSE)**:

- Sends snapshot every few seconds
- Format: `event: snapshot\ndata: {...}\n\n`
- Useful for dashboards or lightweight observability tools

#### Stream Format Example

```
event: snapshot
data: {"status": "up", "items": [{"name": "db", "status": "up", "critical": true}]}

event: snapshot  
data: {"status": "degraded", "items": [{"name": "db", "status": "down", "critical": false}]}
```

### Health Check Configuration

#### Check Intervals
- **Basic checks** (`/ping`, `/livez`): Can be very frequent (1-5s)
- **Readiness checks** (`/readyz`): Moderate frequency (5-10s)  
- **Health checks** (`/healthz`): Based on dependency polling intervals

#### Timeout Configuration
- **Fast checks**: 100-500ms timeout
- **Complex checks**: 1-5s timeout
- **Dependency checks**: Configurable per dependency

## Integration Patterns

### Container Orchestration

#### Kubernetes Integration
```yaml
livenessProbe:
  httpGet:
    path: /livez
    port: 8080
  periodSeconds: 10
  
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  periodSeconds: 5
```

#### Docker Compose Integration
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
  interval: 30s
  timeout: 10s
  retries: 3
```

### Load Balancer Integration

#### Active Health Checks
- Use `/readyz` for traffic routing decisions
- Use `/ping` for basic connectivity
- Avoid `/livez` for load balancer checks (restart-oriented)

#### Passive Health Checks
- Monitor application-level error rates
- Complement active checks with response time monitoring

### Monitoring Integration

#### Metrics Collection
- Export health status as metrics
- Track check duration and success rates
- Monitor dependency failure patterns

#### Alerting Rules
- Alert on critical dependency failures
- Track service degradation patterns
- Monitor check availability and performance

## Implementation Considerations

### Performance Impact
- Keep basic checks (`/ping`, `/livez`) extremely lightweight
- Cache dependency states appropriately for `/healthz`
- Implement circuit breakers for expensive dependency checks

### Security Considerations  
- Consider authentication for detailed health endpoints
- Filter sensitive information from health responses
- Rate limit health check endpoints to prevent abuse

### Error Handling
- Graceful degradation when health checks fail
- Consistent error response formats
- Proper timeout handling for all checks

### Concurrency
- Health checks should be non-blocking
- Parallel dependency evaluation where possible
- Proper synchronization for shared state

## Operational Benefits

### Clear Separation of Concerns
- Distinct endpoints for different operational needs
- Predictable behavior across all services
- Standardized tooling integration

### Improved Observability
- Rich health information for debugging
- Historical health trend analysis
- Dependency relationship visibility

### Better Automation
- Standardized HTTP semantics across all services
- Consistent integration with orchestration platforms
- Reliable automated decision making

## Expected Outcomes

Consistent, machine-readable health endpoints that can be safely consumed by:
- Container orchestration systems (Kubernetes, Docker, Nomad)
- Load balancers and traffic management
- Monitoring and alerting systems
- Operational dashboards and tools
- Automated deployment and scaling systems

## Next Steps

This semantic standard will be validated through:

1. Implementation across initial microservices
2. Integration testing with orchestration platforms
3. Load testing of health check performance
4. Monitoring system integration validation
5. Operational feedback and refinement

