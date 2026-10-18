// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Durable execution is opt-in; invalid bounds and unsupported stores fail before
// work starts while existing custom stores remain usable through legacy calls.
func TestLeasedRuns(t *testing.T) {
	t.Run("policy bounds", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			change func(*ExecutionPolicy)
			valid  bool
		}{
			{"default", func(*ExecutionPolicy) {}, true},
			{"minimum", func(p *ExecutionPolicy) {
				p.HandlerTimeout = time.Second
				p.LeaseDuration = 6 * time.Second
				p.MaxAttempts = 1
				p.RetryDelay = time.Second
			}, true},
			{"maximum", func(p *ExecutionPolicy) {
				p.HandlerTimeout = 10 * time.Minute
				p.LeaseDuration = 15 * time.Minute
				p.MaxAttempts = 32
				p.RetryDelay = time.Hour
			}, true},
			{"short handler", func(p *ExecutionPolicy) { p.HandlerTimeout = time.Second - time.Millisecond }, false},
			{"long handler", func(p *ExecutionPolicy) { p.HandlerTimeout = 10*time.Minute + time.Millisecond }, false},
			{"short lease", func(p *ExecutionPolicy) { p.LeaseDuration = p.HandlerTimeout + 5*time.Second - time.Millisecond }, false},
			{"long lease", func(p *ExecutionPolicy) { p.LeaseDuration = 15*time.Minute + time.Millisecond }, false},
			{"zero attempts", func(p *ExecutionPolicy) { p.MaxAttempts = 0 }, false},
			{"excess attempts", func(p *ExecutionPolicy) { p.MaxAttempts = 33 }, false},
			{"short delay", func(p *ExecutionPolicy) { p.RetryDelay = time.Second - time.Millisecond }, false},
			{"long delay", func(p *ExecutionPolicy) { p.RetryDelay = time.Hour + time.Millisecond }, false},
			{"fractional millisecond", func(p *ExecutionPolicy) { p.HandlerTimeout += time.Nanosecond }, false},
		} {
			t.Run(test.name, func(t *testing.T) {
				p := DefaultExecutionPolicy()
				test.change(&p)

				err := p.Validate()
				if (err == nil) != test.valid {
					t.Fatalf("policy validity = %v", err)
				}
			})
		}
	})
	t.Run("capability before work", func(t *testing.T) {
		for _, enabled := range []bool{false, true} {
			runner := New(NewFakeStore(), Config{Enabled: enabled}, &FakeLogger{}, WithDurableExecution(DefaultExecutionPolicy()))

			err := runner.Start(context.Background())
			if !errors.Is(err, ErrDurableUnsupported) {
				t.Fatalf("Start: %v", err)
			}

			err = runner.Tick(context.Background())
			if !errors.Is(err, ErrDurableUnsupported) {
				t.Fatalf("Tick: %v", err)
			}
		}
	})
}
