package i18n

import (
	"embed"
	"testing"
)

//go:embed testdata
var testFS embed.FS

func TestTranslatorLoadFromFS(t *testing.T) {
	tr := New()
	err := tr.LoadFromFS(testFS, "testdata")
	if err != nil {
		t.Fatalf("LoadFromFS failed: %v", err)
	}

	if !tr.HasLocale("en") {
		t.Error("expected locale 'en' to be loaded")
	}
	if !tr.HasLocale("es") {
		t.Error("expected locale 'es' to be loaded")
	}
}

func TestTranslatorGet(t *testing.T) {
	tr := New()
	if err := tr.LoadFromFS(testFS, "testdata"); err != nil {
		t.Fatalf("LoadFromFS failed: %v", err)
	}

	tests := []struct {
		name     string
		locale   string
		key      string
		expected string
	}{
		{
			name:     "simple key in english",
			locale:   "en",
			key:      "common.search",
			expected: "Search",
		},
		{
			name:     "simple key in spanish",
			locale:   "es",
			key:      "common.search",
			expected: "Buscar",
		},
		{
			name:     "nested key",
			locale:   "en",
			key:      "user.name",
			expected: "Name",
		},
		{
			name:     "fallback to english when key missing in locale",
			locale:   "es",
			key:      "user.only_english",
			expected: "Only in English",
		},
		{
			name:     "missing key returns key itself",
			locale:   "en",
			key:      "missing.key.path",
			expected: "missing.key.path",
		},
		{
			name:     "unknown locale falls back to english",
			locale:   "fr",
			key:      "common.search",
			expected: "Search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.Get(tt.locale, tt.key)
			if result != tt.expected {
				t.Errorf("Get(%q, %q) = %q, want %q", tt.locale, tt.key, result, tt.expected)
			}
		})
	}
}

func TestTranslatorAvailableLocales(t *testing.T) {
	tr := New()
	if err := tr.LoadFromFS(testFS, "testdata"); err != nil {
		t.Fatalf("LoadFromFS failed: %v", err)
	}

	locales := tr.AvailableLocales()
	if len(locales) < 2 {
		t.Errorf("expected at least 2 locales, got %d", len(locales))
	}

	found := make(map[string]bool)
	for _, loc := range locales {
		found[loc] = true
	}

	if !found["en"] {
		t.Error("expected 'en' in available locales")
	}
	if !found["es"] {
		t.Error("expected 'es' in available locales")
	}
}

func TestTranslatorSetDefaultLocale(t *testing.T) {
	tr := New()
	if err := tr.LoadFromFS(testFS, "testdata"); err != nil {
		t.Fatalf("LoadFromFS failed: %v", err)
	}

	tr.SetDefaultLocale("es")
	if tr.DefaultLocaleValue() != "es" {
		t.Errorf("expected default locale 'es', got %q", tr.DefaultLocaleValue())
	}

	result := tr.Get("fr", "common.search")
	if result != "Buscar" {
		t.Errorf("expected Spanish fallback 'Buscar', got %q", result)
	}
}

func TestTranslatorTranslateFunc(t *testing.T) {
	tr := New()
	if err := tr.LoadFromFS(testFS, "testdata"); err != nil {
		t.Fatalf("LoadFromFS failed: %v", err)
	}

	fn := tr.TranslateFunc("es")
	result := fn("common.search")
	if result != "Buscar" {
		t.Errorf("TranslateFunc(es)(common.search) = %q, want 'Buscar'", result)
	}
}
