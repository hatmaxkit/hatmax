<p align="center">
  <img src="docs/img/hatmax.png" width="600">
</p>

# HatMax

**HatMax** is a declarative Go service generator.
It builds a coherent monorepo of microservices from a single `hatmax.yml` definition.

#### How It Works
HatMax parses the `hatmax.yml` file to understand the desired services, models, and APIs.
It then scaffolds the directory structure and generates the boilerplate code for each service, including models, repositories, handlers, and configuration.

A command-line interface will help to bootstrap and evolve projects.
It can generate the initial `hatmax.yml` template and provides guided commands to extend it in a structured way.
You can always edit the YAML manually, but the CLI ensures consistency as the system grows.

#### Purpose
To keep evolving systems consistent without hiding complexity behind frameworks.
You describe what you need; HatMax generates code you own and understand.

#### Approach
HatMax focuses on clarity, ownership, and long-term maintainability.

- **Declarative, not prescriptive:** the architecture lives in a single versioned file. The generator reflects it faithfully instead of enforcing hidden conventions.
- **Standard library first:** everything starts with Go’s stdlib and code you own. When a well-established alternative exists, HatMax may offer it as an optional implementation, allowing you to choose the approach that best fits your environment.
- **Transparent code:** generated services are plain Go, readable and modifiable, with no hidden layers or reflection magic.


From a single declarative spec, HatMax produces:
- The full monorepo structure.
- Boilerplate for each service: models, repositories, handlers, and optional application layers
- Shared operational components: logging, configuration, metrics, and health checks
- Deployment descriptors: Docker Compose, Nomad, or similar manifests
- Testing scaffolds for unit and integration tests
