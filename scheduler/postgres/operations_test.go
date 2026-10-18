// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Operators inspect bounded pages, prune only terminal work and explicitly adopt
// quiesced legacy attempts without losing their counters or future retry waits.
func TestWorkOperations(t *testing.T) {
	t.Run("inspection and retention", runRetention)
	t.Run("quiesced adoption", runAdoption)
	t.Run("future policy", runPolicy)
}

func runRetention(t *testing.T) {
	store, db, slot := leasedFixture(t, "")
	ctx := context.Background()

	_, err := store.ClaimRun(ctx, "job", slot, scheduler.DefaultExecutionPolicy())
	if err != nil {
		t.Fatal(err)
	}

	expireLease(t, db)

	for index, status := range []string{"pending", "running", "retry_wait", "success", "failed"} {
		_, err = db.Exec(`INSERT INTO scheduled_jobs (id,name,task_type,next_run_at) VALUES ($1,$1,'task',clock_timestamp());
 `, fmt.Sprintf("job-%d", index))
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec(`INSERT INTO job_runs (id,job_id,scheduled_for,status,finished_at,error,output)
 SELECT $1,id,next_run_at,$3,CASE WHEN $3 IN ('success','failed') THEN clock_timestamp()-interval '1 hour' END,
 'unsafe legacy detail','{"private":"payload"}' FROM scheduled_jobs WHERE id=$2`, fmt.Sprintf("run-%d", index), fmt.Sprintf("job-%d", index), status)
		if err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}

	query := RunQuery{Limit: 2}
	for {
		page, err := store.ListRuns(ctx, query)
		if err != nil {
			t.Fatal(err)
		}

		if len(page) == 0 {
			break
		}

		for _, run := range page {
			if seen[run.ID] {
				t.Fatal("keyset repeated a run")
			}

			seen[run.ID] = true
			if !run.Durable && run.FailureCode != "" {
				t.Fatal("legacy error exposed as classified diagnostic")
			}

			if run.Durable && (run.State != "expired" || run.EligibleAt == nil) {
				t.Fatalf("expired lease omitted: %s", run.State)
			}
		}

		last := page[len(page)-1]
		query.After = RunCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}

	if len(seen) != 6 {
		t.Fatalf("inspection rows=%d", len(seen))
	}

	for _, bad := range []RunQuery{{Limit: 101}, {Limit: -1}, {State: "unknown"}, {After: RunCursor{ID: "incomplete"}}} {
		_, err = store.ListRuns(ctx, bad)
		if !errors.Is(err, ErrRunQuery) {
			t.Fatalf("unbounded query accepted: %v", err)
		}
	}

	page, err := store.ListRuns(ctx, RunQuery{JobID: "job", State: "expired"})
	if err != nil || len(page) != 1 {
		t.Fatalf("filtered inspection rows=%d error=%v", len(page), err)
	}

	deadline, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	count, err := store.PruneRuns(deadline, time.Now(), 1)
	if err != nil || count != 1 {
		t.Fatalf("bounded prune=%d %v", count, err)
	}

	count, err = store.PruneRuns(deadline, time.Now(), 1000)
	if err != nil || count != 1 {
		t.Fatalf("remaining terminal prune=%d %v", count, err)
	}

	page, err = store.ListRuns(ctx, RunQuery{})
	if err != nil || len(page) != 4 {
		t.Fatalf("pruning lost unfinished work: %d %v", len(page), err)
	}

	for _, limit := range []int{-1, 0, 1001} {
		_, err = store.PruneRuns(ctx, time.Now(), limit)
		if !errors.Is(err, ErrRunQuery) {
			t.Fatalf("unbounded prune accepted: %v", err)
		}
	}

	canceled, stop := context.WithCancel(ctx)
	stop()

	_, err = store.ListRuns(canceled, RunQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("inspection ignored cancellation: %v", err)
	}

	_, err = store.PruneRuns(canceled, time.Now(), 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pruning ignored cancellation: %v", err)
	}
}

