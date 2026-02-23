package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocaleFromCookie(t *testing.T) {
	cfg := LocaleConfig{
		Default:   "en",
		Available: []string{"en", "es", "pl", "de"},
	}

	handler := Locale(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := GetLocale(r.Context())
		w.Write([]byte(locale))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "locale", Value: "es"})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "es" {
		t.Errorf("expected 'es', got %q", rec.Body.String())
	}
}

func TestLocaleFromAcceptLanguage(t *testing.T) {
	cfg := LocaleConfig{
		Default:   "en",
		Available: []string{"en", "es", "pl", "de"},
	}

	handler := Locale(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := GetLocale(r.Context())
		w.Write([]byte(locale))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "de" {
		t.Errorf("expected 'de', got %q", rec.Body.String())
	}
}

func TestLocaleDefault(t *testing.T) {
	cfg := LocaleConfig{
		Default:   "en",
		Available: []string{"en", "es", "pl", "de"},
	}

	handler := Locale(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := GetLocale(r.Context())
		w.Write([]byte(locale))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "en" {
		t.Errorf("expected 'en', got %q", rec.Body.String())
	}
}

func TestLocaleCookieOverridesHeader(t *testing.T) {
	cfg := LocaleConfig{
		Default:   "en",
		Available: []string{"en", "es", "pl", "de"},
	}

	handler := Locale(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := GetLocale(r.Context())
		w.Write([]byte(locale))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "locale", Value: "pl"})
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "pl" {
		t.Errorf("expected 'pl', got %q", rec.Body.String())
	}
}

func TestLocaleUnavailableFallsBack(t *testing.T) {
	cfg := LocaleConfig{
		Default:   "en",
		Available: []string{"en", "es"},
	}

	handler := Locale(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := GetLocale(r.Context())
		w.Write([]byte(locale))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,de;q=0.8")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "en" {
		t.Errorf("expected 'en', got %q", rec.Body.String())
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header   string
		expected []string
	}{
		{"en", []string{"en"}},
		{"en-US", []string{"en"}},
		{"en-US,en;q=0.9", []string{"en", "en"}},
		{"de-DE,de;q=0.9,en;q=0.8", []string{"de", "de", "en"}},
		{"pl-PL, pl;q=0.9, en;q=0.8", []string{"pl", "pl", "en"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := parseAcceptLanguage(tt.header)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d locales, got %d", len(tt.expected), len(result))

				return
			}

			for i, loc := range result {
				if loc != tt.expected[i] {
					t.Errorf("at index %d: expected %q, got %q", i, tt.expected[i], loc)
				}
			}
		})
	}
}

func TestGetLocaleNilContext(t *testing.T) {
	locale := GetLocale(nil)
	if locale != "en" {
		t.Errorf("expected 'en', got %q", locale)
	}
}

func TestSetLocaleCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	SetLocaleCookie(rec, "de")

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "locale" {
		t.Errorf("expected cookie name 'locale', got %q", cookie.Name)
	}

	if cookie.Value != "de" {
		t.Errorf("expected cookie value 'de', got %q", cookie.Value)
	}

	if !cookie.HttpOnly {
		t.Error("expected HttpOnly flag")
	}
}
