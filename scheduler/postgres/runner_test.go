// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
	"hatmax.adrianpk.com/testhelper"
)

// TestPanicRun checks the failure boundary against persisted runs, not only the
// fake store. Healthy work must complete after failed jobs fill every worker slot.
func TestPanicRun(t *testing.T) {
	for _, workers := range []int{1, 2} {
		t.Run(fmt.Sprintf("workers %d", workers), func(t *testing.T) {
			db, _, cleanup := testhelper.SetupTestDB(t)
			defer cleanup()

			ctx := context.Background()

			_, err := db.ExecContext(ctx, Schema)
			if err != nil {
				t.Fatal(err)
			}

			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			for index, id := range []string{"failed-first", "failed-second", "healthy"} {
				_, err = db.ExecContext(ctx, `INSERT INTO scheduled_jobs
					(id, name, task_type, next_run_at) VALUES ($1, $1, $2, $3)`, id, id,
					now.Add(time.Duration(index-3)*time.Minute))
				if err != nil {
					t.Fatal(err)
				}
			}

			runner := scheduler.New(NewStore(db), scheduler.Config{Workers: workers, BatchSize: 3, RetryAttempts: 1},
				&scheduler.FakeLogger{}, scheduler.WithClock(scheduler.NewFakeClock(now)))
			for _, name := range []string{"failed-first", "failed-second"} {
				runner.Register(name, func(_ context.Context, _ scheduler.Job) scheduler.Result {
					panic("broken job")
				})
			}

			runner.Register("healthy", func(_ context.Context, _ scheduler.Job) scheduler.Result {
				return scheduler.Result{Output: map[string]any{"ok": true}}
			})

			err = runner.Tick(ctx)
			if err != nil {
				t.Fatal(err)
			}

			rows, err := db.QueryContext(ctx, `SELECT job_id, status, COALESCE(error, ''),
				started_at, finished_at, output FROM job_runs ORDER BY job_id`)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			count := 0

			for rows.Next() {
				var (
					id, status, detail string
					started, finished  time.Time
					output             []byte
				)

				err = rows.Scan(&id, &status, &detail, &started, &finished, &output)
				if err != nil {
					t.Fatal(err)
				}

				count++

				if !started.Equal(now) || !finished.Equal(now) {
					t.Fatalf("job %s has incomplete timestamps: %v / %v", id, started, finished)
				}

				if id == "healthy" {
					var result map[string]bool

					decodeErr := json.Unmarshal(output, &result)
					if status != "success" || detail != "" || decodeErr != nil || !result["ok"] {
						t.Fatalf("healthy run status=%s error=%s output=%s", status, detail, output)
					}
				} else if status != "failed" || !strings.Contains(detail, "handler panic: broken job") || output != nil {
					t.Fatalf("failed run %s status=%s error=%s output=%s", id, status, detail, output)
				}
			}

			err = rows.Err()
			if err != nil {
				t.Fatal(err)
			}

			if count != 3 {
				t.Fatalf("persisted %d runs, want 3", count)
			}
		})
	}
}
