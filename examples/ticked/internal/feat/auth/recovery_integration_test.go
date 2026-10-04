//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/mailer"
)

type mailboxFixture struct {
	db       *sql.DB
	q        *Queries
	base     *core.Service
	svc      *core.RecoveryService
	issued   *core.MailboxIssue
	settings config.RecoverySettings
}

func newMailboxFixture(t *testing.T, options config.RecoveryConfig) mailboxFixture {
	t.Helper()
	db, q, base, _ := sessionFixture(t)
	svc, err := core.NewRecoveryService(base, q, options, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}
	issue, err := svc.RequestMailboxVerification(t.Context(), "session@example.com")
	if err != nil {
		t.Fatal(err)
	}
	settings, err := options.RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}
	return mailboxFixture{db: db, q: q, base: base, svc: svc, issued: issue, settings: settings}
}
func mailboxCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	err := db.QueryRowContext(t.Context(), query, args...).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Real transactions establish one-use ownership, durable denial accounting,
// reissue/consume serialization, final-state rollback and complete invalidation.
func TestMailboxTransactions(t *testing.T) {
	t.Run("complete invalidation", func(t *testing.T) {
		db, q, base, enrollment, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
		svc, err := core.NewRecoveryService(base, q, config.RecoveryConfig{}, "mailbox-v1")
		if err != nil {
			t.Fatal(err)
		}
		issue, err := svc.RequestMailboxVerification(t.Context(), "session@example.com")
		if err != nil {
			t.Fatal(err)
		}
		before, err := q.GetUserByEmail(t.Context(), "session@example.com")
		if err != nil {
			t.Fatal(err)
		}
		result, err := svc.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
		if err != nil || result == nil {
			t.Fatalf("confirm: %v", err)
		}
		current, err := q.GetUserByID(t.Context(), before.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.AuthVersion != before.AuthVersion+1 || current.MailboxVerifiedAt == nil || current.PasswordHash != before.PasswordHash || !current.Active {
			t.Fatal("invalid verification effects")
		}
		if mailboxCount(t, db, "SELECT count(*) FROM sessions") != 0 || mailboxCount(t, db, "SELECT count(*) FROM auth_pending") != 0 || mailboxCount(t, db, "SELECT count(*) FROM recovery_notices") != 1 {
			t.Fatal("incomplete atomic invalidation")
		}
		_, err = enrollment.FinishWebAuthnEnrollment(t.Context(), challenge.Token, enrollmentResponse(t, challenge), enrollmentRequirement())
		if err == nil {
			t.Fatal("old enrollment survived")
		}
		_, err = svc.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
		if err == nil {
			t.Fatal("token replay succeeded")
		}
		_, err = svc.RequestMailboxVerification(t.Context(), current.Email)
		if err == nil {
			t.Fatal("verified mailbox reissued")
		}
		_, err = base.ValidateSession(t.Context(), issue.Token.Bearer(), PasswordRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("mailbox token authenticated")
		}
	})
	t.Run("failure budgets survive reissue", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		wrong := f.issued.Token.ID() + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		for range 5 {
			_, err := f.svc.ConfirmMailboxVerification(t.Context(), wrong)
			if err == nil {
				t.Fatal("wrong secret accepted")
			}
		}
		if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens") != 5 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 5 {
			t.Fatal("failure budget was refunded")
		}
		_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if !errors.Is(err, core.ErrRecoveryAttempts) {
			t.Fatalf("token exhaustion: %v", err)
		}
		issue, err := f.svc.RequestMailboxVerification(t.Context(), "session@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 5 {
			t.Fatal("reissue reset shared budget")
		}
		_, err = f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if err == nil {
			t.Fatal("replaced bearer survived")
		}
		_, err = f.svc.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("issuance budget and capacity", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		for range 2 {
			_, err := f.svc.RequestMailboxVerification(t.Context(), "session@example.com")
			if err != nil {
				t.Fatal(err)
			}
		}
		_, err := f.svc.RequestMailboxVerification(t.Context(), "session@example.com")
		if !errors.Is(err, core.ErrRecoveryAttempts) {
			t.Fatalf("issuance budget: %v", err)
		}
		if mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens") != 1 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=1") != 3 {
			t.Fatal("issuance bound lost")
		}
	})
	t.Run("shared completion budget", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{CompletionAttempts: 1})
		wrong := f.issued.Token.ID() + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		_, err := f.svc.ConfirmMailboxVerification(t.Context(), wrong)
		if err == nil {
			t.Fatal("wrong secret accepted")
		}
		replacement, err := f.svc.RequestMailboxVerification(t.Context(), "session@example.com")
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.svc.ConfirmMailboxVerification(t.Context(), replacement.Token.Bearer())
		if !errors.Is(err, core.ErrRecoveryAttempts) {
			t.Fatal("replacement refunded shared budget")
		}
		if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 0 {
			t.Fatal("denial changed token/final state")
		}
	})
	t.Run("notification capacity rollback", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		sessionSQL(t, f.db, "INSERT INTO recovery_notices(id,user_id,kind,destination,created_at) SELECT 'notice-'||n::text,u.id,1,u.email,clock_timestamp()+n*interval '1 microsecond' FROM users u CROSS JOIN generate_series(1,20) n")
		_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if !errors.Is(err, core.ErrRecoveryCapacity) {
			t.Fatalf("notice capacity: %v", err)
		}
		if mailboxCount(t, f.db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 20 {
			t.Fatal("capacity failure partially committed")
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := f.svc.ConfirmMailboxVerification(ctx, f.issued.Token.Bearer())
		if err == nil || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens") != 0 {
			t.Fatal("cancelled work committed")
		}
	})
	t.Run("state isolation", func(t *testing.T) {
		for _, tc := range []struct{ name, sql string }{
			{"expired", "UPDATE mailbox_tokens SET created_at=created_at-interval '25 hours',expires_at=expires_at-interval '25 hours'"},
			{"policy", "UPDATE mailbox_tokens SET policy_revision='other'"},
			{"purpose", "UPDATE mailbox_tokens SET purpose=2,expires_at=created_at+interval '1 hour'"},
			{"target", "UPDATE users SET email='changed@example.com'"},
			{"version", "UPDATE users SET auth_version=auth_version+1"},
			{"inactive", "UPDATE users SET active=false"},
			{"consumed", "UPDATE mailbox_tokens SET consumed_at=created_at"},
			{"revoked", "UPDATE mailbox_tokens SET revoked_at=created_at"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f := newMailboxFixture(t, config.RecoveryConfig{})
				sessionSQL(t, f.db, tc.sql)
				_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
				if err == nil {
					t.Fatal("stale authority accepted")
				}
				if mailboxCount(t, f.db, "SELECT count(*) FROM users WHERE mailbox_verified_at IS NOT NULL") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 0 {
					t.Fatal("denial mutated final state")
				}
			})
		}
	})
	t.Run("double consume", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				<-start
				_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
				results <- err
			})
		}
		close(start)
		wg.Wait()
		close(results)
		successes := 0
		for err := range results {
			if err == nil {
				successes++
			}
		}
		if successes != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != 2 {
			t.Fatal("concurrent verification committed twice")
		}
	})
	t.Run("reissue versus consume", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		start := make(chan struct{})
		results := make(chan error, 2)
		var replacement *core.MailboxIssue
		var wg sync.WaitGroup
		wg.Go(func() {
			<-start
			_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
			results <- err
		})
		wg.Go(func() {
			<-start
			var err error
			replacement, err = f.svc.RequestMailboxVerification(t.Context(), "session@example.com")
			results <- err
		})
		close(start)
		wg.Wait()
		close(results)
		successes := 0
		for err := range results {
			if err == nil {
				successes++
			}
		}
		if successes != 1 {
			t.Fatal("reissue and old consume both committed")
		}
		if replacement != nil {
			_, err := f.svc.ConfirmMailboxVerification(t.Context(), replacement.Token.Bearer())
			if err != nil {
				t.Fatal(err)
			}
		}
		if mailboxCount(t, f.db, "SELECT auth_version FROM users") != 2 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 1 {
			t.Fatal("race broke final state")
		}
	})
	t.Run("rollback", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		sessionSQL(t, f.db, "CREATE FUNCTION reject_mailbox_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$; CREATE TRIGGER reject_mailbox_cleanup BEFORE DELETE ON sessions FOR EACH ROW EXECUTE FUNCTION reject_mailbox_cleanup()")
		_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if err == nil {
			t.Fatal("injected failure succeeded")
		}
		if mailboxCount(t, f.db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM users WHERE mailbox_verified_at IS NOT NULL") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE consumed_at IS NOT NULL") != 0 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens") != 1 {
			t.Fatal("partial mutation or attempt refund")
		}
		sessionSQL(t, f.db, "DROP TRIGGER reject_mailbox_cleanup ON sessions")
		_, err = f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("late transaction rollback", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{})
		sessionSQL(t, f.db, "CREATE FUNCTION reject_mailbox_notice() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced notice failure'; END $$; CREATE TRIGGER reject_mailbox_notice BEFORE INSERT ON recovery_notices FOR EACH ROW EXECUTE FUNCTION reject_mailbox_notice()")
		_, err := f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
		if err == nil {
			t.Fatal("late failure succeeded")
		}
		if mailboxCount(t, f.db, "SELECT count(*) FROM sessions") != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE consumed_at IS NOT NULL") != 0 {
			t.Fatal("late failure partially committed")
		}
	})
	t.Run("time after lock", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{VerificationTTL: "1m"})
		sessionSQL(t, f.db, "WITH anchor AS (SELECT date_trunc('microseconds',clock_timestamp())-interval '59 seconds' AS moment) UPDATE mailbox_tokens SET created_at=anchor.moment,expires_at=anchor.moment+interval '1 minute' FROM anchor")
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		holder, err := f.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()
		_, err = holder.ExecContext(ctx, "SELECT id FROM users FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() { _, err := f.svc.ConfirmMailboxVerification(ctx, f.issued.Token.Bearer()); result <- err }()
		waitSessionLock(t, ctx, f.db, holder)
		time.Sleep(1100 * time.Millisecond)
		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}
		if <-result == nil || mailboxCount(t, f.db, "SELECT count(*) FROM users WHERE mailbox_verified_at IS NOT NULL") != 0 {
			t.Fatal("expired token revived after lock wait")
		}
	})
	t.Run("retention cleanup", func(t *testing.T) {
		f := newMailboxFixture(t, config.RecoveryConfig{CleanupBatch: 1})
		removed, err := f.svc.CleanupMailboxTokens(t.Context())
		if err != nil || removed != 0 {
			t.Fatal("live token removed")
		}
		sessionSQL(t, f.db, "UPDATE mailbox_tokens SET created_at=created_at-interval '50 hours',expires_at=expires_at-interval '50 hours'")
		removed, err = f.svc.CleanupMailboxTokens(t.Context())
		if err != nil || removed != 1 {
			t.Fatalf("cleanup: %d %v", removed, err)
		}
	})
	t.Run("preserves actual MFA", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		actor := finishAssertion(t, f, beginAssertion(t, f), 1)
		_ = beginAssertion(t, f)
		var before string
		err := f.db.QueryRowContext(t.Context(), "SELECT row_to_json(a)::text FROM authenticators a").Scan(&before)
		if err != nil {
			t.Fatal(err)
		}
		svc, err := core.NewRecoveryService(f.base, f.q, config.RecoveryConfig{}, "mailbox-v1")
		if err != nil {
			t.Fatal(err)
		}
		issue, err := svc.RequestMailboxVerification(t.Context(), "session@example.com")
		if err != nil {
			t.Fatal(err)
		}
		_, err = svc.ConfirmMailboxVerification(t.Context(), issue.Token.Bearer())
		if err != nil {
			t.Fatal(err)
		}
		var after string
		err = f.db.QueryRowContext(t.Context(), "SELECT row_to_json(a)::text FROM authenticators a").Scan(&after)
		if err != nil || before != after {
			t.Fatal("confirmed factor/replay state changed")
		}
		_, err = f.base.ValidateSession(t.Context(), actor.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("old MFA session survived")
		}
		outcome, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := outcome.CompletedSession(); ok {
			t.Fatal("mailbox verification bypassed MFA")
		}
		_ = finishAssertion(t, f, beginAssertion(t, f), 2)
	})
}

