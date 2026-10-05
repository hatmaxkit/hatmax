//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/middleware"
	"hatmax.adrianpk.com/ui"
	baseweb "hatmax.adrianpk.com/web"
)

type ingressHTTPResult struct {
	status  int
	header  http.Header
	body    string
	elapsed time.Duration
}

func ingressHTTP(t *testing.T, client *http.Client, method, endpoint, contentType, body string, headers map[string]string) ingressHTTPResult {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Content-Type", contentType)

	origin, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Origin", origin.Scheme+"://"+origin.Host)

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	start := time.Now()

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	return ingressHTTPResult{status: response.StatusCode, header: response.Header, body: string(data), elapsed: time.Since(start)}
}

func ingressServer(t *testing.T, cfg *config.Config, next http.Handler, trusted bool) (*httptest.Server, *AuthenticationIngress) {
	t.Helper()

	guard, err := NewAuthenticationIngress(cfg)
	if err != nil {
		t.Fatal(err)
	}

	handler := guard.Middleware(next)
	if trusted {
		handler = middleware.ProxyHeaders(netip.MustParsePrefix("127.0.0.0/8"))(handler)
	}

	server := httptest.NewUnstartedServer(handler)
	server.Config.ReadHeaderTimeout = guard.settings.WorkTimeout
	server.Config.ReadTimeout = guard.settings.WorkTimeout
	server.Config.WriteTimeout = guard.settings.Acknowledgment + time.Second
	server.Start()
	t.Cleanup(server.Close)
	t.Cleanup(guard.Close)

	return server, guard
}

