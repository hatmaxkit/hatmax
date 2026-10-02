// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/config"
)

type Runner struct {
	store    JobStore
	settings SettingsProvider
	handlers map[string]Handler
	clock    Clock
	cfg      Config
	log      Logger

	mu sync.RWMutex

	lifecycleMu sync.Mutex
	state       runnerState
	runCtx      context.Context
	cancel      context.CancelFunc
	done        chan struct{}
}

type runnerState uint8

const (
	runnerIdle runnerState = iota
	runnerRunning
	runnerStopping
	runnerStopped
)

// ErrStopped means this runner has begun shutdown. Construct a new runner to restart.
var ErrStopped = errors.New("scheduler: runner is stopping or stopped")

type Option func(*Runner)

func WithClock(c Clock) Option {
	return func(r *Runner) {
		r.clock = c
	}
}

func WithSettings(settings SettingsProvider) Option {
	return func(r *Runner) {
		r.settings = settings
	}
}

func New(store JobStore, cfg Config, log Logger, opts ...Option) *Runner {
	r := &Runner{
		store:    store,
		handlers: make(map[string]Handler),
		clock:    realClock{},
		cfg:      cfg.WithDefaults(),
		log:      log,
	}
	for _, opt := range opts {
		opt(r)
	}

	return r
}

func NewWithConfig(store JobStore, settings SettingsProvider, cfg *config.Config, log Logger, opts ...Option) *Runner {
	opts = append(opts, WithSettings(settings))

	return New(store, ConfigFromRoot(cfg), log, opts...)
}

func (r *Runner) SetClock(c Clock) {
	r.clock = c
}

func (r *Runner) SetSettings(s SettingsProvider) {
	r.settings = s
}

func (r *Runner) Register(taskType string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[taskType] = handler
}

// Start launches at most one polling loop. A canceled context is rejected before
// startup. Repeated active starts are no-ops; shutdown is terminal for this instance.
func (r *Runner) Start(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	if !r.cfg.Enabled {
		r.log.Info("scheduler: disabled")

		return nil
	}

	r.lifecycleMu.Lock()

	if r.state == runnerRunning && r.runCtx.Err() != nil {
		r.state = runnerStopping
	}

	switch r.state {
	case runnerRunning:
		r.lifecycleMu.Unlock()

		return nil
	case runnerStopping, runnerStopped:
		r.lifecycleMu.Unlock()

		return ErrStopped
	}

	runCtx, cancel := context.WithCancel(ctx)
	r.runCtx = runCtx
	r.cancel = cancel
	r.done = make(chan struct{})
	r.state = runnerRunning

	go r.run(runCtx)
	r.lifecycleMu.Unlock()

	r.log.Info("scheduler: started")

	return nil
}

// Stop cancels polling and active work, then waits within ctx's deadline. It is
// safe before Start and on repeated or concurrent calls. A timeout does not mean
// handlers have exited; another Stop can wait for the same shutdown to complete.
func (r *Runner) Stop(ctx context.Context) error {
	r.lifecycleMu.Lock()
	if r.state == runnerIdle {
		r.lifecycleMu.Unlock()

		return nil
	}

	if r.state == runnerRunning {
		r.state = runnerStopping
		r.cancel()
	}

	done := r.done
	r.lifecycleMu.Unlock()

	// Prefer completed shutdown over an already canceled caller context.
	select {
	case <-done:
		return nil
	default:
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) finish() {
	r.log.Info("scheduler: stopped")

	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()

	r.cancel()
	r.state = runnerStopped
	close(r.done)
}

func (r *Runner) run(ctx context.Context) {
	defer r.finish()

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	r.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	if r.isPaused(ctx) {
		return
	}

	err := r.Tick(ctx)
	if err != nil {
		r.log.Errorf("scheduler: %v", err)
	}
}

func (r *Runner) isPaused(ctx context.Context) bool {
	if r.settings == nil {
		return false
	}

	paused, _ := r.settings.GetBool(ctx, SettingPaused)

	return paused
}

// Tick runs one caller-owned batch. Cancellation stops admission of further jobs;
// already admitted handlers must return before Tick does. Stop only joins the
// background loop, not independent calls to Tick.
func (r *Runner) Tick(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return err
	}

	now := r.clock.Now()

	jobs, err := r.store.ListDue(ctx, now, r.cfg.BatchSize)
	if err != nil {
		return err
	}

	err = ctx.Err()
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	r.log.Infof("scheduler: processing %d jobs", len(jobs))

	if r.cfg.Workers <= 1 {
		for _, job := range jobs {
			err = ctx.Err()
			if err != nil {
				return err
			}

			r.process(ctx, job)
		}

		return ctx.Err()
	}

	sem := make(chan struct{}, r.cfg.Workers)

	var wg sync.WaitGroup

dispatch:
	for _, job := range jobs {
		if ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
			break dispatch
		case sem <- struct{}{}:
		}

		wg.Add(1)

		go func(j Job) {
			defer wg.Done()
			defer func() { <-sem }()

			r.process(ctx, j)
		}(job)
	}

	wg.Wait()

	return ctx.Err()
}

