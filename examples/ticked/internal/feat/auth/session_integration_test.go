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
	"fmt"
	"sync"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
)

func sessionFixture(t *testing.T) (*sql.DB, *Queries, *core.Service, *core.IssuedSession) {
	t.Helper()
	db, q, cfg := credentialDatabase(t)

	svc, err := core.NewService(q, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q, cfg.CredentialAdmission), securityObservationsForTest(t), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Signup(t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	issued, err := testSignin(svc, t.Context(), "session@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	return db, q, svc, issued
}

func sessionSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
}

func storedSession(t *testing.T, db *sql.DB, issued *core.IssuedSession) dal.Session {
	t.Helper()

	digest, err := core.ParseSessionToken(issued.Token)
	if err != nil {
		t.Fatal(err)
	}

	stored, err := dal.New(db).GetSessionByDigest(t.Context(), digest[:])
	if err != nil {
		t.Fatal(err)
	}

	return stored
}

func waitSessionLock(t *testing.T, ctx context.Context, db *sql.DB, holder *sql.Tx) {
	t.Helper()

	var pid int

	err := holder.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid)
	if err != nil {
		t.Fatal(err)
	}

	for {
		var blocked bool

		err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1::integer=ANY(pg_blocking_pids(pid)))", pid).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}

		if blocked {
			return
		}

		select {
		case <-ctx.Done():
			t.Fatal("validation did not block on the owned row lock")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// Real transactions establish digest-only storage, current state and time after
// row waits, coalesced activity, rollback and bounded cleanup.
func TestSessionTransactions(t *testing.T) {
	t.Run("digest and owned metadata", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		stored := storedSession(t, db, issued)

		digest, err := core.ParseSessionToken(issued.Token)
		if err != nil || string(stored.TokenDigest) != string(digest[:]) || len(stored.TokenDigest) != 32 {
			t.Fatal("wrong stored digest")
		}

		var plaintextColumns int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='sessions' AND column_name='token'").Scan(&plaintextColumns)
		if err != nil || plaintextColumns != 0 {
			t.Fatal("plaintext bearer column exists")
		}

		validated, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if err != nil || validated.Session.ID != issued.ID || validated.User.ID != issued.UserID {
			t.Fatalf("validate: %v", err)
		}

		sessionSQL(t, db, "UPDATE users SET roles=ARRAY['reader'] WHERE id=$1", issued.UserID)

		first, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal(err)
		}

		first.User.Roles[0] = "administrator"
		first.Session.ID = "mutated"

		second, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if err != nil || second.User.Roles[0] != "reader" || second.Session.ID != issued.ID {
			t.Fatal("returned metadata aliases persistent state")
		}

		err = svc.Signout(t.Context(), issued.Token)
		if err != nil {
			t.Fatal(err)
		}

		second, err = svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if second != nil || !errors.Is(err, core.ErrSessionNotFound) {
			t.Fatal("revoked bearer accepted")
		}
	})
	t.Run("coalesced and concurrent activity", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		sessionSQL(t, db, "UPDATE sessions SET authenticated_at=clock_timestamp()-interval '2 minutes',proof_verified_at=clock_timestamp()-interval '150 seconds', created_at=clock_timestamp()-interval '3 minutes',last_activity_at=clock_timestamp()-interval '90 seconds' WHERE id=$1", issued.ID)
		before := storedSession(t, db, issued)

		_, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal(err)
		}

		poll := storedSession(t, db, issued)
		if !poll.LastActivityAt.Equal(before.LastActivityAt) {
			t.Fatal("polling renewed inactivity")
		}

		sessionSQL(t, db, `CREATE TABLE activity_count (id text);
   CREATE FUNCTION count_activity() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO activity_count VALUES(NEW.id); RETURN NEW; END $$;
   CREATE TRIGGER count_activity AFTER UPDATE ON sessions FOR EACH ROW EXECUTE FUNCTION count_activity();`)

		failures := make(chan error, 8)

		var workers sync.WaitGroup
		for range 8 {
			workers.Go(func() {
				_, validateErr := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.RelevantActivity)
				failures <- validateErr
			})
		}

		workers.Wait()
		close(failures)

		for validateErr := range failures {
			if validateErr != nil {
				t.Fatal(validateErr)
			}
		}

		after := storedSession(t, db, issued)

		var writes int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM activity_count").Scan(&writes)
		if err != nil || writes != 1 || !after.LastActivityAt.After(before.LastActivityAt) || !after.ExpiresAt.Equal(before.ExpiresAt) || !after.AuthenticatedAt.Equal(before.AuthenticatedAt) {
			t.Fatalf("activity bounds: writes %d, error %v", writes, err)
		}
	})
	t.Run("activation cycle after lock wait", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetUserForAuth(ctx, issued.UserID)
		if err != nil {
			t.Fatal(err)
		}

		result := make(chan error, 1)

		go func() {
			_, validateErr := svc.ValidateSession(ctx, issued.Token, PasswordRequirement(), core.RelevantActivity)
			result <- validateErr
		}()

		waitSessionLock(t, ctx, db, holder)

		for _, active := range []bool{false, true} {
			_, err = holder.ExecContext(ctx, "UPDATE users SET active=$2,auth_version=auth_version+1 WHERE id=$1", issued.UserID, active)
			if err != nil {
				t.Fatal(err)
			}
		}

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrCredentialChanged) {
			t.Fatalf("old session survived activation cycle: %v", err)
		}

		stored := storedSession(t, db, issued)
		if !stored.LastActivityAt.Equal(issued.LastActivityAt) {
			t.Fatal("stale state renewed activity")
		}
	})
	t.Run("expiry during session lock wait", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		sessionSQL(t, db, "UPDATE sessions SET created_at=clock_timestamp()-interval '6 minutes',authenticated_at=clock_timestamp()-interval '5 minutes',proof_verified_at=clock_timestamp()-interval '330 seconds',last_activity_at=clock_timestamp()-interval '239 seconds',inactivity_us=240000000 WHERE id=$1", issued.ID)
		before := storedSession(t, db, issued)

		metadata, err := toAuthSession(before)
		if err != nil {
			t.Fatal(err)
		}

		var now time.Time

		err = db.QueryRowContext(t.Context(), "SELECT clock_timestamp()").Scan(&now)
		if err != nil || metadata.Check(now) != nil {
			t.Fatal("fixture was expired before lock wait")
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetSessionForUpdate(ctx, before.TokenDigest)
		if err != nil {
			t.Fatal(err)
		}

		result := make(chan error, 1)

		go func() {
			_, validateErr := svc.ValidateSession(ctx, issued.Token, PasswordRequirement(), core.RelevantActivity)
			result <- validateErr
		}()

		waitSessionLock(t, ctx, db, holder)

		for {
			var expired bool

			err = db.QueryRowContext(ctx, "SELECT clock_timestamp()>=$1::timestamptz", before.LastActivityAt.Add(4*time.Minute)).Scan(&expired)
			if err != nil {
				t.Fatal(err)
			}

			if expired {
				break
			}

			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(10 * time.Millisecond):
			}
		}

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrSessionExpired) {
			t.Fatalf("lock wait revived expired session: %v", err)
		}

		after := storedSession(t, db, issued)
		if !after.LastActivityAt.Equal(before.LastActivityAt) {
			t.Fatal("expired activity was renewed")
		}
	})
	t.Run("activity commit rollback", func(t *testing.T) {
		db, _, svc, issued := sessionFixture(t)
		sessionSQL(t, db, "UPDATE sessions SET created_at=clock_timestamp()-interval '3 minutes',authenticated_at=clock_timestamp()-interval '2 minutes',proof_verified_at=clock_timestamp()-interval '150 seconds',last_activity_at=clock_timestamp()-interval '90 seconds' WHERE id=$1", issued.ID)
		before := storedSession(t, db, issued)
		sessionSQL(t, db, `CREATE FUNCTION reject_activity() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced activity commit failure'; END $$;
   CREATE CONSTRAINT TRIGGER reject_activity AFTER UPDATE ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_activity();`)

		validated, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.RelevantActivity)
		if err == nil || validated != nil {
			t.Fatal("failed commit returned authorized state")
		}

		after := storedSession(t, db, issued)
		if !after.LastActivityAt.Equal(before.LastActivityAt) {
			t.Fatal("failed commit retained activity")
		}
	})
	t.Run("role invalidation", func(t *testing.T) {
		_, q, svc, issued := sessionFixture(t)

		err := q.UpdateUserRoles(t.Context(), issued.UserID, []string{"reader"}, time.Now())
		if err != nil {
			t.Fatal(err)
		}

		validated, err := svc.ValidateSession(t.Context(), issued.Token, PasswordRequirement(), core.NoActivity)
		if validated != nil || !errors.Is(err, core.ErrCredentialChanged) {
			t.Fatalf("role change retained session: %v", err)
		}
	})
	t.Run("bounded idle and absolute cleanup", func(t *testing.T) {
		db, q, _, issued := sessionFixture(t)

		for i := range 5 {
			token, err := crypto.GenerateSecureToken(32)
			if err != nil {
				t.Fatal(err)
			}

			digest, err := core.ParseSessionToken(token)
			if err != nil {
				t.Fatal(err)
			}

			now := time.Now().UTC().Truncate(time.Microsecond)
			proof := now.Add(-2 * time.Hour)

			expiry := now.Add(time.Hour)
			if i%2 == 0 {
				expiry = now.Add(-time.Hour)
			}

			_, err = dal.New(db).CreateSession(t.Context(), dal.CreateSessionParams{ID: fmt.Sprintf("expired-%d", i), UserID: issued.UserID, TokenDigest: digest[:], AuthVersion: issued.AuthVersion, PolicyRevision: PasswordRequirement().Revision, Generation: 1, ProofMethod: int16(core.PasswordProof), ProofVerifiedAt: proof, AuthenticatedAt: proof, CreatedAt: proof, LastActivityAt: proof, ExpiresAt: expiry, InactivityUs: time.Minute.Microseconds()})
			if err != nil {
				t.Fatal(err)
			}
		}

		removed, err := q.DeleteExpiredSessions(t.Context(), 2)
		if err != nil || removed != 2 {
			t.Fatalf("batch bound: %d, %v", removed, err)
		}

		removed, err = q.DeleteExpiredSessions(t.Context(), 1000)
		if err != nil || removed != 3 {
			t.Fatalf("remaining expired rows: %d, %v", removed, err)
		}

		_ = storedSession(t, db, issued)
		for _, limit := range []int{0, 1001} {
			removed, err = q.DeleteExpiredSessions(t.Context(), limit)
			if err == nil || removed != 0 {
				t.Fatal("invalid batch reached storage")
			}
		}
	})
}
