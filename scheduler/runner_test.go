package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hatmaxkit/hatmax/config"
)

var errTest = errors.New("test error")

func TestNew(t *testing.T) {
	store := NewFakeStore()
	log := &FakeLogger{}
	cfg := Config{Enabled: true, Interval: time.Second}

	r := New(store, cfg, log)

	if r.store != store {
		t.Error("store not set")
	}

	if r.handlers == nil {
		t.Error("handlers map not initialized")
	}

	if r.cfg.Interval != time.Second {
		t.Error("config not applied")
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{
			name: "all zeros get defaults",
			cfg:  Config{},
			want: Config{
				Interval:      time.Minute,
				BatchSize:     20,
				Workers:       1,
				RetryAttempts: 3,
				RetryBackoff:  time.Minute,
			},
		},
		{
			name: "custom values preserved",
			cfg: Config{
				Enabled:       true,
				Interval:      5 * time.Second,
				BatchSize:     10,
				Workers:       4,
				RetryAttempts: 5,
				RetryBackoff:  2 * time.Minute,
			},
			want: Config{
				Enabled:       true,
				Interval:      5 * time.Second,
				BatchSize:     10,
				Workers:       4,
				RetryAttempts: 5,
				RetryBackoff:  2 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.WithDefaults()
			if got != tt.want {
				t.Errorf("WithDefaults() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRunner_Register(t *testing.T) {
	r := New(NewFakeStore(), Config{}, &FakeLogger{})

	r.Register("test-task", func(ctx context.Context, job Job) Result {
		return Result{}
	})

	if _, ok := r.handlers["test-task"]; !ok {
		t.Error("handler not registered")
	}
}

func TestRunner_StartStop(t *testing.T) {
	r := New(NewFakeStore(), Config{Enabled: true, Interval: 100 * time.Millisecond}, &FakeLogger{})

	ctx := context.Background()

	err := r.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = r.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRunner_StartDisabled(t *testing.T) {
	r := New(NewFakeStore(), Config{Enabled: false}, &FakeLogger{})

	ctx := context.Background()

	err := r.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	// Should return immediately without starting goroutine
}

func TestRunner_Tick(t *testing.T) {
	store := NewFakeStore()
	clock := NewFakeClock(time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC))
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true}, log)
	r.SetClock(clock)

	var processedJobs []string

	r.Register("email", func(ctx context.Context, job Job) Result {
		processedJobs = append(processedJobs, job.ID)

		return Result{Output: map[string]any{"sent": true}}
	})

	store.AddJob(Job{
		ID:           "job-1",
		Name:         "daily-email",
		TaskType:     "email",
		Payload:      json.RawMessage(`{"to":"test@example.com"}`),
		ScheduledFor: clock.Now().Add(-time.Minute),
	})

	ctx := context.Background()

	err := r.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}

	if len(processedJobs) != 1 {
		t.Errorf("expected 1 job processed, got %d", len(processedJobs))
	}

	if processedJobs[0] != "job-1" {
		t.Errorf("expected job-1, got %s", processedJobs[0])
	}

	runs := store.AllRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	if runs[0].Status != "success" {
		t.Errorf("expected status success, got %s", runs[0].Status)
	}
}

func TestRunner_TickUnknownTaskType(t *testing.T) {
	store := NewFakeStore()
	clock := NewFakeClock(time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC))
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true}, log)
	r.SetClock(clock)

	store.AddJob(Job{
		ID:           "job-1",
		TaskType:     "unknown-type",
		ScheduledFor: clock.Now(),
	})

	ctx := context.Background()
	r.Tick(ctx)

	runs := store.AllRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	if runs[0].Status != "failed" {
		t.Errorf("expected status failed, got %s", runs[0].Status)
	}

	if runs[0].Error == "" {
		t.Error("expected error message for unknown task type")
	}
}

func TestRunner_TickHandlerError(t *testing.T) {
	store := NewFakeStore()
	clock := NewFakeClock(time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC))
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true}, log)
	r.SetClock(clock)

	r.Register("failing", func(ctx context.Context, job Job) Result {
		return Result{Err: errTest}
	})

	store.AddJob(Job{
		ID:           "job-1",
		TaskType:     "failing",
		ScheduledFor: clock.Now(),
	})

	ctx := context.Background()
	r.Tick(ctx)

	runs := store.AllRuns()
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	if runs[0].Status != "failed" {
		t.Errorf("expected status failed, got %s", runs[0].Status)
	}
}

