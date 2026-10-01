// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

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

// JobStore owns slot claims and completion. ListDue excludes claimed slots;
// CreateRun enforces one run per (jobID, scheduledFor). MarkSuccess and MarkFailed
// atomically record the outcome and advance the schedule, or retire a one-shot job.
// Recurrence uses the finish time, skips missed slots, and also advances after failure.
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
