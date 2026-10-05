//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

// Hooks synchronize actual completed cryptography without faking persistence.
type resetBoundary struct {
	*Queries
	reset  func(context.Context, core.MailboxRecord)
	change func(context.Context, core.PasswordChangeAuthorization)
}

func (q resetBoundary) CompletePasswordReset(ctx context.Context, p core.MailboxRecord, hash, policy string, settings config.RecoverySettings) (*core.PasswordReset, error) {
	if q.reset != nil {
		q.reset(ctx, p)
	}

	return q.Queries.CompletePasswordReset(ctx, p, hash, policy, settings)
}
func (q resetBoundary) CommitPasswordChange(ctx context.Context, p core.PasswordChangeAuthorization, hash string, policy core.PasswordChangePolicy, settings config.RecoverySettings) (*core.PasswordChanged, error) {
	if q.change != nil {
		q.change(ctx, p)
	}

	return q.Queries.CommitPasswordChange(ctx, p, hash, policy, settings)
}

// Every state mutation is isolated in a real schema; denial cannot replace the
// credential or create access/notification, even when a token was once valid.
func testResetEligibility(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"inactive", "UPDATE users SET active=false"},
		{"unverified", "UPDATE users SET mailbox_verified_at=NULL"},
		{"future verification", "UPDATE users SET mailbox_verified_at=clock_timestamp()+interval '1 day'"},
		{"target", "UPDATE users SET email='different@example.com'"},
		{"version", "UPDATE users SET auth_version=auth_version+1"},
		{"policy", "UPDATE mailbox_tokens SET policy_revision='other' WHERE purpose=2"},
		{"expired", "UPDATE mailbox_tokens SET created_at=created_at-interval '2 hours',expires_at=expires_at-interval '2 hours' WHERE purpose=2"},
		{"consumed", "UPDATE mailbox_tokens SET consumed_at=created_at WHERE purpose=2"},
		{"revoked", "UPDATE mailbox_tokens SET revoked_at=created_at WHERE purpose=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResetFixture(t, config.RecoveryConfig{})

			before, err := f.q.GetUserByID(t.Context(), f.actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			sessionSQL(t, f.db, tc.sql)

			_, err = f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
			if err == nil {
				t.Fatal("stale reset authorized")
			}

			after, err := f.q.GetUserByID(t.Context(), f.actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			if before.PasswordHash != after.PasswordHash || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 0 {
				t.Fatal("denial admitted work or changed credential")
			}
		})
	}

	for _, tc := range []struct{ name, sql string }{
		{"inactive", "UPDATE users SET active=false"},
		{"unverified", "UPDATE users SET mailbox_verified_at=NULL"},
		{"future verification", "UPDATE users SET mailbox_verified_at=clock_timestamp()+interval '1 day'"},
	} {
		t.Run("issue "+tc.name, func(t *testing.T) {
			f := newResetFixture(t, config.RecoveryConfig{})
			sessionSQL(t, f.db, tc.sql)

			_, err := f.service.RequestPasswordReset(t.Context(), "session@example.com")
			if !errors.Is(err, core.ErrRecoveryUnavailable) {
				t.Fatalf("issuance: %v", err)
			}

			if mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE purpose=2") != 1 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=2") != 1 {
				t.Fatal("denial replaced issued slot or spent issuance")
			}
		})
	}
}

