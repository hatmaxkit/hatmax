<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Test Helper

`testhelper` builds a Postgres database for a Go test. The implementation
note is [testhelper/readme.md](../../../testhelper/readme.md).

`SetupTestDB(t)` returns a `*sql.DB`, a schema name, and a cleanup function.
When `DB_HOST` is set, it uses that server. `DB_PORT` defaults to `5432`,
`DB_USER` and `DB_PASSWORD` default to `postgres`, and `DB_NAME` defaults to
`postgres`. The schema name starts with `test_`, the current Unix time in
nanoseconds, and 8 random characters. Cleanup drops that schema.
The server role needs permission to create/drop schemas in the selected
database. Every pooled connection receives its schema through `search_path`.
Cleanup closes the returned pool first and attempts `DROP SCHEMA ... CASCADE`
through another connection. Open/drop errors are ignored on that path; confirm
removal when testing isolation. Use only disposable test servers.

When `DB_HOST` is empty, `SetupTestDB` starts a Postgres testcontainer and
returns its database using schema `public`. This path starts `postgres:16-alpine`
and requires a functioning container runtime. Cleanup closes the pool and
terminates only that container; termination errors are logged through the test.

`SetupTestDBWithConfig(t)` returns a `*config.Config` pointed at the same
kind of database, plus a cleanup function.
It creates no consumer-owned pool. Close pools you open from the configuration
before invoking cleanup. Register the supplied cleanup with `t.Cleanup` even
when a test can fail early. Setup uses background contexts; test command timeouts
bound the process rather than individual setup operations.

`TestLogger` returns `log.NewTestLogger("error")`.
