//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	cryptoutil "hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/model"
	"net/url"
	"strings"
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

	code, err := previousTOTPCode(t, secret)
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
	t.Run("concurrent winners", testFallbackRaces)
	t.Run("transaction rollback", testFallbackRollback)
	t.Run("durable budget", testFallbackBudget)
	t.Run("set regeneration", testFallbackRegeneration)
	t.Run("state conflicts", testFallbackConflicts)
	t.Run("salted verifiers", testFallbackVerifiers)
	t.Run("post-lock proof time", testFallbackLockTime)
	t.Run("post-write proof time", testFallbackWriteTime)
	t.Run("pending capacity and policy", testFallbackCapacity)
	t.Run("current proof binding", testFallbackProofBinding)
	t.Run("subject ownership", testFallbackOwnership)
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

type fallbackBoundary struct {
	*Queries
	before func(core.FallbackReservation, core.FallbackCompletion)
	deny   bool
	issue  func(core.BackupIssueReservation, core.BackupSet, core.SessionRecord)
}

func (q fallbackBoundary) CompleteFallback(ctx context.Context, r core.FallbackReservation, c core.FallbackCompletion, required core.AccessRequirement, settings config.FallbackSettings, limit int) (*core.Session, error) {
	if q.before != nil {
		q.before(r, c)
	}

	if q.deny {
		return nil, core.ErrFallback
	}

	return q.Queries.CompleteFallback(ctx, r, c, required, settings, limit)
}
func (q fallbackBoundary) ReplaceBackupSet(ctx context.Context, r core.BackupIssueReservation, set core.BackupSet, s core.SessionRecord, required core.AccessRequirement, settings config.FallbackSettings) (*core.Session, error) {
	if q.issue != nil {
		q.issue(r, set, s)
	}

	return q.Queries.ReplaceBackupSet(ctx, r, set, s, required, settings)
}
func fallbackBarrier(t *testing.T, ready <-chan struct{}) {
	t.Helper()

	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("actual verifier did not reach commit boundary")
	}
}
func backupFixture(t *testing.T) (fallbackFixture, []string, *core.IssuedSession) {
	t.Helper()
	f := newFallbackFixture(t, config.FallbackConfig{BackupCodes: 2})

	codes, actor, err := f.svc.IssueBackupCodes(t.Context(), finishTOTP(t, f).Token, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return f, codes, actor
}

// Both real verifiers reach the final transaction with the same replay snapshot.
func testFallbackRaces(t *testing.T) {
	for _, method := range []core.FallbackMethod{core.FallbackTOTP, core.FallbackBackup} {
		name := "one TOTP step winner"
		if method == core.FallbackBackup {
			name = "one backup code winner"
		}

		t.Run(name, func(t *testing.T) {
			f := newFallbackFixture(t, config.FallbackConfig{BackupCodes: 2})

			code := fallbackCode(t, f)
			if method == core.FallbackBackup {
				codes, _, err := f.svc.IssueBackupCodes(t.Context(), finishTOTP(t, f).Token, fallbackRequirement())
				if err != nil {
					t.Fatal(err)
				}

				code = codes[0]
			}

			challenges := []*core.FallbackChallenge{beginFallback(t, f, method), beginFallback(t, f, method)}
			ready := make(chan struct{}, 2)
			release := make(chan struct{})
			svc := fallbackService(t, f.base, fallbackBoundary{Queries: f.q, before: func(_ core.FallbackReservation, _ core.FallbackCompletion) { ready <- struct{}{}; <-release }}, config.FallbackConfig{BackupCodes: 2})
			results := make(chan bool, 2)

			for _, p := range challenges {
				go func() {
					issued, err := svc.FinishFallback(t.Context(), p.Token, code, fallbackRequirement())
					results <- err == nil && issued != nil
				}()
			}

			for range 2 {
				fallbackBarrier(t, ready)
			}

			close(release)

			winners := 0

			for range 2 {
				if <-results {
					winners++
				}
			}

			if winners != 1 {
				t.Fatalf("replay winners %d", winners)
			}
		})
	}

	t.Run("one actor rotation", func(t *testing.T) {
		f, codes, actor := backupFixture(t)

		p := make([]*core.FallbackChallenge, 2)
		for i := range p {
			var err error

			p[i], err = f.svc.BeginFallbackStepUp(t.Context(), actor.Token, "a distinct safe password", core.FallbackBackup, fallbackRequirement())
			if err != nil {
				t.Fatal(err)
			}
		}

		ready := make(chan struct{}, 2)
		release := make(chan struct{})
		svc := fallbackService(t, f.base, fallbackBoundary{Queries: f.q, before: func(_ core.FallbackReservation, _ core.FallbackCompletion) { ready <- struct{}{}; <-release }}, config.FallbackConfig{BackupCodes: 2})
		results := make(chan bool, 2)

		for i := range p {
			go func() {
				issued, err := svc.FinishFallback(t.Context(), p[i].Token, codes[i], fallbackRequirement())
				results <- err == nil && issued != nil
			}()
		}

		for range 2 {
			fallbackBarrier(t, ready)
		}

		close(release)

		winners := 0

		for range 2 {
			if <-results {
				winners++
			}
		}

		if winners != 1 {
			t.Fatalf("rotation winners %d", winners)
		}

		var remaining int

		err := f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&remaining)
		if err != nil || remaining != 1 {
			t.Fatalf("losing rotation consumed a code: %d %v", remaining, err)
		}
	})
	t.Run("one set regeneration", func(t *testing.T) {
		f, _, actor := backupFixture(t)
		ready := make(chan struct{}, 2)
		release := make(chan struct{})
		svc := fallbackService(t, f.base, fallbackBoundary{Queries: f.q, issue: func(_ core.BackupIssueReservation, _ core.BackupSet, _ core.SessionRecord) {
			ready <- struct{}{}

			<-release
		}}, config.FallbackConfig{BackupCodes: 2})
		results := make(chan bool, 2)

		for range 2 {
			go func() {
				codes, issued, err := svc.IssueBackupCodes(t.Context(), actor.Token, fallbackRequirement())
				results <- err == nil && issued != nil && len(codes) == 2
			}()
		}

		for range 2 {
			fallbackBarrier(t, ready)
		}

		close(release)

		winners := 0

		for range 2 {
			if <-results {
				winners++
			}
		}

		if winners != 1 {
			t.Fatalf("set winners %d", winners)
		}
	})
}

