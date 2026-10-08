<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Seed

`seed` skips recorded seeder names and records successful runs. The implementation note is
[seed/readme.md](../../../seed/readme.md).

## Seeder

`Seeder` is `Name() string` and `Seed(ctx) error`. `Name` is the tracking id.

## Runner

`NewRunner(provider, seeders, log)` keeps the seeders in the given order.
`Register` appends another seeder.

`Start` builds a `Tracker` from `provider.GetDB()` on the first call. It
creates `_seeds` when missing. Each seeder is skipped when its name is
already in that table. A seed error or a tracking error stops the remaining
seeders. A seed that succeeds and then fails to record its name can run again
on the next start. `Run` calls `Start`.

The provider must expose a live non-nil connection before `Start`; the runner
does not guard a nil pool. It neither starts nor closes that pool. There is no
transaction joining a seeder's work and its tracking insert, and no concurrent
runner lock. Coordinate one runner and make each seeder safe to retry after
partial work. A name is not a content checksum; changing code does not replay
a recorded name.

## Tracker

`_seeds` has `id` and `applied_at`. `IsApplied` counts rows for that id.
`MarkApplied` inserts the id and returns a database error when the id is
already present.

## References

`Ref` is a string. `NewRefMap` returns an empty map. `Register` stores a UUID
for a ref and overwrites an existing ref. `Resolve` returns `unresolved
reference: <ref>` when the ref is absent. `MustResolve` panics with that
error. `Has` reports whether the ref is present.

References are an in-memory map without synchronization or persistence. A
skipped seeder does not reconstruct them after restart. Later seeders must
reload durable IDs when they depend on previously applied records.
