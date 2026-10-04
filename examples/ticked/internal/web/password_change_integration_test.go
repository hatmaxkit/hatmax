//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
)

// Production forms, actual storage/verifier and captured failed mail establish
// cookie revocation and safe success after committed notification intent.
func TestPasswordChangeTransportTransactions(t *testing.T) {
	db, q, base := mailboxHTTPDatabase(t)

	user, err := base.Signup(t.Context(), "user@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	outcome, err := base.Signin(t.Context(), user.Email, "a distinct safe password", authfeat.PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	actor, ok := outcome.CompletedSession()
	if !ok {
		t.Fatal("missing actor")
	}

	s, err := core.NewRecoveryService(base, q, config.RecoveryConfig{}, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}

	captured := &mailboxHTTPMail{failure: true}

	delivery, err := authfeat.NewMailboxDelivery(s, q, captured, log.NewTestLogger("error"), "https://example.com", false)
	if err != nil {
		t.Fatal(err)
	}

	policy := core.PasswordChangePolicy{Requirement: authfeat.PasswordRequirement(), AllowPassword: true}
	policy.Requirement.Proof = core.RequirePhishingResistantMFA

	h, err := NewPasswordChangeHandler(delivery, policy)
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	h.RegisterRoutes(router)

	request := func(method, origin, token string, fields url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://example.com/account/password", strings.NewReader(fields.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", origin)

		if token != "" {
			r.AddCookie(&http.Cookie{Name: core.SessionCookieName, Value: token})
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		return w
	}
	for _, tc := range []struct {
		name, method, origin, token string
		fields                      url.Values
		code                        int
	}{
		{"GET", http.MethodGet, "", actor.Token, nil, 200},
		{"foreign origin", http.MethodPost, "https://foreign.example", actor.Token, url.Values{"password": {"a new complete password"}}, 403},
		{"forged subject", http.MethodPost, "https://example.com", actor.Token, url.Values{"password": {"a new complete password"}, "subject": {"other"}}, 400},
		{"forged bearer", http.MethodPost, "https://example.com", "invalid", url.Values{"password": {"a new complete password"}}, 403},
		{"short candidate", http.MethodPost, "https://example.com", actor.Token, url.Values{"password": {"short"}}, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := request(tc.method, tc.origin, tc.token, tc.fields)
			if w.Code != tc.code || len(w.Result().Cookies()) != 0 {
				t.Fatalf("denial response: %d", w.Code)
			}

			var version int
			err := db.QueryRowContext(t.Context(), "SELECT auth_version FROM users").Scan(&version)
			if err != nil {
				t.Fatal(err)
			}

			if version != 1 || len(captured.messages) != 0 {
				t.Fatal("denial changed state or sent mail")
			}
		})
	}

	const password = "a new complete password 😀"

	w := request(http.MethodPost, "https://example.com", actor.Token, url.Values{"password": {password}})
	if w.Code != 200 || strings.Contains(w.Body.String(), "private provider") || strings.Contains(w.Body.String(), password) || len(captured.messages) != 1 {
		t.Fatalf("commit/mail failure response: %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 || cookies[0].Value != "" {
		t.Fatal("cookie not safely cleared")
	}

	if _, err = base.ValidateSession(t.Context(), actor.Token, authfeat.PasswordRequirement(), core.NoActivity); err == nil {
		t.Fatal("old cookie remains valid")
	}

	var version, notices, attempts int
	if err = db.QueryRowContext(t.Context(), "SELECT auth_version FROM users").Scan(&version); err != nil {
		t.Fatal(err)
	}

	if err = db.QueryRowContext(t.Context(), "SELECT count(*),sum(attempts) FROM recovery_notices WHERE kind=2 AND delivered_at IS NULL").Scan(&notices, &attempts); err != nil {
		t.Fatal(err)
	}

	if version != 2 || notices != 1 || attempts != 1 {
		t.Fatal("committed notice not retained")
	}

	if strings.Contains(captured.messages[0].Text, password) || strings.Contains(captured.messages[0].Text, actor.Token) {
		t.Fatal("notification contains credential or bearer")
	}

	captured.failure = false

	if err = delivery.DispatchMailboxNotices(t.Context(), actor.UserID, 1); err != nil {
		t.Fatal(err)
	}

	if len(captured.messages) != 2 || captured.messages[1].Subject != "Your Ticked password was changed" {
		t.Fatal("wrong notification retry")
	}

	if w = request(http.MethodPost, "https://example.com", actor.Token, url.Values{"password": {password}}); w.Code != 403 {
		t.Fatal("replayed actor changed password")
	}

	outcome, err = base.Signin(t.Context(), user.Email, password, authfeat.PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok = outcome.CompletedSession(); !ok {
		t.Fatal("new credential does not authenticate")
	}
}