// A database failure after one-use writes rolls them back while admission stays spent.
func testFallbackRollback(t *testing.T) {
	for _, method := range []core.FallbackMethod{core.FallbackTOTP, core.FallbackBackup} {
		name := "TOTP rollback"
		if method == core.FallbackBackup {
			name = "backup rollback"
		}

		t.Run(name, func(t *testing.T) {
			f := newFallbackFixture(t, config.FallbackConfig{BackupCodes: 2})

			code := fallbackCode(t, f)
			if method == core.FallbackBackup {
				codes, _, err := f.svc.IssueBackupCodes(t.Context(), finishTOTP(t, f).Token, fallbackRequirement())
				if err != nil {
					t.Fatal(err)
				}

				code = codes[0]
			}

			var accepted, revision int64

			err := f.db.QueryRowContext(t.Context(), "SELECT accepted_step,replay_revision FROM totp_authenticators").Scan(&accepted, &revision)
			if err != nil {
				t.Fatal(err)
			}

			p := beginFallback(t, f, method)
			sessionSQL(t, f.db, `CREATE FUNCTION reject_fallback() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced write failure'; END $$`)
			sessionSQL(t, f.db, `CREATE TRIGGER reject_fallback BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION reject_fallback()`)

			issued, err := f.svc.FinishFallback(t.Context(), p.Token, code, fallbackRequirement())
			if err == nil || issued != nil {
				t.Fatal("failed completion issued a bearer")
			}

			digest, _, err := core.ParseFallbackToken(p.Token)
			if err != nil {
				t.Fatal(err)
			}

			var attempts, count int

			err = f.db.QueryRowContext(t.Context(), "SELECT attempts FROM auth_pending WHERE digest=$1", digest[:]).Scan(&attempts)
			if err != nil || attempts != 1 {
				t.Fatalf("admission refunded %d %v", attempts, err)
			}

			if method == core.FallbackTOTP {
				var step, replay int64

				err = f.db.QueryRowContext(t.Context(), "SELECT accepted_step,replay_revision FROM totp_authenticators").Scan(&step, &replay)
				if err != nil || step != accepted || replay != revision {
					t.Fatal("failed completion consumed TOTP")
				}
			} else {
				err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&count)
				if err != nil || count != 2 {
					t.Fatal("failed completion consumed backup")
				}
			}

			sessionSQL(t, f.db, "DROP TRIGGER reject_fallback ON sessions")

			issued, err = f.svc.FinishFallback(t.Context(), p.Token, code, fallbackRequirement())
			if err != nil || issued == nil {
				t.Fatalf("rolled-back code unavailable: %v", err)
			}
		})
	}

	t.Run("set rollback", func(t *testing.T) {
		f, codes, actor := backupFixture(t)
		sessionSQL(t, f.db, `CREATE FUNCTION reject_set() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced set failure'; END $$`)
		sessionSQL(t, f.db, `CREATE TRIGGER reject_set BEFORE INSERT ON backup_codes FOR EACH ROW EXECUTE FUNCTION reject_set()`)

		leaked, newActor, err := f.svc.IssueBackupCodes(t.Context(), actor.Token, fallbackRequirement())
		if err == nil || leaked != nil || newActor != nil {
			t.Fatal("failed set replacement exposed material")
		}

		_, err = f.base.ValidateSession(t.Context(), actor.Token, fallbackRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal("failed set change revoked actor")
		}

		p := beginFallback(t, f, core.FallbackBackup)

		issued, err := f.svc.FinishFallback(t.Context(), p.Token, codes[0], fallbackRequirement())
		if err != nil || issued == nil {
			t.Fatalf("old set lost after rollback: %v", err)
		}
	})
}