func (r *Runner) process(ctx context.Context, job Job) {
	if ctx.Err() != nil {
		return
	}

	if job.RunID == "" {
		job.Attempt = 1
		job.MaxAttempts = r.cfg.RetryAttempts
	}

	runID, err := r.claim(ctx, job)
	if err != nil {
		r.log.Errorf("scheduler: cannot claim job %s attempt %d: %v", job.ID, job.Attempt, err)

		return
	}

	job.RunID = runID

	r.mu.RLock()
	handler, ok := r.handlers[job.TaskType]
	r.mu.RUnlock()

	if !ok {
		err := r.store.MarkFailed(ctx, runID, r.clock.Now(), "unknown task type: "+job.TaskType)
		if err != nil {
			r.log.Errorf("scheduler: cannot mark failed %s: %v", runID, err)
		}

		r.log.Errorf("scheduler: unknown task type: %s", job.TaskType)

		return
	}

	result := invokeHandler(ctx, handler, job)

	if result.Failed() {
		r.fail(ctx, job, runID, result.Err)

		return
	}

	output := result.Output
	if output == nil {
		output = map[string]any{}
	}

	outputBytes, _ := json.Marshal(output)

	err = r.store.MarkSuccess(ctx, runID, r.clock.Now(), outputBytes)
	if err != nil {
		r.log.Errorf("scheduler: cannot mark success %s: %v", runID, err)
	}
}

func (r *Runner) claim(ctx context.Context, job Job) (string, error) {
	if job.RunID != "" {
		return job.RunID, r.store.ClaimRetry(ctx, job.RunID, job.Attempt, r.clock.Now())
	}

	runID := uuid.New().String()

	err := r.store.CreateRun(ctx, job.ID, runID, job.ScheduledFor)
	if err != nil {
		return "", err
	}

	return runID, r.store.MarkRunning(ctx, runID, r.clock.Now())
}

func (r *Runner) fail(ctx context.Context, job Job, runID string, failure error) {
	finishedAt := r.clock.Now()

	var err error
	if job.Attempt < job.MaxAttempts {
		err = r.store.MarkRetry(ctx, runID, job.MaxAttempts, finishedAt, finishedAt.Add(r.cfg.RetryBackoff), failure.Error())
		if err != nil {
			r.log.Errorf("scheduler: cannot mark retry %s: %v", runID, err)
		}
	} else {
		err = r.store.MarkFailed(ctx, runID, finishedAt, failure.Error())
		if err != nil {
			r.log.Errorf("scheduler: cannot mark failed %s: %v", runID, err)
		}
	}

	r.log.Errorf("scheduler: job %s attempt %d failed: %v", job.ID, job.Attempt, failure)
}

// Recover only application handler calls, on the goroutine invoking them.
// Store, clock, and logger failures remain outside this job failure boundary.
// Completion tracking also distinguishes legacy nil panic values from success.
func invokeHandler(ctx context.Context, handler Handler, job Job) (result Result) {
	completed := false

	defer func() {
		value := recover()
		if !completed {
			result = Result{Err: fmt.Errorf("handler panic: %v", value)}
		}
	}()

	result = handler(ctx, job)
	completed = true

	return result
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }
