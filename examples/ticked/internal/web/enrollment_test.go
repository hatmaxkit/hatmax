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

type enrollmentTransport struct {
	calls    int
	fail     error
	required auth.AccessRequirement
	token    string
}

func (s *enrollmentTransport) BeginWebAuthnEnrollment(_ context.Context, _, _ string, r auth.AccessRequirement) (*auth.EnrollmentChallenge, error) {
	s.calls++
	s.required = r

	return &auth.EnrollmentChallenge{Token: "restricted-enrollment"}, s.fail
}
func (s *enrollmentTransport) FinishWebAuthnEnrollment(_ context.Context, token string, _ []byte, r auth.AccessRequirement) (*auth.Authenticator, error) {
	s.calls++
	s.required = r
	s.token = token

	return &auth.Authenticator{ID: "confirmed"}, s.fail
}

// Transport evidence covers trusted policy, same-origin/body admission and cookie
// isolation. The fake covers HTTP only; protocol verification has separate tests.
func TestEnrollmentHTTP(t *testing.T) {
	tests := []struct {
		name, path, origin, body string
		fail                     error
		wantStatus, wantCalls    int
	}{
		{"begin", "begin", "http://example.com", `{"email":"user@example.com","password":"a safe password"}`, nil, 200, 1},
		{"finish", "finish", "http://example.com", `{}`, nil, 200, 1},
		{"missing origin", "begin", "", `{}`, nil, 403, 0},
		{"foreign origin", "begin", "http://evil.example.com", `{}`, nil, 403, 0},
		{"policy injection", "begin", "http://example.com", `{"proof":1}`, nil, 403, 0},
		{"trailing JSON", "begin", "http://example.com", `{} {}`, nil, 403, 0},
		{"body bound", "finish", "http://example.com", strings.Repeat("x", auth.MaxEnrollmentBody+1), nil, 403, 0},
		{"database failure", "finish", "http://example.com", `{}`, errors.New("private SQL details"), 403, 1},
		{"budget", "finish", "http://example.com", `{}`, auth.ErrEnrollmentAttempts, 429, 1},
	}
	required := auth.AccessRequirement{Proof: auth.RequirePhishingResistantMFA, Revision: "server-v1", MaxAge: 5 * time.Minute}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &enrollmentTransport{fail: test.fail}

			handler, err := NewEnrollmentHandler(svc, required)
			if err != nil {
				t.Fatal(err)
			}

			router := chi.NewRouter()
			handler.RegisterRoutes(router)

			request := httptest.NewRequest(http.MethodPost, "http://example.com/authenticators/enrollment/"+test.path, strings.NewReader(test.body))
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Enrollment-Token", "restricted-enrollment")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, request)

			if w.Code != test.wantStatus || svc.calls != test.wantCalls || strings.Contains(w.Body.String(), "private SQL") {
				t.Fatalf("transport bounds: status %d calls %d", w.Code, svc.calls)
			}

			if svc.calls != 0 && svc.required != required {
				t.Fatal("request weakened trusted policy")
			}

			for _, cookie := range w.Result().Cookies() {
				if cookie.MaxAge >= 0 || test.path != "finish" || test.fail != nil {
					t.Fatal("enrollment response issued or changed authority")
				}
			}

			if test.path == "finish" && svc.calls != 0 && svc.token != "restricted-enrollment" {
				t.Fatal("pending bearer was not read from its dedicated header")
			}
		})
	}
}