// Shared durable budgets survive pending reissue and count canonical missing-code failures.
func testFallbackBudget(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{Limits: config.AuthenticatorConfig{SubjectAttempts: 2}})
	// Setup spent one attempt. One admitted failure exhausts the same subject budget.
	p := beginFallback(t, f, core.FallbackTOTP)

	issued, err := f.svc.FinishFallback(t.Context(), p.Token, "abcdef", fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("incorrect code accepted")
	}

	p = beginFallback(t, f, core.FallbackTOTP)

	issued, err = f.svc.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
	if !errors.Is(err, core.ErrEnrollmentAttempts) || issued != nil {
		t.Fatalf("reissue bypassed budget: %v", err)
	}

	var (
		attempts int
		cooldown bool
	)

	err = f.db.QueryRowContext(t.Context(), "SELECT attempts,cooldown_until IS NOT NULL FROM auth_factor_budgets").Scan(&attempts, &cooldown)
	if err != nil || attempts != 2 || !cooldown {
		t.Fatal("exhaustion not durable")
	}
}

// Regeneration invalidates old pending, codes and sessions but preserves current actor proof age.
func testFallbackRegeneration(t *testing.T) {
	f, codes, actor := backupFixture(t)
	old := beginFallback(t, f, core.FallbackBackup)

	fresh, rotated, err := f.svc.IssueBackupCodes(t.Context(), actor.Token, fallbackRequirement())
	if err != nil || len(fresh) != 2 || rotated == nil {
		t.Fatal(err)
	}

	issued, err := f.svc.FinishFallback(t.Context(), old.Token, codes[0], fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("old pending survived set change")
	}

	p := beginFallback(t, f, core.FallbackBackup)

	issued, err = f.svc.FinishFallback(t.Context(), p.Token, codes[0], fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("old code survived set change")
	}

	p = beginFallback(t, f, core.FallbackBackup)

	backupActor, err := f.svc.FinishFallback(t.Context(), p.Token, fresh[0], fallbackRequirement())
	if err != nil || backupActor == nil {
		t.Fatal(err)
	}

	leaked, newActor, err := f.svc.IssueBackupCodes(t.Context(), backupActor.Token, fallbackRequirement())
	if err == nil || leaked != nil || newActor != nil {
		t.Fatal("backup actor regenerated its constituent")
	}
	// Fresh password reauthentication yields password proof only, never retained MFA.
	result, err := f.base.Reauthenticate(t.Context(), rotated.Token, "a distinct safe password", PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	pwd, ok := result.CompletedSession()
	if !ok || pwd.Proof.Method != core.PasswordProof || !pwd.Proof.FactorAt.IsZero() {
		t.Fatal("password refreshed factor proof")
	}

	leaked, newActor, err = f.svc.IssueBackupCodes(t.Context(), pwd.Token, fallbackRequirement())
	if err == nil || leaked != nil || newActor != nil {
		t.Fatal("password-only actor issued codes")
	}
}

// Current state is mutated after real verification, before the production commit.
func testFallbackConflicts(t *testing.T) {
	tests := []struct{ name, query string }{
		{"security revision", "UPDATE totp_authenticators SET revision=revision+1"},
		{"replay revision", "UPDATE totp_authenticators SET replay_revision=replay_revision+1"},
		{"step already accepted", "UPDATE totp_authenticators SET accepted_step=accepted_step+2"},
		{"factor removed", "DELETE FROM totp_authenticators"},
		{"account version", "UPDATE users SET auth_version=auth_version+1"},
		{"inactive subject", "UPDATE users SET active=false"},
		{"expired lease", "UPDATE auth_pending SET lease_until=clock_timestamp()"},
		{"policy revision", "UPDATE auth_pending SET policy_revision='changed'"},
		{"tampered seed", "UPDATE totp_authenticators SET envelope=set_byte(envelope,60,get_byte(envelope,60)#1)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFallbackFixture(t, config.FallbackConfig{})
			p := beginFallback(t, f, core.FallbackTOTP)
			svc := fallbackService(t, f.base, fallbackBoundary{Queries: f.q, before: func(_ core.FallbackReservation, _ core.FallbackCompletion) { sessionSQL(t, f.db, test.query) }}, config.FallbackConfig{})

			issued, err := svc.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
			if err == nil || issued != nil {
				t.Fatal("stale actual proof issued access")
			}

			var sessions int

			err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
			if err != nil || sessions != 0 {
				t.Fatal("denied transaction inserted access")
			}
		})
	}

	t.Run("accepted setup step", func(t *testing.T) {
		f := newFallbackFixture(t, config.FallbackConfig{})

		var step int64

		err := f.db.QueryRowContext(t.Context(), "SELECT accepted_step FROM totp_authenticators").Scan(&step)
		if err != nil {
			t.Fatal(err)
		}

		code, err := totp.GenerateCode(f.secret, time.Unix(step*30, 0))
		if err != nil {
			t.Fatal(err)
		}

		issued, err := f.svc.FinishFallback(t.Context(), beginFallback(t, f, core.FallbackTOTP).Token, code, fallbackRequirement())
		if err == nil || issued != nil {
			t.Fatal("setup step reused for access")
		}

		err = f.svc.ConfirmTOTPSetup(t.Context(), f.setup.Token, code, fallbackRequirement())
		if err == nil {
			t.Fatal("setup pending replayed")
		}
	})
	t.Run("phishing resistant policy", func(t *testing.T) {
		f := newFallbackFixture(t, config.FallbackConfig{})
		for _, method := range []core.FallbackMethod{core.FallbackTOTP, core.FallbackBackup} {
			// Wrong password must not reach its KDF when the trusted profile is unsupported.
			p, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "wrong password", method, webAuthnRequirement())
			if !errors.Is(err, core.ErrSessionProof) || p != nil {
				t.Fatal("unsupported profile reached password work")
			}
		}

		p := beginFallback(t, f, core.FallbackTOTP)

		issued, err := f.svc.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), webAuthnRequirement())
		if err == nil || issued != nil {
			t.Fatal("finish substituted a stronger policy")
		}
	})
	t.Run("key unavailable", func(t *testing.T) {
		f := newFallbackFixture(t, config.FallbackConfig{})

		svc, err := core.NewFallbackService(f.base, f.q, config.FallbackConfig{Issuer: "Ticked test"}, core.SeedKeys{Active: "other", Keys: map[string][]byte{"other": make([]byte, 32)}})
		if err != nil {
			t.Fatal(err)
		}

		p := beginFallback(t, f, core.FallbackTOTP)

		issued, err := svc.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
		if err == nil || issued != nil {
			t.Fatal("missing seed identity accepted")
		}
	})
}

