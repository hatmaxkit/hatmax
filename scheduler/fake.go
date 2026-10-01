// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package scheduler

import (
	"context"
	"fmt"
	"sort"
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
	mu      sync.Mutex
	Jobs    []Job
	Runs    map[string]*FakeRun
	retired map[string]bool
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
		Runs:    make(map[string]*FakeRun),
		retired: make(map[string]bool),
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
		if !s.retired[j.ID] && !j.ScheduledFor.After(now) && !s.hasSlot(j.ID, j.ScheduledFor) {
			due = append(due, j)
		}
	}

	sort.SliceStable(due, func(i, j int) bool { return due[i].ScheduledFor.Before(due[j].ScheduledFor) })

	if limit >= 0 && len(due) > limit {
		due = due[:limit]
	}

	return due, nil
}

func (s *FakeStore) CreateRun(ctx context.Context, jobID, runID string, scheduledFor time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.Runs[runID]; exists || s.hasSlot(jobID, scheduledFor) {
		return fmt.Errorf("scheduler: run or job slot already exists")
	}

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

	return s.finish(runID, finishedAt, "success", output, "")
}

func (s *FakeStore) MarkFailed(ctx context.Context, runID string, finishedAt time.Time, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.finish(runID, finishedAt, "failed", nil, errMsg)
}

// The caller holds mu so the outcome and job transition share one atomic boundary.
func (s *FakeStore) finish(runID string, finishedAt time.Time, status string, output []byte, detail string) error {
	run, ok := s.Runs[runID]
	if !ok || run.Status == "success" || run.Status == "failed" {
		return nil
	}

	for index, job := range s.Jobs {
		if job.ID != run.JobID || s.retired[job.ID] || !job.ScheduledFor.Equal(run.ScheduledFor) {
			continue
		}

		if job.Schedule == nil {
			s.retired[job.ID] = true
		} else {
			next := job.Schedule.Next(finishedAt)
			if !next.After(finishedAt) || !next.After(run.ScheduledFor) {
				return fmt.Errorf("scheduler: next slot must follow completion")
			}

			s.Jobs[index].ScheduledFor = next
		}

		break
	}

	run.Status = status
	run.FinishedAt = finishedAt
	run.Output = output
	run.Error = detail

	return nil
}

// The caller holds mu. All run states reserve their scheduled slot.
func (s *FakeStore) hasSlot(jobID string, scheduledFor time.Time) bool {
	for _, run := range s.Runs {
		if run.JobID == jobID && run.ScheduledFor.Equal(scheduledFor) {
			return true
		}
	}

	return false
}

func (s *FakeStore) UpdateNextRun(ctx context.Context, jobID string, lastRun, nextRun time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, j := range s.Jobs {
		if j.ID == jobID {
			if nextRun.IsZero() {
				s.retired[j.ID] = true
			} else {
				delete(s.retired, j.ID)
				s.Jobs[i].ScheduledFor = nextRun
			}

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
