// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Real PostgreSQL claims protect one slot across replicas, process death,
// configuration changes, stale completions and terminal schedule advancement.
func TestLeasedRuns(t *testing.T) {
	if schema := os.Getenv("SCHEDULER_CRASH_SCHEMA"); schema != "" {
		leasedCrashWorker(t, schema)

		return
	}

	t.Run("replicas and fencing", leasedReplicas)
	t.Run("killed worker", leasedInterruption)
	t.Run("retry and exhaustion", leasedExhaustion)
	t.Run("safe diagnostics", leasedDiagnostics)
	t.Run("atomic terminal write", leasedAtomic)
	t.Run("cooperative shutdown", leasedShutdown)
}

func leasedFixture(t *testing.T, spec string) (*Store, *sql.DB, time.Time) {
	t.Helper()

	slot := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	_, db := slotStore(t, "postgres", scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: slot}, spec)

	return NewStore(db), db, slot
}

func expireLease(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`UPDATE job_runs SET lease_expires_at=clock_timestamp()-interval '1 second' WHERE status='running'`)
	if err != nil {
		t.Fatal(err)
	}
}

func leasedReplicas(t *testing.T) {
	store, db, slot := leasedFixture(t, "")
	ctx := context.Background()
	policy := scheduler.DefaultExecutionPolicy()

	var (
		wins   atomic.Int32
		group  sync.WaitGroup
		winner scheduler.RunClaim
	)

	for range 12 {
		group.Go(func() {
			claim, err := NewStore(db).ClaimRun(ctx, "job", slot, policy)
			if err == nil {
				winner = claim

				wins.Add(1)
			} else if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Error(err)
			}
		})
	}

	group.Wait()

	if wins.Load() != 1 || winner.Attempt != 1 || winner.Policy != policy {
		t.Fatalf("claims=%d attempt=%d policy=%+v", wins.Load(), winner.Attempt, winner.Policy)
	}

	for _, test := range []struct {
		name  string
		write func() error
	}{
		{"running", func() error { return store.MarkRunning(ctx, winner.RunID, time.Now()) }},
		{"retry claim", func() error { return store.ClaimRetry(ctx, winner.RunID, 2, time.Now()) }},
		{"retry result", func() error {
			return store.MarkRetry(ctx, winner.RunID, 8, time.Now(), time.Now().Add(time.Minute), "unsafe")
		}},
		{"success", func() error { return store.MarkSuccess(ctx, winner.RunID, time.Now(), []byte(`{}`)) }},
		{"failure", func() error { return store.MarkFailed(ctx, winner.RunID, time.Now(), "unsafe") }},
		{"schedule", func() error { return store.UpdateNextRun(ctx, "job", slot, slot.Add(time.Hour)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.write()
			if err == nil {
				t.Fatal("unfenced write accepted")
			}
		})
	}

	expireLease(t, db)

	err := store.CompleteRun(ctx, winner, []byte(`{}`), "")
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("expired completion: %v", err)
	}

	changed := policy
	changed.MaxAttempts = 1

	replacement, err := NewStore(db).ClaimRun(ctx, "job", slot, changed)
	if err != nil {
		t.Fatal(err)
	}

	if replacement.RunID != winner.RunID || replacement.Attempt != 2 || replacement.Token == winner.Token || replacement.Policy != policy {
		t.Fatal("replacement changed identity or frozen policy")
	}

	err = store.CompleteRun(ctx, winner, nil, scheduler.FailureHandler)
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("stale completion: %v", err)
	}

	for _, test := range []struct {
		name   string
		change func(*scheduler.RunClaim)
	}{
		{"attempt", func(c *scheduler.RunClaim) { c.Attempt++ }},
		{"slot", func(c *scheduler.RunClaim) { c.ScheduledFor = c.ScheduledFor.Add(time.Second) }},
		{"token", func(c *scheduler.RunClaim) { c.Token = "invalid" }},
		{"job", func(c *scheduler.RunClaim) { c.JobID = "missing" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			bad := replacement
			test.change(&bad)

			err := store.CompleteRun(ctx, bad, []byte(`{}`), "")
			if !errors.Is(err, scheduler.ErrStaleClaim) {
				t.Fatalf("bad claim accepted: %v", err)
			}
		})
	}

	err = store.CompleteRun(ctx, replacement, []byte(`{"ok":true}`), "")
	if err != nil {
		t.Fatal(err)
	}

	err = store.CompleteRun(ctx, replacement, []byte(`{}`), "")
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("repeated completion: %v", err)
	}

	err = store.MarkFailed(ctx, winner.RunID, time.Now(), "unsafe")
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("legacy terminal overwrite: %v", err)
	}

	var (
		status  string
		enabled bool
	)

	err = db.QueryRow(`SELECT r.status,j.enabled FROM job_runs r JOIN scheduled_jobs j ON j.id=r.job_id`).Scan(&status, &enabled)
	if err != nil || status != "success" || enabled {
		t.Fatalf("terminal=%s enabled=%t error=%v", status, enabled, err)
	}
}

