// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListDue(ctx context.Context, now time.Time, limit int) ([]scheduler.Job, error) {
	query := `
		SELECT j.id, j.name, j.task_type, j.payload, j.next_run_at, j.metadata, j.schedule_spec, j.schedule_tz
		FROM scheduled_jobs j
		WHERE j.enabled = true AND j.next_run_at <= $1
		AND NOT EXISTS (SELECT 1 FROM job_runs r WHERE r.job_id = j.id AND r.scheduled_for = j.next_run_at)
		ORDER BY j.next_run_at
		FOR UPDATE OF j SKIP LOCKED
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []scheduler.Job

	for rows.Next() {
		var (
			j             scheduler.Job
			metadataBytes []byte
			spec, zone    string
		)

		err = rows.Scan(&j.ID, &j.Name, &j.TaskType, &j.Payload, &j.ScheduledFor, &metadataBytes, &spec, &zone)
		if err != nil {
			return nil, err
		}

		j.Schedule, err = parseSchedule(spec, zone)
		if err != nil {
			return nil, fmt.Errorf("job %s: %w", j.ID, err)
		}

		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &j.Metadata)
		}

		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

func (s *Store) CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error {
	query := `
		INSERT INTO job_runs (id, job_id, scheduled_for, status, attempt, created_at, updated_at)
		VALUES ($1, $2, $3, 'pending', 1, $4, $4)
	`
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, query, runID, jobID, scheduledFor, now)

	return err
}

func (s *Store) MarkRunning(ctx context.Context, runID string, startedAt time.Time) error {
	query := `UPDATE job_runs SET status = 'running', started_at = $2, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, runID, startedAt)

	return err
}

func (s *Store) MarkSuccess(ctx context.Context, runID string, finishedAt time.Time, output []byte) error {
	return s.finish(ctx, runID, finishedAt, "success", output, "")
}

func (s *Store) MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error {
	return s.finish(ctx, runID, finishedAt, "failed", nil, errMsg)
}

// Lock the run and job together so completion cannot leave an exhausted due slot.
// A manually replaced or disabled schedule belongs to the caller and is preserved.
func (s *Store) finish(ctx context.Context, runID string, finishedAt time.Time, status string, output []byte, detail string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		jobID, currentStatus, spec, zone string
		scheduledFor, currentSlot        time.Time
		enabled                          bool
	)

	err = tx.QueryRowContext(ctx, `SELECT j.id, r.status, r.scheduled_for, j.next_run_at,
		j.schedule_spec, j.schedule_tz, j.enabled FROM job_runs r
		JOIN scheduled_jobs j ON j.id = r.job_id WHERE r.id = $1
		FOR UPDATE OF j, r`, runID).Scan(&jobID, &currentStatus, &scheduledFor, &currentSlot, &spec, &zone, &enabled)
	if err != nil {
		return err
	}

	if currentStatus == "success" || currentStatus == "failed" {
		return tx.Commit()
	}

	if enabled && currentSlot.Equal(scheduledFor) {
		schedule, err := parseSchedule(spec, zone)
		if err != nil {
			return err
		}

		var next any

		if schedule != nil {
			nextSlot := schedule.Next(finishedAt)
			if !nextSlot.After(finishedAt) || !nextSlot.After(scheduledFor) {
				return fmt.Errorf("scheduler: next slot must follow completion")
			}

			next = nextSlot
		}

		_, err = tx.ExecContext(ctx, `UPDATE scheduled_jobs SET last_run_at = $2,
			next_run_at = COALESCE($3::timestamptz, next_run_at), enabled = ($3::timestamptz IS NOT NULL),
			updated_at = $4 WHERE id = $1`, jobID, scheduledFor, next, finishedAt)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `UPDATE job_runs SET status = $2, finished_at = $3,
		output = $4, error = NULLIF($5, ''), updated_at = $3 WHERE id = $1`, runID, status, finishedAt, output, detail)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error {
	query := `UPDATE scheduled_jobs SET last_run_at = $2, next_run_at = COALESCE($3::timestamptz, next_run_at),
		enabled = ($3::timestamptz IS NOT NULL), updated_at = $4 WHERE id = $1`
	now := time.Now().UTC()

	var next any
	if !nextRun.IsZero() {
		next = nextRun
	}

	_, err := s.db.ExecContext(ctx, query, jobID, lastRun, next, now)

	return err
}
