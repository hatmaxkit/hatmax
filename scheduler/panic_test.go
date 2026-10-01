// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestWorkerPanic runs the real worker fan-out in a subprocess. An uncontained
// goroutine panic must fail this test, not terminate the parent test suite.
func TestWorkerPanic(t *testing.T) {
	for _, workers := range []int{1, 2} {
		t.Run(fmt.Sprintf("workers %d", workers), func(t *testing.T) {
			output, err := panicProcess(t, workers)
			if err != nil {
				t.Fatalf("scheduler child failed: %v\n%s", err, output)
			}
		})
	}
}

// TestPanicProcess is selected only by TestWorkerPanic's isolated subprocess.
// Filling all slots with failed jobs verifies a later healthy job can run.
func TestPanicProcess(t *testing.T) {
	value := os.Getenv("HATMAX_SCHEDULER_PANIC_WORKERS")
	if value == "" {
		t.Skip("subprocess helper")
	}

	workers, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}

	store := NewFakeStore()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner := New(store, Config{Workers: workers, BatchSize: 3, RetryAttempts: 1}, &FakeLogger{}, WithClock(NewFakeClock(now)))

	if os.Getenv("HATMAX_SCHEDULER_STORE_PANIC") == "1" {
		runner.store = &panicStore{FakeStore: store, panicList: true}
		runner.tick(context.Background())

		return
	}

	runner.Register("panic", func(_ context.Context, _ Job) Result {
		panic("broken job")
	})
	runner.Register("healthy", func(_ context.Context, _ Job) Result {
		return Result{}
	})

	for _, job := range []Job{
		{ID: "failed-first", TaskType: "panic", ScheduledFor: now},
		{ID: "failed-second", TaskType: "panic", ScheduledFor: now},
		{ID: "healthy", TaskType: "healthy", ScheduledFor: now},
	} {
		store.AddJob(job)
	}

	runner.tick(context.Background())

	runs := store.AllRuns()
	if len(runs) != 3 {
		t.Fatalf("recorded %d runs, want 3", len(runs))
	}

	for _, run := range runs {
		want := "failed"
		if run.JobID == "healthy" {
			want = "success"
		} else if !strings.Contains(run.Error, "handler panic: broken job") {
			t.Fatalf("panic detail missing: %q", run.Error)
		}

		if run.Status != want || !run.StartedAt.Equal(now) || !run.FinishedAt.Equal(now) {
			t.Fatalf("job %s status=%s start=%v finish=%v", run.JobID, run.Status, run.StartedAt, run.FinishedAt)
		}
	}
}

// TestHandlerPanic covers arbitrary panic values without losing normal results.
func TestHandlerPanic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"string", "broken job"},
		{"error", errors.New("broken job")},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := invokeHandler(context.Background(), func(_ context.Context, _ Job) Result {
				panic(tc.value)
			}, Job{})
			if !result.Failed() || result.Output != nil || !strings.HasPrefix(result.Err.Error(), "handler panic: ") {
				t.Fatalf("panic result %+v", result)
			}
		})
	}

	for _, tc := range []struct {
		name   string
		result Result
	}{
		{"success", Result{Output: map[string]any{"ok": true}}},
		{"failure", Result{Err: errTest}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := invokeHandler(context.Background(), func(_ context.Context, _ Job) Result {
				return tc.result
			}, Job{})
			if got.Err != tc.result.Err || len(got.Output) != len(tc.result.Output) || got.Output["ok"] != tc.result.Output["ok"] {
				t.Fatalf("normal result changed: %+v", got)
			}
		})
	}
}

// TestStorePanic proves recovery is not a catch-all around scheduler state.
func TestStorePanic(t *testing.T) {
	output, err := panicProcess(t, 1, "HATMAX_SCHEDULER_STORE_PANIC=1")
	if err == nil || !strings.Contains(string(output), "panic: store failure") {
		t.Fatalf("store panic was masked: %v\n%s", err, output)
	}
}

// TestFailedRunWrite keeps a failed persistence operation observable rather than
// falsely asserting that the panic's run was durably marked failed.
func TestFailedRunWrite(t *testing.T) {
	for _, handlerPanic := range []bool{false, true} {
		t.Run(fmt.Sprintf("panic %t", handlerPanic), func(t *testing.T) {
			store := &panicStore{FakeStore: NewFakeStore(), markErr: errors.New("failed-state write unavailable")}
			logger := &panicLogger{}
			runner := New(store, Config{RetryAttempts: 1}, logger)
			runner.Register("broken", func(_ context.Context, _ Job) Result {
				if handlerPanic {
					panic("broken job")
				}

				return Result{Err: errTest}
			})
			store.AddJob(Job{ID: "job", TaskType: "broken"})

			err := runner.Tick(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			runs := store.AllRuns()
			if len(runs) != 1 || runs[0].Status != "running" {
				t.Fatalf("failed persistence falsely marked a completed run: %+v", runs)
			}

			if got := strings.Join(logger.messages, "\n"); !strings.Contains(got, "cannot mark failed") ||
				!strings.Contains(got, "failed-state write unavailable") {
				t.Fatalf("persistence failure hidden: %s", got)
			}
		})
	}
}

type panicStore struct {
	*FakeStore
	markErr   error
	listErr   error
	panicList bool
}

func (s *panicStore) MarkFailed(_ context.Context, _ string, _ time.Time, _ string) error {
	return s.markErr
}

func (s *panicStore) ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error) {
	if s.panicList {
		panic("store failure")
	}

	if s.listErr != nil {
		return nil, s.listErr
	}

	return s.FakeStore.ListDue(ctx, now, limit)
}

type panicLogger struct {
	FakeLogger
	mu       sync.Mutex
	messages []string
}

func (l *panicLogger) Errorf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

func panicProcess(t *testing.T, workers int, env ...string) ([]byte, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPanicProcess$", "-test.timeout=5s")

	cmd.Env = append(os.Environ(), fmt.Sprintf("HATMAX_SCHEDULER_PANIC_WORKERS=%d", workers))
	cmd.Env = append(cmd.Env, "GORACE="+os.Getenv("GORACE")+" atexit_sleep_ms=0")
	cmd.Env = append(cmd.Env, env...)

	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("scheduler subprocess timed out: %v\n%s", ctx.Err(), output)
	}

	return output, err
}

// TestStoreError preserves the public Tick error from ListDue unchanged.
func TestStoreError(t *testing.T) {
	store := &panicStore{FakeStore: NewFakeStore(), listErr: errTest}
	runner := New(store, Config{}, &FakeLogger{})

	err := runner.Tick(context.Background())
	if err != errTest {
		t.Fatalf("store error changed: %v", err)
	}

	if len(store.AllRuns()) != 0 {
		t.Fatal("store error fabricated a job run")
	}
}
