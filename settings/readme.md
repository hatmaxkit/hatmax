# settings

Runtime key-value configuration with schema validation.

## Overview

This package provides dynamic application settings stored in a database,
as opposed to `config/` which handles static configuration from files/env.

Use cases:
- Site name, logo URL, feature toggles
- User-configurable preferences
- Settings editable via admin UI

## API

### Schema

Define what settings exist and their constraints:

```go
schema := settings.Schema{
    Key:      "site.name",
    Type:     settings.String,
    Default:  "My Site",
    Required: true,
}

schema := settings.Schema{
    Key:     "site.max_uploads",
    Type:    settings.Int,
    Default: "10",
    Min:     ptr(1),
    Max:     ptr(100),
}

schema := settings.Schema{
    Key:     "site.theme",
    Type:    settings.Enum,
    Default: "light",
    Options: []string{"light", "dark", "auto"},
}
```

### Registry

Register schemas at startup:

```go
reg := settings.NewRegistry()
reg.Register(settings.Schema{...})
reg.Register(settings.Schema{...})

// Query
schema, ok := reg.Get("site.name")
all := reg.All()
siteSettings := reg.ByPrefix("site.")
```

### Store

Interface for persistence (implement for your database):

```go
type Store interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    All(ctx context.Context) ([]Value, error)
    Delete(ctx context.Context, key string) error
}
```

### Service

High-level API combining registry + store:

```go
svc := settings.NewService(registry, store)

// Type-safe getters (returns default if not set)
name, err := svc.GetString(ctx, "site.name")
max, err := svc.GetInt(ctx, "site.max_uploads")
enabled, err := svc.GetBool(ctx, "feature.dark_mode")

// Set validates against schema before persisting
err := svc.Set(ctx, "site.name", "New Name")
```

## Example

```go
// 1. Define schemas
reg := settings.NewRegistry()
reg.Register(settings.Schema{
    Key:      "site.name",
    Type:     settings.String,
    Default:  "Hatmax",
    Required: true,
})
reg.Register(settings.Schema{
    Key:     "site.items_per_page",
    Type:    settings.Int,
    Default: "20",
    Min:     ptr(5),
    Max:     ptr(100),
})

// 2. Create service with store
store := NewPostgresStore(db) // your implementation
svc := settings.NewService(reg, store)

// 3. Use in handlers
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
    perPage, _ := h.settings.GetInt(r.Context(), "site.items_per_page")
    // ...
}
```

## Schema vs Config

| Aspect | config/ | settings/ |
|--------|---------|-----------|
| Source | YAML, env vars, flags | Database |
| When | Startup (static) | Runtime (dynamic) |
| Who changes | DevOps, deploy | Admin users, UI |
| Examples | DB connection, port | Site name, theme |
