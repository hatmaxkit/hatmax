package scheduler

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Runner struct {
	store    JobStore
	settings SettingsProvider
	handlers map[string]Handler
	clock    Clock
	cfg      Config
	log      Logger

	mu   sync.RWMutex
	stop chan struct{}
	wg   sync.WaitGroup
}

func New(store JobStore, cfg Config, log Logger) *Runner {
	return &Runner{
		store:    store,
		handlers: make(map[string]Handler),
		clock:    realClock{},
		cfg:      cfg.WithDefaults(),
		log:      log,
		stop:     make(chan struct{}),
	}
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

func (r *Runner) Start(ctx context.Context) error {
	if !r.cfg.Enabled {
		r.log.Info("scheduler: disabled")
		return nil
	}

	r.wg.Add(1)
	go r.run(ctx)
	r.log.Info("scheduler: started")
	return nil
}

func (r *Runner) Stop(ctx context.Context) error {
	close(r.stop)
	r.wg.Wait()
	r.log.Info("scheduler: stopped")
	return nil
}

func (r *Runner) run(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	r.tick(ctx)

	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.tick(ctx)
		}
	}
}

func (r *Runner) tick(ctx context.Context) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Errorf("scheduler: panic: %v", rec)
		}
	}()

	if r.isPaused(ctx) {
		return
	}

	if err := r.Tick(ctx); err != nil {
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

func (r *Runner) Tick(ctx context.Context) error {
	now := r.clock.Now()
	jobs, err := r.store.ListDue(ctx, now, r.cfg.BatchSize)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		return nil
	}

	r.log.Infof("scheduler: processing %d jobs", len(jobs))

	if r.cfg.Workers <= 1 {
		for _, job := range jobs {
			r.process(ctx, job)
		}
		return nil
	}

	sem := make(chan struct{}, r.cfg.Workers)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j Job) {
			defer wg.Done()
			defer func() { <-sem }()
			r.process(ctx, j)
		}(job)
	}
	wg.Wait()

	return nil
}

func (r *Runner) process(ctx context.Context, job Job) {
	runID := uuid.New().String()
	now := r.clock.Now()

	if err := r.store.CreateRun(ctx, job.ID, runID, job.ScheduledFor); err != nil {
		r.log.Errorf("scheduler: cannot create run for job %s: %v", job.ID, err)
		return
	}

	if err := r.store.MarkRunning(ctx, runID, now); err != nil {
		r.log.Errorf("scheduler: cannot mark running %s: %v", runID, err)
		return
	}

	r.mu.RLock()
	handler, ok := r.handlers[job.TaskType]
	r.mu.RUnlock()

	if !ok {
		r.store.MarkFailed(ctx, runID, r.clock.Now(), "unknown task type: "+job.TaskType)
		r.log.Errorf("scheduler: unknown task type: %s", job.TaskType)
		return
	}

	result := handler(ctx, job)

	if result.Failed() {
		r.store.MarkFailed(ctx, runID, r.clock.Now(), result.Err.Error())
		r.log.Errorf("scheduler: job %s failed: %v", job.ID, result.Err)
		return
	}

	output := result.Output
	if output == nil {
		output = map[string]any{}
	}
	outputBytes, _ := json.Marshal(output)
	r.store.MarkSuccess(ctx, runID, r.clock.Now(), outputBytes)
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }
