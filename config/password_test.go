// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import "testing"

// Credential settings fail at startup rather than silently weakening policy or
// allowing records to select work outside the configured resource envelope.
func TestPasswordSettings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*AuthConfig)
	}{
		{"minimum", func(c *AuthConfig) { c.PasswordMinLen = 14 }},
		{"inverted lengths", func(c *AuthConfig) { c.PasswordMinLen = 65; c.PasswordMaxLen = 64 }},
		{"small maximum", func(c *AuthConfig) { c.PasswordMaxLen = 63 }},
		{"large maximum", func(c *AuthConfig) { c.PasswordMaxLen = 1025 }},
		{"small byte cap", func(c *AuthConfig) { c.PasswordMaxBytes = 4095 }},
		{"large byte cap", func(c *AuthConfig) { c.PasswordMaxBytes = 4097 }},
		{"negative checker timeout", func(c *AuthConfig) { c.PasswordCheckTimeout = "-1s" }},
		{"zero checker timeout", func(c *AuthConfig) { c.PasswordCheckTimeout = "0s" }},
		{"invalid checker timeout", func(c *AuthConfig) { c.PasswordCheckTimeout = "invalid" }},
		{"large checker timeout", func(c *AuthConfig) { c.PasswordCheckTimeout = "31s" }},
		{"negative work timeout", func(c *AuthConfig) { c.PasswordTimeout = "-1s" }},
		{"zero work timeout", func(c *AuthConfig) { c.PasswordTimeout = "0s" }},
		{"large work timeout", func(c *AuthConfig) { c.PasswordTimeout = "31s" }},
		{"memory ceiling", func(c *AuthConfig) { c.ArgonMemoryKiB = 262145 }},
		{"work ceiling", func(c *AuthConfig) { c.ArgonIterations = 11 }},
		{"memory narrowing", func(c *AuthConfig) { c.ArgonMemoryKiB = 1<<32 + 65536 }},
		{"lane narrowing", func(c *AuthConfig) { c.ArgonParallelism = 257 }},
		{"lane ceiling", func(c *AuthConfig) { c.ArgonParallelism = 9 }},
		{"aggregate memory", func(c *AuthConfig) { c.PasswordMaxConcurrent = 9 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := New()
			tc.change(&cfg.Auth)

			_, err := cfg.Auth.PasswordSettings()
			if err == nil {
				t.Fatal("invalid credential settings accepted")
			}
		})
	}
}