// Real Argon2id records have independent salts and cannot cross purpose/owner/ID.
func testFallbackVerifiers(t *testing.T) {
	f, codes, actor := backupFixture(t)

	rows, err := f.db.QueryContext(t.Context(), "SELECT verifier FROM backup_codes ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}

	var records []string

	for rows.Next() {
		var record string

		err = rows.Scan(&record)
		if err != nil {
			t.Fatal(err)
		}

		records = append(records, record)
	}

	err = rows.Err()
	rows.Close()

	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 2 || strings.Split(records[0], "$")[4] == strings.Split(records[1], "$")[4] {
		t.Fatal("backup salts shared")
	}

	parsed, err := cryptoutil.ParseBackupCode(codes[0])
	if err != nil {
		t.Fatal(err)
	}

	var record string

	err = f.db.QueryRowContext(t.Context(), "SELECT verifier FROM backup_codes WHERE id=$1", parsed.ID).Scan(&record)
	if err != nil {
		t.Fatal(err)
	}

	verifier, err := model.NewPasswordVerifier(model.PasswordVerifierConfig{MemoryKiB: 19456, Iterations: 2, Parallelism: 1})
	if err != nil {
		t.Fatal(err)
	}

	input, err := cryptoutil.BackupVerifierInput(actor.UserID, parsed)
	if err != nil {
		t.Fatal(err)
	}

	err = verifier.Verify(t.Context(), record, input)
	if err != nil {
		t.Fatal(err)
	}

	foreign, err := cryptoutil.BackupVerifierInput("another-subject", parsed)
	if err != nil {
		t.Fatal(err)
	}

	if err = verifier.Verify(t.Context(), record, foreign); !errors.Is(err, model.ErrPasswordMismatch) {
		t.Fatal("verifier did not bind owner")
	}

	if err = verifier.Verify(t.Context(), record, parsed.Secret); !errors.Is(err, model.ErrPasswordMismatch) {
		t.Fatal("verifier accepted password-purpose input")
	}

	other, err := cryptoutil.NewBackupSecret()
	if err != nil {
		t.Fatal(err)
	}

	replaced := "backup1." + other.ID + "." + parsed.Secret

	rebound, err := cryptoutil.ParseBackupCode(replaced)
	if err != nil {
		t.Fatal(err)
	}

	foreign, err = cryptoutil.BackupVerifierInput(actor.UserID, rebound)
	if err != nil {
		t.Fatal(err)
	}

	if err = verifier.Verify(t.Context(), record, foreign); !errors.Is(err, model.ErrPasswordMismatch) {
		t.Fatal("verifier did not bind code ID")
	}
	// The PHC parser enforces stored costs before KDF admission; a corrupt record spends only the durable attempt.
	sessionSQL(t, f.db, "UPDATE backup_codes SET verifier=replace(verifier,'m=19456','m=999999') WHERE id=$1", parsed.ID)

	issued, err := f.svc.FinishFallback(t.Context(), beginFallback(t, f, core.FallbackBackup).Token, codes[0], fallbackRequirement())
	if !errors.Is(err, model.ErrPasswordRecord) || issued != nil {
		t.Fatalf("out-of-budget PHC accepted: %v", err)
	}
}

// A fresh clock after an actual row lock wait rejects stale constituent proof.
func testFallbackLockTime(t *testing.T) {
	f, codes, _ := backupFixture(t)
	required := fallbackRequirement()
	required.MaxAge = time.Second

	p, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackBackup, required)
	if err != nil {
		t.Fatal(err)
	}

	var (
		reserved core.FallbackReservation
		command  core.FallbackCompletion
	)

	capture := fallbackService(t, f.base, fallbackBoundary{Queries: f.q, deny: true, before: func(r core.FallbackReservation, c core.FallbackCompletion) { reserved = r; command = c }}, config.FallbackConfig{BackupCodes: 2})

	issued, err := capture.FinishFallback(t.Context(), p.Token, codes[0], required)
	if err == nil || issued != nil || reserved.Pending.Revision < 1 {
		t.Fatal("capture failed")
	}

	sessionSQL(t, f.db, "UPDATE auth_pending SET lease_until=$2 WHERE digest=$1", reserved.Pending.Digest[:], reserved.Pending.LeaseUntil)

	settings, err := (config.FallbackConfig{Issuer: "Ticked test", BackupCodes: 2}).Settings()
	if err != nil {
		t.Fatal(err)
	}

	err = command.Check(reserved, time.Now().UTC(), required, settings)
	if err != nil {
		t.Fatalf("fixture invalid before lock: %v", err)
	}

	holder, err := f.db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback()

	var id string

	err = holder.QueryRowContext(t.Context(), "SELECT id FROM backup_sets FOR UPDATE").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	type result struct {
		session *core.Session
		err     error
	}

	done := make(chan result, 1)

	go func() {
		session, err := f.q.CompleteFallback(ctx, reserved, command, required, settings, 10)
		done <- result{session, err}
	}()

	waitSessionLock(t, ctx, f.db, holder)
	waitProofExpiry(t, ctx, f.db, reserved.Pending.PasswordAt.Add(time.Second))

	err = holder.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	outcome := <-done
	if !errors.Is(outcome.err, core.ErrSessionProofExpired) || outcome.session != nil {
		t.Fatalf("oldest constituent renewed after lock: %v", outcome.err)
	}

	var remaining int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&remaining)
	if err != nil || remaining != 2 {
		t.Fatal("post-lock denial consumed code")
	}
}

