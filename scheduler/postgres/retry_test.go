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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Persisted waits survive new stores and runners without moving the scheduled slot
// or resetting the budget. Only terminal results consume the recurring occurrence.
func TestRetryRun(t *testing.T) {
	for _, recurring := range []bool{false, true} {
		for _, succeeds := range []bool{false, true} {
			t.Run(fmt.Sprintf("recurring %t success %t", recurring, succeeds), func(t *testing.T) {
				ctx := context.Background()
				now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

				spec := ""
				if recurring {
					spec = `{"type":"interval","every":"1m"}`
				}

				_, db := slotStore(t, "postgres", scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now}, spec)
				clock := scheduler.NewFakeClock(now)

				var runID string

				calls := 0

				for attempt := 1; attempt <= 3; attempt++ {
					limit := 3
					if attempt > 1 {
						limit = 1 // New configuration cannot reset or change an existing slot's budget.
					}

					runner := scheduler.New(NewStore(db), scheduler.Config{RetryAttempts: limit, RetryBackoff: 5 * time.Minute},
						&scheduler.FakeLogger{}, scheduler.WithClock(clock))
					runner.Register("task", func(_ context.Context, job scheduler.Job) scheduler.Result {
						calls++
						if job.Attempt != calls || job.MaxAttempts != 3 || !job.ScheduledFor.Equal(now) {
							t.Errorf("handler job = %+v, calls = %d", job, calls)
						}

						clock.Advance(10 * time.Second)

						if succeeds && job.Attempt == 3 {
							return scheduler.Result{Output: map[string]any{"ok": true}}
						}

						return scheduler.Result{Err: errors.New("transient failure")}
					})

					err := runner.Tick(ctx)
					if err != nil {
						t.Fatal(err)
					}

					finishedAt := clock.Now()

					var (
						id, status, detail string
						count, recorded    int
						slot, next         time.Time
						finished           time.Time
						retryAt, last      sql.NullTime
						enabled            bool
						output             []byte
					)

					err = db.QueryRowContext(ctx, `SELECT r.id, r.status, r.attempt, r.scheduled_for, r.finished_at,
						r.retry_at, COALESCE(r.error, ''), r.output, j.next_run_at, j.last_run_at, j.enabled
						FROM job_runs r JOIN scheduled_jobs j ON j.id = r.job_id`).Scan(&id, &status, &recorded,
						&slot, &finished, &retryAt, &detail, &output, &next, &last, &enabled)
					if err != nil {
						t.Fatal(err)
					}

					if runID == "" {
						runID = id
					}

					if id != runID || recorded != attempt || calls != attempt || !slot.Equal(now) || !finished.Equal(finishedAt) {
						t.Fatalf("persisted attempt id=%s count=%d calls=%d slot=%v finish=%v", id, recorded, calls, slot, finished)
					}

					if attempt < 3 {
						deadline := finishedAt.Add(5 * time.Minute)
						if status != "retry_wait" || !retryAt.Valid || !retryAt.Time.Equal(deadline) ||
							!enabled || last.Valid || !next.Equal(now) || detail != "transient failure" {
							t.Fatalf("retry changed the slot: %s retry=%v last=%v next=%v enabled=%t", status, retryAt, last, next, enabled)
						}

						clock.Set(deadline.Add(-time.Microsecond))

						err = runner.Tick(ctx)
						if err != nil || calls != attempt {
							t.Fatalf("early retry: calls=%d error=%v", calls, err)
						}

						clock.Set(deadline)

						continue
					}

					want := "failed"
					if succeeds {
						want = "success"

						var result map[string]bool

						decodeErr := json.Unmarshal(output, &result)
						if detail != "" || decodeErr != nil || !result["ok"] {
							t.Fatalf("success retains failure or loses output: %q / %s", detail, output)
						}
					} else if detail != "transient failure" || output != nil {
						t.Fatalf("terminal failure detail=%q output=%s", detail, output)
					}

					if status != want || retryAt.Valid || !last.Valid || !last.Time.Equal(now) || enabled != recurring {
						t.Fatalf("terminal run %s retry=%v last=%v enabled=%t", status, retryAt, last, enabled)
					}

					if recurring && !next.Equal(finishedAt.Add(time.Minute)) {
						t.Fatalf("recurrence advanced from the wrong time: %v", next)
					}

					err = runner.Tick(ctx)
					if err != nil || calls != 3 {
						t.Fatalf("terminal slot repeated: calls=%d error=%v", calls, err)
					}

					err = db.QueryRowContext(ctx, `SELECT count(*) FROM job_runs`).Scan(&count)
					if err != nil || count != 1 {
						t.Fatalf("slot split into %d runs: %v", count, err)
					}

					if recurring {
						due, dueErr := NewStore(db).ListDue(ctx, next, 1)
						if dueErr != nil || len(due) != 1 || due[0].Attempt != 1 || due[0].RunID != "" || due[0].MaxAttempts != 0 {
							t.Fatalf("new slot inherited an old retry budget: %v / %v", due, dueErr)
						}
					}
				}
			})
		}
	}
}

