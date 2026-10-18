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
	"time"

	"hatmax.adrianpk.com/scheduler"
)

var ErrRunQuery = errors.New("scheduler: invalid run query or batch limit")

// RunCursor is an immutable keyset position from a previous ListRuns result.
type RunCursor struct {
	CreatedAt time.Time
	ID        string
}

// RunQuery limits inspection to at most 100 rows (50 by default).
// State is empty, pending, running, expired, retry_wait, success or failed.
type RunQuery struct {
	JobID string
	State string
	Limit int
	After RunCursor
}

// RunInfo excludes ownership tokens, payloads, output and legacy raw errors.
// UpdatedAt and Attempt provide the observed revision for quiesced adoption.
type RunInfo struct {
	ID           string
	JobID        string
	TaskType     string
	State        string
	Durable      bool
	Attempt      int
	MaxAttempts  int
	ScheduledFor time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	EligibleAt   *time.Time
	FailureCode  scheduler.FailureCode
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) ListRuns(ctx context.Context, query RunQuery) ([]RunInfo, error) {
	if query.Limit == 0 {
		query.Limit = 50
	}

	if query.Limit < 1 || query.Limit > 100 || (query.After.ID == "") != query.After.CreatedAt.IsZero() {
		return nil, ErrRunQuery
	}

	switch query.State {
	case "", "pending", "running", "expired", "retry_wait", "success", "failed":
	default:
		return nil, ErrRunQuery
	}

	var after any
	if !query.After.CreatedAt.IsZero() {
		after = query.After.CreatedAt
	}

	rows, err := s.db.QueryContext(ctx, `SELECT r.id,r.job_id,j.task_type,
 CASE WHEN r.durable AND r.status='running' AND r.lease_expires_at<=clock_timestamp() THEN 'expired' ELSE r.status END,
 r.durable,r.attempt,COALESCE(r.retry_limit,0),r.scheduled_for,r.started_at,r.finished_at,
 COALESCE(r.retry_at,r.lease_expires_at),CASE WHEN r.durable THEN COALESCE(r.failure_code,'') ELSE '' END,r.created_at,r.updated_at
 FROM job_runs r JOIN scheduled_jobs j ON j.id=r.job_id
 WHERE ($1='' OR r.job_id=$1) AND ($2='' OR $2=CASE WHEN r.durable AND r.status='running'
 AND r.lease_expires_at<=clock_timestamp() THEN 'expired' ELSE r.status END)
 AND ($3::timestamptz IS NULL OR (r.created_at,r.id)>($3::timestamptz,$4))
 ORDER BY r.created_at,r.id LIMIT $5`, query.JobID, query.State, after, query.After.ID, query.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunInfo

	for rows.Next() {
		var run RunInfo

		err = rows.Scan(&run.ID, &run.JobID, &run.TaskType, &run.State, &run.Durable, &run.Attempt, &run.MaxAttempts,
			&run.ScheduledFor, &run.StartedAt, &run.FinishedAt, &run.EligibleAt, &run.FailureCode, &run.CreatedAt, &run.UpdatedAt)
		if err != nil {
			return nil, err
		}

		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// PruneRuns removes at most 1000 terminal rows older than before. It uses the
// caller's context, preserves every unfinished run and never starts a service.
func (s *Store) PruneRuns(ctx context.Context, before time.Time, limit int) (int64, error) {
	if before.IsZero() || limit < 1 || limit > 1000 {
		return 0, ErrRunQuery
	}

	result, err := s.db.ExecContext(ctx, `WITH expired AS (SELECT id FROM job_runs
 WHERE status IN ('success','failed') AND finished_at<$1 ORDER BY finished_at,id
 FOR UPDATE SKIP LOCKED LIMIT $2) DELETE FROM job_runs r USING expired e WHERE r.id=e.id`, before, limit)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func policyJSON(policy scheduler.ExecutionPolicy) ([]byte, error) {
	return json.Marshal(storedPolicy{Handler: policy.HandlerTimeout.Milliseconds(), Lease: policy.LeaseDuration.Milliseconds(),
		Attempts: policy.MaxAttempts, Delay: policy.RetryDelay.Milliseconds()})
}

// ConfigureDurable changes the policy for future slots only. Existing runs keep
// their frozen snapshots. Applying this configuration opts the job out of legacy polling.
func (s *Store) ConfigureDurable(ctx context.Context, jobID string, policy scheduler.ExecutionPolicy) error {
	err := policy.Validate()
	if err != nil {
		return err
	}

	data, err := policyJSON(policy)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `UPDATE scheduled_jobs SET durable_policy=$2,updated_at=clock_timestamp() WHERE id=$1`, jobID, data)

	return claimed(result, err)
}

// AdoptLegacyRun requires all old workers to be quiesced by the caller. It matches
// the inspected revision, preserves cumulative attempts and refuses newer slots,
// terminal runs and exhausted budgets. The abandoned attempt remains consumed.
func (s *Store) AdoptLegacyRun(ctx context.Context, runID string, attempt int, updatedAt time.Time, policy scheduler.ExecutionPolicy) error {
	err := policy.Validate()
	if err != nil {
		return err
	}

	if attempt < 1 || attempt >= policy.MaxAttempts || updatedAt.IsZero() {
		return scheduler.ErrRunClaim
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var jobID string

	err = tx.QueryRowContext(ctx, `SELECT job_id FROM job_runs WHERE id=$1`, runID).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.ErrRunClaim
	}

	if err != nil {
		return err
	}

	job, err := lockJob(ctx, tx, jobID)
	if err != nil {
		return err
	}

	if !job.enabled {
		return scheduler.ErrRunClaim
	}

	var found string

	err = tx.QueryRowContext(ctx, `SELECT id FROM job_runs WHERE id=$1 AND NOT durable AND attempt=$2 AND updated_at=$3
 AND scheduled_for=$4 AND status IN ('pending','running','retry_wait') FOR UPDATE`, runID, attempt, updatedAt, job.slot).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.ErrRunClaim
	}

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE job_runs SET durable=true,status='retry_wait',handler_timeout_ms=$2,lease_duration_ms=$3,
 retry_delay_ms=$4,retry_limit=$5,retry_at=GREATEST(COALESCE(retry_at,clock_timestamp()),clock_timestamp()),finished_at=NULL,
 lease_token=NULL,lease_expires_at=NULL,error=NULL,output=NULL,failure_code='legacy_adopted',updated_at=clock_timestamp() WHERE id=$1`,
		runID, policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.RetryDelay.Milliseconds(), policy.MaxAttempts)
	if err != nil {
		return err
	}

	data, err := policyJSON(policy)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE scheduled_jobs SET durable_policy=$2,updated_at=clock_timestamp() WHERE id=$1`, jobID, data)
	if err != nil {
		return err
	}

	return tx.Commit()
}