func TestRunner_TickWithWorkers(t *testing.T) {
	store := NewFakeStore()
	clock := NewFakeClock(time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC))
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true, Workers: 3}, log)
	r.SetClock(clock)

	processed := make(chan string, 5)

	r.Register("parallel", func(ctx context.Context, job Job) Result {
		processed <- job.ID

		return Result{}
	})

	for i := 0; i < 5; i++ {
		store.AddJob(Job{
			ID:           "job-" + string(rune('a'+i)),
			TaskType:     "parallel",
			ScheduledFor: clock.Now(),
		})
	}

	ctx := context.Background()
	r.Tick(ctx)

	close(processed)

	count := 0
	for range processed {
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 jobs processed, got %d", count)
	}
}

func TestRunner_TickNoJobs(t *testing.T) {
	store := NewFakeStore()
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true}, log)

	ctx := context.Background()

	err := r.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick() error = %v", err)
	}

	runs := store.AllRuns()
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestRunner_SetSettings(t *testing.T) {
	r := New(NewFakeStore(), Config{}, &FakeLogger{})

	settings := &fakeSettings{paused: true}
	r.SetSettings(settings)

	if r.settings != settings {
		t.Error("settings not set")
	}
}

func TestRunner_IsPaused(t *testing.T) {
	store := NewFakeStore()
	clock := NewFakeClock(time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC))
	log := &FakeLogger{}

	r := New(store, Config{Enabled: true}, log)
	r.SetClock(clock)
	r.SetSettings(&fakeSettings{paused: true})

	r.Register("test", func(ctx context.Context, job Job) Result {
		return Result{}
	})

	store.AddJob(Job{
		ID:           "job-1",
		TaskType:     "test",
		ScheduledFor: clock.Now(),
	})

	// tick internally checks isPaused
	r.tick(context.Background())

	runs := store.AllRuns()
	if len(runs) != 0 {
		t.Errorf("expected 0 runs when paused, got %d", len(runs))
	}
}

func TestConfigFromRoot(t *testing.T) {
	root := config.New()
	root.Scheduler.Enabled = true
	root.Scheduler.Interval = "30s"
	root.Scheduler.BatchSize = 7
	root.Scheduler.Workers = 2
	root.Scheduler.RetryAttempts = 4
	root.Scheduler.RetryBackoff = "2m"

	got := ConfigFromRoot(root)

	if !got.Enabled {
		t.Error("Enabled not mapped")
	}

	if got.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", got.Interval)
	}

	if got.BatchSize != 7 {
		t.Errorf("BatchSize = %d, want 7", got.BatchSize)
	}

	if got.Workers != 2 {
		t.Errorf("Workers = %d, want 2", got.Workers)
	}

	if got.RetryAttempts != 4 {
		t.Errorf("RetryAttempts = %d, want 4", got.RetryAttempts)
	}

	if got.RetryBackoff != 2*time.Minute {
		t.Errorf("RetryBackoff = %v, want 2m", got.RetryBackoff)
	}
}

func TestNewWithConfig(t *testing.T) {
	root := config.New()
	root.Scheduler.Enabled = true
	root.Scheduler.Interval = "45s"

	settings := &fakeSettings{paused: true}
	r := NewWithConfig(NewFakeStore(), settings, root, &FakeLogger{})

	if !r.cfg.Enabled {
		t.Error("Enabled not mapped from root config")
	}

	if r.cfg.Interval != 45*time.Second {
		t.Errorf("Interval = %v, want 45s", r.cfg.Interval)
	}

	if r.settings != settings {
		t.Error("settings not set from constructor")
	}
}

type fakeSettings struct {
	paused bool
}

func (s *fakeSettings) GetBool(ctx context.Context, key string) (bool, error) {
	if key == SettingPaused {
		return s.paused, nil
	}

	return false, nil
}

func (s *fakeSettings) GetInt(ctx context.Context, key string) (int, error) {
	return 0, nil
}
