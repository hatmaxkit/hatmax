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
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type entryQueries struct {
	*Queries
	lookups atomic.Int32
	after   func(context.Context) error
}

func (q *entryQueries) GetUserByEmail(ctx context.Context, email string) (*core.User, error) {
	q.lookups.Add(1)
	user, err := q.Queries.GetUserByEmail(ctx, email)
	if err == nil && q.after != nil {
		err = q.after(ctx)
	}
	return user, err
}

type entryFixture struct {
	db         *sql.DB
	q          *entryQueries
	cfg        *config.Config
	base       *core.Service
	enrollment *core.WebAuthnService
	fallback   *core.FallbackService
}

func newEntryFixture(t *testing.T, attempts int) entryFixture {
	t.Helper()
	db, q, cfg := credentialDatabase(t)
	cfg.CredentialAdmission.PasswordAttempts = attempts
	cfg.CredentialAdmission.RegistrationAttempts = 2
	observed := &entryQueries{Queries: q}
	base, err := core.NewService(observed, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q, cfg.CredentialAdmission), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}
	enrollment, err := core.NewWebAuthnService(base, observed, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := core.NewFallbackService(base, observed, config.FallbackConfig{Issuer: "Example"}, core.SeedKeys{Active: "fixture", Keys: map[string][]byte{"fixture": make([]byte, 32)}})
	if err != nil {
		t.Fatal(err)
	}
	return entryFixture{db, observed, cfg, base, enrollment, fallback}
}
func entryProofCount(t *testing.T, f entryFixture, want int) {
	t.Helper()
	got := mailboxCount(t, f.db, "SELECT coalesce(sum(attempts),0) FROM credential_admissions WHERE namespace='credential-fixture' AND purpose=2")
	if got != want {
		t.Fatalf("proof charges = %d; want %d", got, want)
	}
}

// This fault loses only the admission result after an actual PostgreSQL commit.
// It cannot manufacture proof, roll back the committed charge or admit a retry.
type lostEntryAdmission struct{ *CredentialAdmissionStore }

func (q lostEntryAdmission) ChargeCredentialAdmission(ctx context.Context, identity string, purpose core.CredentialAdmissionPurpose, cfg config.CredentialAdmissionSettings) error {
	err := q.CredentialAdmissionStore.ChargeCredentialAdmission(ctx, identity, purpose, cfg)
	if err != nil {
		return err
	}
	return context.DeadlineExceeded
}