func runAdoption(t *testing.T) {
	for _, status := range []string{"pending", "running", "retry_wait", "success", "superseded"} {
		t.Run(status, func(t *testing.T) {
			store, db, slot := leasedFixture(t, "")
			ctx := context.Background()

			state := status
			if status == "superseded" {
				state = "running"
			}

			_, err := db.Exec(`INSERT INTO job_runs (id,job_id,scheduled_for,status,attempt,retry_at,finished_at,error)
 VALUES ('old','job',$1,$2,2,CASE WHEN $2='retry_wait' THEN clock_timestamp()+interval '1 hour' END,
 CASE WHEN $2='success' THEN clock_timestamp() END,'legacy diagnostic')`, slot, state)
			if err != nil {
				t.Fatal(err)
			}
			// Reapplying the upgrade retains the legacy row and cannot claim it implicitly.
			for range 2 {
				_, err = db.Exec(Schema)
				if err != nil {
					t.Fatal(err)
				}
			}

			jobs, err := store.ListClaimable(ctx, 10)
			if err != nil || len(jobs) != 0 {
				t.Fatalf("legacy work silently adopted: %d %v", len(jobs), err)
			}

			runs, err := store.ListRuns(ctx, RunQuery{})
			if err != nil || len(runs) != 1 {
				t.Fatalf("legacy inspection: %v", err)
			}

			observed := runs[0]

			if status == "superseded" {
				_, err = db.Exec(`UPDATE scheduled_jobs SET next_run_at=next_run_at+interval '1 day'`)
				if err != nil {
					t.Fatal(err)
				}
			}

			policy := scheduler.DefaultExecutionPolicy()
			policy.MaxAttempts = 4

			err = store.AdoptLegacyRun(ctx, "old", observed.Attempt, observed.UpdatedAt.Add(time.Second), policy)
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("stale revision adopted: %v", err)
			}

			exhausted := policy
			exhausted.MaxAttempts = 2

			err = store.AdoptLegacyRun(ctx, "old", observed.Attempt, observed.UpdatedAt, exhausted)
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("exhausted work adopted: %v", err)
			}

			err = store.AdoptLegacyRun(ctx, "old", observed.Attempt, observed.UpdatedAt, policy)
			if status == "success" || status == "superseded" {
				if !errors.Is(err, scheduler.ErrRunClaim) {
					t.Fatalf("ineligible adoption: %v", err)
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			err = store.AdoptLegacyRun(ctx, "old", observed.Attempt, observed.UpdatedAt, policy)
			if !errors.Is(err, scheduler.ErrRunClaim) {
				t.Fatalf("duplicate adoption: %v", err)
			}

			var (
				attempt int
				detail  sql.NullString
				code    string
			)

			err = db.QueryRow(`SELECT attempt,error,failure_code FROM job_runs`).Scan(&attempt, &detail, &code)
			if err != nil || attempt != 2 || detail.Valid || code != "legacy_adopted" {
				t.Fatalf("adoption changed history: attempt=%d detail=%t code=%s error=%v", attempt, detail.Valid, code, err)
			}

			if status == "retry_wait" {
				_, err = store.ClaimRun(ctx, "job", slot, policy)
				if !errors.Is(err, scheduler.ErrRunClaim) {
					t.Fatalf("adoption discarded future wait: %v", err)
				}

				_, err = db.Exec(`UPDATE job_runs SET retry_at=clock_timestamp()-interval '1 second'`)
				if err != nil {
					t.Fatal(err)
				}
			}

			claim, err := store.ClaimRun(ctx, "job", slot, policy)
			if err != nil || claim.Attempt != 3 || claim.Policy != policy {
				t.Fatalf("adopted attempt=%d error=%v", claim.Attempt, err)
			}

			err = store.CompleteRun(ctx, claim, []byte(`{}`), "")
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func runPolicy(t *testing.T) {
	store, db, slot := leasedFixture(t, `{"type":"interval","every":"1m"}`)
	ctx := context.Background()
	policy := scheduler.DefaultExecutionPolicy()

	first, err := store.ClaimRun(ctx, "job", slot, policy)
	if err != nil {
		t.Fatal(err)
	}

	future := policy
	future.MaxAttempts = 1

	err = store.ConfigureDurable(ctx, "job", future)
	if err != nil {
		t.Fatal(err)
	}

	expireLease(t, db)

	retry, err := store.ClaimRun(ctx, "job", slot, future)
	if err != nil || retry.Policy != policy || retry.Attempt != 2 {
		t.Fatalf("configuration reset active policy: %v", err)
	}

	err = store.CompleteRun(ctx, retry, []byte(`{}`), "")
	if err != nil {
		t.Fatal(err)
	}

	err = store.CompleteRun(ctx, first, nil, scheduler.FailureHandler)
	if !errors.Is(err, scheduler.ErrStaleClaim) {
		t.Fatalf("old owner advanced schedule: %v", err)
	}

	next := time.Now().UTC().Add(-time.Second).Truncate(time.Microsecond)

	_, err = db.Exec(`UPDATE scheduled_jobs SET next_run_at=$1`, next)
	if err != nil {
		t.Fatal(err)
	}

	newSlot, err := store.ClaimRun(ctx, "job", next, policy)
	if err != nil || newSlot.Policy != future || newSlot.Attempt != 1 {
		t.Fatalf("new slot lost configured policy: %v", err)
	}
}
