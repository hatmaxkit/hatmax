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
	Attempt      int
	Metadata     map[string]string
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
