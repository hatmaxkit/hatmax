package scheduler

import (
	"context"
	"sync"
	"time"
)

type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{now: t}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type FakeStore struct {
	mu   sync.Mutex
	Jobs []Job
	Runs map[string]*FakeRun
}

type FakeRun struct {
	ID           string
	JobID        string
	ScheduledFor time.Time
	StartedAt    time.Time
	FinishedAt   time.Time
	Status       string
	Output       []byte
	Error        string
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		Runs: make(map[string]*FakeRun),
	}
}

func (s *FakeStore) AddJob(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Jobs = append(s.Jobs, job)
}

func (s *FakeStore) ListDue(ctx context.Context, now time.Time, limit int) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []Job
	for _, j := range s.Jobs {
		if !j.ScheduledFor.After(now) {
			due = append(due, j)
			if len(due) >= limit {
				break
			}
		}
	}
	return due, nil
}

func (s *FakeStore) CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Runs[runID] = &FakeRun{
		ID:           runID,
		JobID:        jobID,
		ScheduledFor: scheduledFor,
		Status:       "pending",
	}
	return nil
}

func (s *FakeStore) MarkRunning(ctx context.Context, runID string, startedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run, ok := s.Runs[runID]; ok {
		run.Status = "running"
		run.StartedAt = startedAt
	}
	return nil
}

func (s *FakeStore) MarkSuccess(ctx context.Context, runID string, finishedAt time.Time, output []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run, ok := s.Runs[runID]; ok {
		run.Status = "success"
		run.FinishedAt = finishedAt
		run.Output = output
	}
	return nil
}

func (s *FakeStore) MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run, ok := s.Runs[runID]; ok {
		run.Status = "failed"
		run.FinishedAt = finishedAt
		run.Error = errMsg
	}
	return nil
}

func (s *FakeStore) UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.Jobs {
		if j.ID == jobID {
			s.Jobs[i].ScheduledFor = nextRun
			break
		}
	}
	return nil
}

func (s *FakeStore) GetRun(runID string) *FakeRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Runs[runID]
}

func (s *FakeStore) AllRuns() []*FakeRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	runs := make([]*FakeRun, 0, len(s.Runs))
	for _, r := range s.Runs {
		runs = append(runs, r)
	}
	return runs
}

type FakeLogger struct {
	mu       sync.Mutex
	Messages []string
}

func (l *FakeLogger) Info(v ...any)                  {}
func (l *FakeLogger) Infof(format string, a ...any)  {}
func (l *FakeLogger) Error(v ...any)                 {}
func (l *FakeLogger) Errorf(format string, a ...any) {}
