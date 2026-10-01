// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
	"hatmax.adrianpk.com/testhelper"
)

// Both stores must retire one-shot slots and advance recurring slots after all outcomes.
func TestSlotTransition(t *testing.T) {
	for _, backend := range []string{"fake", "postgres"} {
		for _, schedule := range []struct {
			name string
			spec string
		}{
			{"once", ""},
			{"interval", `{"type":"interval","every":"30m"}`},
			{"daily", `{"type":"daily","hour":1}`},
			{"weekly", `{"type":"weekly","day":4,"hour":1}`},
		} {
			for _, outcome := range []string{"success", "failure", "panic", "unknown"} {
				t.Run(backend+"/"+schedule.name+"/"+outcome, func(t *testing.T) {
					ctx := context.Background()
					now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

					rule, err := parseSchedule(schedule.spec, "UTC")
					if err != nil {
						t.Fatal(err)
					}

					job := scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now, Schedule: rule}
					store, db := slotStore(t, backend, job, schedule.spec)
					clock := scheduler.NewFakeClock(now)
					runner := scheduler.New(store, scheduler.Config{Workers: 2, RetryAttempts: 1}, &scheduler.FakeLogger{}, scheduler.WithClock(clock))

					var calls atomic.Int32

					if outcome != "unknown" {
						runner.Register("task", func(context.Context, scheduler.Job) scheduler.Result {
							calls.Add(1)

							if outcome == "panic" {
								panic("broken handler")
							}

							if outcome == "failure" {
								return scheduler.Result{Err: errors.New("handler failed")}
							}

							return scheduler.Result{Output: map[string]any{"ok": true}}
						})
					}

					for range 2 {
						err = runner.Tick(ctx)
						if err != nil {
							t.Fatal(err)
						}
					}

					due, err := store.ListDue(ctx, now, 1)
					if err != nil || len(due) != 0 {
						t.Fatalf("completed slot remains due: %v / %v", due, err)
					}

					err = store.CreateRun(ctx, job.ID, "duplicate", now)
					if err == nil {
						t.Fatal("duplicate completed slot was accepted")
					}

					wantCalls := int32(1)

					if rule != nil {
						clock.Set(rule.Next(now))

						err = runner.Tick(ctx)
						if err != nil {
							t.Fatal(err)
						}

						wantCalls = 2
					}

					if outcome == "unknown" {
						wantCalls = 0
					}

					if calls.Load() != wantCalls {
						t.Fatalf("handler calls = %d, want %d", calls.Load(), wantCalls)
					}

					if db != nil {
						var (
							enabled    bool
							last, next time.Time
							count      int
						)

						err = db.QueryRowContext(ctx, `SELECT enabled, last_run_at, next_run_at FROM scheduled_jobs WHERE id = 'job'`).Scan(&enabled, &last, &next)
						if err != nil {
							t.Fatal(err)
						}

						if enabled != (rule != nil) || !last.Equal(clock.Now()) || (rule != nil && !next.Equal(rule.Next(clock.Now()))) {
							t.Fatalf("persisted schedule enabled=%t last=%v next=%v", enabled, last, next)
						}

						status := "success"
						if outcome != "success" {
							status = "failed"
						}

						err = db.QueryRowContext(ctx, `SELECT count(*) FROM job_runs WHERE status = $1`, status).Scan(&count)
						if err != nil {
							t.Fatal(err)
						}

						wantRuns := 1
						if rule != nil {
							wantRuns = 2
						}

						if count != wantRuns {
							t.Fatalf("terminal runs = %d, want %d", count, wantRuns)
						}
					}
				})
			}
		}
	}
}

func slotStore(t *testing.T, backend string, job scheduler.Job, spec string) (scheduler.JobStore, *sql.DB) {
	t.Helper()

	if backend == "fake" {
		store := scheduler.NewFakeStore()
		store.AddJob(job)

		return store, nil
	}

	db, _, cleanup := testhelper.SetupTestDB(t)
	t.Cleanup(cleanup)

	_, err := db.ExecContext(context.Background(), Schema)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.ExecContext(context.Background(), `INSERT INTO scheduled_jobs (id, name, task_type, next_run_at, schedule_spec)
		VALUES ($1, $1, $2, $3, $4)`, job.ID, job.TaskType, job.ScheduledFor, spec)
	if err != nil {
		t.Fatal(err)
	}

	return NewStore(db), db
}

