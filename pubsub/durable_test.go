// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package pubsub

import (
	"testing"
	"time"
)

// Explicit history and policy bounds prevent accidentally unbounded delivery.
func TestBoundedDeliveries(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*DurableOptions)
		valid  bool
	}{
		{"defaults", func(*DurableOptions) {}, true},
		{"minimum", func(o *DurableOptions) { o.Policy = DeliveryPolicy{time.Second, 6 * time.Second, 1, time.Second} }, true},
		{"maximum", func(o *DurableOptions) { o.Policy = DeliveryPolicy{10 * time.Minute, 15 * time.Minute, 32, time.Hour} }, true},
		{"history required", func(o *DurableOptions) { o.History = "" }, false},
		{"short deadline", func(o *DurableOptions) { o.Policy.HandlerTimeout = time.Second - time.Millisecond }, false},
		{"long deadline", func(o *DurableOptions) { o.Policy.HandlerTimeout = 10*time.Minute + time.Millisecond }, false},
		{"short lease", func(o *DurableOptions) {
			o.Policy.LeaseDuration = o.Policy.HandlerTimeout + 5*time.Second - time.Millisecond
		}, false},
		{"long lease", func(o *DurableOptions) { o.Policy.LeaseDuration = 15*time.Minute + time.Millisecond }, false},
		{"zero attempts", func(o *DurableOptions) { o.Policy.MaxAttempts = 0 }, false},
		{"excess attempts", func(o *DurableOptions) { o.Policy.MaxAttempts = 33 }, false},
		{"short delay", func(o *DurableOptions) { o.Policy.RetryDelay = time.Second - time.Millisecond }, false},
		{"long delay", func(o *DurableOptions) { o.Policy.RetryDelay = time.Hour + time.Millisecond }, false},
		{"fractional millisecond", func(o *DurableOptions) { o.Policy.RetryDelay += time.Nanosecond }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := DurableOptions{DefaultDeliveryPolicy(), RetainedHistory}
			test.change(&opts)

			err := opts.Validate()
			if (err == nil) != test.valid {
				t.Fatalf("valid=%t error=%v", test.valid, err)
			}
		})
	}
}
