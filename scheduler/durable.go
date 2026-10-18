// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrDurableUnsupported = errors.New("scheduler: store does not support durable execution")
	ErrInvalidPolicy      = errors.New("scheduler: invalid durable execution policy")
	ErrStaleClaim         = errors.New("scheduler: stale run claim")
)

// ExecutionPolicy is frozen on the first claim of a logical scheduled slot.
// Durations must be whole milliseconds. Cancellation of handlers is cooperative.
type ExecutionPolicy struct {
	HandlerTimeout time.Duration
	LeaseDuration  time.Duration
	MaxAttempts    int
	RetryDelay     time.Duration
}

func DefaultExecutionPolicy() ExecutionPolicy {
	return ExecutionPolicy{HandlerTimeout: 10 * time.Second, LeaseDuration: 30 * time.Second, MaxAttempts: 8, RetryDelay: 30 * time.Second}
}

func (p ExecutionPolicy) Validate() error {
	if p.HandlerTimeout < time.Second || p.HandlerTimeout > 10*time.Minute ||
		p.LeaseDuration < p.HandlerTimeout+5*time.Second || p.LeaseDuration > 15*time.Minute ||
		p.MaxAttempts < 1 || p.MaxAttempts > 32 || p.RetryDelay < time.Second || p.RetryDelay > time.Hour ||
		p.HandlerTimeout%time.Millisecond != 0 || p.LeaseDuration%time.Millisecond != 0 || p.RetryDelay%time.Millisecond != 0 {
		return ErrInvalidPolicy
	}

	return nil
}

// RunClaim is an opaque ownership proof. Do not publish its Token in diagnostics.
type RunClaim struct {
	RunID        string
	JobID        string
	ScheduledFor time.Time
	Attempt      int
	Token        string
	ExpiresAt    time.Time
	Policy       ExecutionPolicy
}

// FailureCode is safe retained diagnostic data, never arbitrary handler text.
type FailureCode string

const (
	FailureHandler      FailureCode = "handler_error"
	FailurePanic        FailureCode = "handler_panic"
	FailureDeadline     FailureCode = "handler_deadline"
	FailureUnknownTask  FailureCode = "unknown_task"
	FailureOutput       FailureCode = "invalid_output"
	FailureLeaseExpired FailureCode = "lease_expired"
)

func (c FailureCode) Valid() bool {
	switch c {
	case "", FailureHandler, FailurePanic, FailureDeadline, FailureUnknownTask, FailureOutput, FailureLeaseExpired:
		return true
	default:
		return false
	}
}

// DurableStore supplements JobStore without changing existing custom stores.
// Polling transfers no ownership. ClaimRun uses the database clock and atomically
// creates or reclaims an attempt. CompleteRun fences every write and advances a
// terminal slot in the same transaction. A claim miss may finalize exhaustion.
type DurableStore interface {
	ListClaimable(ctx context.Context, limit int) ([]Job, error)
	ClaimRun(ctx context.Context, jobID string, scheduledFor time.Time, policy ExecutionPolicy) (RunClaim, error)
	CompleteRun(ctx context.Context, claim RunClaim, output []byte, failure FailureCode) error
}

// WithDurableExecution opts this runner into leased execution. Start and Tick
// reject invalid policy or an unsupported store before admitting any work.
func WithDurableExecution(policy ExecutionPolicy) Option {
	return func(r *Runner) { r.durable = &policy }
}

func (r *Runner) validateDurable() error {
	if r.durable == nil {
		return nil
	}

	err := r.durable.Validate()
	if err != nil {
		return err
	}

	if _, ok := r.store.(DurableStore); !ok {
		return ErrDurableUnsupported
	}

	return nil
}

func (r *Runner) processDurable(ctx context.Context, job Job) {
	store := r.store.(DurableStore)

	claim, err := store.ClaimRun(ctx, job.ID, job.ScheduledFor, *r.durable)
	if err != nil {
		if !errors.Is(err, ErrRunClaim) {
			r.log.Errorf("scheduler: durable claim failed: %v", err)
		}

		return
	}

	job.RunID, job.Attempt, job.MaxAttempts = claim.RunID, claim.Attempt, claim.Policy.MaxAttempts

	r.mu.RLock()
	handler, ok := r.handlers[job.TaskType]
	r.mu.RUnlock()

	failure := FailureUnknownTask

	var output []byte

	if ok {
		handlerCtx, cancel := context.WithTimeout(ctx, claim.Policy.HandlerTimeout)
		result, code := invokeDurableHandler(handlerCtx, handler, job)

		cancel()

		failure = code
		if failure == "" {
			if result.Output == nil {
				result.Output = map[string]any{}
			}

			output, err = json.Marshal(result.Output)
			if err != nil || len(output) > 8192 {
				failure, output = FailureOutput, nil
			}
		}
	}

	err = store.CompleteRun(ctx, claim, output, failure)
	if err != nil {
		r.log.Errorf("scheduler: durable completion failed: %v", err)
	}
}

// Recover on the invoking goroutine without formatting or persisting panic data.
func invokeDurableHandler(ctx context.Context, handler Handler, job Job) (result Result, code FailureCode) {
	completed := false

	defer func() {
		recover()

		if !completed {
			result, code = Result{}, FailurePanic
		}
	}()

	result = handler(ctx, job)
	completed = true

	switch {
	case ctx.Err() != nil:
		code = FailureDeadline
	case result.Err != nil:
		code = FailureHandler
	}

	return result, code
}
