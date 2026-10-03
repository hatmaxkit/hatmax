// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Construction rejects invalid durations and unsafe lifecycle relationships.
func TestSessionSettings(t *testing.T) {
	tests := []struct {
		name   string
		config AuthConfig
		valid  bool
	}{
		{name: "defaults", valid: true},
		{name: "minimum", config: AuthConfig{SessionRecentProofAge: "1m", SessionTTL: "1m", SessionInactivityTTL: "1m", SessionActivityInterval: "15s", SessionCleanupBatch: 1}, valid: true},
		{name: "maximum", config: AuthConfig{SessionTTL: "720h", SessionInactivityTTL: "720h", SessionActivityInterval: "5m", SessionTimeout: "30s", SessionCleanupBatch: 1000}, valid: true},
		{name: "malformed", config: AuthConfig{SessionTTL: "forever"}},
		{name: "zero duration", config: AuthConfig{SessionTTL: "0s"}},
		{name: "negative", config: AuthConfig{SessionTimeout: "-1s"}},
		{name: "overflow", config: AuthConfig{SessionTTL: "99999999999999999h"}},
		{name: "short absolute", config: AuthConfig{SessionTTL: "59s"}},
		{name: "long absolute", config: AuthConfig{SessionTTL: "721h"}},
		{name: "idle exceeds absolute", config: AuthConfig{SessionTTL: "10m"}},
		{name: "short idle", config: AuthConfig{SessionInactivityTTL: "59s"}},
		{name: "subsecond cadence", config: AuthConfig{SessionActivityInterval: "999ms"}},
		{name: "long cadence", config: AuthConfig{SessionActivityInterval: "6m"}},
		{name: "cadence ratio", config: AuthConfig{SessionInactivityTTL: "1m", SessionActivityInterval: "16s"}},
		{name: "storage precision", config: AuthConfig{SessionTTL: "24h1ns"}},
		{name: "timeout cap", config: AuthConfig{SessionTimeout: "31s"}},
		{name: "recent minimum", config: AuthConfig{SessionRecentProofAge: "1s", SessionMaxPerSubject: 1, SessionPageSize: 1}, valid: true},
		{name: "recent maximum", config: AuthConfig{SessionRecentProofAge: "24h", SessionMaxPerSubject: 100, SessionPageSize: 100}, valid: true},
		{name: "recent short", config: AuthConfig{SessionRecentProofAge: "999ms"}},
		{name: "recent precision", config: AuthConfig{SessionRecentProofAge: "1s1ns"}},
		{name: "recent exceeds ttl", config: AuthConfig{SessionRecentProofAge: "25h"}},
		{name: "capacity negative", config: AuthConfig{SessionMaxPerSubject: -1}},
		{name: "capacity exceeds", config: AuthConfig{SessionMaxPerSubject: 101}},
		{name: "page negative", config: AuthConfig{SessionPageSize: -1}},
		{name: "page exceeds", config: AuthConfig{SessionPageSize: 101}},
		{name: "negative batch", config: AuthConfig{SessionCleanupBatch: -1}},
		{name: "batch cap", config: AuthConfig{SessionCleanupBatch: 1001}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings, err := test.config.SessionSettings()
			if (err == nil) != test.valid {
				t.Fatalf("settings: %v", err)
			}

			if !test.valid && settings != (SessionSettings{}) {
				t.Fatal("invalid settings escaped")
			}

			if test.name == "defaults" && (settings.TTL != 24*time.Hour || settings.InactivityTTL != 30*time.Minute || settings.ActivityInterval != time.Minute || settings.Timeout != 5*time.Second || settings.CleanupBatch != 1000 || settings.RecentProofAge != 5*time.Minute || settings.MaxPerSubject != 10 || settings.PageSize != 50) {
				t.Fatal("unexpected defaults")
			}
		})
	}
}
