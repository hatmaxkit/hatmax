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

type fallbackTransport struct {
	calls    int
	required auth.AccessRequirement
	method   auth.FallbackMethod
	failure  error
}

func (s *fallbackTransport) called(r auth.AccessRequirement) { s.calls++; s.required = r }
func (s *fallbackTransport) BeginTOTPSetup(_ context.Context, _, _ string, r auth.AccessRequirement) (*auth.TOTPSetup, error) {
	s.called(r)

	return &auth.TOTPSetup{Token: "totpset1." + strings.Repeat("A", 43), URL: "otpauth://test"}, s.failure
}
func (s *fallbackTransport) ConfirmTOTPSetup(_ context.Context, _, _ string, r auth.AccessRequirement) error {
	s.called(r)

	return s.failure
}
func (s *fallbackTransport) BeginFallbackAuthentication(_ context.Context, _, _ string, m auth.FallbackMethod, r auth.AccessRequirement) (*auth.FallbackChallenge, error) {
	s.called(r)
	s.method = m

	return &auth.FallbackChallenge{Token: "fallback1." + strings.Repeat("A", 43)}, s.failure
}
func (s *fallbackTransport) BeginFallbackStepUp(_ context.Context, _, _ string, m auth.FallbackMethod, r auth.AccessRequirement) (*auth.FallbackChallenge, error) {
	s.called(r)
	s.method = m

	return &auth.FallbackChallenge{Token: "fallstep1." + strings.Repeat("A", 43)}, s.failure
}
func (s *fallbackTransport) FinishFallback(_ context.Context, _, _ string, r auth.AccessRequirement) (*auth.IssuedSession, error) {
	s.called(r)

	return &auth.IssuedSession{Session: auth.Session{ID: "safe", ExpiresAt: time.Now().Add(time.Hour)}, Token: "issued-bearer"}, s.failure
}
func (s *fallbackTransport) IssueBackupCodes(_ context.Context, _ string, r auth.AccessRequirement) ([]string, *auth.IssuedSession, error) {
	s.called(r)

	return []string{"immediate-code"}, &auth.IssuedSession{Session: auth.Session{ID: "safe", ExpiresAt: time.Now().Add(time.Hour)}, Token: "issued-bearer"}, s.failure
}
func (s *fallbackTransport) ValidateSession(_ context.Context, _ string, r auth.AccessRequirement, _ auth.SessionActivity) (*auth.ValidatedSession, error) {
	s.called(r)

	return &auth.ValidatedSession{Session: auth.Session{ID: "safe"}}, s.failure
}

// Transport fakes establish cookie/purpose/policy boundaries, not actual proof.
// Real cryptography and atomic persistence are covered by TestFallbackTransactions.
func TestFallbackHTTP(t *testing.T) {
	tests := []struct {
		name, path, body, prefix, origin string
		failure                          error
		status, calls                    int
		cookie                           bool
		method                           auth.FallbackMethod
		management                       bool
	}{
		{name: "initial setup", path: "totp/setup/begin", body: `{"email":"e","password":"p"}`, origin: "http://example.com", status: 200, calls: 1},
		{name: "setup confirmation", path: "totp/setup/finish", body: `{"code":"123456"}`, prefix: "totpset1.", origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "TOTP begin", path: "totp/authentication/begin", body: `{"email":"e","password":"p"}`, origin: "http://example.com", status: 200, calls: 1, method: auth.FallbackTOTP},
		{name: "backup begin", path: "backup/authentication/begin", body: `{"email":"e","password":"p"}`, origin: "http://example.com", status: 200, calls: 1, method: auth.FallbackBackup},
		{name: "bound step-up", path: "totp/step-up/begin", body: `{"password":"p"}`, origin: "http://example.com", status: 200, calls: 1, method: auth.FallbackTOTP},
		{name: "finish", path: "fallback/authentication/finish", body: `{"code":"123456"}`, prefix: "fallback1.", origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "step-up finish", path: "fallback/step-up/finish", body: `{"code":"123456"}`, prefix: "fallstep1.", origin: "http://example.com", status: 200, calls: 1, cookie: true},
		{name: "backup issue", path: "backup/issue", body: `{}`, origin: "http://example.com", status: 200, calls: 1, cookie: true, management: true},
		{name: "null command", path: "backup/issue", body: `null`, origin: "http://example.com", status: 403},
		{name: "policy injection", path: "totp/authentication/begin", body: `{"proof":3}`, origin: "http://example.com", status: 403},
		{name: "foreign origin", path: "totp/authentication/begin", body: `{}`, origin: "http://other.com", status: 403},
		{name: "missing origin", path: "totp/authentication/begin", body: `{}`, status: 403},
		{name: "trailing JSON", path: "backup/issue", body: `{} {}`, origin: "http://example.com", status: 403},
		{name: "excessive code", path: "fallback/authentication/finish", body: `{"code":"` + strings.Repeat("a", 129) + `"}`, prefix: "fallback1.", origin: "http://example.com", status: 403},
		{name: "excessive body", path: "fallback/authentication/finish", body: strings.Repeat("a", auth.MaxEnrollmentBody+1), prefix: "fallback1.", origin: "http://example.com", status: 403},
		{name: "setup as sign-in", path: "fallback/authentication/finish", body: `{"code":"123456"}`, prefix: "totpset1.", origin: "http://example.com", status: 403},
		{name: "sign-in as step-up", path: "fallback/step-up/finish", body: `{"code":"123456"}`, prefix: "fallback1.", origin: "http://example.com", status: 403},
		{name: "step-up as sign-in", path: "fallback/authentication/finish", body: `{"code":"123456"}`, prefix: "fallstep1.", origin: "http://example.com", status: 403},
		{name: "failed commit", path: "fallback/authentication/finish", body: `{"code":"123456"}`, prefix: "fallback1.", origin: "http://example.com", failure: errors.New("private SQL detail"), status: 403, calls: 1},
		{name: "durable budget", path: "fallback/authentication/finish", body: `{"code":"123456"}`, prefix: "fallback1.", origin: "http://example.com", failure: auth.ErrEnrollmentAttempts, status: 429, calls: 1},
	}
	access := auth.AccessRequirement{Proof: auth.RequireMFA, Revision: "server-policy", MaxAge: 5 * time.Minute}
	management := access
	management.Proof = auth.RequirePhishingResistantMFA

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &fallbackTransport{failure: test.failure}

			h, err := NewFallbackHandler(svc, svc, access, management)
			if err != nil {
				t.Fatal(err)
			}

			router := chi.NewRouter()
			h.RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodPost, "http://example.com/authenticators/"+test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", test.origin)

			if test.prefix != "" {
				request.Header.Set("X-Fallback-Token", test.prefix+strings.Repeat("A", 43))
			}

			request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "existing-bearer"})

			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.status || svc.calls != test.calls || svc.method != test.method {
				t.Fatalf("status %d calls %d method %d", response.Code, svc.calls, svc.method)
			}

			if strings.Contains(response.Body.String(), "issued-bearer") || strings.Contains(response.Body.String(), "private SQL") {
				t.Fatal("response leaked bearer or error details")
			}

			cookies := response.Result().Cookies()
			if (len(cookies) != 0) != test.cookie {
				t.Fatal("pending/failure changed cookie")
			}

			required := access
			if test.management {
				required = management
			}

			if test.calls != 0 && (svc.required != required || response.Header().Get("Cache-Control") != "no-store") {
				t.Fatal("trusted policy/cache lost")
			}

			if test.cookie && test.path != "totp/setup/finish" && (!cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Value != "issued-bearer") {
				t.Fatal("unsafe cookie")
			}
		})
	}
}