func resetCountingService(t *testing.T, f resetFixture, checker core.PasswordChecker) *core.RecoveryService {
	t.Helper()

	cfg := config.New()
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1

	base, err := core.NewService(f.q, cfg, checker, log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	return recoveryForChange(t, base, f.q, config.RecoveryConfig{})
}

// Malformed/unknown/wrong-purpose inputs do no candidate work. A wrong secret
// for a known reset spends durable token/shared attempts before returning denial.
func testResetAdmission(t *testing.T) {
	f := newResetFixture(t, config.RecoveryConfig{})

	var checked atomic.Int32

	s := resetCountingService(t, f, changeChecker{func(context.Context, string) (bool, error) {
		checked.Add(1)
		return false, nil
	}})

	_, err := f.base.Signup(t.Context(), "unverified@example.com", "another safe long password")
	if err != nil {
		t.Fatal(err)
	}

	verify, err := s.RequestMailboxVerification(t.Context(), "unverified@example.com")
	if err != nil {
		t.Fatal(err)
	}

	wrong := f.issue.Token.Bearer()

	replacement := "A"
	if wrong[37] == 'A' {
		replacement = "B"
	}

	wrong = wrong[:37] + replacement + wrong[38:]
	for _, tc := range []struct{ name, token, password string }{
		{"malformed", "bad", changedPassword},
		{"oversized", f.issue.Token.Bearer(), strings.Repeat("x", 4097)},
		{"unknown", model.NewID() + "." + strings.Repeat("A", 43), changedPassword},
		{"verification purpose", verify.Token.Bearer(), changedPassword},
		{"wrong secret", wrong, changedPassword},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.ResetPassword(t.Context(), tc.token, tc.password)
			if err == nil {
				t.Fatal("invalid token authorized")
			}

			if checked.Load() != 0 {
				t.Fatal("unproven secret reached checker/KDF")
			}
		})
	}

	_, err = s.ConfirmMailboxVerification(t.Context(), f.issue.Token.Bearer())
	if err == nil {
		t.Fatal("reset token granted mailbox verification")
	}

	if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 1 || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE id=$1", verify.Token.ID()) != 0 {
		t.Fatal("purpose/wrong-secret accounting failed")
	}

	_, err = s.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
	if err != nil || checked.Load() != 1 {
		t.Fatalf("actual token completion: %v", err)
	}
}

func testResetBudgets(t *testing.T) {
	f := newResetFixture(t, config.RecoveryConfig{CompletionAttempts: 2})

	_, err := f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), "short")
	if !errors.Is(err, core.ErrPasswordTooShort) {
		t.Fatalf("candidate: %v", err)
	}

	replacement, err := f.service.RequestPasswordReset(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.service.ResetPassword(t.Context(), replacement.Token.Bearer(), changedPassword)
	if !errors.Is(err, core.ErrRecoveryAttempts) {
		t.Fatalf("replacement refunded completion: %v", err)
	}

	_, err = f.service.ChangePassword(t.Context(), f.actor.Token, changedPassword, passwordChangePolicy())
	if !errors.Is(err, core.ErrRecoveryAttempts) {
		t.Fatalf("shared change budget: %v", err)
	}

	if mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 2 || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 0 {
		t.Fatal("budget reset/refund")
	}

	t.Run("token exhaustion", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{TokenAttempts: 1})

		_, err := f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), "short")
		if !errors.Is(err, core.ErrPasswordTooShort) {
			t.Fatal(err)
		}

		_, err = f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
		if !errors.Is(err, core.ErrRecoveryAttempts) {
			t.Fatal("token attempt refunded")
		}
	})
	t.Run("issuance exhaustion", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{IssuanceAttempts: 1})

		_, err := f.service.RequestPasswordReset(t.Context(), "session@example.com")
		if !errors.Is(err, core.ErrRecoveryAttempts) {
			t.Fatal("issuance limit ignored")
		}

		_, err = f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
		if err != nil {
			t.Fatal("issuance denial invalidated predecessor")
		}
	})
}

