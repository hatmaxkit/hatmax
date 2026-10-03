// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"fmt"
	"time"
)

// SessionSettings is the validated immutable lifecycle configuration.
type SessionSettings struct {
	TTL              time.Duration
	InactivityTTL    time.Duration
	ActivityInterval time.Duration
	Timeout          time.Duration
	CleanupBatch     int
}

// SessionSettings resolves absent defaults and rejects malformed or inconsistent
// lifecycle settings. Stored durations use whole microseconds with PostgreSQL.
func (c AuthConfig) SessionSettings() (SessionSettings, error) {
	values := []struct {
		name, raw, fallback string
		max                 time.Duration
	}{
		{"session_ttl", c.SessionTTL, "24h", 30 * 24 * time.Hour},
		{"session_inactivity_ttl", c.SessionInactivityTTL, "30m", 30 * 24 * time.Hour},
		{"session_activity_interval", c.SessionActivityInterval, "1m", 5 * time.Minute},
		{"session_timeout", c.SessionTimeout, "5s", 30 * time.Second},
	}

	resolved := make([]time.Duration, len(values))
	for i, value := range values {
		raw := value.raw
		if raw == "" {
			raw = value.fallback
		}

		duration, err := time.ParseDuration(raw)
		if err != nil || duration <= 0 || duration > value.max || (i < 3 && duration%time.Microsecond != 0) {
			return SessionSettings{}, fmt.Errorf("auth.%s is invalid or exceeds its supported duration", value.name)
		}

		resolved[i] = duration
	}

	if resolved[0] < time.Minute || resolved[1] < time.Minute || resolved[1] > resolved[0] || resolved[2] < time.Second || resolved[2] > resolved[1]/4 {
		return SessionSettings{}, fmt.Errorf("auth session lifetimes and activity interval are inconsistent")
	}

	batch := c.SessionCleanupBatch
	if batch == 0 {
		batch = 1000
	}

	if batch < 1 || batch > 1000 {
		return SessionSettings{}, fmt.Errorf("auth.session_cleanup_batch must be between 1 and 1000")
	}

	return SessionSettings{TTL: resolved[0], InactivityTTL: resolved[1], ActivityInterval: resolved[2], Timeout: resolved[3], CleanupBatch: batch}, nil
}
