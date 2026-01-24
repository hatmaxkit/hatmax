package i18n

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const DefaultLocale = "en"

type Translator struct {
	translations map[string]map[string]any
	mu           sync.RWMutex
	defaultLoc   string
	available    []string
}

func New() *Translator {
	return &Translator{
		translations: make(map[string]map[string]any),
		defaultLoc:   DefaultLocale,
		available:    []string{},
	}
}

func (t *Translator) LoadFromFS(fs embed.FS, basePath string) error {
	entries, err := fs.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("read i18n directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}

		locale := strings.TrimSuffix(strings.TrimSuffix(name, ".yml"), ".yaml")
		filePath := basePath + "/" + name

		data, err := fs.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}

		if err := t.loadLocale(locale, data); err != nil {
			return fmt.Errorf("parse %s: %w", filePath, err)
		}

		t.available = append(t.available, locale)
	}

	return nil
}

func (t *Translator) loadLocale(locale string, data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.translations[locale] = make(map[string]any)
	flatten("", raw, t.translations[locale])

	return nil
}

func flatten(prefix string, src map[string]any, dest map[string]any) {
	for k, v := range src {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]any:
			flatten(key, val, dest)
		default:
			dest[key] = val
		}
	}
}

func (t *Translator) Get(locale, key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if trans, ok := t.translations[locale]; ok {
		if val, ok := trans[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
			return fmt.Sprintf("%v", val)
		}
	}

	if locale != t.defaultLoc {
		if trans, ok := t.translations[t.defaultLoc]; ok {
			if val, ok := trans[key]; ok {
				if s, ok := val.(string); ok {
					return s
				}
				return fmt.Sprintf("%v", val)
			}
		}
	}

	return key
}

func (t *Translator) SetDefaultLocale(locale string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.defaultLoc = locale
}

func (t *Translator) DefaultLocaleValue() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.defaultLoc
}

func (t *Translator) AvailableLocales() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]string, len(t.available))
	copy(result, t.available)
	return result
}

func (t *Translator) HasLocale(locale string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.translations[locale]
	return ok
}

func (t *Translator) TranslateFunc(locale string) func(string) string {
	return func(key string) string {
		return t.Get(locale, key)
	}
}
