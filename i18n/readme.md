# i18n

Internationalization support with YAML translation files.

## Usage

```go
//go:embed locales
var localesFS embed.FS

// Load translations
translator := i18n.New()
translator.LoadFromFS(localesFS, "locales")

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

## Translation Files

YAML files named by locale (e.g., `en.yml`, `es.yml`):

```yaml
common:
  search: "Search"
  back: "Back"

user:
  name: "Name"
  email: "Email"
```

Access via dot notation: `common.search`, `user.name`.
