// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Reject canceled startup before it can launch a background polling loop.
func TestStartCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := New(NewFakeStore(), Config{Enabled: true}, &FakeLogger{})

	err := runner.Start(ctx)
	if !errors.Is(err, context.Canceled) {
		_ = runner.Stop(context.Background())

		t.Fatalf("Start returned %v, want context.Canceled", err)
	}
}

// Repeated cleanup is safe even when startup never happened.
func TestStopRepeat(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}

		t.Run(name, func(t *testing.T) {
			runner := New(NewFakeStore(), Config{Enabled: enabled}, &FakeLogger{})
			for range 2 {
				err := runner.Stop(context.Background())
				if err != nil {
					t.Fatal(err)
				}
			}

			if !enabled {
				err := runner.Start(context.Background())
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Concurrent starts share one immediate poll, and concurrent stops join the
// same loop. Cleanup before startup does not consume an unused runner.
func TestLifecycleCalls(t *testing.T) {
	store := &lifecycleStore{FakeStore: NewFakeStore(), entered: make(chan struct{}, 64)}
	runner := New(store, Config{Enabled: true, Interval: time.Hour}, &FakeLogger{})

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = runner.Stop(ctx)
	})

	err := runner.Stop(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var calls sync.WaitGroup
	for range 32 {
		calls.Go(func() {
			startErr := runner.Start(context.Background())
			if startErr != nil {
				t.Error(startErr)
			}
		})
	}

	calls.Wait()
	waitLifecycle(t, store.entered)

	for range 32 {
		calls.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			stopErr := runner.Stop(ctx)
			if stopErr != nil {
				t.Error(stopErr)
			}
		})
	}

	calls.Wait()

	if store.polls.Load() != 1 {
		t.Fatalf("started %d polls, want one", store.polls.Load())
	}

	err = runner.Start(context.Background())
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("restart returned %v, want ErrStopped", err)
	}
}

// Parent cancellation reaches a blocked store and ends the loop without Stop.
// A rejected startup leaves the runner available for a later valid start.
func TestStartContext(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(fmt.Sprintf("rejected first %t", rejected), func(t *testing.T) {
			store := &lifecycleStore{
				FakeStore: NewFakeStore(), entered: make(chan struct{}, 4), waitContext: true,
			}
			runner := New(store, Config{Enabled: true, Interval: time.Millisecond}, &FakeLogger{})

			if rejected {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()

				err := runner.Start(ctx)
				if !errors.Is(err, context.DeadlineExceeded) || store.polls.Load() != 0 {
					t.Fatalf("expired startup: err=%v polls=%d", err, store.polls.Load())
				}
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := runner.Start(ctx)
			if err != nil {
				t.Fatal(err)
			}

			waitLifecycle(t, store.entered)
			cancel()

			// The done signal proves cancellation itself completed shutdown.
			runner.lifecycleMu.Lock()
			done := runner.done
			runner.lifecycleMu.Unlock()
			waitLifecycle(t, done)

			err = runner.Stop(ctx)
			if err != nil || store.polls.Load() != 1 {
				t.Fatalf("canceled shutdown: err=%v polls=%d", err, store.polls.Load())
			}
		})
	}
}

// Racing lifecycle calls cannot duplicate loops or panic. Only a start that
// linearizes before shutdown may succeed; every later start reports ErrStopped.
func TestLifecycleRace(t *testing.T) {
	store := &lifecycleStore{FakeStore: NewFakeStore(), entered: make(chan struct{}, 64)}
	runner := New(store, Config{Enabled: true, Interval: time.Hour}, &FakeLogger{})

	var calls sync.WaitGroup

	for range 32 {
		calls.Go(func() {
			err := runner.Start(context.Background())
			if err != nil && !errors.Is(err, ErrStopped) {
				t.Error(err)
			}
		})
		calls.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err := runner.Stop(ctx)
			if err != nil {
				t.Error(err)
			}
		})
	}

	calls.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.Stop(ctx)
	if err != nil || store.polls.Load() > 1 {
		t.Fatalf("racing calls: err=%v polls=%d", err, store.polls.Load())
	}
}

