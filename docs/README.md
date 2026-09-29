# Hatmax Documentation

Hatmax is a composable Go toolkit for building web applications. The User
Guide is the path from a first running process to an application that stores
data, signs in, does work outside the request, and reads a runtime setting.
Supporting guides explain focused tasks, package contracts, and the design
decisions that connect the packages.

## User Guide

The [User Guide](tutorials/user-guide/README.md) is the learning path. Follow
it in order when building a first application, or open one chapter when
returning to a step you already know.

Focused tutorials that are not part of the main learning path are listed in
the [Tutorials index](tutorials/README.md).

## How-to Guides

The [How-to Guides](how-to/README.md) are direct procedures for known goals,
including application setup, Postgres, migrations, authentication, mail,
background jobs, pubsub, image storage, and database-backed tests.

## Reference

The [Reference](reference/README.md) states exported contracts, options,
errors, limits, and lifecycle behavior. Start with
[Terminology](reference/terminology/README.md) when a Hatmax name is unfamiliar.

Package `readme.md` files remain next to the code. They are implementation
notes. They do not override exported code or the reference.

## Explanation

The [Explanation index](explanation/README.md) describes component ordering,
configuration boundaries, Postgres-first infrastructure, adapter ownership,
server-rendered HTMX, and failure boundaries.

## Additional Reference

- [Package Map](reference/package-map/README.md) summarizes the public package
  surface.
- [Gallery](reference/gallery/README.md) shows the example application.
- The project [changelog](../CHANGELOG.md) records release-worthy changes.
