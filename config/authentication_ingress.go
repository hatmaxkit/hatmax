// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"errors"
	"time"
)

// AuthenticationIngressConfig bounds common authentication HTTP admission.
type AuthenticationIngressConfig struct {
	PeerRequests   int    `koanf:"peer_requests"`
	PeerWindow     string `koanf:"peer_window"`
	MaxPeers       int    `koanf:"max_peers"`
	MaxActive      int    `koanf:"max_active"`
	CleanupBatch   int    `koanf:"cleanup_batch"`
	Acknowledgment string `koanf:"acknowledgment"`
}

// AuthenticationIngressSettings is an immutable finite work/response snapshot.
type AuthenticationIngressSettings struct {
	PeerRequests, MaxPeers, MaxActive, CleanupBatch int
	PeerWindow, Acknowledgment, WorkTimeout         time.Duration
}

// AuthenticationIngressSettings includes actual service work plus observation/margin.
func (c *Config) AuthenticationIngressSettings() (AuthenticationIngressSettings, error) {
	s := AuthenticationIngressSettings{PeerRequests: 12, PeerWindow: time.Minute, MaxPeers: 1024, MaxActive: 32, CleanupBatch: 128, Acknowledgment: 6 * time.Second, WorkTimeout: 5 * time.Second}
	if c == nil {
		return s, errors.New("authentication ingress configuration is required")
	}

	for _, value := range []string{c.Auth.PasswordTimeout, c.Authenticator.Timeout, c.Recovery.Timeout} {
		if value == "" {
			continue
		}

		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 || d > 30*time.Second {
			return s, errors.New("invalid authentication work deadline")
		}

		s.WorkTimeout = max(s.WorkTimeout, d)
	}

	for _, v := range []struct {
		value   int
		target  *int
		maximum int
	}{{c.AuthenticationIngress.PeerRequests, &s.PeerRequests, 1000}, {c.AuthenticationIngress.MaxPeers, &s.MaxPeers, 10000}, {c.AuthenticationIngress.MaxActive, &s.MaxActive, 128}, {c.AuthenticationIngress.CleanupBatch, &s.CleanupBatch, 1000}} {
		if v.value != 0 {
			if v.value < 1 || v.value > v.maximum {
				return s, errors.New("invalid authentication ingress limit")
			}

			*v.target = v.value
		}
	}

	if c.AuthenticationIngress.PeerWindow != "" {
		d, err := time.ParseDuration(c.AuthenticationIngress.PeerWindow)
		if err != nil || d < time.Second || d > time.Hour {
			return s, errors.New("invalid authentication peer window")
		}

		s.PeerWindow = d
	}

	if c.AuthenticationIngress.Acknowledgment != "" {
		d, err := time.ParseDuration(c.AuthenticationIngress.Acknowledgment)
		if err != nil {
			return s, errors.New("invalid authentication acknowledgment")
		}

		s.Acknowledgment = d
	}

	if s.Acknowledgment < s.WorkTimeout+200*time.Millisecond || s.Acknowledgment > 31*time.Second {
		return s, errors.New("authentication acknowledgment must include work, observation and margin")
	}

	return s, nil
}
