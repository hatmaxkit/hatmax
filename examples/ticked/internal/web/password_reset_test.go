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
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
)

type resetTransport struct {
	requests, completions  int
	email, token, password string
	err                    error
}

func (s *resetTransport) RequestPasswordReset(_ context.Context, email string) error {
	s.requests++
	s.email = email

	return s.err
}
func (s *resetTransport) ResetPassword(_ context.Context, token, password string) error {
	s.completions++
	s.token = token
	s.password = password

	return s.err
}

type resetValidator struct {
	calls       int
	roles       []string
	requirement auth.AccessRequirement
}

func (v *resetValidator) ValidateSession(_ context.Context, _ string, r auth.AccessRequirement, _ auth.SessionActivity) (*auth.ValidatedSession, error) {
	v.calls++
	v.requirement = r

	return &auth.ValidatedSession{User: &auth.User{Roles: v.roles}}, nil
}

// Fakes establish routing/input/error policy only. Tagged production PostgreSQL
// tests separately establish current session proof and operator role authority.
func TestPasswordResetTransport(t *testing.T) {
	token := "123e4567-e89b-42d3-a456-426614174000." + strings.Repeat("A", 43)
	for _, tc := range []struct {
		name, method, path, cookie, origin string
		fields                             url.Values
		roles                              []string
		failure                            error
		code, requests, completions        int
	}{
		{"request form", "GET", "/account/password/reset", "", "", nil, nil, nil, 200, 0, 0},
		{"confirm form", "GET", "/account/password/reset/confirm", "", "", nil, nil, nil, 200, 0, 0},
		{"neutral request", "POST", "/account/password/reset", "", "https://example.com", url.Values{"email": {"user@example.com"}}, nil, errors.New("private secret"), 202, 1, 0},
		{"complete", "POST", "/account/password/reset/confirm", "old", "https://example.com", url.Values{"token": {token}, "password": {"complete\nUnicode 😀 password"}}, nil, nil, 200, 0, 1},
		{"failure", "POST", "/account/password/reset/confirm", "old", "https://example.com", url.Values{"token": {token}, "password": {"password candidate"}}, nil, errors.New("private secret"), 403, 0, 1},
		{"extra account", "POST", "/account/password/reset/confirm", "", "https://example.com", url.Values{"token": {token}, "password": {"password candidate"}, "subject": {"other"}}, nil, nil, 403, 0, 0},
		{"duplicate", "POST", "/account/password/reset/confirm", "", "https://example.com", url.Values{"token": {token, token}, "password": {"password candidate"}}, nil, nil, 403, 0, 0},
		{"query", "POST", "/account/password/reset/confirm?token=other", "", "https://example.com", url.Values{"token": {token}, "password": {"password candidate"}}, nil, nil, 403, 0, 0},
		{"oversized", "POST", "/account/password/reset/confirm", "", "https://example.com", url.Values{"token": {token}, "password": {strings.Repeat("x", 4097)}}, nil, nil, 403, 0, 0},
		{"foreign origin", "POST", "/account/password/reset/confirm", "", "https://foreign.example", url.Values{"token": {token}, "password": {"password candidate"}}, nil, nil, 403, 0, 0},
		{"operator missing", "POST", "/admin/password-reset", "", "https://example.com", url.Values{"email": {"user@example.com"}}, nil, nil, 303, 0, 0},
		{"operator role", "POST", "/admin/password-reset", "actor", "https://example.com", url.Values{"email": {"user@example.com"}}, []string{"user"}, nil, 403, 0, 0},
		{"operator request", "POST", "/admin/password-reset", "actor", "https://example.com", url.Values{"email": {"user@example.com"}}, []string{"admin"}, nil, 202, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &resetTransport{err: tc.failure}
			validator := &resetValidator{roles: tc.roles}
			policy := auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "test-v1", MaxAge: 5 * time.Minute}

			h, err := NewPasswordResetHandler(service, validator, policy)
			if err != nil {
				t.Fatal(err)
			}

			h.admission.acknowledgment = time.Millisecond
			router := chi.NewRouter()
			h.RegisterRoutes(router)

			r := httptest.NewRequest(tc.method, "https://example.com"+tc.path, strings.NewReader(tc.fields.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", tc.origin)

			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: tc.cookie})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != tc.code || service.requests != tc.requests || service.completions != tc.completions {
				t.Fatalf("response %d; requests %d; completions %d", w.Code, service.requests, service.completions)
			}

			if strings.Contains(w.Body.String(), "private secret") || strings.Contains(w.Body.String(), token) {
				t.Fatal("secret leaked")
			}

			if tc.name == "complete" {
				if service.token != token || service.password != "complete\nUnicode 😀 password" {
					t.Fatal("candidate altered")
				}

				cookies := w.Result().Cookies()
				if len(cookies) != 1 || cookies[0].MaxAge != -1 || cookies[0].Value != "" {
					t.Fatal("success did not clear cookie")
				}
			} else if len(w.Result().Cookies()) != 0 {
				t.Fatal("failure or initiation changed cookies")
			}

			if tc.name == "operator request" && validator.requirement != policy {
				t.Fatal("request changed operator policy")
			}
		})
	}
}

// A saturated operator route is rejected before actual session/account lookup.
func TestResetOperatorAdmission(t *testing.T) {
	validator := &resetValidator{roles: []string{"admin"}}

	h, err := NewPasswordResetHandler(&resetTransport{}, validator, auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "test-v1", MaxAge: time.Minute})
	if err != nil {
		t.Fatal(err)
	}

	if h.admission.acknowledgment != 6*time.Second {
		t.Fatal("production response bound changed")
	}

	h.admission.active = 32
	router := chi.NewRouter()
	h.RegisterRoutes(router)

	r := httptest.NewRequest("POST", "https://example.com/admin/password-reset", strings.NewReader("email=user%40example.com"))
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "actor"})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != 429 || validator.calls != 0 {
		t.Fatal("capacity failed before-lookup contract")
	}
}
