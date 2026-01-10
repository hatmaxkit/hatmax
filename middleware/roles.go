package middleware

import (
	"context"
	"net/http"

	"github.com/hatmaxkit/hatmax/auth"
)

// SessionValidator validates session tokens and returns users.
type SessionValidator interface {
	ValidateSession(ctx context.Context, token string) (*auth.User, error)
}

// RequireRole returns a middleware that checks if the authenticated user has the specified role.
// Returns 401 Unauthorized if no user is in the context, 403 Forbidden if the user lacks the role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.GetUser(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !user.HasRole(role) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoles returns a middleware that validates authentication and checks if the user has any of the specified roles.
// Combines auth.RequireAuth + RequireAnyRole in a single middleware.
// Redirects to /signin if not authenticated, returns 403 Forbidden if the user lacks all roles.
func RequireRoles(svc SessionValidator, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}

			user, err := svc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				auth.ClearSessionCookie(w)
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}

			if !user.HasAnyRole(roles...) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ctx := auth.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAnyRole returns a middleware that checks if the authenticated user has any of the specified roles.
// Returns 401 Unauthorized if no user is in the context, 403 Forbidden if the user lacks all roles.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.GetUser(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !user.HasAnyRole(roles...) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
