# HatMax

`HatMax` is a Go-based monorepo generator. It's designed to build a monorepo of microservices from a single `hatmax.yml` definition file.

## How it Works

The generator parses the `hatmax.yml` file to understand the desired services, models, and APIs. It then scaffolds the directory structure and generates the boilerplate code for each service.

## Purpose

HatMax provides a declarative scaffold for microservice ecosystems that evolve over time. Rather than managing services that grow organically without structure, you maintain a clear configuration that reflects your current architecture and can be iterated upon.

The generator approach means you get straightforward Go code that you understand and control, rather than being locked into framework conventions. Services are generated with production concerns in mind, including operational transparency, dependency management, and integration patterns that work well in container environments.

## Approach

Microservices need structure to remain manageable as they evolve. By maintaining a declarative view of your architecture, you can iterate while keeping the system coherent. The monorepo model supports this by making cross-service changes explicit and reviewable.
