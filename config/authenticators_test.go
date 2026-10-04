// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Constructor bounds fail closed and origins are copied rather than retaining
// mutable caller configuration that could change the trusted RP profile.
func TestAuthenticatorSettings(t *testing.T) {
	good := func() AuthenticatorConfig {
		return AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}}
	}

	tests := []struct {
		name   string
		change func(*AuthenticatorConfig)
	}{
		{"missing RP", func(c *AuthenticatorConfig) { c.RPID = "" }},
		{"missing origins", func(c *AuthenticatorConfig) { c.Origins = nil }},
		{"duplicate origins", func(c *AuthenticatorConfig) { c.Origins = append(c.Origins, c.Origins[0]) }},
		{"unrelated origin", func(c *AuthenticatorConfig) { c.Origins = []string{"https://evil-example.com"} }},
		{"insecure origin", func(c *AuthenticatorConfig) {
			c.Origins = []string{"http://example.com"}
			c.LocalhostDevelopment = true
		}},
		{"origin credentials", func(c *AuthenticatorConfig) { c.Origins = []string{"https://user@example.com"} }},
		{"origin path", func(c *AuthenticatorConfig) { c.Origins = []string{"https://example.com/path"} }},
		{"pending TTL", func(c *AuthenticatorConfig) { c.PendingTTL = "11m" }},
		{"duration syntax", func(c *AuthenticatorConfig) { c.Timeout = "invalid" }},
		{"timeout", func(c *AuthenticatorConfig) { c.Timeout = "31s" }},
		{"lease", func(c *AuthenticatorConfig) { c.Lease = "4s" }},
		{"pending cap", func(c *AuthenticatorConfig) { c.MaxPending = 11 }},
		{"factor cap", func(c *AuthenticatorConfig) { c.MaxAuthenticators = 21 }},
		{"pending attempts", func(c *AuthenticatorConfig) { c.PendingAttempts = 11 }},
		{"subject attempts", func(c *AuthenticatorConfig) { c.SubjectAttempts = 101 }},
		{"protocol concurrency", func(c *AuthenticatorConfig) { c.MaxConcurrent = 3 }},
		{"cleanup", func(c *AuthenticatorConfig) { c.CleanupBatch = 1001 }},
		{"negative capacity", func(c *AuthenticatorConfig) { c.MaxPending = -1 }},
		{"recent proof", func(c *AuthenticatorConfig) { c.RecentProofAge = "0s" }},
		{"budget window", func(c *AuthenticatorConfig) { c.BudgetWindow = "0s" }},
		{"cooldown", func(c *AuthenticatorConfig) { c.Cooldown = "2h" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := good()
			test.change(&cfg)

			_, err := cfg.EnrollmentSettings()
			if err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}

	cfg := good()

	settings, err := cfg.EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}

	if settings.PendingTTL != 5*time.Minute || settings.MaxConcurrent != 2 || settings.MaxPending != 5 || settings.PendingAttempts != 5 || settings.SubjectAttempts != 10 || settings.RPBinding == ([32]byte{}) {
		t.Fatal("wrong default enrollment bounds")
	}

	cfg.Origins[0] = "https://login.example.com"
	if settings.Origins[0] != "https://example.com" {
		t.Fatal("configuration aliases caller origins")
	}

	changed, err := cfg.EnrollmentSettings()
	if err != nil || changed.RPBinding == settings.RPBinding {
		t.Fatal("origin change did not invalidate RP binding")
	}

	cfg = AuthenticatorConfig{RPID: "localhost", RPName: "Local", Origins: []string{"http://localhost:8080"}}

	_, err = cfg.EnrollmentSettings()
	if err == nil {
		t.Fatal("implicit insecure development origin")
	}

	cfg.LocalhostDevelopment = true

	_, err = cfg.EnrollmentSettings()
	if err != nil {
		t.Fatal(err)
	}
}
