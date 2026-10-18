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
	"time"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/scheduler"
)

var _ scheduler.DurableStore = (*Store)(nil)

// ListClaimable closes its finite result set before any handler is called.
// PostgreSQL time, not a worker's clock, controls retry and lease eligibility.
func (s *Store) ListClaimable(ctx context.Context, limit int) ([]scheduler.Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT j.id,j.name,j.task_type,j.payload,j.next_run_at,j.metadata,j.schedule_spec,j.schedule_tz
 FROM scheduled_jobs j LEFT JOIN job_runs r ON r.job_id=j.id AND r.scheduled_for=j.next_run_at
 WHERE j.enabled AND j.next_run_at <= clock_timestamp() AND (r.id IS NULL OR (r.durable AND
 ((r.status='running' AND r.lease_expires_at <= clock_timestamp()) OR (r.status='retry_wait' AND r.retry_at <= clock_timestamp()))))
 ORDER BY COALESCE(r.retry_at,r.lease_expires_at,j.next_run_at),j.id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []scheduler.Job

	for rows.Next() {
		var (
			job        scheduler.Job
			metadata   []byte
			spec, zone string
		)

		err = rows.Scan(&job.ID, &job.Name, &job.TaskType, &job.Payload, &job.ScheduledFor, &metadata, &spec, &zone)
		if err != nil {
			return nil, err
		}

		job.Schedule, err = parseSchedule(spec, zone)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(metadata, &job.Metadata)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

type lockedJob struct {
	id, spec, zone string
	slot           time.Time
	enabled        bool
	policy         []byte
}

func lockJob(ctx context.Context, tx *sql.Tx, id string) (lockedJob, error) {
	var job lockedJob

	err := tx.QueryRowContext(ctx, `SELECT id,schedule_spec,schedule_tz,next_run_at,enabled,durable_policy
 FROM scheduled_jobs WHERE id=$1 FOR UPDATE`, id).Scan(&job.id, &job.spec, &job.zone, &job.slot, &job.enabled, &job.policy)

	return job, err
}

type storedPolicy struct {
	Handler  int64 `json:"handler_timeout_ms"`
	Lease    int64 `json:"lease_duration_ms"`
	Attempts int   `json:"max_attempts"`
	Delay    int64 `json:"retry_delay_ms"`
}

func (p storedPolicy) execution() (scheduler.ExecutionPolicy, error) {
	// Validate integer bounds before multiplying untrusted persisted durations.
	if p.Handler < 1000 || p.Handler > 600000 || p.Lease < p.Handler+5000 || p.Lease > 900000 ||
		p.Attempts < 1 || p.Attempts > 32 || p.Delay < 1000 || p.Delay > 3600000 {
		return scheduler.ExecutionPolicy{}, scheduler.ErrInvalidPolicy
	}

	return scheduler.ExecutionPolicy{HandlerTimeout: time.Duration(p.Handler) * time.Millisecond,
		LeaseDuration: time.Duration(p.Lease) * time.Millisecond, MaxAttempts: p.Attempts, RetryDelay: time.Duration(p.Delay) * time.Millisecond}, nil
}

// ClaimRun serializes on the job before the run, so slot creation, replacement
// ownership and exhausted recurrence all have a single transactional boundary.
func (s *Store) ClaimRun(ctx context.Context, jobID string, slot time.Time, policy scheduler.ExecutionPolicy) (scheduler.RunClaim, error) {
	err := policy.Validate()
	if err != nil {
		return scheduler.RunClaim{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return scheduler.RunClaim{}, err
	}
	defer tx.Rollback()

	job, err := lockJob(ctx, tx, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.RunClaim{}, scheduler.ErrRunClaim
	}

	if err != nil {
		return scheduler.RunClaim{}, err
	}

	if !job.enabled || !job.slot.Equal(slot) {
		return scheduler.RunClaim{}, scheduler.ErrRunClaim
	}

	var (
		claim            scheduler.RunClaim
		status           string
		durable          bool
		retryAt, expires sql.NullTime
		stored           storedPolicy
	)

	err = tx.QueryRowContext(ctx, `SELECT id,status,durable,attempt,retry_at,lease_expires_at,
 COALESCE(handler_timeout_ms,0),COALESCE(lease_duration_ms,0),COALESCE(retry_limit,0),COALESCE(retry_delay_ms,0)
 FROM job_runs WHERE job_id=$1 AND scheduled_for=$2 FOR UPDATE`, jobID, slot).Scan(&claim.RunID, &status, &durable, &claim.Attempt, &retryAt, &expires, &stored.Handler, &stored.Lease, &stored.Attempts, &stored.Delay)

	fresh := errors.Is(err, sql.ErrNoRows)
	if err != nil && !fresh {
		return scheduler.RunClaim{}, err
	}

	var now time.Time

	err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now)
	if err != nil {
		return scheduler.RunClaim{}, err
	}

	if slot.After(now) {
		return scheduler.RunClaim{}, scheduler.ErrRunClaim
	}

	if fresh {
		if len(job.policy) > 0 {
			err = json.Unmarshal(job.policy, &stored)
			if err != nil {
				return scheduler.RunClaim{}, scheduler.ErrInvalidPolicy
			}

			policy, err = stored.execution()
			if err != nil {
				return scheduler.RunClaim{}, err
			}
		}

		if len(job.policy) == 0 {
			_, err = tx.ExecContext(ctx, `UPDATE scheduled_jobs SET durable_policy=jsonb_build_object(
 'handler_timeout_ms',$2::bigint,'lease_duration_ms',$3::bigint,'max_attempts',$4::int,'retry_delay_ms',$5::bigint),updated_at=$6 WHERE id=$1`,
				jobID, policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.MaxAttempts, policy.RetryDelay.Milliseconds(), now)
			if err != nil {
				return scheduler.RunClaim{}, err
			}
		}

		claim.RunID, claim.Attempt = uuid.NewString(), 1
	} else {
		if !durable || !((status == "retry_wait" && retryAt.Valid && !retryAt.Time.After(now)) ||
			(status == "running" && expires.Valid && !expires.Time.After(now))) {
			return scheduler.RunClaim{}, scheduler.ErrRunClaim
		}

		policy, err = stored.execution()
		if err != nil {
			return scheduler.RunClaim{}, err
		}

		if claim.Attempt >= policy.MaxAttempts {
			err = finishDurable(ctx, tx, job, claim.RunID, now, nil, scheduler.FailureLeaseExpired)
			if err != nil {
				return scheduler.RunClaim{}, err
			}

			err = tx.Commit()
			if err != nil {
				return scheduler.RunClaim{}, err
			}

			return scheduler.RunClaim{}, scheduler.ErrRunClaim
		}

		claim.Attempt++
	}

	claim.JobID, claim.ScheduledFor, claim.Policy = jobID, slot, policy

	claim.Token, claim.ExpiresAt = uuid.NewString(), now.Add(policy.LeaseDuration)
	if fresh {
		_, err = tx.ExecContext(ctx, `INSERT INTO job_runs (id,job_id,scheduled_for,status,attempt,retry_limit,durable,
 handler_timeout_ms,lease_duration_ms,retry_delay_ms,lease_token,lease_expires_at,started_at,created_at,updated_at)
 VALUES ($1,$2,$3,'running',1,$4,true,$5,$6,$7,$8,$9,$10,$10,$10)`, claim.RunID, jobID, slot, policy.MaxAttempts,
			policy.HandlerTimeout.Milliseconds(), policy.LeaseDuration.Milliseconds(), policy.RetryDelay.Milliseconds(), claim.Token, claim.ExpiresAt, now)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE job_runs SET status='running',attempt=$2,started_at=$3,finished_at=NULL,
 retry_at=NULL,lease_token=$4,lease_expires_at=$5,updated_at=$3,
 failure_code=CASE WHEN status='running' THEN 'lease_expired' ELSE failure_code END WHERE id=$1`, claim.RunID, claim.Attempt, now, claim.Token, claim.ExpiresAt)
	}

	if err != nil {
		return scheduler.RunClaim{}, err
	}

	err = tx.Commit()

	return claim, err
}

// CompleteRun accepts only a live generation and never trusts policy values
// supplied back by the caller. Late completions cannot advance recurrence.
func (s *Store) CompleteRun(ctx context.Context, claim scheduler.RunClaim, output []byte, failure scheduler.FailureCode) error {
	if !failure.Valid() {
		return fmt.Errorf("scheduler: invalid failure code")
	}

	if failure == "" && (len(output) > 8192 || !json.Valid(output)) {
		failure, output = scheduler.FailureOutput, nil
	}

	if failure != "" {
		output = nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	job, err := lockJob(ctx, tx, claim.JobID)
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.ErrStaleClaim
	}

	if err != nil {
		return err
	}

	var (
		status, token         string
		slot, expires         time.Time
		attempt, limit, delay int
	)

	err = tx.QueryRowContext(ctx, `SELECT status,COALESCE(lease_token,''),scheduled_for,COALESCE(lease_expires_at,'epoch'),attempt,retry_limit,retry_delay_ms
 FROM job_runs WHERE id=$1 AND job_id=$2 AND durable FOR UPDATE`, claim.RunID, claim.JobID).Scan(&status, &token, &slot, &expires, &attempt, &limit, &delay)
	if errors.Is(err, sql.ErrNoRows) {
		return scheduler.ErrStaleClaim
	}

	if err != nil {
		return err
	}

	var now time.Time

	err = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now)
	if err != nil {
		return err
	}

	if status != "running" || claim.Token == "" || token != claim.Token || attempt != claim.Attempt ||
		!slot.Equal(claim.ScheduledFor) || !expires.After(now) {
		return scheduler.ErrStaleClaim
	}

	if failure == "" {
		var size int

		err = tx.QueryRowContext(ctx, `SELECT octet_length($1::jsonb::text)`, output).Scan(&size)
		if err != nil {
			return err
		}

		if size > 8192 {
			failure, output = scheduler.FailureOutput, nil
		}
	}

	if failure != "" && failure != scheduler.FailureUnknownTask && attempt < limit {
		_, err = tx.ExecContext(ctx, `UPDATE job_runs SET status='retry_wait',retry_at=$2,finished_at=NULL,
 lease_token=NULL,lease_expires_at=NULL,failure_code=$3,error=NULL,output=NULL,updated_at=$4 WHERE id=$1`,
			claim.RunID, now.Add(time.Duration(delay)*time.Millisecond), failure, now)
	} else {
		err = finishDurable(ctx, tx, job, claim.RunID, now, output, failure)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

func finishDurable(ctx context.Context, tx *sql.Tx, job lockedJob, runID string, now time.Time, output []byte, failure scheduler.FailureCode) error {
	var slot time.Time

	err := tx.QueryRowContext(ctx, `SELECT scheduled_for FROM job_runs WHERE id=$1`, runID).Scan(&slot)
	if err != nil {
		return err
	}

	if job.enabled && job.slot.Equal(slot) {
		schedule, err := parseSchedule(job.spec, job.zone)
		if err != nil {
			return err
		}

		var next any

		if schedule != nil {
			nextSlot := schedule.Next(now)
			if !nextSlot.After(now) || !nextSlot.After(slot) {
				return fmt.Errorf("scheduler: next slot must follow completion")
			}

			next = nextSlot
		}

		_, err = tx.ExecContext(ctx, `UPDATE scheduled_jobs SET last_run_at=$2,next_run_at=COALESCE($3::timestamptz,next_run_at),
 enabled=($3::timestamptz IS NOT NULL),updated_at=$4 WHERE id=$1`, job.id, slot, next, now)
		if err != nil {
			return err
		}
	}

	status := "success"
	if failure != "" {
		status = "failed"
	}

	_, err = tx.ExecContext(ctx, `UPDATE job_runs SET status=$2,finished_at=$3,output=$4,failure_code=NULLIF($5,''),
 error=NULL,retry_at=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$3 WHERE id=$1`, runID, status, now, output, failure)

	return err
}
