// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Finite defaults and hard bounds reject unlimited or malformed callbacks.
func TestSecurityObservationSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     SecurityObservationConfig
		invalid bool
	}{
		{"defaults", SecurityObservationConfig{}, false},
		{"lower bounds", SecurityObservationConfig{Timeout: "1ms", Concurrency: 1}, false},
		{"upper bounds", SecurityObservationConfig{Timeout: "100ms", Concurrency: 16}, false},
		{"zero timeout", SecurityObservationConfig{Timeout: "0s"}, true},
		{"short timeout", SecurityObservationConfig{Timeout: "999us"}, true},
		{"long timeout", SecurityObservationConfig{Timeout: "101ms"}, true},
		{"invalid timeout", SecurityObservationConfig{Timeout: "unknown"}, true},
		{"negative concurrency", SecurityObservationConfig{Concurrency: -1}, true},
		{"large concurrency", SecurityObservationConfig{Concurrency: 17}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings, err := tc.cfg.SecurityObservationSettings()
			if (err != nil) != tc.invalid {
				t.Fatalf("invalid=%v: %v", tc.invalid, err)
			}

			cfg := New()
			cfg.SecurityObservation = tc.cfg

			err = cfg.Validate()
			if (err != nil) != tc.invalid {
				t.Fatalf("configuration validation: %v", err)
			}

			if tc.name == "defaults" && (settings.Timeout != 100*time.Millisecond || settings.Concurrency != 2) {
				t.Fatal("wrong defaults")
			}
		})
	}
}
