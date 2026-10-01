// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Canceled execution cannot fabricate a durable completion. The interrupted
// claimed slot stays excluded, while later work remains available for a new runner.
func TestStopClaimed(t *testing.T) {
	for _, workers := range []int{1, 2} {
		t.Run(fmt.Sprintf("workers %d", workers), func(t *testing.T) {
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			store, db := slotStore(t, "postgres", scheduler.Job{ID: "job", TaskType: "blocked", ScheduledFor: now}, "")
			runner := scheduler.New(store, scheduler.Config{Enabled: true, Workers: workers}, &scheduler.FakeLogger{},
				scheduler.WithClock(scheduler.NewFakeClock(now)))
			entered := make(chan struct{})
			release := make(chan struct{})

			var releaseOnce sync.Once

			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				_ = runner.Stop(ctx)
			})
			runner.Register("blocked", func(ctx context.Context, _ scheduler.Job) scheduler.Result {
				close(entered)
				<-release

				return scheduler.Result{Err: ctx.Err()}
			})

			err := runner.Start(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("handler did not start")
			}

			// An expired caller bounds the wait but still initiates shutdown.
			waitCtx, cancelWait := context.WithCancel(context.Background())
			cancelWait()

			_ = runner.Stop(waitCtx)

			releaseOnce.Do(func() { close(release) })

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			err = runner.Stop(ctx)
			if err != nil {
				t.Fatal(err)
			}

			var (
				status   string
				finished sql.NullTime
				last     sql.NullTime
				enabled  bool
			)

			err = db.QueryRowContext(ctx, `SELECT r.status, r.finished_at, j.last_run_at, j.enabled
				FROM job_runs r JOIN scheduled_jobs j ON j.id = r.job_id`).Scan(&status, &finished, &last, &enabled)
			if err != nil {
				t.Fatal(err)
			}

			if status != "running" || finished.Valid || last.Valid || !enabled {
				t.Fatalf("cancellation fabricated completion: %s finished=%v last=%v enabled=%t", status, finished, last, enabled)
			}

			_, err = db.ExecContext(ctx, `INSERT INTO scheduled_jobs (id, name, task_type, next_run_at)
				VALUES ('later', 'later', 'healthy', $1)`, now)
			if err != nil {
				t.Fatal(err)
			}

			due, err := store.ListDue(ctx, now, 1)
			if err != nil || len(due) != 1 || due[0].ID != "later" {
				t.Fatalf("interrupted claim blocked later work: %v / %v", due, err)
			}
		})
	}
}
