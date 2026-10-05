//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/ui"
	baseweb "hatmax.adrianpk.com/web"
)

func securityLogEvents(t *testing.T, capture *ingressLog) []core.SecurityEvent {
	t.Helper()
	capture.mu.Lock()
	defer capture.mu.Unlock()

	events := []core.SecurityEvent{}

	for _, line := range capture.lines {
		if !strings.HasPrefix(line, "Authentication security event: ") {
			continue
		}

		var event core.SecurityEvent

		err := json.Unmarshal([]byte(strings.TrimPrefix(line, "Authentication security event: ")), &event)
		if err != nil {
			t.Fatal(err)
		}

		err = event.Check()
		if err != nil {
			t.Fatal(err)
		}

		events = append(events, event)
	}

	return events
}

// Real production handlers, SQL and the application observer preserve neutral
// responses and cookie authority while recording exactly one redacted core event.
func TestAuthenticationObservationTransactions(t *testing.T) {
	db, q, _ := mailboxHTTPDatabase(t)
	cfg := config.New()
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1
	cfg.AuthenticationIngress.PeerRequests = 1000
	cfg.CredentialAdmission.PasswordAttempts = 20
	capture := &ingressLog{}

	observations, err := core.NewSecurityObservations(authfeat.SecurityLogger{Logger: capture}, cfg.SecurityObservation)
	if err != nil {
		t.Fatal(err)
	}

	base, err := core.NewService(q, cfg, authfeat.NewPasswordChecker(), webCredentialAdmission(t, q, cfg.CredentialAdmission), observations, capture)
	if err != nil {
		t.Fatal(err)
	}

	templates := baseweb.NewTemplateManager(testAssetsFS, capture, baseweb.WithFuncMap(ui.FuncMap()))

	err = templates.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	handler := &Handler{authSvc: authfeat.NewService(base, q, capture), log: capture, tmpl: templates}
	router := chi.NewRouter()
	router.Post("/signup", handler.handleSignup)
	router.Post("/signin", handler.handleSignin)
	router.Get("/signin", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	server, _ := ingressServer(t, cfg, router, false)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	password := "a distinct safe password"
	email := "observed@example.com"
	signup := url.Values{"email": {email}, "password": {password}, "confirm_password": {password}}.Encode()
	first := ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", signup, nil)

	duplicate := ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", signup, nil)
	if first.status != http.StatusSeeOther || duplicate.status != first.status || duplicate.header.Get("Location") != first.header.Get("Location") || first.header.Get("Set-Cookie") != "" || duplicate.header.Get("Set-Cookie") != "" {
		t.Fatal("observation changed neutral registration")
	}

	events := securityLogEvents(t, capture)
	if len(events) != 2 || events[0].Operation != core.SecurityRegistration || events[0].Outcome != core.SecurityCommitted || events[1].Outcome != core.SecurityPolicyRejected {
		t.Fatal("registration event provenance or duplicate helper emission")
	}

	malformed := ingressHTTP(t, client, "POST", server.URL+"/signin?token=private-link", "application/x-www-form-urlencoded", "email=raw%40example.com&password=private-password&password=duplicate", nil)
	if malformed.status != http.StatusBadRequest || len(securityLogEvents(t, capture)) != 2 {
		t.Fatal("syntax refusal reached a core mutation")
	}

	_ = ingressHTTP(t, client, "GET", server.URL+"/signin?token=private-link", "", "", nil)
	if len(securityLogEvents(t, capture)) != 2 {
		t.Fatal("page GET emitted security mutation")
	}

	wrong := url.Values{"email": {email}, "password": {"wrong private password"}}.Encode()

	denied := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", wrong, nil)
	if denied.status != http.StatusForbidden || denied.header.Get("Set-Cookie") != "" {
		t.Fatal("denied event authorized cookie")
	}

	signin := url.Values{"email": {email}, "password": {password}}.Encode()

	accepted := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", signin, nil)
	if accepted.status != http.StatusOK || accepted.header.Get("HX-Redirect") != "/list-items" || accepted.header.Get("Set-Cookie") == "" {
		t.Fatal("observer prevented actual session cookie")
	}

	response := http.Response{Header: accepted.header}

	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("observed login lost safe cookie properties")
	}

	_, err = base.ValidateSession(t.Context(), cookies[0].Value, authfeat.PasswordRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal("observed cookie has no actual authority")
	}

	events = securityLogEvents(t, capture)
	if len(events) != 4 || events[2].Outcome != core.SecurityInvalidProof || events[2].Proof != 0 || events[3].Outcome != core.SecurityAuthenticated || events[3].Proof != core.PasswordProof || events[3].Record == "" {
		t.Fatal("HTTP event outcome/proof provenance")
	}

	var sessions int

	err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions WHERE id=$1", events[3].Record).Scan(&sessions)
	if err != nil || sessions != 1 {
		t.Fatal("authenticated event has no committed session")
	}

	capture.mu.Lock()
	lines := strings.Join(capture.lines, "\n")
	capture.mu.Unlock()

	for _, secret := range []string{email, password, "wrong private password", "private-link", "private-password", accepted.header.Get("Set-Cookie")} {
		if strings.Contains(lines, secret) {
			t.Fatal("application logs leaked credential input or bearer")
		}
	}

	if observations.Diagnostics().Delivered != 4 {
		t.Fatal("expected timely application observations")
	}
}
