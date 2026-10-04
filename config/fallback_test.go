// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import "testing"

// OTP consumers need finite admission but no WebAuthn RP identity.
func TestFallbackBounds(t *testing.T) {
	tests := []struct {
		name  string
		cfg   FallbackConfig
		valid bool
	}{
		{name: "defaults without RP", cfg: FallbackConfig{Issuer: "Example"}, valid: true},
		{name: "strict step", cfg: FallbackConfig{Issuer: "Example", StrictStep: true, BackupCodes: 10}, valid: true},
		{name: "missing issuer"},
		{name: "control issuer", cfg: FallbackConfig{Issuer: "Example\n"}},
		{name: "negative codes", cfg: FallbackConfig{Issuer: "Example", BackupCodes: -1}},
		{name: "excessive codes", cfg: FallbackConfig{Issuer: "Example", BackupCodes: 11}},
		{name: "excessive capacity", cfg: FallbackConfig{Issuer: "Example", Limits: AuthenticatorConfig{MaxPending: 11}}},
		{name: "short lease", cfg: FallbackConfig{Issuer: "Example", Limits: AuthenticatorConfig{Lease: "1ms"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings, err := test.cfg.Settings()
			if (err == nil) != test.valid {
				t.Fatalf("settings: %v", err)
			}

			if test.valid && (settings.Skew > 1 || settings.BackupCodes < 1 || settings.BackupCodes > 10 || settings.RPID != "") {
				t.Fatal("invalid finite snapshot")
			}
		})
	}
}