// SQL writes can consume enough time to expire authority before the final commit.
func testFallbackWriteTime(t *testing.T) {
	f, codes, _ := backupFixture(t)
	required := fallbackRequirement()
	required.MaxAge = time.Second

	p, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackBackup, required)
	if err != nil {
		t.Fatal(err)
	}

	sessionSQL(t, f.db, `CREATE FUNCTION delay_fallback() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1.1); RETURN NEW; END $$`)
	sessionSQL(t, f.db, `CREATE TRIGGER delay_fallback AFTER INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION delay_fallback()`)

	issued, err := f.svc.FinishFallback(t.Context(), p.Token, codes[0], required)
	if !errors.Is(err, core.ErrSessionProofExpired) || issued != nil {
		t.Fatalf("post-write authority survived expiry: %v", err)
	}

	var remaining int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&remaining)
	if err != nil || remaining != 2 {
		t.Fatal("post-write denial consumed code")
	}
}

// Pending retention is shared across methods, and changed step policy cannot reuse a ceremony.
func testFallbackCapacity(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{Limits: config.AuthenticatorConfig{MaxPending: 1}})
	p := beginFallback(t, f, core.FallbackTOTP)

	_, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackTOTP, fallbackRequirement())
	if !errors.Is(err, core.ErrEnrollmentCapacity) {
		t.Fatalf("live pending displaced: %v", err)
	}

	strict := fallbackService(t, f.base, f.q, config.FallbackConfig{StrictStep: true})

	issued, err := strict.FinishFallback(t.Context(), p.Token, fallbackCode(t, f), fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("changed skew reused captured policy")
	}

	digest, _, err := core.ParseFallbackToken(p.Token)
	if err != nil {
		t.Fatal(err)
	}

	sessionSQL(t, f.db, "UPDATE auth_pending SET password_at=clock_timestamp()-interval '6 minutes',created_at=clock_timestamp()-interval '6 minutes',expires_at=clock_timestamp()-interval '1 minute' WHERE digest=$1", digest[:])

	_, err = f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", "a distinct safe password", core.FallbackTOTP, fallbackRequirement())
	if err != nil {
		t.Fatal("expired pending not reclaimed")
	}

	count, err := f.svc.CleanupPending(t.Context())
	if err != nil || count != 0 {
		t.Fatal("cleanup removed live pending")
	}

	sessionSQL(t, f.db, "UPDATE auth_pending SET password_at=clock_timestamp()-interval '6 minutes',created_at=clock_timestamp()-interval '6 minutes',expires_at=clock_timestamp()-interval '1 minute'")

	count, err = f.svc.CleanupPending(t.Context())
	if err != nil || count != 1 {
		t.Fatal("cleanup missed expired pending")
	}
}

