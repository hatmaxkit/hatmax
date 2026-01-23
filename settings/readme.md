# settings

Runtime key-value configuration with schema validation.

## Usage

```go
// Define schemas
reg := settings.NewRegistry()
reg.Register(settings.Schema{
    Key:      "site.name",
    Type:     settings.String,
    Default:  "My Site",
    Required: true,
})
reg.Register(settings.Schema{
    Key:     "site.items_per_page",
    Type:    settings.Int,
    Default: "20",
    Min:     ptr(1),
    Max:     ptr(100),
})

// Create service
svc := settings.NewService(reg, postgresStore)

// Read (returns default if not set)
name, _ := svc.GetString(ctx, "site.name")
limit, _ := svc.GetInt(ctx, "site.items_per_page")
enabled, _ := svc.GetBool(ctx, "feature.dark_mode")

// Write (validates against schema)
svc.Set(ctx, "site.name", "New Name")
```

## API

```go
type Store interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    All(ctx context.Context) ([]Value, error)
    Delete(ctx context.Context, key string) error
}
```

## Notes

Complements `config/`: config is static (YAML, env vars, flags at startup), settings is dynamic (DB, editable at runtime via admin UI).
