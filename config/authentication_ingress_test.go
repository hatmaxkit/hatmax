// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"testing"
	"time"
)

// Admission captures finite settings and rejects acknowledgment shorter than work.
func TestAuthenticationIngressSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*Config)
		invalid bool
	}{
		{"defaults", func(*Config) {}, false},
		{"negative requests", func(c *Config) { c.AuthenticationIngress.PeerRequests = -1 }, true},
		{"large requests", func(c *Config) { c.AuthenticationIngress.PeerRequests = 1001 }, true},
		{"negative peers", func(c *Config) { c.AuthenticationIngress.MaxPeers = -1 }, true},
		{"large peers", func(c *Config) { c.AuthenticationIngress.MaxPeers = 10001 }, true},
		{"negative active", func(c *Config) { c.AuthenticationIngress.MaxActive = -1 }, true},
		{"large active", func(c *Config) { c.AuthenticationIngress.MaxActive = 129 }, true},
		{"negative batch", func(c *Config) { c.AuthenticationIngress.CleanupBatch = -1 }, true},
		{"large batch", func(c *Config) { c.AuthenticationIngress.CleanupBatch = 1001 }, true},
		{"short peer window", func(c *Config) { c.AuthenticationIngress.PeerWindow = "1ms" }, true},
		{"large peer window", func(c *Config) { c.AuthenticationIngress.PeerWindow = "2h" }, true},
		{"malformed peer window", func(c *Config) { c.AuthenticationIngress.PeerWindow = "bad" }, true},
		{"short acknowledgment", func(c *Config) { c.AuthenticationIngress.Acknowledgment = "5s" }, true},
		{"large acknowledgment", func(c *Config) { c.AuthenticationIngress.Acknowledgment = "32s" }, true},
		{"invalid acknowledgment", func(c *Config) { c.AuthenticationIngress.Acknowledgment = "bad" }, true},
		{"credential work", func(c *Config) { c.Auth.PasswordTimeout = "10s" }, true},
		{"factor work", func(c *Config) { c.Authenticator.Timeout = "10s" }, true},
		{"recovery work", func(c *Config) { c.Recovery.Timeout = "10s" }, true},
		{"invalid work", func(c *Config) { c.Auth.PasswordTimeout = "bad" }, true},
		{"excessive work", func(c *Config) { c.Auth.PasswordTimeout = "31s" }, true},
		{"explicit long target", func(c *Config) { c.Authenticator.Timeout = "30s"; c.AuthenticationIngress.Acknowledgment = "30.2s" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := New()
			tc.change(cfg)

			s, err := cfg.AuthenticationIngressSettings()
			if (err != nil) != tc.invalid {
				t.Fatalf("configuration error = %v", err)
			}

			if err == nil && (s.Acknowledgment < s.WorkTimeout+200*time.Millisecond || s.MaxActive < 1 || s.MaxPeers < 1) {
				t.Fatal("finite snapshot invalid")
			}
		})
	}

	var cfg *Config

	_, err := cfg.AuthenticationIngressSettings()
	if err == nil {
		t.Fatal("nil configuration accepted")
	}
}