// Both stores reject early, stale, invalid, concurrent, disabled, and terminal
// claims. An attempt belongs to exactly one claimant even after polling races.
func TestRetryClaim(t *testing.T) {
	for _, backend := range []string{"fake", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			store, _ := slotStore(t, backend, scheduler.Job{ID: "job", ScheduledFor: now}, "")

			err := store.CreateRun(ctx, "job", "run", now)
			if err != nil {
				t.Fatal(err)
			}

			err = store.MarkRunning(ctx, "run", now)
			if err != nil {
				t.Fatal(err)
			}

			canceled, cancel := context.WithCancel(ctx)
			cancel()

			err = store.MarkRetry(canceled, "run", 3, now, now.Add(time.Minute), "failed")
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled retry write succeeded: %v", err)
			}

			err = store.MarkRetry(ctx, "run", 3, now, now, "failed")
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("nonpositive delay accepted: %v", err)
			}

			deadline := now.Add(time.Minute)
			for _, limit := range []int{-1, 0, 1} {
				err = store.MarkRetry(ctx, "run", limit, now, deadline, "failed")
				if !errors.Is(err, scheduler.ErrRunClaim) {
					t.Fatalf("exhausted or invalid limit %d accepted: %v", limit, err)
				}
			}

			err = store.MarkRetry(ctx, "run", 3, now, deadline, "failed")
			if err != nil {
				t.Fatal(err)
			}

			err = store.ClaimRetry(canceled, "run", 2, deadline)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled retry claim succeeded: %v", err)
			}

			for _, test := range []struct {
				name    string
				attempt int
				at      time.Time
			}{
				{"early", 2, deadline.Add(-time.Microsecond)},
				{"invalid", 0, deadline},
				{"stale", 1, deadline},
				{"skipped", 3, deadline},
			} {
				t.Run(test.name, func(t *testing.T) {
					err = store.ClaimRetry(ctx, "run", test.attempt, test.at)
					if !errors.Is(err, scheduler.ErrRunClaim) {
						t.Fatalf("ineligible claim accepted: %v", err)
					}
				})
			}

			var (
				claims atomic.Int32
				group  sync.WaitGroup
			)
			for range 16 {
				group.Go(func() {
					claimErr := store.ClaimRetry(ctx, "run", 2, deadline)
					if claimErr == nil {
						claims.Add(1)
					} else if !errors.Is(claimErr, scheduler.ErrRunClaim) {
						t.Error(claimErr)
					}
				})
			}

			group.Wait()

			if claims.Load() != 1 {
				t.Fatalf("successful claims = %d, want 1", claims.Load())
			}

			err = store.MarkRunning(ctx, "run", deadline)
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("pending claim bypassed retry ownership: %v", err)
			}

			for _, limit := range []int{2, 4} {
				err = store.MarkRetry(ctx, "run", limit, deadline, deadline.Add(time.Minute), "failed again")
				if !errors.Is(err, scheduler.ErrRunClaim) {
					t.Fatalf("fixed budget changed to %d: %v", limit, err)
				}
			}

			err = store.MarkRetry(ctx, "run", 3, deadline, deadline.Add(time.Minute), "failed again")
			if err != nil {
				t.Fatal(err)
			}

			err = store.UpdateNextRun(ctx, "job", now, time.Time{})
			if err != nil {
				t.Fatal(err)
			}

			err = store.ClaimRetry(ctx, "run", 3, deadline.Add(time.Minute))
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("disabled job retried: %v", err)
			}

			err = store.UpdateNextRun(ctx, "job", now, now)
			if err != nil {
				t.Fatal(err)
			}

			err = store.MarkFailed(ctx, "run", deadline, "terminal")
			if err != nil {
				t.Fatal(err)
			}

			err = store.ClaimRetry(ctx, "run", 3, deadline.Add(time.Minute))
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("terminal run revived: %v", err)
			}
		})
	}
}