func testResetRaces(t *testing.T) {
	t.Run("double completion", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})
		start := make(chan struct{})
		results := make(chan error, 2)

		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				<-start

				_, e := f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
				results <- e
			})
		}

		close(start)
		wg.Wait()
		close(results)

		winners := 0

		for err := range results {
			if err == nil {
				winners++
			}
		}

		if winners != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion)+1 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 1 {
			t.Fatal("multiple completion winners")
		}
	})
	t.Run("reissue after actual KDF", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})
		boundary := resetBoundary{Queries: f.q, reset: func(ctx context.Context, _ core.MailboxRecord) {
			_, err := f.service.RequestPasswordReset(ctx, "session@example.com")
			if err != nil {
				t.Fatal(err)
			}
		}}
		s := recoveryForChange(t, f.base, boundary, config.RecoveryConfig{})

		_, err := s.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
		if err == nil || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 2 {
			t.Fatal("reissue resurrected old authorization or refunded attempt")
		}
	})
	t.Run("reset versus change", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})
		entered := make(chan struct{}, 2)
		release := make(chan struct{})
		barrier := func(ctx context.Context) {
			entered <- struct{}{}

			select {
			case <-release:
			case <-ctx.Done():
			}
		}
		boundary := resetBoundary{Queries: f.q, reset: func(ctx context.Context, _ core.MailboxRecord) { barrier(ctx) }, change: func(ctx context.Context, _ core.PasswordChangeAuthorization) { barrier(ctx) }}
		s := recoveryForChange(t, f.base, boundary, config.RecoveryConfig{})

		ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
		defer cancel()

		results := make(chan error, 2)

		go func() { _, e := s.ResetPassword(ctx, f.issue.Token.Bearer(), changedPassword); results <- e }()
		go func() {
			_, e := s.ChangePassword(ctx, f.actor.Token, changedPassword, passwordChangePolicy())
			results <- e
		}()

		for range 2 {
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}

		close(release)

		winners := 0

		for range 2 {
			if <-results == nil {
				winners++
			}
		}

		if winners != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion)+1 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind IN(2,3)") != 1 {
			t.Fatal("competing credential snapshots both committed")
		}
	})
}

func testResetFinalChecks(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"inactive", "UPDATE users SET active=false"},
		{"mailbox changed", "UPDATE users SET email='different@example.com'"},
		{"verification cleared", "UPDATE users SET mailbox_verified_at=NULL"},
		{"version changed", "UPDATE users SET auth_version=auth_version+1"},
		{"policy changed", "UPDATE mailbox_tokens SET policy_revision='other' WHERE purpose=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResetFixture(t, config.RecoveryConfig{})

			p, err := f.q.ReservePasswordReset(t.Context(), f.issue.Token, "mailbox-v1", f.settings)
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			holder, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer holder.Rollback()

			_, err = dal.New(holder).GetUserForAuth(ctx, f.actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			encoded := replacementHash(t)
			results := make(chan error, 1)

			go func() { _, e := f.q.CompletePasswordReset(ctx, *p, encoded, "mailbox-v1", f.settings); results <- e }()

			waitSessionLock(t, ctx, f.db, holder)

			_, err = holder.ExecContext(ctx, tc.sql)
			if err != nil {
				t.Fatal(err)
			}

			err = holder.Commit()
			if err != nil {
				t.Fatal(err)
			}

			err = <-results
			if err == nil || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 1 {
				t.Fatal("post-lock current state not rechecked")
			}
		})
	}

	t.Run("post-lock lease expiry", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{Timeout: "1s", Lease: "1s"})

		p, err := f.q.ReservePasswordReset(t.Context(), f.issue.Token, "mailbox-v1", f.settings)
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 6*time.Second)
		defer cancel()

		holder, err := f.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetUserForAuth(ctx, f.actor.UserID)
		if err != nil {
			t.Fatal(err)
		}

		encoded := replacementHash(t)
		results := make(chan error, 1)

		go func() { _, e := f.q.CompletePasswordReset(ctx, *p, encoded, "mailbox-v1", f.settings); results <- e }()

		waitSessionLock(t, ctx, f.db, holder)

		timer := time.NewTimer(time.Until(p.LeaseUntil.Add(100 * time.Millisecond)))
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-results
		if err == nil || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) {
			t.Fatal("expired lease committed")
		}
	})
	t.Run("post-write token expiry", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})
		sessionSQL(t, f.db, "WITH sampled AS (SELECT clock_timestamp() AS now) UPDATE mailbox_tokens SET created_at=sampled.now-interval '1 hour'+interval '1 second',expires_at=sampled.now+interval '1 second' FROM sampled WHERE purpose=2")
		boundary := resetBoundary{Queries: f.q, reset: func(context.Context, core.MailboxRecord) {
			sessionSQL(t, f.db, "CREATE FUNCTION slow_reset_notice() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1.1); RETURN NEW; END $$; CREATE TRIGGER slow_reset_notice BEFORE INSERT ON recovery_notices FOR EACH ROW EXECUTE FUNCTION slow_reset_notice()")
		}}

		_, err := recoveryForChange(t, f.base, boundary, config.RecoveryConfig{}).ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
		if err == nil || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE purpose=2 AND consumed_at IS NOT NULL") != 0 {
			t.Fatal("post-write expiry partially committed")
		}
	})
}

func testResetFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check func(context.Context, string) (bool, error)
		want  error
	}{
		{"rejected", func(context.Context, string) (bool, error) { return true, nil }, core.ErrPasswordDisallowed},
		{"checker error", func(context.Context, string) (bool, error) { return false, errors.New("private diagnostic") }, core.ErrPasswordCheckFailed},
		{"cancelled", func(ctx context.Context, _ string) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newResetFixture(t, config.RecoveryConfig{})
			s := resetCountingService(t, f, changeChecker{tc.check})

			ctx := t.Context()
			if tc.name == "cancelled" {
				work, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()

				ctx = work
			}

			_, err := s.ResetPassword(ctx, f.issue.Token.Bearer(), changedPassword)
			if !errors.Is(err, tc.want) {
				t.Fatalf("candidate failure: %v", err)
			}

			if mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 0 {
				t.Fatal("failure refunded admission or mutated credential")
			}
		})
	}

	for _, target := range []string{"users", "mailbox_tokens", "sessions", "auth_pending", "recovery_notices"} {
		t.Run("rollback "+target, func(t *testing.T) {
			f := newResetFixture(t, config.RecoveryConfig{})

			enrollment, err := core.NewAuthenticatorService(f.base, f.q, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
			if err != nil {
				t.Fatal(err)
			}

			_, err = enrollment.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
			if err != nil {
				t.Fatal(err)
			}

			before, err := f.q.GetUserByID(t.Context(), f.actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			op := "UPDATE"
			if target == "sessions" || target == "auth_pending" {
				op = "DELETE"
			}

			if target == "recovery_notices" {
				op = "INSERT"
			}

			boundary := resetBoundary{Queries: f.q, reset: func(context.Context, core.MailboxRecord) {
				sessionSQL(t, f.db, "CREATE FUNCTION reject_reset() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$; CREATE TRIGGER reject_reset BEFORE "+op+" ON "+target+" FOR EACH ROW EXECUTE FUNCTION reject_reset()")
			}}

			_, err = recoveryForChange(t, f.base, boundary, config.RecoveryConfig{}).ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
			if err == nil {
				t.Fatal("forced failure committed")
			}

			after, err := f.q.GetUserByID(t.Context(), f.actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			if before.PasswordHash != after.PasswordHash || before.AuthVersion != after.AuthVersion ||
				mailboxCount(t, f.db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM auth_pending") != 1 ||
				mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE purpose=2 AND consumed_at IS NOT NULL") != 0 ||
				mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 1 ||
				mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE kind=3") != 0 {
				t.Fatal("reset partially committed or refunded admission")
			}
		})
	}

	t.Run("notice capacity", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})
		sessionSQL(t, f.db, "INSERT INTO recovery_notices(id,user_id,kind,destination,created_at) SELECT 'notice-'||n::text,u.id,1,u.email,clock_timestamp()+n*interval '1 microsecond' FROM users u CROSS JOIN generate_series(1,19) n")

		_, err := f.service.ResetPassword(t.Context(), f.issue.Token.Bearer(), changedPassword)
		if !errors.Is(err, core.ErrRecoveryCapacity) || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) || mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE purpose=2 AND consumed_at IS NOT NULL") != 0 {
			t.Fatal("notice capacity broke atomicity")
		}
	})
}