func leasedCrashWorker(t *testing.T, schema string) {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), schema)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewStore(db)

	jobs, err := store.ListClaimable(context.Background(), 1)
	if err != nil || len(jobs) != 1 {
		t.Fatal("worker cannot find scheduled slot")
	}

	claim, err := store.ClaimRun(context.Background(), jobs[0].ID, jobs[0].ScheduledFor, scheduler.DefaultExecutionPolicy())
	if err != nil {
		t.Fatal(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(claim)
	if err != nil {
		t.Fatal(err)
	}

	select {}
}

func leasedInterruption(t *testing.T) {
	store, db, slot := leasedFixture(t, "")

	var schema string

	err := db.QueryRow(`SELECT current_schema()`).Scan(&schema)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLeasedRuns$")

	cmd.Env = append(os.Environ(), "SCHEDULER_CRASH_SCHEMA="+schema)

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	var old scheduler.RunClaim

	err = json.NewDecoder(pipe).Decode(&old)
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Process.Kill()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Wait()
	if err == nil {
		t.Fatal("worker unexpectedly exited successfully")
	}

	expireLease(t, db)

	fresh, err := NewStore(db).ClaimRun(context.Background(), "job", slot, scheduler.DefaultExecutionPolicy())
	if err != nil || fresh.RunID != old.RunID || fresh.Attempt != old.Attempt+1 {
		t.Fatalf("restart attempt=%d error=%v", fresh.Attempt, err)
	}

	err = store.CompleteRun(context.Background(), old, []byte(`{}`), "")
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("dead worker completion: %v", err)
	}

	err = store.CompleteRun(context.Background(), fresh, []byte(`{}`), "")
	if err != nil {
		t.Fatal(err)
	}
}

func leasedExhaustion(t *testing.T) {
	for _, recurring := range []bool{false, true} {
		t.Run(fmt.Sprintf("recurring %t", recurring), func(t *testing.T) {
			spec := ""
			if recurring {
				spec = `{"type":"interval","every":"1m"}`
			}

			store, db, slot := leasedFixture(t, spec)
			ctx := context.Background()
			policy := scheduler.DefaultExecutionPolicy()
			policy.MaxAttempts = 2
			policy.RetryDelay = time.Second

			first, err := store.ClaimRun(ctx, "job", slot, policy)
			if err != nil {
				t.Fatal(err)
			}

			err = store.CompleteRun(ctx, first, nil, scheduler.FailureHandler)
			if err != nil {
				t.Fatal(err)
			}

			_, err = store.ClaimRun(ctx, "job", slot, policy)
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("early retry: %v", err)
			}

			_, err = db.Exec(`UPDATE job_runs SET retry_at=clock_timestamp()-interval '1 second'`)
			if err != nil {
				t.Fatal(err)
			}

			changed := policy
			changed.MaxAttempts = 32
			changed.RetryDelay = time.Hour

			second, err := NewStore(db).ClaimRun(ctx, "job", slot, changed)
			if err != nil || second.Attempt != 2 || second.Policy != policy {
				t.Fatalf("retry lost retained policy: %+v %v", second.Policy, err)
			}

			expireLease(t, db)

			for range 2 {
				_, err = store.ClaimRun(ctx, "job", slot, changed)
				if !errors.Is(err, scheduler.ErrRunClaim) {
					t.Fatalf("exhausted claim: %v", err)
				}
			}

			var (
				status, code         string
				attempt              int
				enabled              bool
				next, last, finished time.Time
			)

			err = db.QueryRow(`SELECT r.status,r.failure_code,r.attempt,j.enabled,j.next_run_at,j.last_run_at,r.finished_at
 FROM job_runs r JOIN scheduled_jobs j ON j.id=r.job_id`).Scan(&status, &code, &attempt, &enabled, &next, &last, &finished)
			if err != nil {
				t.Fatal(err)
			}

			if status != "failed" || code != "lease_expired" || attempt != 2 || enabled != recurring || !last.Equal(slot) || (recurring && !next.Equal(finished.Add(time.Minute))) {
				t.Fatalf("terminal=%s code=%s attempt=%d enabled=%t next=%v", status, code, attempt, enabled, next)
			}
		})
	}
}

