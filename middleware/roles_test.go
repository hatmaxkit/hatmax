// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatmax.adrianpk.com/auth"
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

type roleValidator struct {
	result   *auth.ValidatedSession
	activity auth.SessionActivity
	calls    int
}

func (v *roleValidator) ValidateSession(ctx context.Context, token string, activity auth.SessionActivity) (*auth.ValidatedSession, error) {
	v.calls++
	v.activity = activity

	if token == "invalid" {
		return nil, auth.ErrSessionToken
	}

	return v.result, nil
}

// Validation failures and role denial cannot pass context to protected handlers.
// A permitted request forwards trusted activity and all safe identity metadata.
func TestRoleSession(t *testing.T) {
	tests := []struct {
		name, token, role string
		want              int
	}{
		{"missing cookie", "", "admin", http.StatusSeeOther},
		{"invalid cookie", "invalid", "admin", http.StatusSeeOther},
		{"denied role", "valid", "reader", http.StatusForbidden},
		{"allowed role", "valid", "admin", http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &roleValidator{result: &auth.ValidatedSession{User: &auth.User{ID: "user", Roles: []string{test.role}}, Session: auth.Session{ID: "session", UserID: "user"}}}
			handler := RequireRoles(validator, auth.NoActivity, "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				metadata, ok := auth.GetSession(r.Context())

				userID, idFound := auth.GetUserID(r.Context())
				if !ok || !idFound || metadata.ID != "session" || userID != "user" {
					t.Fatal("safe session context missing")
				}

				w.WriteHeader(http.StatusOK)
			}))

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.token != "" {
				request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: test.token})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.want || validator.activity != auth.NoActivity {
				t.Fatal("unexpected role/session behavior")
			}

			if test.token == "" && validator.calls != 0 {
				t.Fatal("missing cookie reached validation")
			}
		})
	}
}
