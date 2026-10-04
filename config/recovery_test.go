// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Invalid configured limits fail construction rather than silently choosing defaults.
func TestRecoverySettings(t *testing.T) {
	defaults, err := (RecoveryConfig{}).RecoverySettings()
	if err != nil || defaults.VerificationTTL != 24*time.Hour || defaults.ResetTTL != time.Hour || defaults.TokenAttempts != 5 {
		t.Fatal("invalid defaults")
	}

	for _, tc := range []struct {
		name string
		cfg  RecoveryConfig
	}{
		{"invalid duration", RecoveryConfig{VerificationTTL: "oops"}},
		{"short verification", RecoveryConfig{VerificationTTL: "59s"}},
		{"long verification", RecoveryConfig{VerificationTTL: "25h"}},
		{"long reset", RecoveryConfig{ResetTTL: "61m"}},
		{"short timeout", RecoveryConfig{Timeout: "500ms"}},
		{"long timeout", RecoveryConfig{Timeout: "31s", Lease: "31s"}},
		{"short lease", RecoveryConfig{Lease: "1s"}},
		{"submicrosecond", RecoveryConfig{VerificationTTL: "1m1ns"}},
		{"negative attempts", RecoveryConfig{TokenAttempts: -1}},
		{"token attempts", RecoveryConfig{TokenAttempts: 11}},
		{"issue attempts", RecoveryConfig{IssuanceAttempts: 11}},
		{"completion attempts", RecoveryConfig{CompletionAttempts: 21}},
		{"cleanup batch", RecoveryConfig{CleanupBatch: 1001}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.cfg.RecoverySettings()
			if err == nil {
				t.Fatal("invalid settings accepted")
			}
		})
	}

	cfg := RecoveryConfig{VerificationTTL: "1m", ResetTTL: "1m", Timeout: "30s", Lease: "30s", TokenAttempts: 10, IssuanceAttempts: 10, CompletionAttempts: 20, CleanupBatch: 1}

	_, err = cfg.RecoverySettings()
	if err != nil {
		t.Fatal(err)
	}

	base := New()

	base.Recovery = RecoveryConfig{TokenAttempts: -1}
	if base.Validate() == nil {
		t.Fatal("config validation ignored recovery policy")
	}
}
