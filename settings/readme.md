<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# settings

Runtime key-value configuration with schema validation.

## Usage

These fragments belong in the bootstrap composition root. Supply `ctx` and
an application-owned `settings.Store`, such as the Guide's memory adapter.
No PostgreSQL store or settings administration UI is supplied by this package.

```go
// Define schemas
reg := settings.NewRegistry()
minItems, maxItems := 1, 100
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
    Min:         &minItems,
    Max:         &maxItems,
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
svc := settings.NewService(reg, settingStore)

// Read (only ErrNotFound selects a default)
name, err := svc.GetString(ctx, "site.name")
if err != nil {
    return err
}

limit, err := svc.GetInt(ctx, "site.items_per_page")
if err != nil {
    return err
}

enabled, err := svc.GetBool(ctx, "feature.dark_mode")
if err != nil {
    return err
}

// Write (validates against schema)
err = svc.Set(ctx, "site.name", "New Name")
if err != nil {
    return err
}
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

`Store.Get` returns `settings.ErrNotFound` only when a key is absent. Wrapped
sentinels are supported through `errors.Is`; translate backend-specific
not-found errors in your adapter. Present empty strings return nil error, and
read or context failures must propagate. Update custom adapters that previously
returned `"", nil` or an unrelated error for absence.

Service getters use defaults only for absence. `GetString` preserves a stored
empty string; `GetInt` and `GetBool` reject it with a parse error. Delete a key
to restore its default. Missing unregistered keys or empty defaults retain
zero-value behavior. Getters do not persist fallback values.

## Value Helpers

The standalone helpers below keep their own empty-to-zero parsing behavior.

```go
// Parse from string (returns zero value if empty)
b, err := settings.ParseBool("true")
n, err := settings.ParseInt("42")

// Format to string
boolText := settings.FormatBool(true)
intText := settings.FormatInt(42)
```

## Notes

Complements `config/`: startup configuration uses YAML, environment and flags.
Settings change through an application-owned store and workflow. Schemas and
reads do not authorize a caller or protect secrets. `Secret` is UI metadata;
`MaxLength` counts bytes. Defaults and stored values are parsed on reads but
their schema bounds are not revalidated. See the
[Configuration Reference](../docs/reference/configuration/README.md#settings).
