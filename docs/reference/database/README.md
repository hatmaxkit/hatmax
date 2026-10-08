<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Database

`db` opens a Postgres connection and applies SQL migrations. The
implementation note is [db/readme.md](../../../db/readme.md).

`Postgres` is the engine name `"postgres"`.

## Connection

`New(assets, engine, cfg, log)` stores the filesystem, the engine name,
`cfg.Database.ConnectionString()`, and `cfg.Database.Schema`. It does not
open a connection.

`Start` opens the connection with the `pgx` driver and pings it. A ping
failure closes the connection and returns `cannot ping database`. A successful
ping stores `*sql.DB` and, when `Schema` is non-empty, runs
`CREATE SCHEMA IF NOT EXISTS` with that name quoted through `pgx.Identifier`.
The connection's `search_path` selects the same literal schema on each pooled
connection. Configured names are not SQL or search-path expressions; mixed
case and embedded quotes are preserved. PostgreSQL identifier length limits
and schema privileges still apply. No existing schema is renamed or migrated.
If an existing deployment relied on unquoted case folding, configure its actual
schema name before upgrading rather than expecting an automatic rename.
A schema error returns `cannot ensure schema` and leaves the connection open.
`Stop` closes the connection when one is stored, and returns nil otherwise.
`GetDB` returns the stored connection, or nil before a successful ping.

After a successful ping, `GetDB` is set before schema creation. A schema error
therefore also leaves a non-nil pool. The caller must call `Stop` after such a
failed start. `app.Start` rolls back only earlier successful components; the
failed component owns its partial-start cleanup. A wrapper can close this pool
before returning that error. `Stop` does not clear the stored pointer; a closed
pool is not a new live connection. Start once per lifecycle and do not overwrite
an open pool by starting it again.

## Migrations

`Migration` has `Datetime`, `Name`, `Up`, and `Down`.

`NewMigrator(provider, assets, engine, log)` stores those values. `SetPath`
replaces the default directory. The default directory is
`assets/migration/<engine>`.

`Start` reads `provider.GetDB()`. A nil connection returns `database
connection not available`. `Stop` returns nil.

A migration file ends in `.sql`. Its name is `<datetime>-<name>.sql`. A name
without `-` is an error. Sections begin with `-- +migrate Up` and
`-- +migrate Down`. Files are ordered by `Datetime`.

The applied set is the `migrations` table: `id`, `datetime`, `name`, and
`created_at`. A migration is pending when `datetime` concatenated with `name`
is absent from that table. An empty `Up` section fails that migration. Each
pending migration runs in its own transaction: the `Up` SQL, then an insert
that calls `gen_random_uuid()`. A failure rolls that transaction back and
stops the remaining files. `Down` is parsed and is not executed.

The datetime is a string sorted lexicographically; there is no datetime-format
validation or content checksum. Use fixed-width unique prefixes. Equal prefixes
have no supported tie-order contract. The tracking key concatenates datetime
and name without a separator, so choose unambiguous names and never rename an
applied file to request a replay. Editing its SQL leaves it applied.

The tracking table is created before files are loaded, outside a migration's
transaction. Earlier successful migrations remain committed if a later one
fails. The runner supplies no cross-process migration lock; coordinate one
migration owner at startup. It does not drop the selected schema or pool.
`gen_random_uuid()` must be available (built in on current PostgreSQL).
