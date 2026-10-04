// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
)

type passwordTransport struct {
	token, password string
	policy          auth.PasswordChangePolicy
	calls           int
	err             error
}

func (s *passwordTransport) ChangePassword(_ context.Context, token, password string, p auth.PasswordChangePolicy) error {
	s.calls++
	s.token = token
	s.password = password
	s.policy = p

	return s.err
}

// The production handler derives identity only from the cookie, rejects mixed
// fields/origins and returns safe errors without echoing passwords or diagnostics.
func TestPasswordChangeTransport(t *testing.T) {
	for _, tc := range []struct {
		name, method, origin, cookie, body, query string
		failure                                   error
		status, calls                             int
	}{
		{"form", http.MethodGet, "", "", "", "", nil, 200, 0},
		{"change", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"complete\nUnicode 😀 password"}}.Encode(), "", nil, 200, 1},
		{"missing actor", http.MethodPost, "https://example.com", "", url.Values{"password": {"new password"}}.Encode(), "", nil, 403, 0},
		{"foreign origin", http.MethodPost, "https://other.example", "actor", url.Values{"password": {"new password"}}.Encode(), "", nil, 403, 0},
		{"account field", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"new password"}, "subject": {"foreign"}}.Encode(), "", nil, 400, 0},
		{"duplicate field", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"one", "two"}}.Encode(), "", nil, 400, 0},
		{"query password", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"new password"}}.Encode(), "?password=other", nil, 400, 0},
		{"oversized", http.MethodPost, "https://example.com", "actor", url.Values{"password": {strings.Repeat("x", 4097)}}.Encode(), "", nil, 400, 0},
		{"safe failure", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"new password"}}.Encode(), "", errors.New("private secret"), 403, 1},
		{"budget", http.MethodPost, "https://example.com", "actor", url.Values{"password": {"new password"}}.Encode(), "", auth.ErrRecoveryAttempts, 429, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &passwordTransport{err: tc.failure}

			h, err := NewPasswordChangeHandler(svc, auth.PasswordChangePolicy{Requirement: auth.AccessRequirement{Revision: "test-v1"}, AllowPassword: true})
			if err != nil {
				t.Fatal(err)
			}

			router := chi.NewRouter()
			h.RegisterRoutes(router)

			r := httptest.NewRequest(tc.method, "https://example.com/account/password"+tc.query, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", tc.origin)

			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: tc.cookie})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != tc.status || svc.calls != tc.calls {
				t.Fatalf("response: %d, calls: %d", w.Code, svc.calls)
			}

			if strings.Contains(w.Body.String(), "private secret") || strings.Contains(w.Body.String(), "complete\nUnicode") {
				t.Fatal("secret in response")
			}

			if tc.name == "change" {
				if svc.password != "complete\nUnicode 😀 password" || svc.token != "actor" || svc.policy.Requirement.Proof != auth.RequirePhishingResistantMFA {
					t.Fatal("input or trusted default changed")
				}

				cookies := w.Result().Cookies()
				if len(cookies) != 1 || cookies[0].Name != auth.SessionCookieName || cookies[0].MaxAge != -1 {
					t.Fatal("success did not clear cookie")
				}
			}

			if tc.name != "change" && len(w.Result().Cookies()) != 0 {
				t.Fatal("failure/GET changed cookie")
			}

			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing cache boundary")
			}
		})
	}
}
