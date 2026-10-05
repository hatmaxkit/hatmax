//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

const changedPassword = "a different complete password"

func passwordChangePolicy() core.PasswordChangePolicy {
	r := PasswordRequirement()
	r.Proof = core.RequirePhishingResistantMFA
	r.MaxAge = 5 * time.Minute

	return core.PasswordChangePolicy{Requirement: r, AllowPassword: true}
}
func recoveryForChange(t *testing.T, base *core.Service, q core.RecoveryQueries, cfg config.RecoveryConfig) *core.RecoveryService {
	t.Helper()

	s, err := core.NewRecoveryService(base, q, cfg, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}

	return s
}
func passwordAuthorization(t *testing.T, q *Queries, actor *core.IssuedSession, p core.PasswordChangePolicy) (*core.PasswordChangeAuthorization, config.RecoverySettings) {
	t.Helper()

	settings, err := (config.RecoveryConfig{}).RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}

	digest, err := core.ParseSessionToken(actor.Token)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := q.AuthorizePasswordChange(t.Context(), digest, p, settings)
	if err != nil {
		t.Fatal(err)
	}

	return snapshot, settings
}
func replacementHash(t *testing.T) string {
	t.Helper()

	v, err := model.NewPasswordVerifier(model.PasswordVerifierConfig{MemoryKiB: 19456, Iterations: 2, Parallelism: 1})
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := v.Hash(t.Context(), changedPassword)
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}

// Real sessions, verifier output and subject-first PostgreSQL transactions
// establish authority, atomic revocation and durable attempts across failures.
func TestPasswordChangeTransactions(t *testing.T) {
	t.Run("complete invalidation", testPasswordChangeComplete)
	t.Run("factor policies", testPasswordChangeFactors)
	t.Run("backup retention", testPasswordChangeBackups)
	t.Run("post-lock factor binding", testPasswordChangeFactorLock)
	t.Run("KDF admission failures", testPasswordChangeKDF)
	t.Run("snapshot isolation", testPasswordChangeSnapshot)
	t.Run("post-lock freshness", testPasswordChangeLockTime)
	t.Run("post-write freshness", testPasswordChangeWriteTime)
	t.Run("concurrent changes", testPasswordChangeRace)
	t.Run("rollback and capacity", testPasswordChangeRollback)
	t.Run("checker failures", testPasswordChangeChecker)
	t.Run("shared completion budget", testPasswordChangeBudget)
}

func testPasswordChangeComplete(t *testing.T) {
	db, q, base, enrollment, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})

	outcome, err := base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	actor, ok := outcome.CompletedSession()
	if !ok {
		t.Fatal("missing actor")
	}

	s := recoveryForChange(t, base, q, config.RecoveryConfig{})

	issue, err := s.RequestMailboxVerification(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}

	sessionSQL(t, db, "UPDATE users SET mailbox_verified_at=clock_timestamp()")

	before, err := q.GetUserByID(t.Context(), actor.UserID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy())
	if err != nil || result == nil {
		t.Fatalf("change: %v", err)
	}

	after, err := q.GetUserByID(t.Context(), actor.UserID)
	if err != nil {
		t.Fatal(err)
	}

	if after.AuthVersion != before.AuthVersion+1 || after.PasswordHash == before.PasswordHash || !after.Active || after.MailboxVerifiedAt == nil || !after.MailboxVerifiedAt.Equal(*before.MailboxVerifiedAt) {
		t.Fatal("incorrect credential/account effects")
	}

	if mailboxCount(t, db, "SELECT count(*) FROM sessions") != 0 || mailboxCount(t, db, "SELECT count(*) FROM auth_pending") != 0 || mailboxCount(t, db, "SELECT count(*) FROM mailbox_tokens WHERE revoked_at IS NOT NULL") != 1 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices WHERE kind=2") != 1 {
		t.Fatal("incomplete invalidation/notice")
	}

	_, err = base.ValidateSession(t.Context(), actor.Token, PasswordRequirement(), core.NoActivity)
	if err == nil {
		t.Fatal("actor survived")
	}

	_, err = enrollment.FinishWebAuthnEnrollment(t.Context(), challenge.Token, enrollmentResponse(t, challenge), enrollmentRequirement())
	if err == nil {
		t.Fatal("pending survived")
	}

	if _, err = s.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer()); err == nil {
		t.Fatal("mailbox slot survived")
	}

	if _, err = base.Signin(t.Context(), before.Email, "a distinct safe password", PasswordRequirement()); !errors.Is(err, core.ErrInvalidPassword) {
		t.Fatalf("old password: %v", err)
	}

	out, err := base.Signin(t.Context(), before.Email, changedPassword, PasswordRequirement())
	if err != nil {
		t.Fatal(err)
	}

	if _, ok = out.CompletedSession(); !ok {
		t.Fatal("new complete password failed")
	}
}

