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
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

type credentialDB struct{ db *sql.DB }

func (p credentialDB) GetDB() *sql.DB { return p.db }

type changingQueries struct {
	*Queries
	change func(context.Context, core.CredentialState) error
}

func (q changingQueries) CreateSession(ctx context.Context, state core.CredentialState, session core.SessionRecord, requirement core.AccessRequirement, limit int) (*core.Session, error) {
	err := q.change(ctx, state)
	if err != nil {
		return nil, err
	}

	return q.Queries.CreateSession(ctx, state, session, requirement, limit)
}

func credentialDatabase(t *testing.T) (*sql.DB, *Queries, *config.Config) {
	t.Helper()

	host := os.Getenv("DB_HOST")
	if host == "" {
		t.Fatal("DB_HOST is required for credential integration tests")
	}

	cfg := config.New()
	cfg.Database.Host = host

	cfg.Database.User = os.Getenv("DB_USER")
	if cfg.Database.User == "" {
		cfg.Database.User = "postgres"
	}

	cfg.Database.Database = os.Getenv("DB_NAME")
	if cfg.Database.Database == "" {
		cfg.Database.Database = "postgres"
	}

	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	if port := os.Getenv("DB_PORT"); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil {
			t.Fatal(err)
		}

		cfg.Database.Port = number
	}

	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1

	root, err := sql.Open("pgx", cfg.Database.ConnectionString())
	if err != nil {
		t.Fatal(err)
	}

	schema := "credential_" + strings.ReplaceAll(model.NewID(), "-", "")

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

	for _, name := range []string{"001-users.sql", "004-authenticators.sql", "005-webauthn-completion.sql", "006-fallback-proof.sql"} {
		migration, readErr := os.ReadFile("../../../assets/migration/postgres/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}

		up := strings.Split(string(migration), "-- +migrate Down")[0]

		_, err = db.ExecContext(t.Context(), up)
		if err != nil {
			t.Fatal(err)
		}
	}

	q := NewQueries(credentialDB{db})

	err = q.Start(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return db, q, cfg
}

// Real PostgreSQL transactions establish persistent uniqueness, version-based
// stale-state rejection, mutation/session serialization and complete rollback.
func TestCredentialTransactions(t *testing.T) {
	t.Run("simultaneous Unicode signup", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)

		svc, err := core.NewService(q, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		password := strings.Repeat("😀", 64)
		start := make(chan struct{})
		failures := make(chan error, 2)

		var workers sync.WaitGroup
		for range 2 {
			workers.Go(func() {
				<-start

				_, signupErr := svc.Signup(t.Context(), "unicode@example.com", password)
				failures <- signupErr
			})
		}

		close(start)
		workers.Wait()
		close(failures)

		successes, conflicts := 0, 0

		for signupErr := range failures {
			switch {
			case signupErr == nil:
				successes++
			case errors.Is(signupErr, core.ErrEmailTaken):
				conflicts++
			default:
				t.Fatalf("unclassified signup result: %v", signupErr)
			}
		}

		var count int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM users").Scan(&count)
		if err != nil || count != 1 || successes != 1 || conflicts != 1 {
			t.Fatalf("uniqueness: users %d, success %d, conflicts %d, error %v", count, successes, conflicts, err)
		}

		session, err := testSignin(svc, t.Context(), "unicode@example.com", password)
		if err != nil || session == nil {
			t.Fatalf("Unicode sign-in failed: %v", err)
		}
	})
	t.Run("conditional replacement", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)

		svc, err := core.NewService(q, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		user, err := svc.Signup(t.Context(), "replace@example.com", "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		_, err = testSignin(svc, t.Context(), user.Email, "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		settings, err := cfg.Auth.PasswordSettings()
		if err != nil {
			t.Fatal(err)
		}

		verifier, err := model.NewPasswordVerifier(settings.Verifier)
		if err != nil {
			t.Fatal(err)
		}

		records := make([]string, 2)
		for i := range records {
			records[i], err = verifier.Hash(t.Context(), fmt.Sprintf("replacement phrase %d", i))
			if err != nil {
				t.Fatal(err)
			}
		}

		state := core.CredentialState{UserID: user.ID, Version: user.AuthVersion}
		start := make(chan struct{})
		failures := make(chan error, 2)

		var workers sync.WaitGroup
		for i := range records {
			workers.Go(func() {
				<-start

				_, replaceErr := q.ReplacePassword(t.Context(), state, records[i], time.Now())
				failures <- replaceErr
			})
		}

		close(start)
		workers.Wait()
		close(failures)

		successes, stale := 0, 0

		for replaceErr := range failures {
			switch {
			case replaceErr == nil:
				successes++
			case errors.Is(replaceErr, core.ErrCredentialChanged):
				stale++
			default:
				t.Fatal(replaceErr)
			}
		}

		current, err := q.GetUserByID(t.Context(), user.ID)
		if err != nil {
			t.Fatal(err)
		}

		var sessions int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil || successes != 1 || stale != 1 || current.AuthVersion != state.Version+1 || sessions != 0 {
			t.Fatalf("conditional replacement failed: success %d, stale %d, version %d, sessions %d, error %v", successes, stale, current.AuthVersion, sessions, err)
		}

		if current.PasswordHash != records[0] && current.PasswordHash != records[1] {
			t.Fatal("partial credential stored")
		}
	})
	t.Run("mutation before session commit", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)

		svc, err := core.NewService(q, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		user, err := svc.Signup(t.Context(), "stale@example.com", "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		settings, err := cfg.Auth.PasswordSettings()
		if err != nil {
			t.Fatal(err)
		}

		verifier, err := model.NewPasswordVerifier(settings.Verifier)
		if err != nil {
			t.Fatal(err)
		}

		replacement, err := verifier.Hash(t.Context(), "replacement safe password")
		if err != nil {
			t.Fatal(err)
		}

		changing := changingQueries{Queries: q, change: func(ctx context.Context, state core.CredentialState) error {
			_, changeErr := q.ReplacePassword(ctx, state, replacement, time.Now())

			return changeErr
		}}

		staleSvc, err := core.NewService(changing, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		session, err := testSignin(staleSvc, t.Context(), user.Email, "original safe password")
		if session != nil || !errors.Is(err, core.ErrCredentialChanged) {
			t.Fatalf("stale proof issued a session: %v", err)
		}

		var sessions int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil || sessions != 0 {
			t.Fatalf("stale proof persisted a session: %v", err)
		}
	})
	t.Run("row lock and activation cycle", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)

		svc, err := core.NewService(q, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		user, err := svc.Signup(t.Context(), "cycle@example.com", "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		holder, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer holder.Rollback()

		_, err = dal.New(holder).GetUserForAuth(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}

		state := core.CredentialState{UserID: user.ID, Version: user.AuthVersion}
		result := make(chan error, 1)

		go func() {
			_, sessionErr := q.CreateSession(ctx, state, core.SessionRecord{Session: core.Session{ID: model.NewID(), UserID: user.ID, AuthVersion: state.Version, PolicyRevision: PasswordRequirement().Revision, Generation: 1, Proof: core.VerifiedProof{Method: core.PasswordProof, VerifiedAt: time.Now().Add(-1500 * time.Millisecond)}, AuthenticatedAt: time.Now().Add(-time.Second), CreatedAt: time.Now().Add(-2 * time.Second), LastActivityAt: time.Now().Add(-time.Second), ExpiresAt: time.Now().Add(time.Hour), InactivityTTL: time.Minute}}, PasswordRequirement(), 10)
			result <- sessionErr
		}()

		for {
			var waiting int

			err = db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%FOR UPDATE%' AND datname=current_database()").Scan(&waiting)
			if err != nil {
				t.Fatal(err)
			}

			if waiting > 0 {
				break
			}

			select {
			case <-ctx.Done():
				t.Fatal("session insertion did not wait on user lock")
			case <-time.After(5 * time.Millisecond):
			}
		}

		for _, active := range []bool{false, true} {
			_, err = holder.ExecContext(ctx, "UPDATE users SET active=$2, auth_version=auth_version+1 WHERE id=$1", user.ID, active)
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
			t.Fatalf("activation cycle accepted stale version: %v", err)
		}

		current, err := q.GetUserByID(ctx, user.ID)
		if err != nil || !current.Active || current.AuthVersion != user.AuthVersion+2 {
			t.Fatalf("activation cycle state: %v", err)
		}
	})
	t.Run("replacement rollback", func(t *testing.T) {
		db, q, cfg := credentialDatabase(t)

		svc, err := core.NewService(q, cfg, NewPasswordChecker(), log.NewTestLogger("error"))
		if err != nil {
			t.Fatal(err)
		}

		user, err := svc.Signup(t.Context(), "rollback@example.com", "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		_, err = testSignin(svc, t.Context(), user.Email, "original safe password")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.ExecContext(t.Context(), "CREATE FUNCTION reject_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced cleanup failure'; END $$; CREATE TRIGGER reject_cleanup BEFORE DELETE ON sessions FOR EACH ROW EXECUTE FUNCTION reject_cleanup()")
		if err != nil {
			t.Fatal(err)
		}

		state := core.CredentialState{UserID: user.ID, Version: user.AuthVersion}

		settings, err := cfg.Auth.PasswordSettings()
		if err != nil {
			t.Fatal(err)
		}

		verifier, err := model.NewPasswordVerifier(settings.Verifier)
		if err != nil {
			t.Fatal(err)
		}

		replacement, err := verifier.Hash(t.Context(), "replacement safe password")
		if err != nil {
			t.Fatal(err)
		}

		updated, err := q.ReplacePassword(t.Context(), state, replacement, time.Now())
		if updated != nil || err == nil {
			t.Fatal("forced failure did not abort replacement")
		}

		current, err := q.GetUserByID(t.Context(), user.ID)
		if err != nil {
			t.Fatal(err)
		}

		var sessions int

		err = db.QueryRowContext(t.Context(), "SELECT count(*) FROM sessions").Scan(&sessions)
		if err != nil || current.AuthVersion != user.AuthVersion || current.PasswordHash != user.PasswordHash || sessions != 1 {
			t.Fatalf("partial durable replacement: version %d, sessions %d, error %v", current.AuthVersion, sessions, err)
		}
	})
}

func testSignin(svc *core.Service, ctx context.Context, email, password string) (*core.IssuedSession, error) {
	result, err := svc.Signin(ctx, email, password, PasswordRequirement())
	if err != nil {
		return nil, err
	}

	issued, ok := result.CompletedSession()
	if !ok {
		return nil, fmt.Errorf("unexpected test authentication outcome")
	}

	return issued, nil
}
