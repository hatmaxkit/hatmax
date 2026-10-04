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
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
)

type assertionTransport struct {
	calls    int
	required auth.AccessRequirement
	fail     error
}

func (s *assertionTransport) BeginWebAuthnAuthentication(_ context.Context, _ string, r auth.AccessRequirement) (*auth.AssertionChallenge, error) {
	s.calls++
	s.required = r

	return &auth.AssertionChallenge{Token: "assert1." + strings.Repeat("A", 43)}, s.fail
}
func (s *assertionTransport) BeginWebAuthnStepUp(_ context.Context, _ string, r auth.AccessRequirement) (*auth.AssertionChallenge, error) {
	s.calls++
	s.required = r

	return &auth.AssertionChallenge{Token: "stepup1." + strings.Repeat("A", 43)}, s.fail
}
func (s *assertionTransport) FinishWebAuthn(_ context.Context, _ string, _ []byte, r auth.AccessRequirement) (*auth.IssuedSession, error) {
	s.calls++
	s.required = r

	return &auth.IssuedSession{Session: auth.Session{ID: "safe", ExpiresAt: time.Now().Add(time.Hour)}, Token: "ordinary-bearer"}, s.fail
}
func (s *assertionTransport) ValidateSession(_ context.Context, _ string, r auth.AccessRequirement, _ auth.SessionActivity) (*auth.ValidatedSession, error) {
	s.calls++
	s.required = r

	return &auth.ValidatedSession{Session: auth.Session{ID: "safe"}}, s.fail
}

// HTTP fakes test transport/cookie isolation only. Actual protocol proof and SQL
// atomicity are independently established by TestWebAuthnTransactions.
func TestWebAuthnHTTP(t *testing.T) {
	tests := []struct {
		name, path, body, token, origin string
		fail                            error
		status, calls                   int
		cookie                          bool
	}{
		{name: "begin", path: "authentication/begin", body: `{"email":"user@example.com"}`, origin: "http://example.com", status: 200, calls: 1},
		{name: "step-up begin", path: "step-up/begin", body: `{}`, origin: "http://example.com", status: 200, calls: 1},
		{name: "finish", path: "authentication/finish", body: `{}`, token: "assert1.", origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "step-up finish", path: "step-up/finish", body: `{}`, token: "stepup1.", origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "missing origin", path: "authentication/begin", body: `{}`, status: 403},
		{name: "foreign origin", path: "authentication/finish", body: `{}`, token: "assert1.", origin: "http://other.com", status: 403},
		{name: "policy injection", path: "authentication/begin", body: `{"email":"user@example.com","proof":1}`, origin: "http://example.com", status: 403},
		{name: "trailing JSON", path: "authentication/begin", body: `{"email":"user@example.com"} {}`, origin: "http://example.com", status: 403},
		{name: "body bound", path: "authentication/finish", body: strings.Repeat("x", auth.MaxEnrollmentBody+1), token: "assert1.", origin: "http://example.com", status: 403},
		{name: "wrong finish purpose", path: "authentication/finish", body: `{}`, token: "stepup1.", origin: "http://example.com", status: 403},
		{name: "wrong step-up purpose", path: "step-up/finish", body: `{}`, token: "assert1.", origin: "http://example.com", status: 403},
		{name: "budget", path: "authentication/finish", body: `{}`, token: "assert1.", origin: "http://example.com", fail: auth.ErrEnrollmentAttempts, status: 429, calls: 1},
		{name: "commit failure", path: "authentication/finish", body: `{}`, token: "assert1.", origin: "http://example.com", fail: errors.New("private SQL details"), status: 403, calls: 1},
	}
	required := auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "server-v1", MaxAge: 5 * time.Minute}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &assertionTransport{fail: test.fail}

			handler, err := NewWebAuthnHandler(service, service, required)
			if err != nil {
				t.Fatal(err)
			}

			router := chi.NewRouter()
			handler.RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodPost, "http://example.com/authenticators/"+test.path, strings.NewReader(test.body))
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Content-Type", "application/json")

			if test.token != "" {
				request.Header.Set("X-Assertion-Token", test.token+strings.Repeat("A", 43))
			}

			request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "existing-bearer"})

			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			if w.Code != test.status || service.calls != test.calls || strings.Contains(w.Body.String(), "private SQL") || strings.Contains(w.Body.String(), "ordinary-bearer") {
				t.Fatalf("response status %d calls %d", w.Code, service.calls)
			}

			if (len(w.Result().Cookies()) != 0) != test.cookie {
				t.Fatal("pending/failure changed cookie")
			}

			if test.cookie {
				cookie := w.Result().Cookies()[0]
				if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Value != "ordinary-bearer" {
					t.Fatal("unsafe issued cookie")
				}
			}

			if service.calls > 0 && (service.required != required || w.Header().Get("Cache-Control") != "no-store") {
				t.Fatal("trusted policy/cache boundary changed")
			}
		})
	}
}

// A strong route revalidates current proof and never clears a usable weak cookie.
func TestWebAuthnRoute(t *testing.T) {
	for _, failure := range []error{nil, auth.ErrSessionProof, auth.ErrSessionProofExpired} {
		service := &assertionTransport{fail: failure}
		required := auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "server-v1", MaxAge: time.Minute}

		handler, err := NewWebAuthnHandler(service, service, required)
		if err != nil {
			t.Fatal(err)
		}

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		request := httptest.NewRequest(http.MethodGet, "http://example.com/authenticators/proof", nil)
		request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "existing-bearer"})

		w := httptest.NewRecorder()
		router.ServeHTTP(w, request)

		status := 200
		if failure != nil {
			status = 403
		}

		if w.Code != status || service.required != required || len(w.Result().Cookies()) != 0 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("strong route proof/cookie boundary changed")
		}
	}
}
