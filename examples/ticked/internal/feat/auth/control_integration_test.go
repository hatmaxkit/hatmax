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
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
	"reflect"
	"sync"
	"testing"
	"time"
)

type rotationWait struct {
	*Queries
	ready   chan core.SessionRecord
	release chan struct{}
}

func (q rotationWait) RotateSession(ctx context.Context, state core.CredentialState, digest core.SessionDigest, generation int64, record core.SessionRecord, required core.AccessRequirement) (*core.Session, error) {
	q.ready <- record

	select {
	case <-q.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return q.Queries.RotateSession(ctx, state, digest, generation, record, required)
}
func rotationService(t *testing.T, q rotationWait, cfg *config.Config) *core.Service {
	t.Helper()

	svc, err := core.NewService(q, cfg, NewPasswordChecker(), credentialAdmissionForTest(t, q.Queries, cfg.CredentialAdmission), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	return svc
}
func waitRotation(t *testing.T, ready <-chan core.SessionRecord) core.SessionRecord {
	t.Helper()

	select {
	case record := <-ready:
		return record
	case <-time.After(5 * time.Second):
		t.Fatal("rotation did not reach storage")

		return core.SessionRecord{}
	}
}

func completedControl(t *testing.T, svc *core.Service, token string) *core.IssuedSession {
	t.Helper()
	outcome, err := svc.Reauthenticate(t.Context(), token, "a distinct safe password", PasswordRequirement())

	issued, ok := outcome.CompletedSession()
	if err != nil || !ok {
		t.Fatalf("reauthenticate: %v", err)
	}

	return issued
}
func sessionCount(t *testing.T, db *sql.DB, subject string) int {
	t.Helper()

	var count int

	err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions WHERE user_id=$1", subject).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}

	return count
}

// Actual password verifier results exercise atomic storage under locks, failures
// and concurrency. Fixtures do not assert or complete an unsupported factor.
func TestControlTransactions(t *testing.T) {
	t.Run("actual rotation and replay", func(t *testing.T) {
		db, _, svc, old := sessionFixture(t)

		var err error

		before := storedSession(t, db, old)
		fresh := completedControl(t, svc, old.Token)

		after := storedSession(t, db, fresh)
		if fresh.Token == old.Token || after.Generation != before.Generation+1 || after.ID != before.ID || !after.CreatedAt.Equal(before.CreatedAt) || !after.ProofVerifiedAt.After(before.ProofVerifiedAt) || !after.AuthenticatedAt.Equal(after.ProofVerifiedAt) {
			t.Fatal("rotation metadata")
		}

		_, err = svc.ValidateSession(t.Context(), old.Token, PasswordRequirement(), core.NoActivity)
		if !errors.Is(err, core.ErrSessionNotFound) {
			t.Fatal("old bearer validates")
		}

		err = svc.Signout(t.Context(), old.Token)
		if !errors.Is(err, core.ErrSessionNotFound) {
			t.Fatal("old signout revoked replacement")
		}

		_, err = svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
		if !errors.Is(err, core.ErrSessionNotFound) {
			t.Fatal("old bearer rotated again")
		}

		_, err = svc.ValidateSession(t.Context(), fresh.Token, PasswordRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal(err)
		}

		if sessionCount(t, db, old.UserID) != 1 {
			t.Fatal("rotation increased retained rows")
		}
	})
	t.Run("competing actual rotations", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)
		gate := rotationWait{Queries: q, ready: make(chan core.SessionRecord, 2), release: make(chan struct{})}
		svc := rotationService(t, gate, cfg)

		_, err := svc.Signup(t.Context(), "race@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		old, err := testSignin(svc, t.Context(), "race@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		type result struct {
			outcome *core.AuthenticationResult
			err     error
		}

		results := make(chan result, 2)

		for range 2 {
			go func() {
				outcome, err := svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
				results <- result{outcome, err}
			}()
		}

		waitRotation(t, gate.ready)
		waitRotation(t, gate.ready)
		close(gate.release)

		completed, rejected := 0, 0

		for range 2 {
			result := <-results
			if issued, ok := result.outcome.CompletedSession(); result.err == nil && ok {
				completed++

				if issued.Generation != 2 {
					t.Fatal("generation")
				}
			} else if result.outcome == nil && (errors.Is(result.err, core.ErrSessionNotFound) || errors.Is(result.err, core.ErrSessionGeneration)) {
				rejected++
			} else {
				t.Fatalf("rotation race: %v", result.err)
			}
		}

		if completed != 1 || rejected != 1 || sessionCount(t, db, old.UserID) != 1 {
			t.Fatal("multiple successful rotations")
		}
	})
	t.Run("rotation commit rollback", func(t *testing.T) {
		db, _, svc, old := sessionFixture(t)
		before := storedSession(t, db, old)
		sessionSQL(t, db, `CREATE FUNCTION reject_rotation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced rotation commit failure'; END $$;
   CREATE CONSTRAINT TRIGGER reject_rotation AFTER UPDATE ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_rotation();`)

		outcome, err := svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
		if err == nil || outcome != nil || !reflect.DeepEqual(before, storedSession(t, db, old)) {
			t.Fatal("partial rotation or issued failed secret")
		}
	})
	t.Run("mutation during verified rotation", func(t *testing.T) {
		for _, mutation := range []string{"signout", "all revocation", "password", "disable cycle"} {
			t.Run(mutation, func(t *testing.T) {
				db, q, cfg := credentialDatabase(t)
				gate := rotationWait{Queries: q, ready: make(chan core.SessionRecord, 1), release: make(chan struct{})}
				svc := rotationService(t, gate, cfg)

				_, err := svc.Signup(t.Context(), "mutation@example.com", "a distinct safe password")
				if err != nil {
					t.Fatal(err)
				}

				old, err := testSignin(svc, t.Context(), "mutation@example.com", "a distinct safe password")
				if err != nil {
					t.Fatal(err)
				}

				result := make(chan error, 1)

				go func() {
					outcome, err := svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
					if outcome != nil {
						err = errors.New("mutated state issued rotation")
					}

					result <- err
				}()

				waitRotation(t, gate.ready)

				if mutation == "all revocation" {
					_, err = svc.RevokeSessions(t.Context(), old.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionAll})
					if err != nil {
						t.Fatal(err)
					}

					close(gate.release)
				} else {
					ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
					defer cancel()

					holder, err := db.BeginTx(ctx, nil)
					if err != nil {
						t.Fatal(err)
					}
					defer holder.Rollback()

					user, err := dal.New(holder).GetUserForAuth(ctx, old.UserID)
					if err != nil {
						t.Fatal(err)
					}

					close(gate.release)
					waitSessionLock(t, ctx, db, holder)

					switch mutation {
					case "signout":
						err = svc.Signout(ctx, old.Token)
					case "password":
						_, err = dal.New(holder).ReplacePassword(ctx, dal.ReplacePasswordParams{ID: user.ID, PasswordHash: user.PasswordHash, UpdatedAt: time.Now()})
						if err == nil {
							err = dal.New(holder).DeleteUserSessions(ctx, user.ID)
						}
					case "disable cycle":
						err = dal.New(holder).UpdateUserActive(ctx, dal.UpdateUserActiveParams{ID: user.ID, Active: false, UpdatedAt: time.Now()})
						if err == nil {
							err = dal.New(holder).UpdateUserActive(ctx, dal.UpdateUserActiveParams{ID: user.ID, Active: true, UpdatedAt: time.Now()})
						}
					}

					if err != nil {
						t.Fatal(err)
					}

					err = holder.Commit()
					if err != nil {
						t.Fatal(err)
					}
				}

				err = <-result
				if !errors.Is(err, core.ErrSessionNotFound) && !errors.Is(err, core.ErrCredentialChanged) {
					t.Fatalf("mutation rotation: %v", err)
				}

				_, err = svc.ValidateSession(t.Context(), old.Token, PasswordRequirement(), core.NoActivity)
				if err == nil {
					t.Fatal("mutation revived old state")
				}
			})
		}
	})
	t.Run("post-lock old expiry", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)
		gate := rotationWait{Queries: q, ready: make(chan core.SessionRecord, 1), release: make(chan struct{})}
		svc := rotationService(t, gate, cfg)

		_, err := svc.Signup(t.Context(), "expiry@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		old, err := testSignin(svc, t.Context(), "expiry@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		sessionSQL(t, db, "WITH security_time AS (SELECT clock_timestamp() AS now) UPDATE sessions SET created_at=security_time.now-interval '10 minutes',proof_verified_at=security_time.now-interval '9 minutes',authenticated_at=security_time.now-interval '8 minutes',last_activity_at=security_time.now-interval '239 seconds',inactivity_us=240000000 FROM security_time WHERE id=$1", old.ID)
		before := storedSession(t, db, old)
		result := make(chan error, 1)

		go func() {
			outcome, err := svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", PasswordRequirement())
			if outcome != nil {
				err = errors.New("expired state issued rotation")
			}

			result <- err
		}()

		waitRotation(t, gate.ready)

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

		close(gate.release)
		waitSessionLock(t, ctx, db, holder)
		waitProofExpiry(t, ctx, db, before.LastActivityAt.Add(4*time.Minute))

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrSessionExpired) {
			t.Fatalf("lock wait expiry: %v", err)
		}

		if !reflect.DeepEqual(before, storedSession(t, db, old)) {
			t.Fatal("expired row renewed")
		}
	})

	t.Run("post-lock replacement proof", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)
		gate := rotationWait{Queries: q, ready: make(chan core.SessionRecord, 1), release: make(chan struct{})}
		svc := rotationService(t, gate, cfg)

		_, err := svc.Signup(t.Context(), "freshness@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		old, err := testSignin(svc, t.Context(), "freshness@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		before := storedSession(t, db, old)
		required := PasswordRequirement()
		required.MaxAge = time.Second
		result := make(chan error, 1)

		go func() {
			outcome, err := svc.Reauthenticate(t.Context(), old.Token, "a distinct safe password", required)
			if outcome != nil {
				err = errors.New("stale proof issued rotation")
			}

			result <- err
		}()

		record := waitRotation(t, gate.ready)

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

		close(gate.release)
		waitSessionLock(t, ctx, db, holder)
		waitProofExpiry(t, ctx, db, record.Proof.VerifiedAt.Add(time.Second))

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		err = <-result
		if !errors.Is(err, core.ErrSessionProofExpired) {
			t.Fatalf("replacement proof: %v", err)
		}

		if !reflect.DeepEqual(before, storedSession(t, db, old)) {
			t.Fatal("stale proof changed session")
		}
	})
	t.Run("management freshness after locks", func(t *testing.T) {
		for _, operation := range []string{"list", "revoke"} {
			t.Run(operation, func(t *testing.T) {
				db, _, svc, actor := sessionFixture(t)
				sessionSQL(t, db, "WITH security_time AS (SELECT clock_timestamp() AS now) UPDATE sessions SET created_at=security_time.now-interval '2 minutes',proof_verified_at=security_time.now-interval '60.5 seconds',authenticated_at=security_time.now-interval '60 seconds' FROM security_time WHERE id=$1", actor.ID)
				before := storedSession(t, db, actor)
				required := PasswordRequirement()
				required.MaxAge = 61 * time.Second

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
					var err error

					if operation == "list" {
						page, e := svc.ListSessions(ctx, actor.Token, required, "")

						err = e
						if page != nil {
							err = errors.New("stale list result")
						}
					} else {
						count, e := svc.RevokeSessions(ctx, actor.Token, required, core.SessionSelection{Scope: core.SessionAll})

						err = e
						if count != 0 {
							err = errors.New("stale revocation result")
						}
					}

					result <- err
				}()

				waitSessionLock(t, ctx, db, holder)
				waitProofExpiry(t, ctx, db, before.ProofVerifiedAt.Add(required.MaxAge))

				err = holder.Commit()
				if err != nil {
					t.Fatal(err)
				}

				err = <-result
				if !errors.Is(err, core.ErrSessionProofExpired) {
					t.Fatalf("management freshness: %v", err)
				}

				if !reflect.DeepEqual(before, storedSession(t, db, actor)) {
					t.Fatal("stale management changed session")
				}
			})
		}
	})

	t.Run("touch waiting on committed rotation", func(t *testing.T) {
		db, _, svc, old := sessionFixture(t)
		sessionSQL(t, db, `CREATE FUNCTION pause_rotation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
    IF NEW.generation > OLD.generation THEN PERFORM pg_advisory_xact_lock(170803); END IF;
    RETURN NEW; END $$;
    CREATE CONSTRAINT TRIGGER pause_rotation AFTER UPDATE ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION pause_rotation();`)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = holder.ExecContext(ctx, "SELECT pg_advisory_xact_lock(170803)")
		if err != nil {
			t.Fatal(err)
		}

		type rotatedResult struct {
			outcome *core.AuthenticationResult
			err     error
		}

		rotation := make(chan rotatedResult, 1)

		go func() {
			outcome, err := svc.Reauthenticate(ctx, old.Token, "a distinct safe password", PasswordRequirement())
			rotation <- rotatedResult{outcome, err}
		}()

		waitSessionLock(t, ctx, db, holder)

		touched := make(chan error, 1)

		go func() {
			validated, err := svc.ValidateSession(ctx, old.Token, PasswordRequirement(), core.RelevantActivity)
			if validated != nil {
				err = errors.New("superseded state authorized touch")
			}

			touched <- err
		}()

		var pid int

		err = holder.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&pid)
		if err != nil {
			t.Fatal(err)
		}

		for {
			var waiting bool

			err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity waiter WHERE EXISTS(
    SELECT 1 FROM pg_stat_activity rotating WHERE rotating.pid=ANY(pg_blocking_pids(waiter.pid))
    AND $1::integer=ANY(pg_blocking_pids(rotating.pid))))`, pid).Scan(&waiting)
			if err != nil {
				t.Fatal(err)
			}

			if waiting {
				break
			}

			select {
			case <-ctx.Done():
				t.Fatal("touch did not wait behind rotation")
			case <-time.After(5 * time.Millisecond):
			}
		}

		err = holder.Commit()
		if err != nil {
			t.Fatal(err)
		}

		result := <-rotation

		fresh, ok := result.outcome.CompletedSession()
		if result.err != nil || !ok {
			t.Fatalf("rotation commit: %v", result.err)
		}

		err = <-touched
		if !errors.Is(err, core.ErrSessionNotFound) {
			t.Fatalf("touch after rotation: %v", err)
		}

		stored := storedSession(t, db, fresh)
		if stored.Generation != 2 || !stored.LastActivityAt.Equal(fresh.LastActivityAt) {
			t.Fatal("stale touch changed replacement")
		}
	})
	t.Run("bounded owned pages and revocation", func(t *testing.T) {
		db, q, svc, actor := sessionFixture(t)
		for range 3 {
			_, err := testSignin(svc, t.Context(), "session@example.com", "a distinct safe password")
			if err != nil {
				t.Fatal(err)
			}
		}

		_, err := svc.Signup(t.Context(), "other@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		foreign, err := testSignin(svc, t.Context(), "other@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		digest, _ := core.ParseSessionToken(actor.Token)
		requirement := PasswordRequirement()
		requirement.MaxAge = time.Minute
		seen := map[string]bool{}

		cursor := ""
		for {
			page, err := q.ListSessions(t.Context(), digest, requirement, 1, cursor)
			if err != nil || len(page.Sessions) != 1 || page.CurrentID != actor.ID {
				t.Fatalf("page: %v", err)
			}

			row := page.Sessions[0]
			if row.UserID != actor.UserID || seen[row.ID] {
				t.Fatal("cross-subject or duplicate page")
			}

			seen[row.ID] = true

			if page.NextCursor == "" {
				break
			}

			cursor = page.NextCursor
		}

		if len(seen) != 4 {
			t.Fatal("missing sessions")
		}

		count, err := svc.RevokeSessions(t.Context(), actor.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionSelected, ID: foreign.ID})
		if count != 0 || !errors.Is(err, core.ErrSessionNotFound) || sessionCount(t, db, foreign.UserID) != 1 {
			t.Fatal("cross-subject deletion")
		}

		count, err = svc.RevokeSessions(t.Context(), actor.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionOthers})
		if err != nil || count != 3 || sessionCount(t, db, actor.UserID) != 1 {
			t.Fatal("others scope")
		}

		count, err = svc.RevokeSessions(t.Context(), actor.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionCurrent})
		if err != nil || count != 1 || sessionCount(t, db, actor.UserID) != 0 || sessionCount(t, db, foreign.UserID) != 1 {
			t.Fatal("current scope")
		}
	})
	t.Run("recent proof and delete rollback", func(t *testing.T) {
		db, _, svc, actor := sessionFixture(t)

		_, err := testSignin(svc, t.Context(), "session@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		sessionSQL(t, db, "WITH security_time AS (SELECT clock_timestamp() AS now) UPDATE sessions SET created_at=security_time.now-interval '10 minutes',proof_verified_at=security_time.now-interval '10 minutes',authenticated_at=security_time.now-interval '10 minutes' FROM security_time WHERE id=$1", actor.ID)

		_, err = svc.ListSessions(t.Context(), actor.Token, PasswordRequirement(), "")
		if !errors.Is(err, core.ErrSessionProofExpired) {
			t.Fatal("stale list proof")
		}

		count, err := svc.RevokeSessions(t.Context(), actor.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionAll})
		if count != 0 || !errors.Is(err, core.ErrSessionProofExpired) || sessionCount(t, db, actor.UserID) != 2 {
			t.Fatal("stale deletion")
		}

		fresh := completedControl(t, svc, actor.Token)
		sessionSQL(t, db, `CREATE FUNCTION reject_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced delete commit failure'; END $$;
   CREATE CONSTRAINT TRIGGER reject_delete AFTER DELETE ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_delete();`)

		count, err = svc.RevokeSessions(t.Context(), fresh.Token, PasswordRequirement(), core.SessionSelection{Scope: core.SessionAll})
		if count != 0 || err == nil || sessionCount(t, db, actor.UserID) != 2 {
			t.Fatal("partial delete or false success")
		}
	})
	t.Run("concurrent retained admission", func(t *testing.T) {
		db, q, _, issued := sessionFixture(t)
		digest, _ := core.ParseSessionToken(issued.Token)
		state := core.CredentialState{UserID: issued.UserID, Version: issued.AuthVersion}
		start := make(chan struct{})
		results := make(chan error, 12)

		var workers sync.WaitGroup
		for range 12 {
			workers.Go(func() {
				token, err := crypto.GenerateSecureToken(32)
				if err != nil {
					results <- err

					return
				}

				next, err := core.ParseSessionToken(token)
				if err != nil {
					results <- err

					return
				}

				record := core.SessionRecord{Session: issued.Session, Digest: next}
				record.ID = model.NewID()

				<-start

				_, err = q.CreateSession(t.Context(), state, record, PasswordRequirement(), 3)
				results <- err
			})
		}

		close(start)
		workers.Wait()
		close(results)

		admitted, denied := 0, 0

		for err := range results {
			if err == nil {
				admitted++
			} else if errors.Is(err, core.ErrSessionCapacity) {
				denied++
			} else {
				t.Fatal(err)
			}
		}

		if admitted != 2 || denied != 10 || sessionCount(t, db, issued.UserID) != 3 {
			t.Fatal("admission cap exceeded")
		}

		_, err := q.ValidateSession(t.Context(), digest, PasswordRequirement(), core.NoActivity, time.Minute)
		if err != nil {
			t.Fatal("capacity evicted actor")
		}
	})
	t.Run("expired reclamation and insert rollback", func(t *testing.T) {
		db, _, svc, old := sessionFixture(t)
		sessionSQL(t, db, "WITH security_time AS (SELECT clock_timestamp() AS now) UPDATE sessions SET created_at=security_time.now-interval '2 hours',proof_verified_at=security_time.now-interval '2 hours',authenticated_at=security_time.now-interval '2 hours',last_activity_at=security_time.now-interval '2 hours',expires_at=security_time.now-interval '1 hour' FROM security_time WHERE id=$1", old.ID)
		sessionSQL(t, db, `CREATE FUNCTION reject_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced insert failure'; END $$;
   CREATE CONSTRAINT TRIGGER reject_insert AFTER INSERT ON sessions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_insert();`)

		outcome, err := svc.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if outcome != nil || err == nil || sessionCount(t, db, old.UserID) != 1 {
			t.Fatal("partial reclamation or issuance")
		}

		sessionSQL(t, db, "DROP TRIGGER reject_insert ON sessions")

		_, err = testSignin(svc, t.Context(), "session@example.com", "a distinct safe password")
		if err != nil {
			t.Fatal(err)
		}

		if sessionCount(t, db, old.UserID) != 1 {
			t.Fatal("expired retained row not reclaimed")
		}
	})
}