// Repeated normal replay updates do not invalidate previously completed security proof.
func testFallbackProofBinding(t *testing.T) {
	f := newFallbackFixture(t, config.FallbackConfig{})
	first := finishTOTP(t, f)
	second := finishTOTP(t, f)

	_, err := f.base.ValidateSession(t.Context(), first.Token, fallbackRequirement(), core.NoActivity)
	if err != nil {
		t.Fatalf("routine replay update invalidated proof: %v", err)
	}

	sessionSQL(t, f.db, "UPDATE totp_authenticators SET revision=revision+1")

	for _, issued := range []*core.IssuedSession{first, second} {
		_, err = f.base.ValidateSession(t.Context(), issued.Token, fallbackRequirement(), core.NoActivity)
		if !errors.Is(err, core.ErrSessionProof) {
			t.Fatal("changed security binding accepted")
		}
	}
}

// The same database contains both accounts; code lookup is scoped to the authenticated subject.
func testFallbackOwnership(t *testing.T) {
	f, codes, _ := backupFixture(t)

	const email = "second@example.com"

	_, err := f.base.Signup(t.Context(), email, "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	setup, err := f.svc.BeginTOTPSetup(t.Context(), email, "a distinct safe password", fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(setup.URL)
	if err != nil {
		t.Fatal(err)
	}

	secret := parsed.Query().Get("secret")

	previous, err := previousTOTPCode(t, secret)
	if err != nil {
		t.Fatal(err)
	}

	err = f.svc.ConfirmTOTPSetup(t.Context(), setup.Token, previous, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	p, err := f.svc.BeginFallbackAuthentication(t.Context(), email, "a distinct safe password", core.FallbackTOTP, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	current, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	actor, err := f.svc.FinishFallback(t.Context(), p.Token, current, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = f.svc.IssueBackupCodes(t.Context(), actor.Token, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	p, err = f.svc.BeginFallbackAuthentication(t.Context(), email, "a distinct safe password", core.FallbackBackup, fallbackRequirement())
	if err != nil {
		t.Fatal(err)
	}

	issued, err := f.svc.FinishFallback(t.Context(), p.Token, codes[0], fallbackRequirement())
	if err == nil || issued != nil {
		t.Fatal("foreign owned code accepted")
	}

	var count int

	err = f.db.QueryRowContext(t.Context(), "SELECT count(*) FROM backup_codes").Scan(&count)
	if err != nil || count != 4 {
		t.Fatal("foreign attempt consumed a code")
	}
}

// Previous-step fixture setup stays away from turnover so transport latency cannot
// move a deliberately older code outside the one-step window before confirmation.
func previousTOTPCode(t *testing.T, secret string) (string, error) {
	t.Helper()

	now := time.Now()

	remaining := 30*time.Second - time.Duration(now.UnixNano()%int64(30*time.Second))
	if remaining < time.Second {
		select {
		case <-time.After(remaining + 10*time.Millisecond):
		case <-t.Context().Done():
			return "", t.Context().Err()
		}

		now = time.Now()
	}

	return totp.GenerateCode(secret, now.Add(-30*time.Second))
}
