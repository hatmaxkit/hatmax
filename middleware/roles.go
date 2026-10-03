// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"context"
	"net/http"

	"hatmax.adrianpk.com/auth"
)

// SessionValidator validates session tokens and returns users.
type SessionValidator interface {
	ValidateSession(ctx context.Context, token string, activity auth.SessionActivity) (*auth.ValidatedSession, error)
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
func RequireRoles(svc SessionValidator, activity auth.SessionActivity, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)

				return
			}

			validated, err := svc.ValidateSession(r.Context(), cookie.Value, activity)
			if err != nil {
				auth.ClearSessionCookie(w)
				http.Redirect(w, r, "/signin", http.StatusSeeOther)

				return
			}

			if !validated.User.HasAnyRole(roles...) {
				http.Error(w, "Forbidden", http.StatusForbidden)

				return
			}

			ctx := auth.WithUser(r.Context(), validated.User)
			ctx = auth.WithUserID(ctx, validated.User.ID)
			ctx = auth.WithSession(ctx, &validated.Session)
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
