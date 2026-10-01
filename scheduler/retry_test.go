// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package scheduler

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// Retries wait without holding a worker, retain their budget across runner restarts,
// and become terminal only on success or exhaustion of the total attempt limit.
func TestRetrySteps(t *testing.T) {
	for _, test := range []struct {
		name     string
		limit    int
		succeed  int
		panics   bool
		attempts int
		status   string
	}{
		{"disabled", 1, 0, false, 1, "failed"},
		{"exhausted", 3, 0, false, 3, "failed"},
		{"default", 0, 0, false, 3, "failed"},
		{"recovered", 3, 2, false, 2, "success"},
		{"panic", 2, 0, true, 2, "failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			clock := NewFakeClock(now)
			store := NewFakeStore()
			store.AddJob(Job{ID: "job", TaskType: "task", ScheduledFor: now})
			store.AddJob(Job{ID: "healthy", TaskType: "healthy", ScheduledFor: now})

			var (
				attempts []int
				runID    string
			)

			healthy := 0

			for attempt := 1; attempt <= test.attempts; attempt++ {
				limit := test.limit
				if attempt > 1 {
					limit = 9 // A restart cannot increase a waiting slot's persisted budget.
				}

				runner := New(store, Config{RetryAttempts: limit, RetryBackoff: time.Minute}, &FakeLogger{}, WithClock(clock))
				runner.Register("task", func(_ context.Context, job Job) Result {
					if runID == "" {
						runID = job.RunID
					}

					if job.RunID == "" || job.RunID != runID {
						t.Errorf("retry changed run identity: %q / %q", job.RunID, runID)
					}

					attempts = append(attempts, job.Attempt)

					if test.panics {
						panic("broken handler")
					}

					if job.Attempt == test.succeed {
						return Result{Output: map[string]any{"ok": true}}
					}

					return Result{Err: errTest}
				})
				runner.Register("healthy", func(context.Context, Job) Result {
					healthy++

					return Result{}
				})

				err := runner.Tick(ctx)
				if err != nil {
					t.Fatal(err)
				}

				if len(attempts) != attempt || attempts[attempt-1] != attempt {
					t.Fatalf("attempts = %v, want consecutive attempts through %d", attempts, attempt)
				}

				if healthy != 1 {
					t.Fatalf("healthy job calls = %d, want 1", healthy)
				}

				clock.Advance(time.Minute - time.Nanosecond)

				err = runner.Tick(ctx)
				if err != nil {
					t.Fatal(err)
				}

				if len(attempts) != attempt {
					t.Fatal("retry ran before its delay elapsed")
				}

				clock.Advance(time.Nanosecond)
			}

			for _, run := range store.AllRuns() {
				if run.JobID == "job" && run.Status != test.status {
					t.Fatalf("terminal status = %s, want %s", run.Status, test.status)
				}
			}

			due, err := store.ListDue(ctx, clock.Now(), 10)
			if err != nil || len(due) != 0 {
				t.Fatalf("terminal jobs remain due: %v / %v", due, err)
			}

			want := make([]int, test.attempts)
			for index := range want {
				want[index] = index + 1
			}

			if !reflect.DeepEqual(attempts, want) || len(store.AllRuns()) != 2 {
				t.Fatalf("attempts = %v, runs = %d", attempts, len(store.AllRuns()))
			}
		})
	}
}

type retryFailureStore struct {
	*FakeStore
}

func (s *retryFailureStore) MarkRetry(context.Context, string, int, time.Time, time.Time, string) error {
	return errTest
}

// A retry-write failure must not pretend the attempt is queued or consume its slot.
func TestRetryWrite(t *testing.T) {
	store := &retryFailureStore{FakeStore: NewFakeStore()}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store.AddJob(Job{ID: "job", TaskType: "task", ScheduledFor: now})

	logger := &panicLogger{}
	runner := New(store, Config{}, logger, WithClock(NewFakeClock(now)))
	calls := 0

	runner.Register("task", func(context.Context, Job) Result {
		calls++

		return Result{Err: errTest}
	})

	err := runner.Tick(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	runs := store.AllRuns()
	if len(runs) != 1 || runs[0].Status != "running" || runs[0].Attempt != 1 || !runs[0].RetryAt.IsZero() {
		t.Fatalf("retry write fabricated completion: %v", runs)
	}

	if !reflect.DeepEqual(logger.messages, []string{
		"scheduler: cannot mark retry " + runs[0].ID + ": test error",
		"scheduler: job job attempt 1 failed: test error",
	}) {
		t.Fatalf("retry write error was hidden: %v", logger.messages)
	}

	err = runner.Tick(context.Background())
	if err != nil || calls != 1 {
		t.Fatalf("running claim was replayed after failed write: calls=%d error=%v", calls, err)
	}
}
