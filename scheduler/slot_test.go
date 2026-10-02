// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Completed one-shot jobs must not execute again or occupy the next due batch.
func TestOneShotSlot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store := NewFakeStore()
	store.AddJob(Job{ID: "once", TaskType: "task", ScheduledFor: now})
	runner := New(store, Config{}, &FakeLogger{}, WithClock(NewFakeClock(now)))
	calls := 0

	runner.Register("task", func(context.Context, Job) Result {
		calls++

		return Result{}
	})

	for range 2 {
		err := runner.Tick(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}

	due, err := store.ListDue(ctx, now, 1)
	if err != nil || len(due) != 0 || calls != 1 {
		t.Fatalf("completed slot: due=%v calls=%d error=%v", due, calls, err)
	}
}

// A fake store must reject the same claimed slot just like PostgreSQL's unique key.
func TestSlotUnique(t *testing.T) {
	store := NewFakeStore()
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	err := store.CreateRun(ctx, "job", "first", now)
	if err != nil {
		t.Fatal(err)
	}

	err = store.CreateRun(ctx, "job", "second", now)
	if err == nil {
		t.Fatal("duplicate job slot was accepted")
	}
}

type terminalStore struct {
	*FakeStore
}

func (store *terminalStore) MarkSuccess(context.Context, string, time.Time, []byte) error {
	return errTest
}

func (store *terminalStore) MarkFailed(context.Context, string, time.Time, string) error {
	return errTest
}

// A terminal write failure must be observable and must not advance the job.
func TestTerminalWrite(t *testing.T) {
	for _, outcome := range []string{"success", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			store := &terminalStore{FakeStore: NewFakeStore()}
			logger := &panicLogger{}
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			store.AddJob(Job{ID: "job", TaskType: "task", ScheduledFor: now, Schedule: Interval{Every: time.Hour}})

			runner := New(store, Config{}, logger, WithClock(NewFakeClock(now)))
			if outcome == "success" {
				runner.Register("task", func(context.Context, Job) Result { return Result{} })
			}

			err := runner.Tick(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			runs := store.AllRuns()
			if len(runs) != 1 || runs[0].Status != "running" || !store.Jobs[0].ScheduledFor.Equal(now) {
				t.Fatalf("terminal write falsely completed the slot: %v", runs)
			}

			if !strings.Contains(strings.Join(logger.messages, "\n"), "cannot mark "+outcome) {
				t.Fatalf("terminal write error was hidden: %v", logger.messages)
			}
		})
	}
}
