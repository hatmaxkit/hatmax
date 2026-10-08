<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# seed

Database seeding with tracking and symbolic references.

## Usage

This application-owned seeder follows the database and migrator. Its migration
creates `users (id text PRIMARY KEY, email text NOT NULL UNIQUE)`. Import
`context`, `database/sql`, `model` and `seed` in the application's seed package.

```go
// Implement Seeder interface
type UserSeeder struct {
    db  *sql.DB
    refs seed.RefMap
}

func (s *UserSeeder) Name() string { return "users" }

func (s *UserSeeder) Seed(ctx context.Context) error {
    id := model.NewID()
    err := s.db.QueryRowContext(ctx, `
        INSERT INTO users (id, email) VALUES ($1, $2)
        ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
        RETURNING id
    `, id, "admin@example.com").Scan(&id)
    if err != nil {
        return err
    }
    s.refs.Register(seed.Ref("admin"), id)
    return nil
}
```

In the startup composition, `database` supplies `GetDB`, and `logger` is the
application logger. Start it only after the database and migration succeed:

```go
refs := seed.NewRefMap()
// Run seeders (tracks in _seeds table, skips already applied)
runner := seed.NewRunner(database, []seed.Seeder{
    &UserSeeder{db: database.GetDB(), refs: refs},
}, logger)
err := runner.Start(ctx)
if err != nil {
    return err
}

// Reference seeded IDs across seeders
adminID, err := refs.Resolve(seed.Ref("admin"))
if err != nil {
    return err
}
```

See: `seeder.go`, `tracker.go`, `ref.go`.

The reference lookup above is for a fresh run. On later starts the runner skips
recorded names and does not rebuild `refs`; load existing IDs from durable
storage before using them. Seeder effects and tracking are separate operations,
so use retry-safe writes and coordinate a single runner. See
[Seed](../docs/reference/seed/README.md) for ownership and failure behavior.