// Actual HTTP connections prove shared peer/active bounds, cancellation and close.
// PostgreSQL is mandatory even when a rejected request must not reach its adapter.
func TestAuthenticationIngressTransactions(t *testing.T) {
	_, queries, base := mailboxHTTPDatabase(t)
	cfg := config.New()
	cfg.AuthenticationIngress.PeerRequests = 1000
	cfg.AuthenticationIngress.MaxActive = 1
	logger := log.NewTestLogger("error")
	real := authfeat.NewService(base, queries, logger)
	observed := &observedIngressAuth{authService: real}
	handler := &Handler{authSvc: observed, log: logger}
	router := chi.NewRouter()
	router.Post("/signin", handler.handleSignin)
	router.Post("/signup", handler.handleSignup)
	server, guard := ingressServer(t, cfg, router, false)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	form := url.Values{"email": {"unknown@example.com"}, "password": {"a distinct safe password"}}.Encode()

	t.Run("active acknowledgment", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/signin", strings.NewReader(form))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", server.URL)

		done := make(chan error, 1)

		go func() {
			response, callErr := client.Do(req)
			if response != nil {
				response.Body.Close()
			}

			done <- callErr
		}()

		deadline := time.Now().Add(3 * time.Second)
		for observed.finishedSignins.Load() == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		if observed.finishedSignins.Load() != 1 {
			t.Fatal("actual password entry did not finish before acknowledgment")
		}

		rejected := ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", "", nil)
		if rejected.status != 429 || observed.signups.Load() != 0 || rejected.elapsed > time.Second {
			t.Fatal("active capacity did not reject before body/account work")
		}

		guard.mu.Lock()
		active := guard.active
		guard.mu.Unlock()

		if active != 1 {
			t.Fatal("acknowledgment released active capacity early")
		}

		cancel()

		err = <-done
		if err == nil {
			t.Fatal("canceled request unexpectedly completed")
		}

		deadline = time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			guard.mu.Lock()
			active = guard.active
			guard.mu.Unlock()

			if active == 0 {
				break
			}

			time.Sleep(time.Millisecond)
		}

		if active != 0 {
			t.Fatal("cancellation retained active capacity")
		}
	})

	t.Run("body and syntax admission", func(t *testing.T) {
		for _, tc := range []struct{ name, path, media, body, origin string }{
			{"duplicate field", "/signin", "application/x-www-form-urlencoded", form + "&password=second", server.URL},
			{"unknown field", "/signin", "application/x-www-form-urlencoded", form + "&proof=password", server.URL},
			{"query injection", "/signin?email=other", "application/x-www-form-urlencoded", form, server.URL},
			{"oversized body", "/signin", "application/x-www-form-urlencoded", strings.Repeat("x", 16<<10+1), server.URL},
			{"invalid identity", "/signin", "application/x-www-form-urlencoded", "email=a%0A%40example.com&password=safe", server.URL},
			{"invalid encoding", "/signin", "application/x-www-form-urlencoded", "email=a%40example.com&password=%FF", server.URL},
			{"invalid media", "/signin", "text/plain", form, server.URL},
			{"foreign origin", "/signin", "application/x-www-form-urlencoded", form, "https://foreign.example.com"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				before := observed.signins.Load()
				result := ingressHTTP(t, client, "POST", server.URL+tc.path, tc.media, tc.body, map[string]string{"Origin": tc.origin})

				want := 400
				if tc.name == "foreign origin" {
					want = 403
				}

				if result.status != want || observed.signins.Load() != before || result.elapsed > time.Second {
					t.Fatal("invalid input reached identity work")
				}
			})
		}
	})

	t.Run("stalled body", func(t *testing.T) {
		conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		err = conn.SetDeadline(time.Now().Add(8 * time.Second))
		if err != nil {
			t.Fatal(err)
		}

		start := time.Now()

		_, err = fmt.Fprintf(conn, "POST /signin HTTP/1.1\r\nHost: %s\r\nOrigin: %s\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: 100\r\n\r\nemail=", strings.TrimPrefix(server.URL, "http://"), server.URL)
		if err != nil {
			t.Fatal(err)
		}

		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}

		response.Body.Close()

		if response.StatusCode != 400 || time.Since(start) < 5*time.Second || time.Since(start) > 7*time.Second || observed.signins.Load() != 1 {
			t.Fatal("body deadline did not bound unparsed ingress")
		}
	})

	t.Run("shared peer admission", func(t *testing.T) {
		cfg := config.New()
		cfg.AuthenticationIngress.PeerRequests = 1
		cfg.AuthenticationIngress.MaxPeers = 1

		var calls atomic.Int32

		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(204) })
		for _, trusted := range []bool{false, true} {
			s, g := ingressServer(t, cfg, next, trusted)
			first := ingressHTTP(t, client, "POST", s.URL+"/signin", "application/x-www-form-urlencoded", "", map[string]string{"X-Forwarded-For": "2001:db8::1"})
			second := ingressHTTP(t, client, "POST", s.URL+"/authenticators/enrollment/begin", "application/json", "{}", map[string]string{"X-Forwarded-For": "2001:0db8:0:0::1"})

			overflow := ingressHTTP(t, client, "POST", s.URL+"/account/password/reset", "application/x-www-form-urlencoded", "", map[string]string{"X-Forwarded-For": "2001:db8::2"})
			if first.status != 204 || second.status != 429 || overflow.status != 429 {
				t.Fatal("route, spoofed header or peer capacity bypassed shared admission")
			}

			if g.Cleanup() != 0 {
				t.Fatal("live peer retired")
			}
		}

		if calls.Load() != 2 {
			t.Fatal("denied peer reached handler")
		}
	})

	t.Run("authentication log isolation", func(t *testing.T) {
		var output bytes.Buffer

		logger := chimiddleware.RequestLogger(&chimiddleware.DefaultLogFormatter{Logger: stdlog.New(&output, "", 0), NoColor: true})
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })

		isolated := httptest.NewServer(authenticationLog(logger, next))
		defer isolated.Close()

		for _, method := range []string{"GET", "POST"} {
			for _, path := range []string{"/signin", "/signup", "/authenticators/enrollment/begin", "/account/password/reset/confirm"} {
				result := ingressHTTP(t, client, method, isolated.URL+path+"?password=private-marker", "text/plain", "", nil)
				if result.status != 204 {
					t.Fatal("log isolation changed route response")
				}
			}
		}

		isolated.Close()

		if output.Len() != 0 {
			t.Fatal("authentication URI reached request logger")
		}

		regular := httptest.NewServer(authenticationLog(logger, next))
		result := ingressHTTP(t, client, "GET", regular.URL+"/ordinary", "", "", nil)
		regular.Close()

		if result.status != 204 || !strings.Contains(output.String(), "/ordinary") {
			t.Fatal("unrelated request logging changed")
		}
	})

	t.Run("shutdown request context", func(t *testing.T) {
		appCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		g, err := NewAuthenticationIngress(cfg)
		if err != nil {
			t.Fatal(err)
		}

		s := httptest.NewUnstartedServer(g.Middleware(router))
		s.Config.BaseContext = func(net.Listener) context.Context { return appCtx }
		s.Config.ReadTimeout = g.settings.WorkTimeout
		s.Config.WriteTimeout = g.settings.Acknowledgment + time.Second

		s.Start()
		defer s.Close()
		defer g.Close()

		before := observed.finishedSignins.Load()

		req, err := http.NewRequestWithContext(t.Context(), "POST", s.URL+"/signin", strings.NewReader(form))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", s.URL)

		type completion struct {
			response *http.Response
			err      error
		}

		done := make(chan completion, 1)

		go func() { response, err := client.Do(req); done <- completion{response, err} }()

		deadline := time.Now().Add(3 * time.Second)
		for observed.finishedSignins.Load() == before && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}

		if observed.finishedSignins.Load() != before+1 {
			t.Fatal("shutdown fixture did not reach acknowledgment")
		}

		g.Close()

		rejected := ingressHTTP(t, client, "POST", s.URL+"/signin", "application/x-www-form-urlencoded", form, nil)
		if rejected.status != 503 {
			t.Fatal("close admitted work before infrastructure cancellation")
		}

		start := time.Now()

		cancel()

		result := <-done
		if result.err != nil {
			t.Fatal(result.err)
		}
		defer result.response.Body.Close()

		if result.response.StatusCode != 503 || len(result.response.Cookies()) != 0 || time.Since(start) > time.Second {
			t.Fatal("server context did not cancel acknowledgment safely")
		}

		g.mu.Lock()
		active := g.active
		g.mu.Unlock()

		if active != 0 {
			t.Fatal("shutdown retained active request")
		}
	})

	t.Run("explicit close", func(t *testing.T) {
		guard.Close()

		before := observed.signins.Load()

		result := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", form, nil)
		if result.status != 503 || observed.signins.Load() != before || result.elapsed > time.Second {
			t.Fatal("closed ingress admitted work")
		}

		result = ingressHTTP(t, client, "GET", server.URL+"/signin", "", "", nil)
		if result.status == 503 {
			t.Fatal("close blocked unrelated safe request")
		}
	})
}

