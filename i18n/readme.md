# i18n

Internationalization support with YAML translation files.

## Usage

```go
//go:embed assets
var assetsFS embed.FS

// Load translations
translator := i18n.New()
translator.LoadFromFS(assetsFS, "assets/locales")

// Get translation
text := translator.Get("es", "common.search") // "Buscar"

// Fallback to default locale if key missing
text := translator.Get("es", "missing.key") // returns key itself

// Template function
fn := translator.TranslateFunc("es")
fn("common.search") // "Buscar"

// Check available locales
locales := translator.AvailableLocales() // ["en", "es"]
translator.HasLocale("de") // false
```

## Directory Structure

Locale files are loaded from the specified path within the embedded filesystem:

```
assets/
└── locales/
    ├── en.yml      # locale "en"
    ├── es.yml      # locale "es"
    └── pt-BR.yaml  # locale "pt-BR" (.yaml also supported)
```

Each filename (without extension) becomes the locale code. Subdirectories are ignored. Only `.yml` and `.yaml` files are processed.

## Translation Files

YAML files use nested keys that are flattened with dot notation:

```yaml
common:
  search: "Search"
  back: "Back"

user:
  name: "Name"
  email: "Email"
```

Access via dot notation: `common.search`, `user.name`.
