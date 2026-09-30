// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireSameOrigin(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		origin     string
		referer    string
		fetchSite  string
		forwarded  string
		tls        bool
		wantStatus int
		wantCalled bool
	}{
		{name: "safe request", method: http.MethodGet, wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "same origin", method: http.MethodPost, origin: "http://example.com", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "same origin referer", method: http.MethodPost, referer: "http://example.com/form", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "explicit default port", method: http.MethodPost, origin: "http://example.com:80", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "direct tls", method: http.MethodPost, origin: "https://example.com", tls: true, wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "forwarded tls", method: http.MethodPost, origin: "https://example.com", forwarded: "https", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "forwarded list", method: http.MethodPost, origin: "https://example.com", forwarded: "https, http", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "missing source", method: http.MethodPost, wantStatus: http.StatusForbidden},
		{name: "cross origin", method: http.MethodPost, origin: "http://attacker.example", wantStatus: http.StatusForbidden},
		{name: "cross scheme", method: http.MethodPost, origin: "https://example.com", wantStatus: http.StatusForbidden},
		{name: "opaque origin", method: http.MethodPost, origin: "null", wantStatus: http.StatusForbidden},
		{name: "user info", method: http.MethodPost, origin: "http://user@example.com", wantStatus: http.StatusForbidden},
		{name: "cross site signal", method: http.MethodPost, origin: "http://example.com", fetchSite: "cross-site", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := RequireSameOrigin(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				called = true

				response.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(test.method, "http://example.com/resource", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Referer", test.referer)
			request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			request.Header.Set("X-Forwarded-Proto", test.forwarded)

			if test.tls {
				request.TLS = &tls.ConnectionState{}
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, test.wantStatus)
			}

			if called != test.wantCalled {
				t.Errorf("called = %t, want %t", called, test.wantCalled)
			}
		})
	}
}