type observedIngressAuth struct {
	authService
	signups, signins, conflicts, finishedSignins atomic.Int32
}

func (s *observedIngressAuth) Signup(ctx context.Context, email, password string) (*core.User, error) {
	s.signups.Add(1)

	user, err := s.authService.Signup(ctx, email, password)
	if errors.Is(err, core.ErrEmailTaken) {
		s.conflicts.Add(1)
	}

	return user, err
}
func (s *observedIngressAuth) Signin(ctx context.Context, email, password string) (*core.AuthenticationResult, error) {
	s.signins.Add(1)

	result, err := s.authService.Signin(ctx, email, password)
	s.finishedSignins.Add(1)

	return result, err
}

type ingressLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *ingressLog) append(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lines = append(l.lines, text)
}
func (l *ingressLog) Debug(v ...any)            { l.append(fmt.Sprint(v...)) }
func (l *ingressLog) Info(v ...any)             { l.append(fmt.Sprint(v...)) }
func (l *ingressLog) Error(v ...any)            { l.append(fmt.Sprint(v...)) }
func (l *ingressLog) Debugf(f string, v ...any) { l.append(fmt.Sprintf(f, v...)) }
func (l *ingressLog) Infof(f string, v ...any)  { l.append(fmt.Sprintf(f, v...)) }
func (l *ingressLog) Errorf(f string, v ...any) { l.append(fmt.Sprintf(f, v...)) }
func (l *ingressLog) With(...any) log.Logger    { return l }