// A handler that ignores cancellation cannot hold Stop past its caller's
// deadline. Subsequent calls wait for that same handler, without restarting.
func TestStopDeadline(t *testing.T) {
	for _, tc := range []struct {
		workers      int
		parentCancel bool
	}{
		{1, false}, {1, true}, {2, false}, {2, true},
	} {
		t.Run(fmt.Sprintf("workers %d parent cancel %t", tc.workers, tc.parentCancel), func(t *testing.T) {
			store := NewFakeStore()
			store.AddJob(Job{ID: "blocked", TaskType: "blocked", ScheduledFor: time.Now().Add(-time.Minute)})
			runner := New(store, Config{Enabled: true, Workers: tc.workers}, &FakeLogger{})
			entered := make(chan struct{})
			release := make(chan struct{})
			canceled := make(chan struct{})

			var releaseOnce sync.Once

			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				_ = runner.Stop(ctx)
			})
			runner.Register("blocked", func(ctx context.Context, _ Job) Result {
				close(entered)
				<-ctx.Done()
				close(canceled)
				<-release

				return Result{Err: ctx.Err()}
			})

			executionCtx, cancelExecution := context.WithCancel(context.Background())
			defer cancelExecution()

			err := runner.Start(executionCtx)
			if err != nil {
				t.Fatal(err)
			}

			waitLifecycle(t, entered)

			if tc.parentCancel {
				cancelExecution()

				startErr := runner.Start(context.Background())
				if !errors.Is(startErr, ErrStopped) {
					t.Fatalf("Start after parent cancellation returned %v", startErr)
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			err = runner.Stop(ctx)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Stop returned %v, want context.DeadlineExceeded", err)
			}

			waitLifecycle(t, canceled)

			err = runner.Start(context.Background())
			if !errors.Is(err, ErrStopped) {
				t.Fatalf("Start during shutdown returned %v", err)
			}

			canceledCtx, cancelWait := context.WithCancel(context.Background())
			cancelWait()

			err = runner.Stop(canceledCtx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled waiter returned %v", err)
			}

			releaseOnce.Do(func() { close(release) })

			joinCtx, cancelJoin := context.WithTimeout(context.Background(), time.Second)
			defer cancelJoin()

			err = runner.Stop(joinCtx)
			if err != nil {
				t.Fatal(err)
			}

			err = runner.Stop(canceledCtx)
			if err != nil {
				t.Fatalf("completed shutdown returned %v", err)
			}
		})
	}
}

// Cancellation prevents additional slot claims, including while parallel
// admission is waiting for a worker. Already admitted handlers are joined.
func TestTickCanceled(t *testing.T) {
	for _, workers := range []int{1, 2} {
		t.Run(fmt.Sprintf("workers %d", workers), func(t *testing.T) {
			store := NewFakeStore()
			for index := range workers + 2 {
				store.AddJob(Job{
					ID: fmt.Sprintf("job-%d", index), TaskType: "blocked",
					ScheduledFor: time.Now().Add(time.Duration(index-10) * time.Minute),
				})
			}

			runner := New(store, Config{Workers: workers}, &FakeLogger{})
			entered := make(chan struct{}, workers)
			release := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			defer close(release)

			runner.Register("blocked", func(ctx context.Context, _ Job) Result {
				entered <- struct{}{}

				<-ctx.Done()
				<-release

				return Result{Err: ctx.Err()}
			})

			finished := make(chan error, 1)

			go func() { finished <- runner.Tick(ctx) }()

			for range workers {
				waitLifecycle(t, entered)
			}

			cancel()

			select {
			case err := <-finished:
				t.Fatalf("Tick returned before admitted handlers finished: %v", err)
			default:
			}

			// One release token per admitted handler leaves the channel open for cleanup.
			for range workers {
				release <- struct{}{}
			}

			select {
			case err := <-finished:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("Tick returned %v, want context.Canceled", err)
				}
			case <-time.After(time.Second):
				t.Fatal("Tick did not finish")
			}

			if len(store.AllRuns()) != workers || len(entered) != 0 {
				t.Fatalf("claimed %d slots, want %d", len(store.AllRuns()), workers)
			}
		})
	}
}

// Check cancellation both before querying and after an adapter returns a batch.
func TestTickAdmission(t *testing.T) {
	for _, duringQuery := range []bool{false, true} {
		t.Run(fmt.Sprintf("during query %t", duringQuery), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			store := &lifecycleStore{FakeStore: NewFakeStore(), entered: make(chan struct{}, 1)}
			store.AddJob(Job{ID: "due", ScheduledFor: time.Now().Add(-time.Minute)})

			if duringQuery {
				store.cancel = cancel
			} else {
				cancel()
			}

			runner := New(store, Config{}, &FakeLogger{})

			err := runner.Tick(ctx)
			if !errors.Is(err, context.Canceled) || len(store.AllRuns()) != 0 {
				t.Fatalf("canceled batch: err=%v runs=%d", err, len(store.AllRuns()))
			}

			wantPolls := int32(0)
			if duringQuery {
				wantPolls = 1
			}

			if store.polls.Load() != wantPolls {
				t.Fatalf("polled %d times, want %d", store.polls.Load(), wantPolls)
			}
		})
	}
}

type lifecycleStore struct {
	*FakeStore
	entered     chan struct{}
	polls       atomic.Int32
	waitContext bool
	cancel      context.CancelFunc
}

func (s *lifecycleStore) ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error) {
	s.polls.Add(1)

	s.entered <- struct{}{}

	if s.waitContext {
		<-ctx.Done()

		return nil, ctx.Err()
	}

	if s.cancel != nil {
		s.cancel()
	}

	return s.FakeStore.ListDue(ctx, now, limit)
}

func waitLifecycle(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("lifecycle signal did not arrive")
	}
}
