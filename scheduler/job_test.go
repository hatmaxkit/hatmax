package scheduler

import (
	"errors"
	"testing"
	"time"
)

var errJobTest = errors.New("test error")

func TestResult_Failed(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{"no error", Result{Output: map[string]any{"ok": true}}, false},
		{"with error", Result{Err: errJobTest}, true},
		{"nil output no error", Result{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Failed(); got != tt.want {
				t.Errorf("Failed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaily_Next(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	base := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		d    Daily
		from time.Time
		want time.Time
	}{
		{
			name: "same day future",
			d:    Daily{Hour: 14, Minute: 30},
			from: base,
			want: time.Date(2026, 2, 18, 14, 30, 0, 0, time.UTC),
		},
		{
			name: "same day past rolls to tomorrow",
			d:    Daily{Hour: 8, Minute: 0},
			from: base,
			want: time.Date(2026, 2, 19, 8, 0, 0, 0, time.UTC),
		},
		{
			name: "with timezone",
			d:    Daily{Hour: 9, Minute: 0, TZ: loc},
			from: time.Date(2026, 2, 18, 15, 0, 0, 0, time.UTC), // 10 AM in NY
			want: time.Date(2026, 2, 19, 14, 0, 0, 0, time.UTC), // 9 AM NY next day
		},
		{
			name: "nil timezone uses UTC",
			d:    Daily{Hour: 12, Minute: 0, TZ: nil},
			from: base,
			want: time.Date(2026, 2, 18, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.d.Next(tt.from)
			if !got.Equal(tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWeekly_Next(t *testing.T) {
	// Wednesday Feb 18, 2026
	base := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		w    Weekly
		from time.Time
		want time.Time
	}{
		{
			name: "same day future time",
			w:    Weekly{Day: time.Wednesday, Hour: 14, Minute: 0},
			from: base,
			want: time.Date(2026, 2, 18, 14, 0, 0, 0, time.UTC),
		},
		{
			name: "same day past time rolls to next week",
			w:    Weekly{Day: time.Wednesday, Hour: 8, Minute: 0},
			from: base,
			want: time.Date(2026, 2, 25, 8, 0, 0, 0, time.UTC),
		},
		{
			name: "future day this week",
			w:    Weekly{Day: time.Friday, Hour: 9, Minute: 0},
			from: base,
			want: time.Date(2026, 2, 20, 9, 0, 0, 0, time.UTC),
		},
		{
			name: "past day rolls to next week",
			w:    Weekly{Day: time.Monday, Hour: 9, Minute: 0},
			from: base,
			want: time.Date(2026, 2, 23, 9, 0, 0, 0, time.UTC),
		},
		{
			name: "nil timezone uses UTC",
			w:    Weekly{Day: time.Thursday, Hour: 10, Minute: 0, TZ: nil},
			from: base,
			want: time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.w.Next(tt.from)
			if !got.Equal(tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterval_Next(t *testing.T) {
	base := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		every time.Duration
		want  time.Time
	}{
		{"1 hour", time.Hour, base.Add(time.Hour)},
		{"30 minutes", 30 * time.Minute, base.Add(30 * time.Minute)},
		{"1 day", 24 * time.Hour, base.AddDate(0, 0, 1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Interval{Every: tt.every}

			got := i.Next(base)
			if !got.Equal(tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}
