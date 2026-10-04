//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"database/sql"
	"net/url"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

func fallbackRequirement() core.AccessRequirement {
	r := PasswordRequirement()
	r.Proof = core.RequireMFA
	r.MaxAge = 5 * time.Minute

	return r
}
func fallbackService(t *testing.T, base *core.Service, q core.FallbackQueries, cfg config.FallbackConfig) *core.FallbackService {
	t.Helper()

	cfg.Issuer = "Ticked test"

	svc, err := core.NewFallbackService(base, q, cfg, core.SeedKeys{Active: "test-v1", Keys: map[string][]byte{"test-v1": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}

	return svc
}

type fallbackFixture struct {
	db     *sql.DB
	q      *Queries
	base   *core.Service
	svc    *core.FallbackService
	secret string
	setup  *core.TOTPSetup
}

func newFallbackFixture(t *testing.T, cfg config.FallbackConfig) fallbackFixture {
	t.Helper()
	db, q, base, _ := sessionFixture(t)
	svc := fallbackService(t, base, q, cfg)

	setup, err := svc.BeginTOTPSetup(t.Context(), "session@example.com", "a distinct safe password", fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(setup.URL)
	if err != nil {
		t.Fatal(err)
	}

	secret := parsed.Query().Get("secret")

	code, err := totp.GenerateCode(secret, time.Now().Add(-30*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	err = svc.ConfirmTOTPSetup(t.Context(), setup.Token, code, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return fallbackFixture{db: db, q: q, base: base, svc: svc, secret: secret, setup: setup}
}
func fallbackCode(t *testing.T, f fallbackFixture) string {
	t.Helper()

	var accepted int64

	err := f.db.QueryRowContext(t.Context(), "SELECT accepted_step FROM totp_authenticators").Scan(&accepted)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix() / 30
	if now <= accepted {
		now = accepted + 1
	}

	code, err := totp.GenerateCode(f.secret, time.Unix(now*30, 0))
	if err != nil {
		t.Fatal(err)
	}

	return code
}
func beginFallback(t *testing.T, f fallbackFixture, method core.FallbackMethod) *core.FallbackChallenge {
	t.Helper()

	p, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", method, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return p
}
func finishTOTP(t *testing.T, f fallbackFixture) *core.IssuedSession {
	t.Helper()
	p := beginFallback(t, f, core.FallbackTOTP)

	issued, err := f.svc.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
	if err != nil || issued == nil {
		t.Fatalf("TOTP completion: %v", err)
	}

	return issued
}

// Real OTP verification, Argon2id and PostgreSQL establish executable fallback access.
func TestFallbackTransactions(t *testing.T) {
	t.Run("TOTP lifecycle", testFallbackTOTP)
	t.Run("backup lifecycle", testFallbackBackup)
}
func testFallbackTOTP(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{})

	var sessions, pending int

	err := f.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending)").Scan(&sessions, &pending)
	if err != nil || sessions != 0 || pending != 0 {
		t.Fatalf("setup issued access: %d/%d %v", sessions, pending, err)
	}

	issued := finishTOTP(t, f)
	if issued.Proof.Method != core.PasswordTOTPProof || issued.Proof.VerifiedAt.After(issued.Proof.FactorAt) {
		t.Fatal("incorrect composite proof")
	}

	_, err = f.base.ValidateSession(t.Context(), issued.Token, fallbackRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("TOTP satisfied phishing resistance")
	}

	_, err = f.svc.BeginTOTPSetup(t.Context(), "session@example.com", "a distinct safe password", fallbackRequirement())
	if err == nil {
		t.Fatal("initial setup accepted established factor")
	}
}
func testFallbackBackup(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{BackupCodes: 2})
	actor := finishTOTP(t, f)

	codes, rotated, err := f.svc.IssueBackupCodes(t.Context(), actor.Token, fallbackRequirement())
	if err != nil || len(codes) != 2 || rotated == nil {
		t.Fatalf("issue: %v", err)
	}

	if !rotated.Proof.VerifiedAt.Equal(actor.Proof.VerifiedAt) || rotated.AuthVersion != actor.AuthVersion+1 || rotated.Generation != actor.Generation+1 {
		t.Fatal("management renewed proof or missed version change")
	}

	_, err = f.base.ValidateSession(t.Context(), actor.Token, fallbackRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("old management bearer retained")
	}

	p := beginFallback(t, f, core.FallbackBackup)

	issued, err := f.svc.FinishFallback(t.Context(), p.Token, codes[0], fallbackRequirement())
	if err != nil || issued == nil || issued.Proof.Method != core.PasswordBackupProof {
		t.Fatalf("backup completion: %v", err)
	}

	_, err = f.base.ValidateSession(t.Context(), issued.Token, fallbackRequirement(), core.NoActivity)
	if err != nil {
		t.Fatal(err)
	}

	p = beginFallback(t, f, core.FallbackBackup)

	issued, err = f.svc.FinishFallback(t.Context(), p.Token, codes[0], fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("backup reused")
	}

	var records int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&records)
	if err != nil || records != 1 {
		t.Fatalf("consumption count %d: %v", records, err)
	}
	// The active set stays bound after consumption; only a set change invalidates proof.
	_, err = dal.New(f.db).LockBackupSet(t.Context(), rotated.UserID)
	if err != nil {
		t.Fatal(err)
	}
}
