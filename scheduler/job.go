// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package scheduler

import (
	"encoding/json"
	"time"
)

type Job struct {
	ID           string
	Name         string
	TaskType     string
	Payload      json.RawMessage
	ScheduledFor time.Time
	// RunID identifies a handler's slot run. Stores populate it for retry admission.
	RunID   string
	Attempt int
	// MaxAttempts is the slot's fixed total budget, retained across retries.
	MaxAttempts int
	Metadata    map[string]string
	// Schedule is nil for one-shot jobs; stores advance recurring jobs on completion.
	Schedule Schedule
}

type Result struct {
	Output map[string]any
	Err    error
}

func (r Result) Failed() bool {
	return r.Err != nil
}

type Schedule interface {
	Next(from time.Time) time.Time
}

type Daily struct {
	Hour   int
	Minute int
	TZ     *time.Location
}

func (d Daily) Next(from time.Time) time.Time {
	loc := d.TZ
	if loc == nil {
		loc = time.UTC
	}

	t := from.In(loc)

	next := time.Date(t.Year(), t.Month(), t.Day(), d.Hour, d.Minute, 0, 0, loc)
	if !next.After(from) {
		next = next.AddDate(0, 0, 1)
	}

	return next.UTC()
}

type Weekly struct {
	Day    time.Weekday
	Hour   int
	Minute int
	TZ     *time.Location
}

func (w Weekly) Next(from time.Time) time.Time {
	loc := w.TZ
	if loc == nil {
		loc = time.UTC
	}

	t := from.In(loc)
	next := time.Date(t.Year(), t.Month(), t.Day(), w.Hour, w.Minute, 0, 0, loc)

	daysUntil := (int(w.Day) - int(t.Weekday()) + 7) % 7
	if daysUntil == 0 && !next.After(from) {
		daysUntil = 7
	}

	next = next.AddDate(0, 0, daysUntil)

	return next.UTC()
}

type Interval struct {
	Every time.Duration
}

func (i Interval) Next(from time.Time) time.Time {
	return from.Add(i.Every)
}
