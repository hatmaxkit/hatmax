package scheduler

import (
	"context"
	"time"
)

type NoopStore struct{}

func (NoopStore) ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error) {
	return nil, nil
}

func (NoopStore) CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error {
	return nil
}

func (NoopStore) MarkRunning(ctx context.Context, runID string, startedAt time.Time) error {
	return nil
}

func (NoopStore) MarkSuccess(ctx context.Context, runID string, finishedAt time.Time, output []byte) error {
	return nil
}

func (NoopStore) MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error {
	return nil
}

func (NoopStore) UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error {
	return nil
}
