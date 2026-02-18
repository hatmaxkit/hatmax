package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestFakeClock(t *testing.T) {
	base := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	c := NewFakeClock(base)

	if !c.Now().Equal(base) {
		t.Errorf("Now() = %v, want %v", c.Now(), base)
	}

	newTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	c.Set(newTime)
	if !c.Now().Equal(newTime) {
		t.Errorf("after Set(), Now() = %v, want %v", c.Now(), newTime)
	}

	c.Advance(time.Hour)
	expected := newTime.Add(time.Hour)
	if !c.Now().Equal(expected) {
		t.Errorf("after Advance(), Now() = %v, want %v", c.Now(), expected)
	}
}

func TestFakeStore_AddJob(t *testing.T) {
	s := NewFakeStore()

	job := Job{ID: "test-1", TaskType: "email"}
	s.AddJob(job)

	if len(s.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(s.Jobs))
	}
	if s.Jobs[0].ID != "test-1" {
		t.Errorf("expected job ID test-1, got %s", s.Jobs[0].ID)
	}
}

func TestFakeStore_ListDue(t *testing.T) {
	s := NewFakeStore()
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	s.AddJob(Job{ID: "past", ScheduledFor: now.Add(-time.Hour)})
	s.AddJob(Job{ID: "now", ScheduledFor: now})
	s.AddJob(Job{ID: "future", ScheduledFor: now.Add(time.Hour)})

	ctx := context.Background()
	due, err := s.ListDue(ctx, now, 10)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}

	if len(due) != 2 {
		t.Errorf("expected 2 due jobs, got %d", len(due))
	}
}

func TestFakeStore_ListDueWithLimit(t *testing.T) {
	s := NewFakeStore()
	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	for i := range 5 {
		s.AddJob(Job{ID: string(rune('a' + i)), ScheduledFor: now.Add(-time.Hour)})
	}

	ctx := context.Background()
	due, _ := s.ListDue(ctx, now, 2)

	if len(due) != 2 {
		t.Errorf("expected 2 due jobs with limit, got %d", len(due))
	}
}

func TestFakeStore_RunLifecycle(t *testing.T) {
	s := NewFakeStore()
	ctx := context.Background()
	now := time.Now()

	s.CreateRun(ctx, "job-1", "run-1", now)

	run := s.GetRun("run-1")
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "pending" {
		t.Errorf("expected pending status, got %s", run.Status)
	}

	s.MarkRunning(ctx, "run-1", now)
	run = s.GetRun("run-1")
	if run.Status != "running" {
		t.Errorf("expected running status, got %s", run.Status)
	}

	s.MarkSuccess(ctx, "run-1", now, []byte(`{"ok":true}`))
	run = s.GetRun("run-1")
	if run.Status != "success" {
		t.Errorf("expected success status, got %s", run.Status)
	}
}

func TestFakeStore_MarkFailed(t *testing.T) {
	s := NewFakeStore()
	ctx := context.Background()
	now := time.Now()

	s.CreateRun(ctx, "job-1", "run-1", now)
	s.MarkFailed(ctx, "run-1", now, "something went wrong")

	run := s.GetRun("run-1")
	if run.Status != "failed" {
		t.Errorf("expected failed status, got %s", run.Status)
	}
	if run.Error != "something went wrong" {
		t.Errorf("expected error message, got %s", run.Error)
	}
}

func TestFakeStore_UpdateNextRun(t *testing.T) {
	s := NewFakeStore()
	ctx := context.Background()
	now := time.Now()
	next := now.Add(24 * time.Hour)

	s.AddJob(Job{ID: "job-1", ScheduledFor: now})
	s.UpdateNextRun(ctx, "job-1", now, next)

	if !s.Jobs[0].ScheduledFor.Equal(next) {
		t.Errorf("expected ScheduledFor to be updated to %v, got %v", next, s.Jobs[0].ScheduledFor)
	}
}

func TestFakeStore_AllRuns(t *testing.T) {
	s := NewFakeStore()
	ctx := context.Background()
	now := time.Now()

	s.CreateRun(ctx, "job-1", "run-1", now)
	s.CreateRun(ctx, "job-2", "run-2", now)

	runs := s.AllRuns()
	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

func TestFakeStore_GetRunNonExistent(t *testing.T) {
	s := NewFakeStore()

	run := s.GetRun("nonexistent")
	if run != nil {
		t.Error("expected nil for nonexistent run")
	}
}

func TestFakeLogger(t *testing.T) {
	l := &FakeLogger{}

	// These should not panic
	l.Info("test")
	l.Infof("test %s", "arg")
	l.Error("test")
	l.Errorf("test %s", "arg")
}

func TestNoopStore(t *testing.T) {
	s := NoopStore{}
	ctx := context.Background()
	now := time.Now()

	jobs, err := s.ListDue(ctx, now, 10)
	if err != nil {
		t.Errorf("ListDue() error = %v", err)
	}
	if jobs != nil {
		t.Errorf("expected nil jobs from noop")
	}

	if err := s.CreateRun(ctx, "j", "r", now); err != nil {
		t.Errorf("CreateRun() error = %v", err)
	}
	if err := s.MarkRunning(ctx, "r", now); err != nil {
		t.Errorf("MarkRunning() error = %v", err)
	}
	if err := s.MarkSuccess(ctx, "r", now, nil); err != nil {
		t.Errorf("MarkSuccess() error = %v", err)
	}
	if err := s.MarkFailed(ctx, "r", now, "err"); err != nil {
		t.Errorf("MarkFailed() error = %v", err)
	}
	if err := s.UpdateNextRun(ctx, "j", now, now); err != nil {
		t.Errorf("UpdateNextRun() error = %v", err)
	}
}
