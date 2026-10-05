//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

type admissionFixture struct {
	db      *sql.DB
	q       *Queries
	key     []byte
	store   *CredentialAdmissionStore
	svc     *core.CredentialAdmission
	options config.CredentialAdmissionConfig
}

func newAdmissionFixture(t *testing.T, cfg config.CredentialAdmissionConfig) admissionFixture {
	t.Helper()
	for _, name := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME"} {
		if os.Getenv(name) == "" {
			t.Fatalf("%s is required for credential admission transactions", name)
		}
	}
	if _, ok := os.LookupEnv("DB_PASSWORD"); !ok {
		t.Fatal("DB_PASSWORD must be explicit and may be empty")
	}
	db, q, _ := credentialDatabase(t)
	data, err := os.ReadFile("../../../assets/migration/postgres/009-credential-admission.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(t.Context(), strings.Split(string(data), "-- +migrate Down")[0])
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewCredentialAdmissionStore(q, "test-app", key)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := core.NewCredentialAdmission(store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return admissionFixture{db, q, key, store, svc, cfg}
}
func admissionExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
}
func admissionRows(t *testing.T, f admissionFixture, want int) {
	t.Helper()
	if mailboxCount(t, f.db, "SELECT count(*) FROM credential_admissions") != want || mailboxCount(t, f.db, "SELECT coalesce(sum(records),0) FROM credential_admission_capacity") != want {
		t.Fatal("capacity drift")
	}
}
func admitFixture(t *testing.T, f admissionFixture, id string, purpose core.CredentialAdmissionPurpose) {
	t.Helper()
	err := f.svc.Admit(t.Context(), id, purpose)
	if err != nil {
		t.Fatal(err)
	}
}

