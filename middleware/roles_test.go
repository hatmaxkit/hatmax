package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hatmaxkit/hatmax/auth"
)

func TestRequireRole(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		user       *auth.User
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "no user in context",
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user without role",
			user:       &auth.User{ID: "1", Roles: []string{"user"}},
			hasUser:    true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user with role",
			user:       &auth.User{ID: "1", Roles: []string{"admin"}},
			hasUser:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user with multiple roles including required",
			user:       &auth.User{ID: "1", Roles: []string{"user", "admin", "moderator"}},
			hasUser:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user with empty roles",
			user:       &auth.User{ID: "1", Roles: []string{}},
			hasUser:    true,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.hasUser {
				ctx := auth.WithUser(req.Context(), tt.user)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("RequireRole() status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireAnyRole(t *testing.T) {
	handler := RequireAnyRole("admin", "moderator")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name       string
		user       *auth.User
		hasUser    bool
		wantStatus int
	}{
		{
			name:       "no user in context",
			hasUser:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "user without any required role",
			user:       &auth.User{ID: "1", Roles: []string{"user", "guest"}},
			hasUser:    true,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "user with first required role",
			user:       &auth.User{ID: "1", Roles: []string{"admin"}},
			hasUser:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user with second required role",
			user:       &auth.User{ID: "1", Roles: []string{"moderator"}},
			hasUser:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user with both required roles",
			user:       &auth.User{ID: "1", Roles: []string{"admin", "moderator"}},
			hasUser:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "user with empty roles",
			user:       &auth.User{ID: "1", Roles: []string{}},
			hasUser:    true,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			if tt.hasUser {
				ctx := auth.WithUser(req.Context(), tt.user)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("RequireAnyRole() status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