// Actual adapters and HTTP responses prove neutral signup and password failures.
// Six-second targets are measured with the production guard, never a fake clock.
func TestPublicAuthenticationTransactions(t *testing.T) {
	db, queries, _ := mailboxHTTPDatabase(t)
	cfg := config.New()
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1
	cfg.CredentialAdmission.PasswordAttempts = 1
	cfg.AuthenticationIngress.PeerRequests = 1000
	capture := &ingressLog{}

	base, err := core.NewService(queries, cfg, authfeat.NewPasswordChecker(), webCredentialAdmission(t, queries, cfg.CredentialAdmission), capture)
	if err != nil {
		t.Fatal(err)
	}

	password := "a distinct safe password"
	for _, email := range []string{"wrong@example.com", "inactive@example.com", "throttled@example.com", "enrollment@example.com", "fallback@example.com", "success@example.com"} {
		user, err := base.Signup(t.Context(), email, password)
		if err != nil {
			t.Fatal(err)
		}

		if strings.HasPrefix(email, "inactive") {
			err = queries.UpdateUserActive(t.Context(), user.ID, false, time.Now())
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	_, err = base.Signin(t.Context(), "throttled@example.com", "incorrect password", authfeat.PasswordRequirement())
	if err == nil {
		t.Fatal("wrong password unexpectedly completed")
	}

	observed := &observedIngressAuth{authService: authfeat.NewService(base, queries, capture)}
	templates := baseweb.NewTemplateManager(testAssetsFS, capture, baseweb.WithFuncMap(ui.FuncMap()))

	err = templates.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	passwordHandler := &Handler{authSvc: observed, log: capture, tmpl: templates}
	router := chi.NewRouter()
	router.Post("/signin", passwordHandler.handleSignin)
	router.Post("/signup", passwordHandler.handleSignup)

	enrollmentSvc, err := core.NewWebAuthnService(base, queries, config.AuthenticatorConfig{RPID: "localhost", RPName: "Test", Origins: []string{"http://localhost"}, LocalhostDevelopment: true})
	if err != nil {
		t.Fatal(err)
	}

	strong := core.AccessRequirement{Proof: core.RequirePhishingResistantMFA, Revision: "ingress-v1", MaxAge: 5 * time.Minute}

	enrollment, err := NewEnrollmentHandler(enrollmentSvc, strong)
	if err != nil {
		t.Fatal(err)
	}

	enrollment.RegisterRoutes(router)

	fallbackSvc, err := core.NewFallbackService(base, queries, config.FallbackConfig{Issuer: "Test"}, core.SeedKeys{Active: "fixture", Keys: map[string][]byte{"fixture": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}

	lower := strong
	lower.Proof = core.RequireMFA

	fallback, err := NewFallbackHandler(fallbackSvc, base, lower, strong)
	if err != nil {
		t.Fatal(err)
	}

	fallback.RegisterRoutes(router)
	server, _ := ingressServer(t, cfg, router, false)
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	t.Run("registration conflict", func(t *testing.T) {
		form := url.Values{"email": {"register@example.com"}, "password": {password}, "confirm_password": {password}}.Encode()
		results := make([]ingressHTTPResult, 2)

		t.Run("concurrent requests", func(t *testing.T) {
			for index := range 2 {
				t.Run(fmt.Sprintf("request %d", index+1), func(t *testing.T) {
					t.Parallel()
					results[index] = ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", form, nil)
				})
			}
		})

		for _, result := range results {
			if result.status != 303 || result.header.Get("HX-Redirect") != "/signin" || result.header.Get("Location") != "/signin" || result.body != "Continue to sign in\n" || result.header.Get("Set-Cookie") != "" || result.elapsed < 6*time.Second || result.elapsed > 8*time.Second {
				t.Fatal("registration account state disclosed or authority granted")
			}
		}

		var users, sessions int

		err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM users WHERE email=$1", "register@example.com").Scan(&users)
		if err != nil {
			t.Fatal(err)
		}

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil {
			t.Fatal(err)
		}

		if users != 1 || sessions != 0 || observed.conflicts.Load() != 1 || observed.signins.Load() != 0 {
			t.Fatal("registration was not one classified winner without automatic login")
		}
	})

	t.Run("neutral password failures", func(t *testing.T) {
		for _, email := range []string{"missing@example.com", "wrong@example.com", "inactive@example.com", "throttled@example.com"} {
			t.Run(strings.Split(email, "@")[0], func(t *testing.T) {
				t.Parallel()

				form := url.Values{"email": {email}, "password": {"incorrect password"}}.Encode()

				result := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", form, nil)
				if result.status != 403 || result.body != "Authentication unavailable\n" || result.header.Get("Set-Cookie") != "" || result.header.Get("HX-Redirect") != "" || result.header.Get("Retry-After") != "" || result.elapsed < 6*time.Second || result.elapsed > 8*time.Second {
					t.Fatal("public password state or timing disclosed")
				}
			})
		}
	})

	t.Run("successful sign-in", func(t *testing.T) {
		form := url.Values{"email": {"success@example.com"}, "password": {password}}.Encode()
		result := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", form, nil)
		response := http.Response{Header: result.header}

		cookies := response.Cookies()
		if result.status != 200 || result.header.Get("HX-Redirect") != "/list-items" || len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
			t.Fatal("successful proof lost the safe session cookie")
		}

		validated, err := base.ValidateSession(t.Context(), cookies[0].Value, authfeat.PasswordRequirement(), core.NoActivity)
		if err != nil || validated.User.Email != "success@example.com" {
			t.Fatal("HTTP cookie did not name a committed actual session")
		}
	})

	t.Run("candidate-only policy", func(t *testing.T) {
		var body string

		for _, email := range []string{"wrong@example.com", "weak-new@example.com"} {
			form := url.Values{"email": {email}, "password": {"short"}, "confirm_password": {"short"}}.Encode()

			result := ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", form, nil)
			if result.status != 400 || result.header.Get("Set-Cookie") != "" || !strings.Contains(result.body, "Password is too short") {
				t.Fatal("candidate-only policy feedback changed")
			}

			if body != "" && body != result.body {
				t.Fatal("policy feedback disclosed account state")
			}

			body = result.body
		}
	})

	t.Run("operating password failure", func(t *testing.T) {
		_, err := db.ExecContext(t.Context(), "ALTER TABLE users RENAME TO unavailable_users")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _, _ = db.ExecContext(context.Background(), "ALTER TABLE unavailable_users RENAME TO users") }()

		form := url.Values{"email": {"lookup-fault@example.com"}, "password": {password}}.Encode()

		result := ingressHTTP(t, client, "POST", server.URL+"/signin", "application/x-www-form-urlencoded", form, nil)
		if result.status != 403 || result.body != "Authentication unavailable\n" || result.header.Get("Set-Cookie") != "" || result.elapsed < 6*time.Second || result.elapsed > 8*time.Second {
			t.Fatal("operating lookup failure changed public response")
		}
	})

	t.Run("public enrollment and fallback", func(t *testing.T) {
		for _, path := range []string{"/authenticators/enrollment/begin", "/authenticators/totp/setup/begin", "/authenticators/totp/authentication/begin", "/authenticators/backup/authentication/begin"} {
			t.Run(path, func(t *testing.T) {
				t.Parallel()

				result := ingressHTTP(t, client, "POST", server.URL+path, "application/json", `{"email":"absent@example.com","password":"a distinct safe password"}`, nil)
				if result.status != 403 || result.body != "Authentication unavailable\n" || result.header.Get("Set-Cookie") != "" || result.elapsed < 6*time.Second || result.elapsed > 8*time.Second {
					t.Fatal("factor password failure changed public response")
				}
			})
		}

		for _, tc := range []struct{ path, email, token string }{
			{"/authenticators/enrollment/begin", "enrollment@example.com", "enroll1."},
			{"/authenticators/totp/setup/begin", "fallback@example.com", "totpset1."},
		} {
			result := ingressHTTP(t, client, "POST", server.URL+tc.path, "application/json", fmt.Sprintf(`{"email":%q,"password":%q}`, tc.email, password), nil)
			if result.status != 200 || !strings.Contains(result.body, tc.token) || result.header.Get("Set-Cookie") != "" {
				t.Fatal("successful restricted challenge changed or granted a session")
			}
		}
	})

	t.Run("database failure redaction", func(t *testing.T) {
		_, err := db.ExecContext(t.Context(), `CREATE FUNCTION fail_public_registration() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'private registration database failure'; END $$; CREATE TRIGGER fail_public_registration BEFORE INSERT ON users FOR EACH ROW EXECUTE FUNCTION fail_public_registration()`)
		if err != nil {
			t.Fatal(err)
		}

		form := url.Values{"email": {"fault@example.com"}, "password": {password}, "confirm_password": {password}}.Encode()

		result := ingressHTTP(t, client, "POST", server.URL+"/signup", "application/x-www-form-urlencoded", form, nil)
		if result.status != 503 || result.body != "Registration unavailable\n" || result.header.Get("Set-Cookie") != "" || result.elapsed < 6*time.Second {
			t.Fatal("backend failure disclosed details or granted authority")
		}

		capture.mu.Lock()
		defer capture.mu.Unlock()

		for _, line := range capture.lines {
			if strings.Contains(line, "@example.com") || strings.Contains(line, password) || strings.Contains(line, "private registration") {
				t.Fatal("credential or database detail reached log")
			}
		}
	})
}
