// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"database/sql"
	"errors"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"testing"
	"time"
)

// Exact equality permits a fresh window only after both trusted boundaries end.
func TestCredentialAdmissionWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		window, cooldown time.Duration
		future           bool
		want             error
	}{
		{"exact equality", 0, 0, false, nil},
		{"window remains", time.Microsecond, 0, false, core.ErrCredentialAdmissionAttempts},
		{"cooldown remains", 0, time.Microsecond, false, core.ErrCredentialAdmissionAttempts},
		{"future state", 0, 0, true, core.ErrCredentialAdmissionState},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start := now.Add(-10 * time.Minute)
			if tc.future {
				start = now.Add(time.Microsecond)
			}

			row := dal.CredentialAdmission{Purpose: 2, WindowStart: start, WindowEnd: now.Add(tc.window), CooldownUs: (10 * time.Minute).Microseconds(), AttemptLimit: 10, Attempts: 10, CooldownUntil: sql.NullTime{Time: now.Add(tc.cooldown), Valid: true}}

			got, err := advanceCredentialRecord(row, now, 10, 10*time.Minute, 10*time.Minute)
			if !errors.Is(err, tc.want) {
				t.Fatalf("boundary: %v", err)
			}

			if tc.want == nil && (got.Attempts != 1 || got.WindowStart != now || got.CooldownUntil.Valid) {
				t.Fatal("equality did not renew")
			}

			if tc.want != nil && (got.Attempts != row.Attempts || got.WindowStart != row.WindowStart || got.CooldownUntil != row.CooldownUntil) {
				t.Fatal("denial changed window")
			}
		})
	}
}
