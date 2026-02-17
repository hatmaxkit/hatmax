# settings

Runtime key-value configuration with schema validation.

## Usage

```go
// Define schemas
reg := settings.NewRegistry()
reg.Register(settings.Schema{
    Key:         "site.name",
    Type:        settings.String,
    Default:     "My Site",
    Label:       "Site Name",
    Description: "The name displayed in the header",
    Required:    true,
    MaxLength:   100,
})
reg.Register(settings.Schema{
    Key:         "site.items_per_page",
    Type:        settings.Int,
    Default:     "20",
    Label:       "Items Per Page",
    Description: "Number of items to show in listings",
    Min:         ptr(1),
    Max:         ptr(100),
})
reg.Register(settings.Schema{
    Key:         "site.theme",
    Type:        settings.Enum,
    Default:     "light",
    Label:       "Theme",
    Description: "Color scheme for the UI",
    Options:     []string{"light", "dark", "auto"},
    Labels:      []string{"Light", "Dark", "System"},
})
reg.Register(settings.Schema{
    Key:         "api.token",
    Type:        settings.String,
    Label:       "API Token",
    Description: "External service API token",
    Secret:      true,
    MaxLength:   255,
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

## Schema Fields

| Field | Type | Description |
|-------|------|-------------|
| Key | string | Unique identifier (use dot prefix for namespacing) |
| Type | Type | String, Int, Bool, or Enum |
| Default | string | Default value when not set |
| Label | string | Human-readable name for UI |
| Description | string | Help text for UI |
| Required | bool | Whether empty values are rejected |
| Secret | bool | Hint for UI to mask value (passwords, tokens) |
| Min | *int | Minimum value for Int type |
| Max | *int | Maximum value for Int type |
| MaxLength | int | Maximum string length (0 = unlimited) |
| Options | []string | Valid values for Enum type |
| Labels | []string | Human-readable labels for Options |

## Organizing by Namespace

Settings use dot-prefixed keys (e.g., `security.require_2fa`). Use `Registry.ByPrefix()` to group them:

```go
securitySettings := reg.ByPrefix("security.")
brandingSettings := reg.ByPrefix("branding.")
```

For UI metadata, use `NamespaceSchema`:

```go
namespaces := []settings.NamespaceSchema{
    {Key: "security", Label: "Security", Description: "Authentication settings"},
    {Key: "branding", Label: "Branding", Description: "App appearance"},
}
```

## Store Interface

```go
type Store interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string) error
    All(ctx context.Context) ([]Value, error)
    Delete(ctx context.Context, key string) error
}
```

## Value Helpers

```go
// Parse from string (returns zero value if empty)
b, err := settings.ParseBool("true")
n, err := settings.ParseInt("42")

// Format to string
s := settings.FormatBool(true)
s := settings.FormatInt(42)
```

## Notes

Complements `config/`: config is static (YAML, env vars, flags at startup), settings is dynamic (DB, editable at runtime via admin UI).