func testPasswordChangeFactors(t *testing.T) {
	t.Run("password requires explicit permission", func(t *testing.T) {
		_, q, base, actor := sessionFixture(t)
		p := passwordChangePolicy()
		p.AllowPassword = false

		_, err := recoveryForChange(t, base, q, config.RecoveryConfig{}).ChangePassword(t.Context(), actor.Token, changedPassword, p)
		if !errors.Is(err, core.ErrSessionProof) {
			t.Fatalf("proof: %v", err)
		}
	})
	t.Run("WebAuthn default and retention", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)
		s := recoveryForChange(t, f.base, f.q, config.RecoveryConfig{})
		// A current password-only session cannot ignore an established factor.
		out, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		weak, _ := out.CompletedSession()
		if _, err = s.ChangePassword(t.Context(), weak.Token, changedPassword, passwordChangePolicy()); !errors.Is(err, core.ErrSessionProof) {
			t.Fatalf("weak proof: %v", err)
		}

		before, err := dal.New(f.db).LockAuthenticator(t.Context(), f.factor.ID)
		if err != nil {
			t.Fatal(err)
		}

		p := passwordChangePolicy()

		p.Requirement.Proof = 0
		if _, err = s.ChangePassword(t.Context(), actor.Token, changedPassword, p); err != nil {
			t.Fatal(err)
		}

		after, err := dal.New(f.db).LockAuthenticator(t.Context(), f.factor.ID)
		if err != nil {
			t.Fatal(err)
		}

		if !sameFactor(factorSnapshot(before), factorSnapshot(after)) {
			t.Fatal("factor/replay state changed")
		}

		ch, err := f.svc.BeginWebAuthnAuthentication(t.Context(), "session@example.com", webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}

		if _, err = f.svc.FinishWebAuthn(t.Context(), ch.Token, signedAssertion(t, f, ch, 2, 5, nil), webAuthnRequirement()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("TOTP requires explicit lower policy", func(t *testing.T) {
		f := newFallbackFixture(t, config.FallbackConfig{})
		actor := finishTOTP(t, f)

		s := recoveryForChange(t, f.base, f.q, config.RecoveryConfig{})
		if _, err := s.ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy()); !errors.Is(err, core.ErrSessionProof) {
			t.Fatalf("strong default: %v", err)
		}

		before, err := dal.New(f.db).LockTOTP(t.Context(), actor.UserID)
		if err != nil {
			t.Fatal(err)
		}

		p := passwordChangePolicy()

		p.Requirement.Proof = core.RequireMFA
		if _, err = s.ChangePassword(t.Context(), actor.Token, changedPassword, p); err != nil {
			t.Fatal(err)
		}

		after, err := dal.New(f.db).LockTOTP(t.Context(), actor.UserID)
		if err != nil {
			t.Fatal(err)
		}

		if before.ID != after.ID || before.Revision != after.Revision || before.AcceptedStep != after.AcceptedStep || before.KeyID != after.KeyID || !bytes.Equal(before.Envelope, after.Envelope) {
			t.Fatal("TOTP state changed")
		}

		pending, err := f.svc.BeginFallbackAuthentication(t.Context(), "session@example.com", changedPassword, core.FallbackTOTP, fallbackRequirement())
		if err != nil {
			t.Fatal(err)
		}

		if _, err = f.svc.FinishFallback(t.Context(), pending.Token, fallbackCode(t, f), fallbackRequirement()); err != nil {
			t.Fatal(err)
		}
	})
}

