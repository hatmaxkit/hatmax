# Goals

Development is organized around two main phases: delivering core functionality first, then expanding capabilities to support more advanced use cases and deployment scenarios.

## Initial

- Declarative **monorepo generation** from a single `hatmax.yml` definition.
- Minimal but extensible **CLI** for bootstrapping projects and generating the monorepo structure.
- **Service generation** for atomic, domain (aggregated), and web layers.
- **Repository implementations** using SQLite and MongoDB, based on aggregate roots.
- Built-in **validation** for models and service definitions.
- **Business frontend service** (HTML + htmx) orchestrating API calls to expose core business functionality, acting as an internal API gateway.
- **Admin frontend service** (HTML + htmx) as a separate service for user management, roles, permissions, and system configuration.
- **Authentication and authorization** with roles and both global and contextual permissions.
- **Deployment support** for Docker Compose and Nomad.

## Future

- **Incremental code generation and synchronization**, allowing new services and handlers to be added without overwriting manually modified code.
- **PostgreSQL repository implementation** as an additional relational backend.
- **Enhanced CLI** for manipulating the declarative spec (`add service`, `add repo`, `add xxx`) while maintaining consistency between YAML and code.
- **Dual authorization system** distinguishing between internal service-to-service calls (from the web frontend) and external API access (SPAs, mobile apps).
- **gRPC inter-service communication** as a configurable alternative for service-to-service communication.
- **Asynchronous request processing** with producers, consumers, and pub/sub using pluggable interfaces, with NATS as the default implementation.
- **Kubernetes support** through basic Helm descriptor generation.

## Observability & Integrations

Important features for production-grade microservice orchestration, not yet prioritized for a specific development phase:

- **Tracing** - Request tracking across services (e.g. Jaeger).
- **Metrics & monitoring** - Collection and analysis of system metrics (e.g. Prometheus, OpenMetrics).
- **Visualization** - Data and metrics dashboards (e.g. Grafana, Kibana).
- **Observability analysis** - Correlation and observational analysis with pluggable interfaces.
- **Error tracking** - Detection and reporting of application errors with configurable backends.
- **Health monitoring** - Service availability and failure detection with configurable watchdog components.
- **Audit service** - Asynchronous microservice for tracking actions and changes across other microservices.
