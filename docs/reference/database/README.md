<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: GPL-3.0-only

This file is part of Hatmax. See COPYING for license terms.
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
`CREATE SCHEMA IF NOT EXISTS` with that name interpolated into the statement.
A schema error returns `cannot ensure schema` and leaves the connection open.
`Stop` closes the connection when one is stored, and returns nil otherwise.
`GetDB` returns the stored connection, or nil before a successful `Start`.

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