// Mutations bypass version increment only to isolate each final recheck. Normal
// production mutations also advance AuthVersion and serialize on the subject.
func testPasswordChangeSnapshot(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"generation", "UPDATE sessions SET generation=generation+1"},
		{"policy", "UPDATE sessions SET policy_revision='changed'"},
		{"version", "UPDATE users SET auth_version=auth_version+1"},
		{"inactive", "UPDATE users SET active=false"},
		{"deleted actor", "DELETE FROM sessions"},
		{"proof age", "UPDATE sessions SET proof_verified_at=proof_verified_at-interval '10 minutes',authenticated_at=authenticated_at-interval '10 minutes',created_at=created_at-interval '10 minutes'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, q, _, actor := sessionFixture(t)
			p := passwordChangePolicy()
			snapshot, settings := passwordAuthorization(t, q, actor, p)

			before, err := q.GetUserByID(t.Context(), actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			sessionSQL(t, db, tc.sql)

			if _, err = q.CommitPasswordChange(t.Context(), *snapshot, replacementHash(t), p, settings); err == nil {
				t.Fatal("stale snapshot authorized")
			}

			after, err := q.GetUserByID(t.Context(), actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			if before.PasswordHash != after.PasswordHash || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 0 || mailboxCount(t, db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 {
				t.Fatal("failure changed credential or refunded admission")
			}
		})
	}

	for _, tc := range []struct{ name, sql string }{
		{"factor revision", "UPDATE authenticators SET revision=revision+1"},
		{"factor removed", "DELETE FROM authenticators"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
			actor := finishAssertion(t, f, beginAssertion(t, f), 1)
			p := passwordChangePolicy()
			snapshot, settings := passwordAuthorization(t, f.q, actor, p)
			sessionSQL(t, f.db, tc.sql)

			if _, err := f.q.CommitPasswordChange(t.Context(), *snapshot, replacementHash(t), p, settings); err == nil {
				t.Fatal("changed factor authorized")
			}
		})
	}

	t.Run("foreign actor cannot select subject", func(t *testing.T) {
		db, q, base, actor := sessionFixture(t)
		if _, err := base.Signup(t.Context(), "other@example.com", "another safe long password"); err != nil {
			t.Fatal(err)
		}

		other, err := testSignin(base, t.Context(), "other@example.com", "another safe long password")
		if err != nil {
			t.Fatal(err)
		}

		p := passwordChangePolicy()
		snapshot, settings := passwordAuthorization(t, q, actor, p)

		snapshot.ActorDigest, _ = core.ParseSessionToken(other.Token)
		if _, err = q.CommitPasswordChange(t.Context(), *snapshot, replacementHash(t), p, settings); err == nil {
			t.Fatal("foreign bearer authorized old subject")
		}

		if mailboxCount(t, db, "SELECT count(*) FROM users WHERE auth_version<>1") != 0 {
			t.Fatal("foreign account mutation")
		}
	})
	t.Run("trusted policy changed", func(t *testing.T) {
		_, q, _, actor := sessionFixture(t)
		p := passwordChangePolicy()
		snapshot, settings := passwordAuthorization(t, q, actor, p)

		p.AllowPassword = false
		if _, err := q.CommitPasswordChange(t.Context(), *snapshot, replacementHash(t), p, settings); err == nil {
			t.Fatal("policy mismatch accepted")
		}
	})
}

