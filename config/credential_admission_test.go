// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Settings reject invalid limits and keep defaults finite for durable storage.
func TestCredentialAdmissionSettings(t *testing.T) {
	tests := []struct {
		name   string
		change func(*CredentialAdmissionConfig)
		bad    bool
	}{
		{"defaults", func(*CredentialAdmissionConfig) {}, false},
		{"negative attempts", func(c *CredentialAdmissionConfig) { c.PasswordAttempts = -1 }, true},
		{"proof maximum", func(c *CredentialAdmissionConfig) { c.PasswordAttempts = 21 }, true},
		{"registration maximum", func(c *CredentialAdmissionConfig) { c.RegistrationAttempts = 11 }, true},
		{"capacity minimum", func(c *CredentialAdmissionConfig) { c.MaxIdentities = 99 }, true},
		{"capacity maximum", func(c *CredentialAdmissionConfig) { c.MaxIdentities = 100001 }, true},
		{"cleanup maximum", func(c *CredentialAdmissionConfig) { c.CleanupBatch = 1001 }, true},
		{"window syntax", func(c *CredentialAdmissionConfig) { c.PasswordWindow = "later" }, true},
		{"window minimum", func(c *CredentialAdmissionConfig) { c.PasswordWindow = "59s" }, true},
		{"cooldown maximum", func(c *CredentialAdmissionConfig) { c.RegistrationCooldown = "61m" }, true},
		{"timeout minimum", func(c *CredentialAdmissionConfig) { c.Timeout = "99ms" }, true},
		{"precision", func(c *CredentialAdmissionConfig) { c.Timeout = "100000001ns" }, true},
		{"explicit bounds", func(c *CredentialAdmissionConfig) {
			c.PasswordAttempts = 20
			c.MaxIdentities = 100
			c.Timeout = "100ms"
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := CredentialAdmissionConfig{}
			tc.change(&c)

			got, err := c.CredentialAdmissionSettings()
			if (err != nil) != tc.bad {
				t.Fatalf("settings: %v", err)
			}

			if !tc.bad && got.Check() != nil {
				t.Fatal("resolved snapshot invalid")
			}
		})
	}

	defaults, err := (CredentialAdmissionConfig{}).CredentialAdmissionSettings()
	if err != nil || defaults.PasswordAttempts != 10 || defaults.PasswordWindow != 10*time.Minute || defaults.RegistrationAttempts != 3 || defaults.MaxIdentities != 10000 || defaults.Timeout != time.Second {
		t.Fatal("unsafe defaults")
	}

	defaults.PasswordAttempts = 0
	if defaults.Check() == nil {
		t.Fatal("unresolved snapshot accepted")
	}
}