func leasedDiagnostics(t *testing.T) {
	for _, test := range []struct {
		name    string
		handler scheduler.Handler
		code    string
	}{
		{"error", func(context.Context, scheduler.Job) scheduler.Result {
			return scheduler.Result{Err: errors.New("sensitive provider details")}
		}, "handler_error"},
		{"panic", func(context.Context, scheduler.Job) scheduler.Result { panic("sensitive payload") }, "handler_panic"},
		{"deadline", func(ctx context.Context, _ scheduler.Job) scheduler.Result {
			<-ctx.Done()

			return scheduler.Result{}
		}, "handler_deadline"},
		{"large output", func(context.Context, scheduler.Job) scheduler.Result {
			return scheduler.Result{Output: map[string]any{"data": strings.Repeat("x", 8192)}}
		}, "invalid_output"},
		{"invalid output", func(context.Context, scheduler.Job) scheduler.Result {
			return scheduler.Result{Output: map[string]any{"data": make(chan int)}}
		}, "invalid_output"},
		{"unknown", nil, "unknown_task"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, db, _ := leasedFixture(t, "")
			policy := scheduler.DefaultExecutionPolicy()
			policy.MaxAttempts = 1
			policy.HandlerTimeout = time.Second

			runner := scheduler.New(store, scheduler.Config{}, &scheduler.FakeLogger{}, scheduler.WithDurableExecution(policy))
			if test.handler != nil {
				runner.Register("task", test.handler)
			}

			err := runner.Tick(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			var (
				status, code  string
				detail, token sql.NullString
				output        []byte
			)

			err = db.QueryRow(`SELECT status,failure_code,error,lease_token,output FROM job_runs`).Scan(&status, &code, &detail, &token, &output)
			if err != nil {
				t.Fatal(err)
			}

			if status != "failed" || code != test.code || detail.Valid || token.Valid || output != nil {
				t.Fatalf("unsafe terminal status=%s code=%s detail=%t token=%t output=%d", status, code, detail.Valid, token.Valid, len(output))
			}
		})
	}
}

func leasedAtomic(t *testing.T) {
	store, db, slot := leasedFixture(t, "")
	ctx := context.Background()

	claim, err := store.ClaimRun(ctx, "job", slot, scheduler.DefaultExecutionPolicy())
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`ALTER TABLE job_runs ADD CONSTRAINT refuse_terminal CHECK(status='running')`)
	if err != nil {
		t.Fatal(err)
	}

	err = store.CompleteRun(ctx, claim, []byte(`{}`), "")
	if err == nil {
		t.Fatal("terminal write unexpectedly passed")
	}

	var (
		enabled bool
		status  string
	)

	err = db.QueryRow(`SELECT enabled,status FROM scheduled_jobs JOIN job_runs ON job_id=scheduled_jobs.id`).Scan(&enabled, &status)
	if err != nil || !enabled || status != "running" {
		t.Fatalf("partial terminal write: enabled=%t status=%s error=%v", enabled, status, err)
	}
}

func leasedShutdown(t *testing.T) {
	store, db, _ := leasedFixture(t, "")
	runner := scheduler.New(store, scheduler.Config{Enabled: true, Interval: time.Millisecond}, &scheduler.FakeLogger{}, scheduler.WithDurableExecution(scheduler.DefaultExecutionPolicy()))
	entered, release, returned := make(chan struct{}), make(chan struct{}), make(chan struct{})

	runner.Register("task", func(ctx context.Context, _ scheduler.Job) scheduler.Result {
		close(entered)
		<-ctx.Done()
		close(returned)
		<-release

		return scheduler.Result{}
	})
	t.Cleanup(func() {
		close(release)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = runner.Stop(ctx)
	})

	err := runner.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("handler not entered")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = runner.Stop(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Stop claimed completion: %v", err)
	}

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("handler missed cancellation")
	}

	var status string

	err = db.QueryRow(`SELECT status FROM job_runs`).Scan(&status)
	if err != nil || status != "running" {
		t.Fatalf("outstanding work status=%s error=%v", status, err)
	}
}