// Real service entrypoints share durable counters before lookup and crypto;
// accepted work remains spent across successful, failed, stale and lost outcomes.
func TestPasswordEntryBudgets(t *testing.T) {
	t.Run("unknown cross-path race", func(t *testing.T) {
		f := newEntryFixture(t, 4)
		second, err := core.NewService(f.q, f.cfg, NewPasswordChecker(), credentialAdmissionForTest(t, f.q.Queries, f.cfg.CredentialAdmission), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}
		operations := []func() error{
			func() error {
				_, err := f.base.Signin(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
				return err
			},
			func() error {
				_, err := second.Signin(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
				return err
			},
			func() error {
				_, err := f.enrollment.BeginWebAuthnEnrollment(t.Context(), "unknown@example.com", "a distinct safe password", enrollmentRequirement())
				return err
			},
			func() error {
				_, err := f.fallback.BeginTOTPSetup(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
				return err
			},
			func() error {
				_, err := f.fallback.BeginFallbackAuthentication(t.Context(), "unknown@example.com", "a distinct safe password", core.FallbackBackup, PasswordRequirement())
				return err
			},
		}
		start := make(chan struct{})
		failures := make(chan error, 24)
		var denied atomic.Int32
		var wg sync.WaitGroup
		for i := range 24 {
			wg.Go(func() {
				<-start
				err := operations[i%len(operations)]()
				if errors.Is(err, core.ErrCredentialAdmissionAttempts) {
					denied.Add(1)
				} else if !errors.Is(err, core.ErrUserNotFound) && !errors.Is(err, core.ErrFallback) {
					failures <- err
				}
			})
		}
		close(start)
		wg.Wait()
		close(failures)
		for err := range failures {
			t.Errorf("unexpected outcome: %v", err)
		}
		if denied.Load() != 20 || f.q.lookups.Load() != 4 {
			t.Fatalf("denials=%d lookups=%d", denied.Load(), f.q.lookups.Load())
		}
		entryProofCount(t, f, 4)
		if mailboxCount(t, f.db, "SELECT count(*) FROM users") != 0 {
			t.Fatal("unknown identity manufactured a user")
		}
		// Registration remains independent and admitted short-password policy failure spends its charge.
		for range 2 {
			_, err = f.base.Signup(t.Context(), "unknown@example.com", "short")
			if !errors.Is(err, core.ErrPasswordTooShort) {
				t.Fatal(err)
			}
		}
		_, err = f.base.Signup(t.Context(), "unknown@example.com", "a distinct safe password")
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatal(err)
		}
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions WHERE purpose=1") != 2 {
			t.Fatal("registration charge drift")
		}
	})
	t.Run("success and session identity", func(t *testing.T) {
		f := newEntryFixture(t, 5)
		_, err := f.base.Signup(t.Context(), "known@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
		issued, err := testSignin(f.base, t.Context(), "known@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.enrollment.BeginWebAuthnEnrollment(t.Context(), "known@example.com", "a distinct safe password", enrollmentRequirement())
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.fallback.BeginTOTPSetup(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.fallback.BeginFallbackAuthentication(t.Context(), "known@example.com", "a distinct safe password", core.FallbackBackup, PasswordRequirement())
		if err == nil {
			t.Fatal("missing backup proof accepted")
		}
		outcome, err := f.base.Reauthenticate(t.Context(), issued.Token, "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := outcome.CompletedSession(); !ok {
			t.Fatal("reauthentication did not rotate")
		}
		entryProofCount(t, f, 5)
		_, err = f.base.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatal(err)
		}
		entryProofCount(t, f, 5)
		if mailboxCount(t, f.db, "SELECT count(*) FROM credential_admissions WHERE purpose=2") != 1 {
			t.Fatal("session identity split its budget")
		}
	})
	t.Run("inactive and invalid record", func(t *testing.T) {
		for _, state := range []string{"inactive", "invalid record"} {
			t.Run(state, func(t *testing.T) {
				f := newEntryFixture(t, 1)
				_, err := f.base.Signup(t.Context(), "known@example.com", "a distinct safe password")
				if err != nil {
					t.Fatal(err)
				}
				if state == "inactive" {
					admissionExec(t, f.db, "UPDATE users SET active=false")
				} else {
					admissionExec(t, f.db, "UPDATE users SET password_hash='invalid'")
				}
				_, err = f.base.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
				if err == nil {
					t.Fatal("invalid state accepted")
				}
				if state == "invalid record" && !errors.Is(err, model.ErrPasswordRecord) {
					t.Fatal(err)
				}
				admissionExec(t, f.db, "UPDATE users SET active=true,auth_version=auth_version+1")
				_, err = f.base.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
				if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
					t.Fatal(err)
				}
				entryProofCount(t, f, 1)
			})
		}
	})
	t.Run("stale session commit", func(t *testing.T) {
		f := newEntryFixture(t, 1)
		_, err := f.base.Signup(t.Context(), "known@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
		changed := changingQueries{Queries: f.q.Queries, change: func(ctx context.Context, _ core.CredentialState) error {
			_, err := f.db.ExecContext(ctx, "UPDATE users SET auth_version=auth_version+1")
			return err
		}}
		svc, err := core.NewService(changed, f.cfg, NewPasswordChecker(), credentialAdmissionForTest(t, f.q.Queries, f.cfg.CredentialAdmission), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrCredentialChanged) {
			t.Fatal(err)
		}
		_, err = f.base.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatal(err)
		}
		entryProofCount(t, f, 1)
		if mailboxCount(t, f.db, "SELECT count(*) FROM sessions") != 0 {
			t.Fatal("stale password proof issued authority")
		}
	})
	t.Run("canceled after charge", func(t *testing.T) {
		f := newEntryFixture(t, 1)
		_, err := f.base.Signup(t.Context(), "known@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		f.q.after = func(context.Context) error { cancel(); return ctx.Err() }
		_, err = f.base.Signin(ctx, "known@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		f.q.after = nil
		_, err = f.base.Signin(t.Context(), "known@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatal(err)
		}
		entryProofCount(t, f, 1)
	})
	t.Run("admission result loss", func(t *testing.T) {
		f := newEntryFixture(t, 1)
		store, err := NewCredentialAdmissionStore(f.q.Queries, "credential-fixture", make([]byte, 32))
		if err != nil {
			t.Fatal(err)
		}
		admission, err := core.NewCredentialAdmission(lostEntryAdmission{store}, f.cfg.CredentialAdmission)
		if err != nil {
			t.Fatal(err)
		}
		svc, err := core.NewService(f.q, f.cfg, NewPasswordChecker(), admission, log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.Signin(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
		if f.q.lookups.Load() != 0 {
			t.Fatal("ambiguous admission reached lookup")
		}
		entryProofCount(t, f, 1)
		_, err = f.base.Signin(t.Context(), "unknown@example.com", "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatal(err)
		}
		entryProofCount(t, f, 1)
	})
	t.Run("checker failure spent", func(t *testing.T) {
		f := newEntryFixture(t, 2)
		f.cfg.CredentialAdmission.RegistrationAttempts = 1
		var calls atomic.Int32
		checker := changeChecker{func(context.Context, string) (bool, error) {
			calls.Add(1)
			return false, errors.New("fixture checker failure")
		}}
		svc, err := core.NewService(f.q, f.cfg, checker, credentialAdmissionForTest(t, f.q.Queries, f.cfg.CredentialAdmission), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}
		for i := range 2 {
			_, err = svc.Signup(t.Context(), "known@example.com", "a distinct safe password")
			want := core.ErrPasswordCheckFailed
			if i == 1 {
				want = core.ErrCredentialAdmissionAttempts
			}
			if !errors.Is(err, want) {
				t.Fatal(err)
			}
		}
		if calls.Load() != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM users") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions WHERE purpose=1") != 1 {
			t.Fatal("denied work retried checker or refunded charge")
		}
	})
	t.Run("raw structure rejects storage", func(t *testing.T) {
		f := newEntryFixture(t, 1)
		for _, tc := range []struct {
			name, email, password string
			want                  error
		}{
			{"identity control", "a\n@example.com", "safe", core.ErrCredentialIdentity},
			{"identity size", strings.Repeat("a", 255), "safe", core.ErrCredentialIdentity},
			{"password encoding", "known@example.com", string([]byte{255}), model.ErrPasswordInput},
			{"password size", "known@example.com", strings.Repeat("a", 4097), model.ErrPasswordInput},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := f.base.Signin(t.Context(), tc.email, tc.password, PasswordRequirement())
				if !errors.Is(err, tc.want) {
					t.Fatal(err)
				}
			})
		}
		if f.q.lookups.Load() != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM credential_admissions") != 0 {
			t.Fatal("invalid structure reached account/admission storage")
		}
	})
	t.Run("whole operation deadline", func(t *testing.T) {
		f := newEntryFixture(t, 1)
		_, err := f.base.Signup(t.Context(), "known@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}
		hold, err := f.db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer hold.Rollback()
		_, err = hold.ExecContext(t.Context(), "SELECT 1 FROM credential_admission_capacity WHERE namespace='credential-fixture' FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
		defer cancel()
		_, err = f.base.Signin(ctx, "known@example.com", "a distinct safe password", PasswordRequirement())
		if err == nil {
			t.Fatal("locked admission escaped caller deadline")
		}
		if f.q.lookups.Load() != 0 {
			t.Fatal("failed admission reached account lookup")
		}
		entryProofCount(t, f, 0)
	})
}
