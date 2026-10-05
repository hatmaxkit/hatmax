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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pquerna/otp/totp"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
)

// Real production forms, PostgreSQL and captured mail establish neutral public
// issuance, one-use cookie-free completion and actual operator proof/role checks.
func TestPasswordResetTransportTransactions(t *testing.T) {
	db, q, base := mailboxHTTPDatabase(t)

	user, err := base.Signup(t.Context(), "user@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	s, err := core.NewRecoveryService(base, q, config.RecoveryConfig{}, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}

	capture := &mailboxHTTPMail{}

	delivery, err := authfeat.NewMailboxDelivery(s, q, capture, log.NewTestLogger("error"), "https://example.com", false)
	if err != nil {
		t.Fatal(err)
	}

	operator := authfeat.PasswordRequirement()
	operator.Proof = core.RequireMFA
	operator.MaxAge = 5 * time.Minute

	h, err := NewPasswordResetHandler(delivery, base, operator)
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	h.RegisterRoutes(router)

	request := func(method, path, origin, cookie string, fields url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://example.com"+path, strings.NewReader(fields.Encode()))
		// Operator journeys use a separate socket client so public admission
		// saturation cannot mask the current-session checks.
		if strings.HasPrefix(path, "/admin/") {
			r.RemoteAddr = "192.0.2.2:1234"
		}

		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: core.SessionCookieName, Value: cookie})
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		return w
	}
	neutral := ""
	initiate := func(email, path, cookie string) {
		t.Helper()

		started := time.Now()
		w := request("POST", path, "https://example.com", cookie, url.Values{"email": {email}})

		elapsed := time.Since(started)
		if w.Code != 202 || elapsed < 6*time.Second || elapsed > 8*time.Second || len(w.Result().Cookies()) != 0 {
			t.Fatalf("public acknowledgment: %d, %s", w.Code, elapsed)
		}

		if neutral == "" {
			neutral = w.Body.String()
		} else if w.Body.String() != neutral {
			t.Fatal("eligibility changed acknowledgment")
		}
	}
	initiate("missing@example.com", "/account/password/reset", "")
	initiate(user.Email, "/account/password/reset", "")

	if len(capture.messages) != 0 {
		t.Fatal("unverified destination received reset")
	}

	verify, err := s.RequestMailboxVerification(t.Context(), user.Email)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.ConfirmMailboxVerification(t.Context(), verify.Token.Bearer())
	if err != nil {
		t.Fatal(err)
	}

	out, err := base.Signin(t.Context(), user.Email, "a distinct safe password", authfeat.PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	actor, ok := out.CompletedSession()
	if !ok {
		t.Fatal("missing current actor")
	}

	_, err = db.ExecContext(t.Context(), "UPDATE users SET active=false WHERE id=$1", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	initiate(user.Email, "/account/password/reset", "")

	_, err = db.ExecContext(t.Context(), "UPDATE users SET active=true WHERE id=$1", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	capture.failure = true

	initiate(user.Email, "/account/password/reset", "")

	if len(capture.messages) != 1 {
		t.Fatal("failed provider issue not committed")
	}

	failedLink := strings.Split(capture.messages[0].Text, "\n")[1]
	failedToken := strings.TrimPrefix(failedLink, "https://example.com/account/password/reset/confirm#token=")
	capture.failure = false

	initiate(user.Email, "/account/password/reset", "")

	if len(capture.messages) != 2 {
		t.Fatal("eligible request not sent")
	}

	link := strings.Split(capture.messages[1].Text, "\n")[1]

	token := strings.TrimPrefix(link, "https://example.com/account/password/reset/confirm#token=")
	if len(token) != 80 || token == failedToken {
		t.Fatal("invalid trusted reset link")
	}
	// Saturated durable issuance is still publicly indistinguishable.
	_, err = db.ExecContext(t.Context(), "UPDATE recovery_budgets SET attempts=3 WHERE user_id=$1 AND kind=2", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	initiate(user.Email, "/account/password/reset", "")

	if len(capture.messages) != 2 {
		t.Fatal("throttled request dispatched")
	}

	for _, tc := range []struct {
		name, method, path, origin string
		fields                     url.Values
		code                       int
	}{
		{"GET", "GET", "/account/password/reset/confirm", "", nil, 200},
		{"foreign origin", "POST", "/account/password/reset/confirm", "https://other.example", url.Values{"token": {token}, "password": {"a complete reset password"}}, 403},
		{"forged subject", "POST", "/account/password/reset/confirm", "https://example.com", url.Values{"token": {token}, "password": {"a complete reset password"}, "subject": {"other"}}, 403},
		{"previous issued link", "POST", "/account/password/reset/confirm", "https://example.com", url.Values{"token": {failedToken}, "password": {"a complete reset password"}}, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := request(tc.method, tc.path, tc.origin, actor.Token, tc.fields)
			if w.Code != tc.code || len(w.Result().Cookies()) != 0 {
				t.Fatalf("unsafe response: %d", w.Code)
			}

			var version int64

			err := db.QueryRowContext(t.Context(), "SELECT auth_version FROM users WHERE id=$1", user.ID).Scan(&version)
			if err != nil {
				t.Fatal(err)
			}

			if version != actor.AuthVersion {
				t.Fatal("GET/denial mutated credential")
			}
		})
	}

	const password = "a complete reset password 😀"

	capture.failure = true

	w := request("POST", "/account/password/reset/confirm", "https://example.com", actor.Token, url.Values{"token": {token}, "password": {password}})
	if w.Code != 200 || strings.Contains(w.Body.String(), token) || strings.Contains(w.Body.String(), password) || strings.Contains(w.Body.String(), "private provider") {
		t.Fatalf("reset response: %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != core.SessionCookieName || cookies[0].Value != "" || cookies[0].MaxAge != -1 {
		t.Fatal("reset did not clear cookie without issuing access")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, authfeat.PasswordRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("old session survived reset")
	}

	var pending int

	err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM recovery_notices WHERE user_id=$1 AND kind=3 AND delivered_at IS NULL", user.ID).Scan(&pending)
	if err != nil || pending != 1 {
		t.Fatal("failed notification intent lost")
	}

	capture.failure = false

	err = delivery.DispatchMailboxNotices(t.Context(), user.ID, 5)
	if err != nil {
		t.Fatal(err)
	}

	w = request("POST", "/account/password/reset/confirm", "https://example.com", "", url.Values{"token": {token}, "password": {password}})
	if w.Code != 403 || len(w.Result().Cookies()) != 0 {
		t.Fatal("reset replay succeeded")
	}

	out, err = base.Signin(t.Context(), user.Email, password, authfeat.PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	_, ok = out.CompletedSession()
	if !ok {
		t.Fatal("new credential failed")
	}

	// Obtain actual qualified operator proof through the production TOTP engine,
	// using an explicitly permitted MFA profile rather than asserted session facts.
	admin, err := base.Signup(t.Context(), "operator@example.com", "a separate operator password")
	if err != nil {
		t.Fatal(err)
	}

	fallback, err := core.NewFallbackService(base, q, config.FallbackConfig{Issuer: "Ticked test"}, core.SeedKeys{Active: "test-v1", Keys: map[string][]byte{"test-v1": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}

	setup, err := fallback.BeginTOTPSetup(t.Context(), admin.Email, "a separate operator password", operator)
	if err != nil {
		t.Fatal(err)
	}

	uri, err := url.Parse(setup.URL)
	if err != nil {
		t.Fatal(err)
	}

	secret := uri.Query().Get("secret")

	code, err := totp.GenerateCode(secret, time.Now().Add(-30*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	err = fallback.ConfirmTOTPSetup(t.Context(), setup.Token, code, operator)
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := fallback.BeginFallbackAuthentication(t.Context(), admin.Email, "a separate operator password", core.FallbackTOTP, operator)
	if err != nil {
		t.Fatal(err)
	}

	code, err = totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	qualified, err := fallback.FinishFallback(t.Context(), challenge.Token, code, operator)
	if err != nil {
		t.Fatal(err)
	}

	w = request("POST", "/admin/password-reset", "https://example.com", qualified.Token, url.Values{"email": {user.Email}})
	if w.Code != 403 {
		t.Fatal("non-operator role accepted")
	}
	// Isolate the current-role check without a version shortcut; supported role
	// mutations also advance the authentication version.
	_, err = db.ExecContext(t.Context(), "UPDATE users SET roles=ARRAY['admin'] WHERE id=$1", admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Renew the target issuance window for this independent operator journey.
	_, err = db.ExecContext(t.Context(), "UPDATE recovery_budgets SET attempts=0 WHERE user_id=$1 AND kind=2", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	prior := len(capture.messages)

	initiate(user.Email, "/admin/password-reset", qualified.Token)

	if len(capture.messages) != prior+1 || !strings.Contains(capture.messages[len(capture.messages)-1].Text, "/account/password/reset/confirm#token=") {
		t.Fatal("operator did not use same mail-only flow")
	}

	strong := operator
	strong.Proof = core.RequirePhishingResistantMFA

	strongHandler, err := NewPasswordResetHandler(delivery, base, strong)
	if err != nil {
		t.Fatal(err)
	}

	strongRouter := chi.NewRouter()
	strongHandler.RegisterRoutes(strongRouter)

	r := httptest.NewRequest("POST", "https://example.com/admin/password-reset", strings.NewReader(url.Values{"email": {user.Email}}.Encode()))
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: core.SessionCookieName, Value: qualified.Token})

	strongResponse := httptest.NewRecorder()
	strongRouter.ServeHTTP(strongResponse, r)

	if strongResponse.Code != 303 || len(capture.messages) != prior+1 {
		t.Fatal("weaker proof bypassed strong operator policy")
	}

	_, err = db.ExecContext(t.Context(), "UPDATE users SET auth_version=auth_version+1 WHERE id=$1", admin.ID)
	if err != nil {
		t.Fatal(err)
	}

	w = request("POST", "/admin/password-reset", "https://example.com", qualified.Token, url.Values{"email": {user.Email}})
	if w.Code != 303 || len(capture.messages) != prior+1 {
		t.Fatal("stale operator accepted")
	}
}
