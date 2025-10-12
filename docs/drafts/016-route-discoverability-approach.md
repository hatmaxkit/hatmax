# Route Discoverability Approach

**Draft Type:** Technical Design  
**Date:** 2025-10-12  
**Status:** Draft

## Overview

This document explores the implementation of route discoverability features for microservices, enabling automatic API documentation, service catalog integration, and dynamic introspection capabilities.

## Problem Space

Microservices often need self-discovery endpoints to expose their available routes for documentation, monitoring, or coordination by other services. This is especially useful in environments where dynamic composition or introspection is required (e.g., service catalogs, test harnesses, admin UIs).

Manually tracking routes is error-prone, and frameworks like Chi already provide route introspection. The solution must remain framework-agnostic but integrate smoothly with Chi or net/http.

## Proposed Solution

Introduce **route discoverability** as an optional feature that publishes all known endpoints, including internal or "virtual" routes, under a well-known path (e.g., `/.well-known/routes`).

Two implementations are proposed:
1. Generic version for `net/http`
2. Optimized version for `chi.Router` using `chi.Walk`

## Implementation Approaches

### net/http Implementation

#### Core Concepts
- Register metadata for each route at handler registration time
- Optionally register "virtual routes" that are not mounted but should appear in discovery (useful for internal APIs or planned endpoints)

#### Integration Pattern

```go
func WithRoutesDiscovery(cfg ...func(*discoverCfg)) Option
```

Default configuration publishes `/.well-known/routes` with all exposed and virtual routes.

#### Usage Example

```go
svc := micro.New(
  ":8080",
  micro.WithPing(),
  micro.WithRoutesDiscovery(),
)

svc.AddVirtualRoute("POST", "/v1/orders", "Create order", false, "orders", "v1")
svc.AddVirtualRoute("GET",  "/v1/orders/{id}", "Get order", false, "orders", "v1")
```

#### Expected Output

```json
[
  {"method": "GET", "path": "/ping", "exposed": true},
  {"method": "POST", "path": "/v1/orders", "exposed": false, "summary": "Create order"}
]
```

### Chi Integration

For services using **Chi**, discovery is simplified via Chi's native introspection.

#### Integration Pattern

```go
func WithRoutesDiscoveryChi(path string) Option
```

This walks all registered routes and publishes them as JSON.

#### Usage Example

```go
svc := micro.New(
  ":8080",
  micro.WithRoutesDiscoveryChi("/.well-known/routes"),
)
svc.Router().Get("/ping", func(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
})
```

#### Expected Output

```json
[
  {"method": "GET", "pattern": "/ping"}
]
```

## Technical Features

### Route Metadata

The discovery system supports rich metadata for each route:

- **method**: HTTP method (GET, POST, etc.)
- **path/pattern**: Route path or pattern
- **exposed**: Whether the route is publicly available
- **summary**: Brief description of the route's purpose
- **internal**: Marker for internal-only routes
- **tags**: Categorization tags for grouping routes

### Virtual Routes

Virtual routes are a key feature for documenting planned or internal APIs:

- Routes that exist conceptually but are not yet implemented
- Internal routes that should be documented but not exposed
- Alternative representations of existing routes

### Framework Integration

#### Generic net/http Support
- Manual route registration with metadata
- Flexible configuration options
- Support for virtual routes
- Custom metadata fields

#### Chi Router Optimization
- Automatic route discovery via `chi.Walk`
- No manual registration required
- Leverages existing route definitions
- Minimal configuration overhead

## Use Cases

### Service Documentation
- Automatic API documentation generation
- Route catalog for developers
- Integration with API documentation tools

### Service Discovery
- Dynamic service catalog population
- Route inventory for load balancers
- Endpoint verification for monitoring

### Development Tools
- Test harness route enumeration
- Admin UI route discovery
- Development environment introspection

### Operational Monitoring
- Route health checking
- Endpoint availability tracking
- API surface area monitoring

## Implementation Considerations

### Security
- Filter sensitive routes in production environments
- Authentication/authorization for discovery endpoints
- Rate limiting for discovery endpoints

### Performance
- Minimal overhead for route registration
- Efficient route enumeration
- Caching of discovery responses

### Extensibility
- Plugin system for custom metadata
- Configurable output formats
- Integration hooks for external systems

## Technical Benefits

### Automatic Documentation
- Self-describing services
- Always up-to-date route information
- Reduced documentation maintenance

### Dynamic Integration
- External tooling can query routes programmatically
- Service mesh integration capabilities
- Monitoring system configuration

### Development Efficiency
- Simplified API exploration
- Reduced manual configuration
- Better tooling support

## Potential Challenges

### Security Exposure
- Possible exposure of sensitive routes
- Information leakage about internal structure
- Need for production filtering

### Maintenance Overhead
- Virtual routes must be kept current
- Metadata accuracy requirements
- Configuration complexity

### Performance Impact
- Route enumeration cost
- Memory usage for metadata storage
- Network overhead for discovery requests

## Configuration Options

### Discovery Endpoint Configuration
- Custom discovery path
- Output format selection (JSON, XML, etc.)
- Filtering rules for route inclusion/exclusion

### Metadata Configuration
- Required vs. optional fields
- Custom metadata schemas
- Validation rules for metadata

### Framework-Specific Options
- Chi-specific configuration
- net/http optimization settings
- Integration with other routers

## Expected Outcomes

A simple and optional discoverability layer compatible with both standard `net/http` and `chi.Router`, allowing microservices to:

- Publish their API surface automatically
- Support external tooling and monitoring
- Provide self-documentation capabilities
- Enable dynamic service discovery

## Next Steps

This approach will be refined through:

1. Prototype implementations for both net/http and Chi
2. Security review of discovery endpoint exposure
3. Performance testing with large route sets
4. Integration testing with external tooling
5. Feedback from initial service implementations