func testPasswordChangeLockTime(t *testing.T) {
	db, q, _, actor := sessionFixture(t)
	p := passwordChangePolicy()
	p.Requirement.MaxAge = time.Second
	snapshot, settings := passwordAuthorization(t, q, actor, p)
	encoded := replacementHash(t)

	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()

	holder, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Rollback()

	if _, err = dal.New(holder).GetUserForAuth(ctx, actor.UserID); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)

	go func() { _, e := q.CommitPasswordChange(ctx, *snapshot, encoded, p, settings); result <- e }()

	waitSessionLock(t, ctx, db, holder)
	// Wait at the required proof boundary, after observing the actual lock wait.
	timer := time.NewTimer(time.Until(actor.Proof.VerifiedAt.Add(time.Second + 100*time.Millisecond)))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	case <-timer.C:
	}

	if err = holder.Commit(); err != nil {
		t.Fatal(err)
	}

	if err = <-result; !errors.Is(err, core.ErrSessionProofExpired) {
		t.Fatalf("post-lock: %v", err)
	}

	if mailboxCount(t, db, "SELECT auth_version FROM users") != 1 {
		t.Fatal("expired proof committed")
	}
}

func testPasswordChangeWriteTime(t *testing.T) {
	db, q, _, actor := sessionFixture(t)
	p := passwordChangePolicy()
	p.Requirement.MaxAge = time.Second
	snapshot, settings := passwordAuthorization(t, q, actor, p)
	sessionSQL(t, db, "CREATE FUNCTION slow_password_notice() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(1.1); RETURN NEW; END $$; CREATE TRIGGER slow_password_notice BEFORE INSERT ON recovery_notices FOR EACH ROW EXECUTE FUNCTION slow_password_notice()")

	if _, err := q.CommitPasswordChange(t.Context(), *snapshot, replacementHash(t), p, settings); !errors.Is(err, core.ErrSessionProofExpired) {
		t.Fatalf("post-write: %v", err)
	}

	if mailboxCount(t, db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 0 {
		t.Fatal("late proof failure partially committed")
	}
}

func testPasswordChangeRace(t *testing.T) {
	db, q, base, actor := sessionFixture(t)
	s := recoveryForChange(t, base, q, config.RecoveryConfig{})
	start := make(chan struct{})
	results := make(chan error, 2)

	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			<-start

			_, err := s.ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy())
			results <- err
		})
	}

	close(start)
	workers.Wait()
	close(results)

	winners := 0

	for err := range results {
		if err == nil {
			winners++
		}
	}

	if winners != 1 || mailboxCount(t, db, "SELECT auth_version FROM users") != 2 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 1 {
		t.Fatal("concurrent changes were not one winner")
	}
}

