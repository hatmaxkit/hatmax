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

type factorTransport struct {
	calls  int
	policy auth.FactorPolicy
	fail   error
	revoke bool
}

func (s *factorTransport) List(_ context.Context, _ string, p auth.FactorPolicy) ([]auth.Factor, error) {
	s.calls++
	s.policy = p

	return []auth.Factor{{FactorSelection: auth.FactorSelection{Kind: auth.FactorWebAuthn, ID: "safe-factor", Revision: 1}}}, s.fail
}
func (s *factorTransport) BeginWebAuthnChange(_ context.Context, _ string, _ auth.FactorSelection, p auth.FactorPolicy) (*auth.EnrollmentChallenge, error) {
	s.calls++
	s.policy = p

	return &auth.EnrollmentChallenge{Token: "change1.transient"}, s.fail
}
func (s *factorTransport) BeginTOTPChange(_ context.Context, _ string, _ auth.FactorSelection, p auth.FactorPolicy) (*auth.TOTPSetup, error) {
	s.calls++
	s.policy = p

	return &auth.TOTPSetup{Token: "change1.transient", URL: "otpauth://transient"}, s.fail
}
func (s *factorTransport) result(p auth.FactorPolicy) (*auth.FactorChangeResult, error) {
	s.calls++
	s.policy = p

	result := &auth.FactorChangeResult{}
	if !s.revoke {
		result.Issued = &auth.IssuedSession{Token: "private-bearer", Session: auth.Session{ExpiresAt: time.Now().Add(time.Hour)}}
	}

	return result, s.fail
}
func (s *factorTransport) FinishWebAuthnChange(_ context.Context, _ string, _ string, _ []byte, p auth.FactorPolicy) (*auth.FactorChangeResult, error) {
	return s.result(p)
}
func (s *factorTransport) FinishTOTPChange(_ context.Context, _ string, _ string, _ string, p auth.FactorPolicy) (*auth.FactorChangeResult, error) {
	return s.result(p)
}
func (s *factorTransport) Remove(_ context.Context, _ string, _ auth.FactorSelection, p auth.FactorPolicy) (*auth.FactorChangeResult, error) {
	return s.result(p)
}

// Transport fakes establish JSON/origin, trusted policy and cookie boundaries;
// actual proof/transaction authority is covered by the real PostgreSQL tests.
func TestFactorHTTP(t *testing.T) {
	tests := []struct {
		name, path, body, origin string
		fail                     error
		status, calls            int
		cookie, revoke           bool
	}{
		{name: "additional begin", path: "webauthn/begin", body: `{}`, origin: "http://example.com", status: 200, calls: 1},
		{name: "TOTP begin", path: "totp/begin", body: `{}`, origin: "http://example.com", status: 200, calls: 1},
		{name: "WebAuthn finish", path: "webauthn/finish", body: `{}`, origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "TOTP finish", path: "totp/finish", body: `{"code":"123456"}`, origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "remove actor proof", path: "remove", body: `{"target":{"kind":1,"id":"own","revision":1}}`, origin: "http://example.com", status: 200, calls: 1, cookie: true, revoke: true},
		{name: "policy injection", path: "webauthn/begin", body: `{"proof":1}`, origin: "http://example.com", status: 403},
		{name: "owner injection", path: "remove", body: `{"target":{"kind":1,"id":"own","revision":1,"userId":"foreign"}}`, origin: "http://example.com", status: 403},
		{name: "non-object", path: "webauthn/begin", body: `null`, origin: "http://example.com", status: 403},
		{name: "trailing JSON", path: "webauthn/begin", body: `{} {}`, origin: "http://example.com", status: 403},
		{name: "missing origin", path: "remove", body: `{}`, status: 403},
		{name: "foreign origin", path: "remove", body: `{}`, origin: "http://foreign.example.com", status: 403},
		{name: "body bound", path: "webauthn/finish", body: strings.Repeat("x", auth.MaxEnrollmentBody+1), origin: "http://example.com", status: 403},
		{name: "budget", path: "webauthn/finish", body: `{}`, origin: "http://example.com", fail: auth.ErrEnrollmentAttempts, status: 429, calls: 1},
		{name: "commit failure", path: "webauthn/finish", body: `{}`, origin: "http://example.com", fail: errors.New("private storage details"), status: 403, calls: 1},
	}
	policy := auth.FactorPolicy{Management: auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "server-v1", MaxAge: 5 * time.Minute}, Access: auth.RequirePhishingResistantMFA}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &factorTransport{fail: test.fail, revoke: test.revoke}

			handler, err := NewFactorHandler(svc, policy, true)
			if err != nil {
				t.Fatal(err)
			}

			router := chi.NewRouter()
			handler.RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodPost, "http://example.com/authenticators/factors/"+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-Factor-Change-Token", "change1.transient")
			request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "actor-bearer"})

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.status || svc.calls != test.calls {
				t.Fatalf("status/calls: %d/%d", response.Code, svc.calls)
			}

			cookies := response.Result().Cookies()
			if (len(cookies) > 0) != test.cookie {
				t.Fatal("failure/begin crossed cookie boundary")
			}

			if test.cookie && test.revoke && cookies[0].MaxAge != -1 {
				t.Fatal("revoked actor cookie retained")
			}

			if strings.Contains(response.Body.String(), "private-bearer") || strings.Contains(response.Body.String(), "private storage") {
				t.Fatal("private data exposed")
			}

			if svc.calls > 0 && svc.policy != policy {
				t.Fatal("client selected management policy")
			}

			if response.Header().Get("Cache-Control") != "no-store" && svc.calls > 0 {
				t.Fatal("response can be cached")
			}
		})
	}

	t.Run("safe presentation", func(t *testing.T) {
		svc := &factorTransport{}

		handler, err := NewFactorHandler(svc, policy, false)
		if err != nil {
			t.Fatal(err)
		}

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		request := httptest.NewRequest(http.MethodGet, "http://example.com/authenticators/manage", nil)
		request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "actor-bearer"})

		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		if response.Code != 200 || !strings.Contains(response.Body.String(), "safe-factor") || strings.Contains(response.Body.String(), "Add TOTP") {
			t.Fatal("safe presentation/configuration mismatch")
		}
	})
}
