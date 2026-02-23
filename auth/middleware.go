package auth

import (
	"net/http"
)

const (
	// SessionCookieName is the default name for the session cookie.
	SessionCookieName = "session"
)

// RequireAuth is a middleware that requires authentication.
// If the user is not authenticated, it redirects to the signin page.
func RequireAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)

				return
			}

			user, err := svc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)

				return
			}

			ctx := WithUser(r.Context(), user)
			ctx = WithUserID(ctx, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth is a middleware that adds the user to the context if authenticated.
// If the user is not authenticated, it continues without adding the user.
func OptionalAuth(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				next.ServeHTTP(w, r)

				return
			}

			user, err := svc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)

				return
			}

			ctx := WithUser(r.Context(), user)
			ctx = WithUserID(ctx, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetSessionCookie sets the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// TOTPEnforcement configures 2FA enforcement behavior.
type TOTPEnforcement struct {
	Enabled   func() bool
	GraceDays func() int
	SetupURL  string
}

// RequireTOTP is a middleware that enforces 2FA setup.
// It should be used after RequireAuth.
func RequireTOTP(cfg TOTPEnforcement) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Enabled == nil || !cfg.Enabled() {
				next.ServeHTTP(w, r)

				return
			}

			user, ok := GetUser(r.Context())
			if !ok {
				next.ServeHTTP(w, r)

				return
			}

			if user.TOTPEnabled {
				next.ServeHTTP(w, r)

				return
			}

			graceDays := 0
			if cfg.GraceDays != nil {
				graceDays = cfg.GraceDays()
			}

			if user.InTOTPGracePeriod(graceDays) {
				next.ServeHTTP(w, r)

				return
			}

			setupURL := cfg.SetupURL
			if setupURL == "" {
				setupURL = "/totp-setup"
			}

			http.Redirect(w, r, setupURL, http.StatusSeeOther)
		})
	}
}