// A claimed slot must not starve later work, even if its run never reaches terminal state.
func TestClaimedBatch(t *testing.T) {
	for _, backend := range []string{"fake", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			job := scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now}

			store, db := slotStore(t, backend, job, "")

			err := store.CreateRun(ctx, job.ID, "claimed", now)
			if err != nil {
				t.Fatal(err)
			}

			due, err := store.ListDue(ctx, now, 1)
			if err != nil || len(due) != 0 {
				t.Fatalf("claimed slot remains due: %v / %v", due, err)
			}

			fresh := scheduler.Job{ID: "fresh", TaskType: "task", ScheduledFor: now.Add(time.Minute)}
			if db == nil {
				store.(*scheduler.FakeStore).AddJob(fresh)
			} else {
				_, err = db.ExecContext(ctx, `INSERT INTO scheduled_jobs (id, name, task_type, next_run_at) VALUES ('fresh', 'fresh', 'task', $1)`, fresh.ScheduledFor)
				if err != nil {
					t.Fatal(err)
				}
			}

			due, err = store.ListDue(ctx, fresh.ScheduledFor, 1)
			if err != nil || len(due) != 1 || due[0].ID != fresh.ID {
				t.Fatalf("claimed slot starved the next batch: %v / %v", due, err)
			}

			var (
				successes atomic.Int32
				wg        sync.WaitGroup
			)
			for index := range 8 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := store.CreateRun(ctx, job.ID, fmt.Sprintf("next-%d", index), now.Add(time.Hour))
					if err == nil {
						successes.Add(1)
					}
				}()
			}

			wg.Wait()

			if successes.Load() != 1 {
				t.Fatalf("concurrent claims = %d, want 1", successes.Load())
			}
		})
	}
}

// Completion is idempotent and cannot overwrite an explicit application reschedule.
func TestCompletionOwner(t *testing.T) {
	for _, backend := range []string{"fake", "postgres"} {
		for _, action := range []string{"repeat", "reschedule", "disable"} {
			t.Run(backend+"/"+action, func(t *testing.T) {
				ctx := context.Background()
				now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				schedule := scheduler.Interval{Every: time.Hour}

				store, _ := slotStore(t, backend, scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now, Schedule: schedule}, `{"type":"interval","every":"1h"}`)

				err := store.CreateRun(ctx, "job", "run", now)
				if err != nil {
					t.Fatal(err)
				}

				next := now.Add(time.Hour)
				if action != "repeat" {
					next = now.Add(24 * time.Hour)
					if action == "disable" {
						next = time.Time{}
					}

					err = store.UpdateNextRun(ctx, "job", now, next)
					if err != nil {
						t.Fatal(err)
					}
				}

				err = store.MarkSuccess(ctx, "run", now, []byte(`{}`))
				if err != nil {
					t.Fatal(err)
				}

				err = store.MarkFailed(ctx, "run", now.Add(48*time.Hour), "late duplicate completion")
				if err != nil {
					t.Fatal(err)
				}

				due, err := store.ListDue(ctx, now.Add(48*time.Hour), 1)
				if err != nil {
					t.Fatal(err)
				}

				if action == "disable" {
					if len(due) != 0 {
						t.Fatal("completion re-enabled a manually disabled job")
					}
				} else if len(due) != 1 || !due[0].ScheduledFor.Equal(next) {
					t.Fatalf("completion changed the next slot: %v, want %v", due, next)
				}
			})
		}
	}
}

// A run-write failure must roll back the job update in the same transaction.
func TestCompletionAtomic(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	store, db := slotStore(t, "postgres", scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now}, "")

	err := store.CreateRun(ctx, "job", "run", now)
	if err != nil {
		t.Fatal(err)
	}

	err = store.MarkRunning(ctx, "run", now)
	if err != nil {
		t.Fatal(err)
	}

	err = store.MarkSuccess(ctx, "run", now, []byte("invalid JSON"))
	if err == nil {
		t.Fatal("invalid result was accepted")
	}

	var (
		enabled bool
		last    sql.NullTime
		status  string
	)

	err = db.QueryRowContext(ctx, `SELECT j.enabled, j.last_run_at, r.status FROM scheduled_jobs j JOIN job_runs r ON j.id = r.job_id`).Scan(&enabled, &last, &status)
	if err != nil {
		t.Fatal(err)
	}

	if !enabled || last.Valid || status != "running" {
		t.Fatalf("partial completion persisted: enabled=%t last=%v status=%s", enabled, last, status)
	}
}
