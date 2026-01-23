# seed

Database seeding with tracking and symbolic references.

## Usage

```go
// Implement Seeder interface
type UserSeeder struct {
    db  *sql.DB
    ref *seed.RefMap
}

func (s *UserSeeder) Name() string { return "users" }

func (s *UserSeeder) Seed(ctx context.Context) error {
    id := model.NewID()
    s.ref.Set("admin", id)
    _, err := s.db.ExecContext(ctx, "INSERT INTO users ...", id)
    return err
}

// Run seeders (tracks in _seeds table, skips already applied)
runner := seed.NewRunner(db, log)
runner.Run(ctx, &UserSeeder{}, &ProductSeeder{})

// Reference seeded IDs across seeders
adminID := refs.Get("admin")
```

See: `seeder.go`, `tracker.go`, `ref.go`.
