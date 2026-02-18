package scheduler

import (
	"context"
	"time"
)

type Handler func(ctx context.Context, job Job) Result

type Scheduler interface {
	Register(taskType string, handler Handler)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Tick(ctx context.Context) error
}

type JobStore interface {
	ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error)
	CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error
	MarkRunning(ctx context.Context, runID string, startedAt time.Time) error
	MarkSuccess(ctx context.Context, runID string, finishedAt time.Time, output []byte) error
	MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error
	UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error
}

type SettingsProvider interface {
	GetBool(ctx context.Context, key string) (bool, error)
	GetInt(ctx context.Context, key string) (int, error)
}

type Clock interface {
	Now() time.Time
}

type Logger interface {
	Info(v ...any)
	Infof(format string, a ...any)
	Error(v ...any)
	Errorf(format string, a ...any)
}
