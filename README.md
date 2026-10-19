<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# HatMax

<p align="center">
  <img src="assets/img/docs/brand/hero.png" width="800">
</p>

[![Go Reference](https://pkg.go.dev/badge/hatmax.adrianpk.com.svg)](https://pkg.go.dev/hatmax.adrianpk.com)
[![CI](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/ci.json)](https://codeberg.org/hatmax/hatmax)
[![coverage](https://img.shields.io/endpoint?url=https://codeberg.org/hatmax/hatmax/raw/branch/main/.badges/coverage.json)](https://codeberg.org/hatmax/hatmax)

**A Go toolkit for server-rendered web applications.**

HatMax provides primitives for common application needs:

- Start and stop components, load configuration, and manage runtime settings.
- Authenticate users and manage sessions, passkeys, and account recovery.
- Store application data in PostgreSQL and apply migrations.
- Render pages, handle forms, validate input, and add HTMX interactions.
- Send email, schedule background work, and publish events.
- Store images and private documents.

These primitives keep infrastructure choices separate from business logic.
For example, an application can move from SMTP to a mail delivery service
without changing how its business logic sends messages. Event delivery and
job persistence follow the same approach.

The `hm` application builder creates and evolves HatMax applications from
conversational requests. It presents a plan for approval before changing files.

## Docs

- [User Guide](docs/tutorials/user-guide/README.md)
- [Bootstrap an Application](docs/how-to/bootstrap-application/README.md)
- [Assisted Generation](docs/tutorials/user-guide/assisted-generation.md)
- [Documentation index](docs/README.md)

## Repos

- [Official repository](https://forge.adrianpk.com/hatmax/hatmax)
- [Codeberg mirror](https://codeberg.org/hatmax/hatmax)
- [GitHub mirror](https://github.com/hatmaxkit/hatmax)

## License

Copyright 2026 Adrian PK. Licensed under the Apache License, Version 2.0
(`Apache-2.0`).

See [LICENSE](LICENSE).