type mailboxCapture struct {
	mu       sync.Mutex
	messages []*mailer.Message
	failure  bool
}

func (m *mailboxCapture) Send(_ context.Context, message *mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *message
	m.messages = append(m.messages, &copy)
	if m.failure {
		return errors.New("provider diagnostic with secret link: " + message.Text)
	}
	return nil
}

// Captured mail plus real storage proves that failed dispatch cannot roll back
// committed authority, retry limits are durable and links use a trusted origin.
func TestMailboxDeliveryTransactions(t *testing.T) {
	f := newMailboxFixture(t, config.RecoveryConfig{})
	capture := &mailboxCapture{failure: true}
	delivery, err := NewMailboxDelivery(f.svc, f.q, capture, log.NewTestLogger("error"), "https://trusted.example.com", false)
	if err != nil {
		t.Fatal(err)
	}
	err = delivery.RequestMailboxVerification(t.Context(), "session@example.com")
	if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "#token=") {
		t.Fatal("dispatch error leaked message")
	}
	sent := capture.messages[0]
	if !strings.Contains(sent.Text, "https://trusted.example.com/account/mailbox/confirm#token=") || sent.To[0].Email != "session@example.com" {
		t.Fatal("untrusted mail destination")
	}
	token := strings.Split(strings.Split(sent.Text, "#token=")[1], "\n")[0]
	_, err = f.svc.ConfirmMailboxVerification(t.Context(), f.issued.Token.Bearer())
	if err == nil {
		t.Fatal("failed reissue revived predecessor")
	}
	err = delivery.ConfirmMailboxVerification(t.Context(), token)
	if err != nil {
		t.Fatal(err)
	}

	var subject string
	err = f.db.QueryRowContext(t.Context(), "SELECT id FROM users").Scan(&subject)
	if err != nil {
		t.Fatal(err)
	}
	for range 7 {
		_ = delivery.DispatchMailboxNotices(t.Context(), subject, 1)
	}
	if mailboxCount(t, f.db, "SELECT attempts FROM recovery_notices") != 5 || mailboxCount(t, f.db, "SELECT count(*) FROM recovery_notices WHERE delivered_at IS NOT NULL") != 0 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != 2 {
		t.Fatal("notification failure restored state or exceeded budget")
	}
	capture.failure = false
	_ = delivery.DispatchMailboxNotices(t.Context(), subject, 1)
	if mailboxCount(t, f.db, "SELECT attempts FROM recovery_notices") != 5 {
		t.Fatal("exhausted notice retried")
	}
	for _, origin := range []string{"http://evil.example.com", "https://user@trusted.example.com", "https://trusted.example.com/path", "https://trusted.example.com?x=y", "https://trusted.example.com#token=x"} {
		_, err = NewMailboxDelivery(f.svc, f.q, capture, log.NewTestLogger("error"), origin, false)
		if err == nil {
			t.Fatal("unsafe trusted origin accepted")
		}
	}
	encoded, err := json.Marshal(f.issued)
	if err != nil || strings.Contains(string(encoded), f.issued.Token.Bearer()) {
		t.Fatal("public serialization leaked bearer")
	}
}
