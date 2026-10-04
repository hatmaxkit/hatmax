// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"hatmax.adrianpk.com/auth"
)

type mailboxTransport struct {
	calls   int
	failure error
}

func (s *mailboxTransport) RequestMailboxVerification(context.Context, string) error {
	s.calls++

	return s.failure
}
func (s *mailboxTransport) ConfirmMailboxVerification(context.Context, string) error {
	s.calls++

	return s.failure
}

// Production routes protect mutations, restrict body shape, group operating and
// unavailable failures and clear cookies only for a committed confirmation.
func TestMailboxHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, origin string
		failure                          error
		status, calls                    int
	}{
		{"request", "POST", "/account/mailbox", "email=user%40example.com", "http://example.com", nil, 202, 1},
		{"unknown mailbox", "POST", "/account/mailbox", "email=absent%40example.com", "http://example.com", auth.ErrRecoveryUnavailable, 202, 1},
		{"mail failure", "POST", "/account/mailbox", "email=user%40example.com", "http://example.com", errors.New("private token diagnostics"), 202, 1},
		{"confirm", "POST", "/account/mailbox/confirm", "token=" + strings.Repeat("a", 80), "http://example.com", nil, 200, 1},
		{"replay", "POST", "/account/mailbox/confirm", "token=" + strings.Repeat("a", 80), "http://example.com", auth.ErrRecoveryUnavailable, 403, 1},
		{"confirm failure", "POST", "/account/mailbox/confirm", "token=" + strings.Repeat("a", 80), "http://example.com", errors.New("private token diagnostics"), 403, 1},
		{"missing origin", "POST", "/account/mailbox", "email=x", "", nil, 403, 0},
		{"foreign origin", "POST", "/account/mailbox", "email=x", "http://evil.example.com", nil, 403, 0},
		{"duplicate", "POST", "/account/mailbox", "email=x&email=y", "http://example.com", nil, 400, 0},
		{"foreign field", "POST", "/account/mailbox", "email=x&purpose=reset", "http://example.com", nil, 400, 0},
		{"large body", "POST", "/account/mailbox", "email=" + strings.Repeat("x", 16<<10), "http://example.com", nil, 400, 0},
		{"GET safety", "GET", "/account/mailbox/confirm?token=private", "", "", nil, 200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mailboxTransport{failure: tc.failure}

			h, err := NewMailboxHandler(svc)
			if err != nil {
				t.Fatal(err)
			}

			h.acknowledgment = time.Millisecond
			router := chi.NewRouter()
			h.RegisterRoutes(router)

			req := httptest.NewRequest(tc.method, "http://example.com"+tc.path, strings.NewReader(tc.body))
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.status || svc.calls != tc.calls || strings.Contains(w.Body.String(), "private") {
				t.Fatalf("response %d calls %d body %s", w.Code, svc.calls, w.Body.String())
			}

			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Fatal("unsafe caching policy")
			}

			for _, c := range w.Result().Cookies() {
				if tc.name != "confirm" || c.Name != auth.SessionCookieName || c.Value != "" || c.MaxAge >= 0 {
					t.Fatal("unexpected authentication cookie")
				}
			}
		})
	}
}

// A real handler deadline pads both accepted and unknown requests equally;
// admission is bounded before lookup and does not trust forwarded identities.
func TestMailboxIngress(t *testing.T) {
	svc := &mailboxTransport{}

	h, err := NewMailboxHandler(svc)
	if err != nil {
		t.Fatal(err)
	}

	if h.acknowledgment != 6*time.Second {
		t.Fatal("public deadline changed")
	}

	req := httptest.NewRequest("POST", "http://example.com/account/mailbox", nil)
	for range 12 {
		if !h.admit(req) {
			t.Fatal("valid admission rejected")
		}

		h.release()
	}

	if h.admit(req) {
		t.Fatal("IP budget exceeded")
	}

	h.windows = map[string]mailboxWindow{}

	h.active = 32
	if h.admit(req) {
		t.Fatal("active capacity exceeded")
	}

	h.active = 0
	req.RemoteAddr = "untrusted"
	req.Header.Set("X-Forwarded-For", "192.0.2.2")

	if h.admit(req) {
		t.Fatal("forwarded identity bypassed admission")
	}

	h.windows = make(map[string]mailboxWindow)
	for i := range 1024 {
		h.windows[time.Unix(int64(i), 0).String()] = mailboxWindow{start: time.Now()}
	}

	req.RemoteAddr = "192.0.2.2:80"
	if h.admit(req) {
		t.Fatal("map capacity exceeded")
	}
}
