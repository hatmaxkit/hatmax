package middleware

import (
	"context"
	"net/http"
	"strings"
)

const (
	LocaleKey        contextKey = "locale"
	LocaleCookieName            = "locale"
)

type LocaleConfig struct {
	Default   string
	Available []string
}

func Locale(cfg LocaleConfig) func(http.Handler) http.Handler {
	available := make(map[string]bool)
	for _, loc := range cfg.Available {
		available[loc] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := detectLocale(r, available, cfg.Default)
			ctx := context.WithValue(r.Context(), LocaleKey, locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func detectLocale(r *http.Request, available map[string]bool, defaultLocale string) string {
	if cookie, err := r.Cookie(LocaleCookieName); err == nil {
		loc := cookie.Value
		if len(available) == 0 || available[loc] {
			return loc
		}
	}

	if accept := r.Header.Get("Accept-Language"); accept != "" {
		for _, locale := range parseAcceptLanguage(accept) {
			if len(available) == 0 || available[locale] {
				return locale
			}
		}
	}

	return defaultLocale
}

func parseAcceptLanguage(header string) []string {
	var locales []string
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, ";"); idx != -1 {
			part = part[:idx]
		}
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idx := strings.Index(part, "-"); idx != -1 {
			part = part[:idx]
		}

		locales = append(locales, strings.ToLower(part))
	}
	return locales
}

func GetLocale(ctx context.Context) string {
	if ctx == nil {
		return "en"
	}
	if locale, ok := ctx.Value(LocaleKey).(string); ok {
		return locale
	}
	return "en"
}

func SetLocaleCookie(w http.ResponseWriter, locale string) {
	http.SetCookie(w, &http.Cookie{
		Name:     LocaleCookieName,
		Value:    locale,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
