<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
-->

# Hatmax User Guide Companion

This application is the versioned companion for the
[Hatmax User Guide](../../docs/tutorials/user-guide/README.md). It demonstrates
configuration, lifecycle assembly, templates, HTMX, request validation,
runtime settings, optional Postgres records, and Postgres pubsub.

Run the application without infrastructure:

```sh
go run .
```

Run it with the database-backed note and event components:

```sh
GUIDE_DATABASE_ENABLED=true go run .
```

The database mode uses the connection in `config.yaml`. It creates
`guide_notes` and the pubsub tables. Use a disposable database when following
the guide.
