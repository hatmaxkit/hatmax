<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Hatmax User Guide Companion

This application is the versioned companion for the
[Hatmax User Guide](../../docs/tutorials/user-guide/README.md). It demonstrates
configuration, lifecycle assembly, templates, HTMX, request validation,
runtime settings, optional Postgres records, and Postgres pubsub.

From the `examples/guide` directory, run the application without infrastructure:

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

The plain mode serves `/ping`, a full page at `/`, and the name-check partial
at `/name`. The application installs same-origin protection, so command-line
POST requests need a matching `Origin` or `Referer` header. For example, while
the process runs on port 8080:

```sh
curl -fsS http://localhost:8080/ping
curl -fsS -H 'Origin: http://localhost:8080' -d 'name=Alice' http://localhost:8080/name
```

The health body is `{"status":"ok"}`; the name partial contains `Accepted Alice`.
The current companion's `main` blocks in `app.Serve` and does not install a
signal-driven graceful-shutdown coordinator. The complete
[bootstrap procedure](../../docs/how-to/bootstrap-application/README.md) shows
how to retain the server and wait for `app.Shutdown` in a process that needs it.