// Applying the schema upgrades older installations without resurrecting completed
// runs or resetting attempt counts, and can safely be repeated.
func TestRetrySchema(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store, db := slotStore(t, "postgres", scheduler.Job{ID: "job", ScheduledFor: now}, "")

	_, err := db.ExecContext(ctx, `ALTER TABLE job_runs DROP COLUMN retry_at, DROP COLUMN retry_limit;
		INSERT INTO job_runs (id, job_id, scheduled_for, status, attempt)
		VALUES ('old', 'job', '2026-01-01T00:00:00Z', 'failed', 1)`)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		_, err = db.ExecContext(ctx, Schema)
		if err != nil {
			t.Fatal(err)
		}
	}

	var (
		status  string
		attempt int
		retryAt sql.NullTime
	)

	err = db.QueryRowContext(ctx, `SELECT status, attempt, retry_at FROM job_runs WHERE id = 'old'`).Scan(&status, &attempt, &retryAt)
	if err != nil || status != "failed" || attempt != 1 || retryAt.Valid {
		t.Fatalf("upgrade changed old run: %s attempt=%d retry=%v error=%v", status, attempt, retryAt, err)
	}

	due, err := store.ListDue(ctx, now.Add(time.Hour), 1)
	if err != nil || len(due) != 0 {
		t.Fatalf("upgrade replayed terminal run: %v / %v", due, err)
	}
}

// A long-overdue slot awaiting retry must not jump ahead of newer ready work;
// both stores order it by the retry deadline without replacing its slot timestamp.
func TestRetryOrder(t *testing.T) {
	for _, backend := range []string{"fake", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			store, db := slotStore(t, backend, scheduler.Job{ID: "job", ScheduledFor: now.Add(-time.Hour)}, "")

			err := store.CreateRun(ctx, "job", "run", now.Add(-time.Hour))
			if err != nil {
				t.Fatal(err)
			}

			err = store.MarkRunning(ctx, "run", now)
			if err != nil {
				t.Fatal(err)
			}

			err = store.MarkRetry(ctx, "run", 3, now, now.Add(time.Minute), "failed")
			if err != nil {
				t.Fatal(err)
			}

			if db != nil {
				_, err = db.ExecContext(ctx, `INSERT INTO scheduled_jobs (id, name, task_type, next_run_at)
					VALUES ('ready', 'ready', 'task', $1)`, now.Add(30*time.Second))
				if err != nil {
					t.Fatal(err)
				}
			} else {
				store.(*scheduler.FakeStore).AddJob(scheduler.Job{ID: "ready", ScheduledFor: now.Add(30 * time.Second)})
			}

			due, err := store.ListDue(ctx, now.Add(time.Minute), 1)
			if err != nil || len(due) != 1 || due[0].ID != "ready" {
				t.Fatalf("retry displaced ready work: %v / %v", due, err)
			}
		})
	}
}
