// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package scheduler

import (
	"context"
	"errors"
	"time"
)

type Handler func(ctx context.Context, job Job) Result

type Scheduler interface {
	Register(taskType string, handler Handler)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Tick(ctx context.Context) error
}

// ErrRunClaim means an attempt was already claimed or is not eligible to run.
var ErrRunClaim = errors.New("scheduler: run attempt is not claimable")

// JobStore owns slot claims, retry waits, and completion. ListDue includes
// unclaimed slots (Attempt 1, empty RunID) and due retry waits (existing RunID,
// next Attempt). CreateRun enforces one run per (jobID, scheduledFor).
// ClaimRetry atomically claims that exact next attempt, only after its deadline.
// MarkRetry persists a failed attempt, fixed total budget, and retry deadline
// without advancing the job. ListDue retains that budget in Job.MaxAttempts.
// MarkSuccess and MarkFailed atomically record the terminal outcome and advance
// the schedule, or retire a one-shot job.
// Recurrence uses the finish time, skips missed slots, and also advances after failure.
type JobStore interface {
	ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error)
	CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error
	MarkRunning(ctx context.Context, runID string, startedAt time.Time) error
	ClaimRetry(ctx context.Context, runID string, attempt int, startedAt time.Time) error
	MarkRetry(ctx context.Context, runID string, maxAttempts int, finishedAt, retryAt time.Time, errMsg string) error
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