// Actual shared transactions prove finite admission, private storage, rollback,
// trusted time, independent purpose and bounded cleanup without fake identities.
func TestCredentialAdmissionTransactions(t *testing.T) {
	t.Run("shared last admission", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		second, err := NewCredentialAdmissionStore(f.q, "test-app", f.key)
		if err != nil {
			t.Fatal(err)
		}
		other, err := core.NewCredentialAdmission(second, f.options)
		if err != nil {
			t.Fatal(err)
		}
		// Both adapters own a key copy; later caller mutation must not split counters.
		f.key[0] ^= 255
		var allowed, denied atomic.Int32
		start := make(chan struct{})
		failures := make(chan error, 24)
		var wg sync.WaitGroup
		for i := range 24 {
			wg.Go(func() {
				<-start
				service := f.svc
				if i%2 == 1 {
					service = other
				}
				err := service.Admit(t.Context(), "unknown@example.com", core.CredentialPasswordProof)
				switch {
				case err == nil:
					allowed.Add(1)
				case errors.Is(err, core.ErrCredentialAdmissionAttempts):
					denied.Add(1)
				default:
					failures <- err
				}
			})
		}
		close(start)
		wg.Wait()
		close(failures)
		for err := range failures {
			t.Fatal(err)
		}
		if allowed.Load() != 10 || denied.Load() != 14 {
			t.Fatalf("allowed %d denied %d", allowed.Load(), denied.Load())
		}
		admissionRows(t, f, 1)
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 10 || mailboxCount(t, f.db, "SELECT count(*) FROM users") != 0 {
			t.Fatal("invalid accounting or invented user")
		}
	})
	t.Run("purpose and restart", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{PasswordAttempts: 1})
		admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		admitFixture(t, f, "a@example.com", core.CredentialRegistration)
		restarted, err := NewCredentialAdmissionStore(f.q, "test-app", f.key)
		if err != nil {
			t.Fatal(err)
		}
		svc, err := core.NewCredentialAdmission(restarted, config.CredentialAdmissionConfig{PasswordAttempts: 20})
		if err != nil {
			t.Fatal(err)
		}
		err = svc.Admit(t.Context(), "a@example.com", core.CredentialPasswordProof)
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatalf("restart/refund: %v", err)
		}
		admissionRows(t, f, 2)
		wrong := append([]byte(nil), f.key...)
		wrong[0] ^= 255
		changed, err := NewCredentialAdmissionStore(f.q, "test-app", wrong)
		if err != nil {
			t.Fatal(err)
		}
		settings, err := f.options.CredentialAdmissionSettings()
		if err != nil {
			t.Fatal(err)
		}
		err = changed.ChargeCredentialAdmission(t.Context(), "fresh@example.com", core.CredentialPasswordProof, settings)
		if !errors.Is(err, core.ErrCredentialAdmissionState) {
			t.Fatalf("key reset admitted: %v", err)
		}
		_, err = changed.CleanupCredentialAdmission(t.Context(), settings)
		if !errors.Is(err, core.ErrCredentialAdmissionState) {
			t.Fatal("foreign key cleaned state")
		}
		admissionRows(t, f, 2)
	})
	t.Run("tightening preserves count", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		for range 3 {
			admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		}
		strict, err := core.NewCredentialAdmission(f.store, config.CredentialAdmissionConfig{PasswordAttempts: 2})
		if err != nil {
			t.Fatal(err)
		}
		for range 2 {
			err = strict.Admit(t.Context(), "a@example.com", core.CredentialPasswordProof)
			if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
				t.Fatal("tightened window admitted")
			}
		}
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 3 || mailboxCount(t, f.db, "SELECT count(*) FROM credential_admissions WHERE cooldown_until IS NOT NULL") != 0 {
			t.Fatal("denial changed accounting")
		}
	})
	t.Run("trusted retirement", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{PasswordAttempts: 1})
		admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		admissionExec(t, f.db, `WITH n AS (SELECT clock_timestamp() AS t) UPDATE credential_admissions SET window_start=n.t-interval '15 minutes',window_end=n.t-interval '5 minutes',cooldown_until=n.t+interval '5 minutes' FROM n`)
		_, err := f.svc.Cleanup(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		err = f.svc.Admit(t.Context(), "a@example.com", core.CredentialPasswordProof)
		if !errors.Is(err, core.ErrCredentialAdmissionAttempts) {
			t.Fatalf("future cooldown: %v", err)
		}
		admissionRows(t, f, 1)
		admissionExec(t, f.db, `WITH n AS (SELECT clock_timestamp() AS t) UPDATE credential_admissions SET window_start=n.t-interval '20 minutes',window_end=n.t-interval '10 minutes',cooldown_until=n.t FROM n`)
		admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 1 {
			t.Fatal("expired window not renewed")
		}
	})
	t.Run("finite capacity", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{MaxIdentities: 100, Timeout: "5s"})
		for i := range 100 {
			admitFixture(t, f, fmt.Sprintf("missing-%d@example.com", i), core.CredentialPasswordProof)
		}
		admissionRows(t, f, 100)
		err := f.svc.Admit(t.Context(), "new@example.com", core.CredentialPasswordProof)
		if !errors.Is(err, core.ErrCredentialAdmissionCapacity) {
			t.Fatalf("capacity: %v", err)
		}
		admitFixture(t, f, "missing-0@example.com", core.CredentialPasswordProof)
		admissionRows(t, f, 100)
		if mailboxCount(t, f.db, "SELECT count(*) FROM users") != 0 {
			t.Fatal("capacity manufactured subjects")
		}
	})
	t.Run("bounded cleanup", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{CleanupBatch: 1})
		for _, id := range []string{"old-a@example.com", "old-b@example.com", "live@example.com"} {
			admitFixture(t, f, id, core.CredentialPasswordProof)
		}
		admissionExec(t, f.db, `WITH n AS (SELECT clock_timestamp() AS t) UPDATE credential_admissions SET window_start=n.t-interval '20 minutes',window_end=n.t-interval '10 minutes',cooldown_until=NULL FROM n WHERE identity_key<>$1`, f.store.digest("identity", "live@example.com"))
		for _, want := range []int{2, 1} {
			count, err := f.svc.Cleanup(t.Context())
			if err != nil || count != 1 {
				t.Fatalf("cleanup %d: %v", count, err)
			}
			admissionRows(t, f, want)
		}
		admitFixture(t, f, "live@example.com", core.CredentialPasswordProof)
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 2 {
			t.Fatal("live window erased")
		}
	})
	t.Run("rollback after row write", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		admitFixture(t, f, "existing@example.com", core.CredentialPasswordProof)
		admissionExec(t, f.db, `CREATE FUNCTION fail_admission() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'owned fault'; END $$; CREATE TRIGGER fail_admission BEFORE UPDATE ON credential_admission_capacity FOR EACH ROW EXECUTE FUNCTION fail_admission()`)
		for _, id := range []string{"existing@example.com", "new@example.com"} {
			err := f.svc.Admit(t.Context(), id, core.CredentialPasswordProof)
			if err == nil || errors.Is(err, core.ErrCredentialAdmissionAttempts) {
				t.Fatal("fault was accepted/classified as expected denial")
			}
		}
		admissionRows(t, f, 1)
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 1 {
			t.Fatal("partial charge persisted")
		}
	})
	t.Run("caller lock deadline", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		admitFixture(t, f, "existing@example.com", core.CredentialPasswordProof)
		tx, err := f.db.BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		_, err = tx.ExecContext(t.Context(), "SELECT namespace FROM credential_admission_capacity FOR UPDATE")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
		defer cancel()
		started := time.Now()
		err = f.svc.Admit(ctx, "new@example.com", core.CredentialPasswordProof)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
			t.Fatalf("lock deadline: %v", err)
		}
		admissionRows(t, f, 1)
	})
	t.Run("cleanup renewal race", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		admissionExec(t, f.db, `WITH n AS (SELECT clock_timestamp() AS t) UPDATE credential_admissions SET window_start=n.t-interval '20 minutes',window_end=n.t-interval '10 minutes',cooldown_until=NULL FROM n`)
		failures := make(chan error, 2)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() { <-start; failures <- f.svc.Admit(t.Context(), "a@example.com", core.CredentialPasswordProof) })
		wg.Go(func() { <-start; _, err := f.svc.Cleanup(t.Context()); failures <- err })
		close(start)
		wg.Wait()
		close(failures)
		for err := range failures {
			if err != nil {
				t.Fatal(err)
			}
		}
		admissionRows(t, f, 1)
		if mailboxCount(t, f.db, "SELECT attempts FROM credential_admissions") != 1 {
			t.Fatal("renewed charge lost")
		}
	})
	t.Run("cleanup rollback", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		admitFixture(t, f, "a@example.com", core.CredentialPasswordProof)
		admissionExec(t, f.db, `WITH n AS (SELECT clock_timestamp() AS t) UPDATE credential_admissions SET window_start=n.t-interval '20 minutes',window_end=n.t-interval '10 minutes',cooldown_until=NULL FROM n`)
		admissionExec(t, f.db, `CREATE FUNCTION fail_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'owned fault'; END $$; CREATE TRIGGER fail_cleanup BEFORE UPDATE ON credential_admission_capacity FOR EACH ROW EXECUTE FUNCTION fail_cleanup()`)
		count, err := f.svc.Cleanup(t.Context())
		if err == nil || count != 0 {
			t.Fatal("failed cleanup accepted")
		}
		admissionRows(t, f, 1)
	})
	t.Run("structural denial", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		for _, id := range []string{"", string([]byte{255}), "a\n@example.com", strings.Repeat("x", 255)} {
			err := f.svc.Admit(t.Context(), id, core.CredentialPasswordProof)
			if !errors.Is(err, core.ErrCredentialIdentity) {
				t.Fatal("invalid identity admitted")
			}
		}
		admissionRows(t, f, 0)
	})
	t.Run("storage measurement", func(t *testing.T) {
		f := newAdmissionFixture(t, config.CredentialAdmissionConfig{})
		started := time.Now()
		for i := range 1000 {
			admitFixture(t, f, fmt.Sprintf("measure-%d@example.com", i), core.CredentialPasswordProof)
		}
		duration := time.Since(started)
		admissionRows(t, f, 1000)
		var rows, indexes int64
		err := f.db.QueryRowContext(t.Context(), "SELECT pg_relation_size('credential_admissions'),pg_indexes_size('credential_admissions')").Scan(&rows, &indexes)
		if err != nil || rows <= 0 || indexes <= 0 {
			t.Fatalf("storage measurement: %v", err)
		}
		t.Logf("fixture: 1000 admissions in %s; heap=%d bytes, indexes=%d bytes", duration, rows, indexes)
	})
}
