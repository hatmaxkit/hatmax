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
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

// Real caller deadlines, owned leases and retained-only cleanup must preserve
// current authority and durable budgets across both recovery purposes.
func TestRecoveryBoundaries(t *testing.T) {
	t.Run("earlier caller deadline", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})

		holder, err := f.db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = holder.ExecContext(t.Context(), "SELECT id FROM users FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
		defer cancel()

		started := time.Now()

		_, err = f.service.ResetPassword(ctx, f.issue.Token.Bearer(), "a complete replacement password")
		if err == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) || time.Since(started) > time.Second {
			t.Fatal("earlier deadline did not bound subject lock wait")
		}

		err = holder.Rollback()
		if err != nil {
			t.Fatal(err)
		}

		if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 0 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 1 || mailboxCount(t, f.db, "SELECT auth_version FROM users") != int(f.actor.AuthVersion) {
			t.Fatal("unadmitted cancellation changed state or attempts")
		}

		_, err = f.base.ValidateSession(t.Context(), f.actor.Token, PasswordRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal("cancellation invalidated current access")
		}
	})
	t.Run("owned lease release", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{})

		token, err := core.ParseRecoveryToken(f.issue.Token.Bearer())
		if err != nil {
			t.Fatal(err)
		}

		first, err := f.q.ReservePasswordReset(t.Context(), token, "mailbox-v1", f.settings)
		if err != nil {
			t.Fatal(err)
		}

		err = f.q.ReleaseMailbox(t.Context(), first.ID, first.Revision)
		if err != nil {
			t.Fatal(err)
		}

		second, err := f.q.ReservePasswordReset(t.Context(), token, "mailbox-v1", f.settings)
		if err != nil {
			t.Fatal(err)
		}

		err = f.q.ReleaseMailbox(t.Context(), first.ID, first.Revision)
		if err != nil {
			t.Fatal(err)
		}

		var lease sql.NullTime

		err = f.db.QueryRowContext(t.Context(), "SELECT lease_until FROM mailbox_tokens WHERE id=$1", second.ID).Scan(&lease)
		if err != nil || !lease.Valid || !lease.Time.Equal(second.LeaseUntil) {
			t.Fatal("stale owner cleared a newer lease")
		}

		_, err = f.q.ReservePasswordReset(t.Context(), token, "mailbox-v1", f.settings)
		if !errors.Is(err, core.ErrRecoveryBusy) {
			t.Fatal("owned live lease did not exclude concurrent work")
		}

		if mailboxCount(t, f.db, "SELECT attempts FROM mailbox_tokens WHERE purpose=2") != 2 || mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 3 {
			t.Fatal("release refunded committed admission")
		}

		err = f.q.ReleaseMailbox(t.Context(), second.ID, second.Revision)
		if err != nil {
			t.Fatal(err)
		}

		var remaining sql.NullTime

		err = f.db.QueryRowContext(t.Context(), "SELECT lease_until FROM mailbox_tokens WHERE id=$1", second.ID).Scan(&remaining)
		if err != nil || remaining.Valid {
			t.Fatal("current owner could not release its lease")
		}
	})
	t.Run("retained cross-purpose cleanup", func(t *testing.T) {
		f := newResetFixture(t, config.RecoveryConfig{CleanupBatch: 1})

		n, err := f.service.CleanupMailboxTokens(t.Context())
		if err != nil || n != 0 {
			t.Fatal("recent terminal or live token removed")
		}

		token, err := core.ParseRecoveryToken(f.issue.Token.Bearer())
		if err != nil {
			t.Fatal(err)
		}

		_, err = f.q.ReservePasswordReset(t.Context(), token, "mailbox-v1", f.settings)
		if err != nil {
			t.Fatal(err)
		}
		// Move only fixture retention times; keep a live owned lease on the
		// revoked reset slot while the consumed verification slot can retire.
		_, err = f.db.ExecContext(t.Context(), `WITH anchor AS (SELECT date_trunc('microseconds',clock_timestamp()) AS moment)
        UPDATE mailbox_tokens SET
        created_at=anchor.moment-CASE WHEN purpose=1 THEN interval '49 hours' ELSE interval '26 hours' END,
        expires_at=anchor.moment-interval '25 hours',
        consumed_at=CASE WHEN purpose=1 THEN anchor.moment-interval '26 hours' ELSE NULL END,
        revoked_at=CASE WHEN purpose=2 THEN anchor.moment-interval '25 hours' ELSE NULL END,
        lease_until=CASE WHEN purpose=2 THEN anchor.moment+interval '1 minute' ELSE NULL END FROM anchor`)
		if err != nil {
			t.Fatal(err)
		}

		n, err = f.service.CleanupMailboxTokens(t.Context())
		if err != nil || n != 1 || mailboxCount(t, f.db, "SELECT count(*) FROM mailbox_tokens WHERE purpose=2") != 1 {
			t.Fatal("cleanup ignored its batch or live lease")
		}

		n, err = f.service.CleanupMailboxTokens(t.Context())
		if err != nil || n != 0 {
			t.Fatal("cleanup removed live leased terminal slot")
		}

		_, err = f.db.ExecContext(t.Context(), "UPDATE mailbox_tokens SET lease_until=NULL WHERE purpose=2")
		if err != nil {
			t.Fatal(err)
		}

		n, err = f.service.CleanupMailboxTokens(t.Context())
		if err != nil || n != 1 {
			t.Fatal("retained released slot did not retire")
		}

		n, err = f.service.CleanupMailboxTokens(t.Context())
		if err != nil || n != 0 {
			t.Fatal("cleanup did not terminate")
		}

		if mailboxCount(t, f.db, "SELECT attempts FROM recovery_budgets WHERE kind=3") != 2 {
			t.Fatal("cleanup refunded admission")
		}

		_, err = f.base.ValidateSession(t.Context(), f.actor.Token, PasswordRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal("cleanup changed current authentication")
		}
	})
}
