//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	authfeat "hatmax.adrianpk.com/examples/ticked/internal/feat/auth"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
	"hatmax.adrianpk.com/model"
)

type mailboxDB struct{ db *sql.DB }

func (d mailboxDB) GetDB() *sql.DB { return d.db }

type mailboxHTTPMail struct {
	messages []*mailer.Message
	failure  bool
}

func (m *mailboxHTTPMail) Send(_ context.Context, message *mailer.Message) error {
	m.messages = append(m.messages, message)
	if m.failure {
		return errors.New("private provider message " + message.Text)
	}
	return nil
}

func mailboxHTTPDatabase(t *testing.T) (*sql.DB, *authfeat.Queries, *core.Service) {
	t.Helper()
	cfg := config.New()
	cfg.Database.Host = os.Getenv("DB_HOST")
	if cfg.Database.Host == "" {
		t.Fatal("DB_HOST is required for mailbox transport integration")
	}
	cfg.Database.User = os.Getenv("DB_USER")
	if cfg.Database.User == "" {
		cfg.Database.User = "postgres"
	}
	cfg.Database.Database = os.Getenv("DB_NAME")
	if cfg.Database.Database == "" {
		cfg.Database.Database = "postgres"
	}
	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	if value := os.Getenv("DB_PORT"); value != "" {
		number, err := strconv.Atoi(value)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Database.Port = number
	}
	cfg.CredentialAdmission.PasswordAttempts = 20
	cfg.CredentialAdmission.RegistrationAttempts = 10
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1
	root, err := sql.Open("pgx", cfg.Database.ConnectionString())
	if err != nil {
		t.Fatal(err)
	}
	schema := "mailbox_" + strings.ReplaceAll(model.NewID(), "-", "")
	_, err = root.ExecContext(t.Context(), "CREATE SCHEMA "+schema)
	if err != nil {
		root.Close()
		t.Fatal(err)
	}
	cfg.Database.Schema = schema
	db, err := sql.Open("pgx", cfg.Database.ConnectionString())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	t.Cleanup(func() {
		db.Close()
		root.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		root.Close()
	})
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
	queries := authfeat.NewQueries(mailboxDB{db})
	err = queries.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	base, err := core.NewService(queries, cfg, authfeat.NewPasswordChecker(), webCredentialAdmission(t, queries, cfg.CredentialAdmission), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}
	return db, queries, base
}

// Production handler, real PostgreSQL and captured mail establish neutral
// initiation including its six-second deadline, GET safety and cookie isolation.
func TestMailboxTransportTransactions(t *testing.T) {
	db, queries, base := mailboxHTTPDatabase(t)
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
		t.Fatal("initial sign-in unavailable")
	}
	svc, err := core.NewRecoveryService(base, queries, config.RecoveryConfig{}, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}
	capture := &mailboxHTTPMail{}
	delivery, err := authfeat.NewMailboxDelivery(svc, queries, capture, log.NewTestLogger("error"), "https://trusted.example.com", false)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewMailboxHandler(delivery)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	request := func(method, path, key, value, origin string) *httptest.ResponseRecorder {
		body := url.Values{}
		if key != "" {
			body.Set(key, value)
		}
		req := httptest.NewRequest(method, "https://trusted.example.com"+path, strings.NewReader(body.Encode())).WithContext(t.Context())
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", origin)
		req.AddCookie(&http.Cookie{Name: core.SessionCookieName, Value: actor.Token})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	var public string
	for _, tc := range []struct {
		name, email string
		failure     bool
	}{{"eligible", user.Email, false}, {"unknown", "missing@example.com", false}, {"provider failure", user.Email, true}} {
		t.Run(tc.name, func(t *testing.T) {
			capture.failure = tc.failure
			start := time.Now()
			w := request("POST", "/account/mailbox", "email", tc.email, "https://trusted.example.com")
			if time.Since(start) < 6*time.Second || w.Code != 202 || len(w.Result().Cookies()) != 0 {
				t.Fatal("neutral acknowledgment changed")
			}
			if public == "" {
				public = w.Body.String()
			} else if public != w.Body.String() {
				t.Fatal("account or provider state disclosed")
			}
		})
	}
	if len(capture.messages) != 2 {
		t.Fatal("unknown account received a message")
	}
	text := capture.messages[1].Text
	if !strings.Contains(text, "https://trusted.example.com/account/mailbox/confirm#token=") {
		t.Fatal("mail link trusted origin changed")
	}
	token := strings.Split(strings.Split(text, "#token=")[1], "\n")[0]
	w := request("GET", "/account/mailbox/confirm", "", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), token) {
		t.Fatal("GET disclosed bearer")
	}
	current, err := queries.GetUserByID(t.Context(), user.ID)
	if err != nil || current.MailboxVerifiedAt != nil {
		t.Fatal("GET mutated ownership")
	}
	w = request("POST", "/account/mailbox/confirm", "token", token, "")
	if w.Code != 403 {
		t.Fatal("unprotected confirmation accepted")
	}
	w = request("POST", "/account/mailbox/confirm", "token", token, "https://trusted.example.com")
	if w.Code != 200 || strings.Contains(w.Body.String(), token) {
		t.Fatalf("confirm: %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != core.SessionCookieName || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatal("confirmation authenticated or failed to clear cookie")
	}
	_, err = base.ValidateSession(t.Context(), actor.Token, authfeat.PasswordRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("old session survived")
	}
	current, err = queries.GetUserByID(t.Context(), user.ID)
	if err != nil || current.MailboxVerifiedAt == nil || current.AuthVersion != user.AuthVersion+1 {
		t.Fatal("verification not committed")
	}
	w = request("POST", "/account/mailbox/confirm", "token", token, "https://trusted.example.com")
	if w.Code != 403 {
		t.Fatal("replay succeeded")
	}
	capture.failure = false
	err = delivery.DispatchMailboxNotices(t.Context(), user.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	var delivered int
	err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM recovery_notices WHERE delivered_at IS NOT NULL").Scan(&delivered)
	if err != nil || delivered != 1 {
		t.Fatal("durable notification retry failed")
	}
}
