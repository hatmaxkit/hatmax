// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package postgres

import (
	"testing"
	"time"

	"hatmax.adrianpk.com/scheduler"
)

// Stored schedule decoding must reject invalid rules before the runner invokes a handler.
func TestStoredSchedule(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, spec, zone string
		want             time.Time
		invalid          bool
	}{
		{name: "once"},
		{name: "whitespace", spec: " \t "},
		{name: "interval", spec: `{"type":"interval","every":"30m"}`, want: from.Add(30 * time.Minute)},
		{name: "daily", spec: `{"type":"daily","hour":9}`, zone: "", want: from.Add(9 * time.Hour)},
		{name: "timezone", spec: `{"type":"daily","hour":9}`, zone: "America/New_York", want: from.Add(14 * time.Hour)},
		{name: "weekly", spec: `{"type":"weekly","day":5,"hour":9}`, zone: "UTC", want: from.Add(33 * time.Hour)},
		{name: "syntax", spec: `{`, invalid: true},
		{name: "type", spec: `{"type":"cron"}`, invalid: true},
		{name: "unknown_field", spec: `{"type":"daily","command":"run"}`, invalid: true},
		{name: "extra_json", spec: `{"type":"daily"} {}`, invalid: true},
		{name: "null", spec: `null`, invalid: true},
		{name: "zero_interval", spec: `{"type":"interval","every":"0s"}`, invalid: true},
		{name: "negative_interval", spec: `{"type":"interval","every":"-1h"}`, invalid: true},
		{name: "bad_interval", spec: `{"type":"interval","every":"bad"}`, invalid: true},
		{name: "interval_calendar", spec: `{"type":"interval","every":"1h","hour":1}`, invalid: true},
		{name: "hour", spec: `{"type":"daily","hour":24}`, invalid: true},
		{name: "minute", spec: `{"type":"daily","minute":60}`, invalid: true},
		{name: "daily_day", spec: `{"type":"daily","day":1}`, invalid: true},
		{name: "calendar_interval", spec: `{"type":"daily","every":"1h"}`, invalid: true},
		{name: "weekday", spec: `{"type":"weekly","day":7}`, invalid: true},
		{name: "zone", spec: `{"type":"daily"}`, zone: "Invalid/Zone", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := parseSchedule(tc.spec, tc.zone)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid schedule accepted")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if tc.want.IsZero() {
				if schedule != nil {
					t.Fatal("empty schedule is not a one-shot job")
				}
			} else if schedule == nil || !schedule.Next(from).Equal(tc.want) {
				t.Fatalf("decoded schedule = %v, want next %v", schedule, tc.want)
			}
		})
	}
}

// A malformed stored schedule must not produce a partially populated due batch.
func TestInvalidStoredJob(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	store, _ := slotStore(t, "postgres", scheduler.Job{ID: "job", TaskType: "task", ScheduledFor: now}, `{"type":"interval","every":"0s"}`)

	jobs, err := store.ListDue(ctx, now, 1)
	if err == nil || jobs != nil {
		t.Fatalf("invalid stored job produced due work: %v / %v", jobs, err)
	}
}