func testPasswordChangeRollback(t *testing.T) {
	for _, target := range []string{"users", "sessions", "auth_pending", "mailbox_tokens", "recovery_notices"} {
		t.Run(target, func(t *testing.T) {
			db, q, base, _, _ := enrollmentFixture(t, config.AuthenticatorConfig{})

			out, err := base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
			if err != nil {
				t.Fatal(err)
			}

			actor, _ := out.CompletedSession()

			s := recoveryForChange(t, base, q, config.RecoveryConfig{})
			if _, err = s.RequestMailboxVerification(t.Context(), "session@example.com"); err != nil {
				t.Fatal(err)
			}

			op := "UPDATE"
			if target == "sessions" || target == "auth_pending" {
				op = "DELETE"
			}

			if target == "recovery_notices" {
				op = "INSERT"
			}

			sessionSQL(t, db, "CREATE FUNCTION reject_password_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$; CREATE TRIGGER reject_password_change BEFORE "+op+" ON "+target+" FOR EACH ROW EXECUTE FUNCTION reject_password_change()")

			before, err := q.GetUserByID(t.Context(), actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			if _, err = s.ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy()); err == nil {
				t.Fatal("forced failure committed")
			}

			after, err := q.GetUserByID(t.Context(), actor.UserID)
			if err != nil {
				t.Fatal(err)
			}

			if before.PasswordHash != after.PasswordHash || before.AuthVersion != after.AuthVersion || mailboxCount(t, db, "SELECT count(*) FROM sessions") != 2 || mailboxCount(t, db, "SELECT count(*) FROM auth_pending") != 1 || mailboxCount(t, db, "SELECT count(*) FROM mailbox_tokens WHERE revoked_at IS NOT NULL") != 0 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 0 || mailboxCount(t, db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 {
				t.Fatal("partial effects or refunded attempts")
			}
		})
	}

	t.Run("notification capacity", func(t *testing.T) {
		db, q, base, actor := sessionFixture(t)
		sessionSQL(t, db, "INSERT INTO recovery_notices(id,user_id,kind,destination,created_at) SELECT 'notice-'||n::text,u.id,1,u.email,clock_timestamp()+n*interval '1 microsecond' FROM users u CROSS JOIN generate_series(1,20) n")

		if _, err := recoveryForChange(t, base, q, config.RecoveryConfig{}).ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy()); !errors.Is(err, core.ErrRecoveryCapacity) {
			t.Fatalf("capacity: %v", err)
		}

		if mailboxCount(t, db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, db, "SELECT count(*) FROM sessions") != 1 {
			t.Fatal("capacity changed security state")
		}
	})
}

type changeChecker struct {
	check func(context.Context, string) (bool, error)
}

func (c changeChecker) Disallowed(ctx context.Context, p string) (bool, error) {
	return c.check(ctx, p)
}

func testPasswordChangeChecker(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check func(context.Context, string) (bool, error)
		want  error
	}{
		{"rejected", func(context.Context, string) (bool, error) { return true, nil }, core.ErrPasswordDisallowed},
		{"failed", func(context.Context, string) (bool, error) { return false, errors.New("private diagnostic") }, core.ErrPasswordCheckFailed},
		{"cancelled", func(ctx context.Context, _ string) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, q, _, actor := sessionFixture(t)
			cfg := config.New()
			cfg.Auth.ArgonMemoryKiB = 19456
			cfg.Auth.ArgonIterations = 2
			cfg.Auth.ArgonParallelism = 1

			base, err := core.NewService(q, cfg, changeChecker{tc.check}, credentialAdmissionForTest(t, q, cfg.CredentialAdmission), securityObservationsForTest(t), log.NewTestLogger("error"))
			if err != nil {
				t.Fatal(err)
			}

			s := recoveryForChange(t, base, q, config.RecoveryConfig{})

			ctx := t.Context()
			if tc.name == "cancelled" {
				work, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()

				ctx = work
			}

			if _, err = s.ChangePassword(ctx, actor.Token, changedPassword, passwordChangePolicy()); !errors.Is(err, tc.want) {
				t.Fatalf("checker: %v", err)
			}

			if mailboxCount(t, db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 {
				t.Fatal("checker failure effects")
			}
		})
	}
}

func testPasswordChangeBudget(t *testing.T) {
	db, q, base, actor := sessionFixture(t)
	s := recoveryForChange(t, base, q, config.RecoveryConfig{CompletionAttempts: 1})

	issue, err := s.RequestMailboxVerification(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err = s.ChangePassword(t.Context(), actor.Token, "short", passwordChangePolicy()); !errors.Is(err, core.ErrPasswordTooShort) {
		t.Fatalf("candidate: %v", err)
	}

	if _, err = s.ChangePassword(t.Context(), actor.Token, changedPassword, passwordChangePolicy()); !errors.Is(err, core.ErrRecoveryAttempts) {
		t.Fatalf("spent budget: %v", err)
	}

	if _, err = s.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer()); !errors.Is(err, core.ErrRecoveryAttempts) {
		t.Fatalf("shared mailbox budget: %v", err)
	}

	if mailboxCount(t, db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 {
		t.Fatal("budget denial mutation")
	}
}
