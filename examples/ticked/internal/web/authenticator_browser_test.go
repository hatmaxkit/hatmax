//go:build browser

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pquerna/otp/totp"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	featureauth "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

type browserDatabase struct{ db *sql.DB }

func (p browserDatabase) GetDB() *sql.DB { return p.db }

func browserEnvironment(t *testing.T, name string, empty bool) string {
	t.Helper()

	value, present := os.LookupEnv(name)
	if !present || !empty && value == "" {
		t.Fatalf("%s is required for browser acceptance", name)
	}

	return value
}

func browserQueries(t *testing.T) (*featureauth.Queries, *config.Config) {
	t.Helper()

	cfg := config.New()
	cfg.Database.Host = browserEnvironment(t, "DB_HOST", false)
	cfg.Database.User = browserEnvironment(t, "DB_USER", false)
	cfg.Database.Database = browserEnvironment(t, "DB_NAME", false)
	cfg.Database.Password = browserEnvironment(t, "DB_PASSWORD", true)

	port, err := strconv.Atoi(browserEnvironment(t, "DB_PORT", false))
	if err != nil {
		t.Fatal(err)
	}

	cfg.Database.Port = port
	cfg.CredentialAdmission.PasswordAttempts = 20
	cfg.CredentialAdmission.RegistrationAttempts = 10
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1

	root, err := sql.Open("pgx", cfg.Database.ConnectionString())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = root.Close() })

	schema := "browser_" + strings.ReplaceAll(model.NewID(), "-", "")

	_, err = root.ExecContext(t.Context(), "CREATE SCHEMA "+schema)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, dropErr := root.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		if dropErr != nil {
			t.Errorf("drop owned browser schema: %v", dropErr)
		}
	})

	cfg.Database.Schema = schema

	db, err := sql.Open("pgx", cfg.Database.ConnectionString())
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = db.Close() })

	for _, name := range []string{"001-users.sql", "004-authenticators.sql", "005-webauthn-completion.sql", "006-fallback-proof.sql", "007-factor-control.sql", "008-account-recovery.sql", "009-credential-admission.sql"} {
		migration, readErr := os.ReadFile("../../assets/migration/postgres/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}

		_, err = db.ExecContext(t.Context(), strings.Split(string(migration), "-- +migrate Down")[0])
		if err != nil {
			t.Fatal(err)
		}
	}

	q := featureauth.NewQueries(browserDatabase{db})

	err = q.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return q, cfg
}

// Actual Chromium navigator operations must pass the production handlers, real
// verifier and PostgreSQL before a strong cookie or factor change is accepted.
func TestAuthenticatorBrowser(t *testing.T) {
	runAuthenticatorBrowser(t, false)
}

func runAuthenticatorBrowser(t *testing.T, recovery bool) {
	t.Helper()
	node := browserEnvironment(t, "NODE_BIN", false)

	chromium := browserEnvironment(t, "CHROMIUM_BIN", false)
	for _, binary := range []string{node, chromium} {
		_, err := exec.LookPath(binary)
		if err != nil {
			t.Fatal(err)
		}
	}

	q, cfg := browserQueries(t)
	logger := log.NewTestLogger("error")

	base, err := auth.NewService(q, cfg, featureauth.NewPasswordChecker(), webCredentialAdmission(t, q, cfg.CredentialAdmission), logger)
	if err != nil {
		t.Fatal(err)
	}

	for _, email := range []string{"passkey@example.com", "totp@example.com", "stepup@example.com"} {
		_, err = base.Signup(t.Context(), email, "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
	}

	cfg.AuthenticationIngress.PeerRequests = 1000
	ingress, err := NewAuthenticationIngress(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ingress.Close)
	router := chi.NewRouter()
	router.Use(ingress.Middleware)

	var recoveryFixture *recoveryBrowserFixture
	if recovery {
		recoveryFixture = newRecoveryBrowserFixture()
		router.Use(recoveryFixture.loseResponse)
	}

	server := httptest.NewUnstartedServer(router)
	t.Cleanup(server.Close)
	origin := "http://" + strings.Replace(server.Listener.Addr().String(), "127.0.0.1", "localhost", 1)

	limits := config.AuthenticatorConfig{RPID: "localhost", RPName: "Ticked browser acceptance", Origins: []string{origin}, LocalhostDevelopment: true}
	if recovery {
		// This composite journey deliberately performs more ceremonies than the
		// default shared ten-attempt window. Keep a finite explicit fixture budget.
		limits.SubjectAttempts = 40
	}

	webAuthn, err := auth.NewWebAuthnService(base, q, limits)
	if err != nil {
		t.Fatal(err)
	}

	fallback, err := auth.NewFallbackService(base, q, config.FallbackConfig{Limits: limits, Issuer: "Ticked browser acceptance"}, auth.SeedKeys{Active: "browser-v1", Keys: map[string][]byte{"browser-v1": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}

	factors, err := auth.NewFactorService(base, q, webAuthn.AuthenticatorService, fallback)
	if err != nil {
		t.Fatal(err)
	}

	strong := featureauth.PasswordRequirement()
	strong.Proof = auth.RequirePhishingResistantMFA
	strong.MaxAge = 5 * time.Minute
	mfa := strong
	mfa.Proof = auth.RequireMFA

	enrollment, err := NewEnrollmentHandler(webAuthn, strong)
	if err != nil {
		t.Fatal(err)
	}

	assertion, err := NewWebAuthnHandler(webAuthn, base, strong)
	if err != nil {
		t.Fatal(err)
	}

	backup, err := NewFallbackHandler(fallback, base, mfa, strong)
	if err != nil {
		t.Fatal(err)
	}

	control, err := NewFactorHandler(factors, auth.FactorPolicy{Management: strong, Access: auth.RequirePhishingResistantMFA}, true)
	if err != nil {
		t.Fatal(err)
	}

	if recovery {
		recoveryFixture.register(t, router, base, q, origin, strong)
	}

	enrollment.RegisterRoutes(router)
	assertion.RegisterRoutes(router)
	backup.RegisterRoutes(router)
	control.RegisterRoutes(router)

	password := &Handler{authSvc: featureauth.NewService(base, q, logger), log: logger}
	router.Post("/signin", password.handleSignin)
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><html lang=en><title>Browser acceptance</title></html>"))
	})
	// Generate a device code using the existing library. This fixture never
	// approves proof or mutates factors; production finish consumes the code.
	router.Post("/__test/otp", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			URL      string `json:"url"`
			Previous bool   `json:"previous"`
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		decodeErr := json.NewDecoder(r.Body).Decode(&input)

		parsed, parseErr := url.Parse(input.URL)
		if decodeErr != nil || parseErr != nil || parsed.Scheme != "otpauth" {
			http.Error(w, "Invalid device input", http.StatusBadRequest)

			return
		}

		at := time.Now()
		if input.Previous {
			at = at.Add(-30 * time.Second)
		}

		code, codeErr := totp.GenerateCode(parsed.Query().Get("secret"), at)
		if codeErr != nil {
			http.Error(w, "Invalid device input", http.StatusBadRequest)

			return
		}

		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Code string `json:"code"`
		}{code})
	})
	server.Start()

	timeout := 150 * time.Second
	if recovery {
		timeout = 240 * time.Second
	}

	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	args := []string{"testdata/authenticators.mjs", chromium, origin, t.TempDir()}
	if recovery {
		args = append(args, "account-recovery")
	}

	command := exec.CommandContext(ctx, node, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("browser acceptance: %v\n%s", err, output)
	}

	t.Logf("%s", output)
}
