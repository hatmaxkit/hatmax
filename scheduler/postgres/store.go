package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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
		SELECT id, name, task_type, payload, next_run_at, metadata
		FROM scheduled_jobs
		WHERE enabled = true AND next_run_at <= $1
		ORDER BY next_run_at
		FOR UPDATE SKIP LOCKED
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
		)

		err = rows.Scan(&j.ID, &j.Name, &j.TaskType, &j.Payload, &j.ScheduledFor, &metadataBytes)
		if err != nil {
			return nil, err
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
	query := `UPDATE job_runs SET status = 'success', finished_at = $2, output = $3, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, runID, finishedAt, output)

	return err
}

func (s *Store) MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error {
	query := `UPDATE job_runs SET status = 'failed', finished_at = $2, error = $3, updated_at = $2 WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, runID, finishedAt, errMsg)

	return err
}

func (s *Store) UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error {
	query := `UPDATE scheduled_jobs SET last_run_at = $2, next_run_at = $3, updated_at = $4 WHERE id = $1`
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, query, jobID, lastRun, nextRun, now)

	return err
}
