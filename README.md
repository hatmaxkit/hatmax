<p align="center">
  <img src="docs/img/hatmax.png" width="1200">
</p>

# HatMax

**HatMax** is a declarative Go service generator.
It builds a coherent monorepo of microservices from a single yaml based definition.

#### How It Works
HatMax parses the `hatmax.yml` file to understand the desired services, models, and APIs.
It then scaffolds the directory structure and generates the boilerplate code for each service, including models, repositories, handlers, and configuration.

A command-line interface will help to bootstrap and evolve projects.
It can generate the initial yaml template and provides guided commands to extend it in a structured way.
You can always edit it manually, but the CLI will ensure consistency as the system grows.

#### Approach
HatMax is built to help you reason clearly and maintain your systems over time, giving you lasting control of your code.

- **Declarative:** the architecture is defined in a single versioned file with no hidden conventions.
- **Standard library first:** everything starts with Go’s stdlib and code you own. When a well-established alternative exists, HatMax may offer it as an optional implementation, allowing you to choose the approach that best fits you.
- **Transparent code:** generated services are plain Go, readable and modifiable, with no hidden layers or reflection magic.


From a single declarative spec, HatMax produces:
- The full monorepo structure.
- Boilerplate for each service: models, repositories, handlers, and application layers
- A presentation layer that can be exposed as a web service.
- Shared operational components: logging, tracing, configuration, metrics, and health checks
- Deployment descriptors: Docker Compose, Nomad, or similar manifests
- Testing scaffolds for unit and integration tests

## Goals

For detailed information about HatMax's objectives, see [Goals](docs/goals.md).

## Misc

![Admin Dashboard](docs/img/gallery/admin-dashboard.png)

See more in the [Gallery](docs/gallery.md).
