// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"errors"
	"time"
)

// RecoveryConfig supplies finite account-token lifetimes and durable admission limits.
type RecoveryConfig struct {
	VerificationTTL    string `koanf:"verification_ttl"`
	ResetTTL           string `koanf:"reset_ttl"`
	Timeout            string `koanf:"timeout"`
	Lease              string `koanf:"lease"`
	TokenAttempts      int    `koanf:"token_attempts"`
	IssuanceAttempts   int    `koanf:"issuance_attempts"`
	CompletionAttempts int    `koanf:"completion_attempts"`
	CleanupBatch       int    `koanf:"cleanup_batch"`
}

// RecoverySettings is an owned validated snapshot. Windows and retention are fixed.
type RecoverySettings struct {
	VerificationTTL, ResetTTL, Timeout, Lease                         time.Duration
	TokenAttempts, IssuanceAttempts, CompletionAttempts, CleanupBatch int
}

// RecoverySettings rejects invalid bounds rather than silently replacing them.
func (c RecoveryConfig) RecoverySettings() (RecoverySettings, error) {
	s := RecoverySettings{}

	durations := []struct {
		input              string
		target             *time.Duration
		fallback, min, max time.Duration
	}{
		{c.VerificationTTL, &s.VerificationTTL, 24 * time.Hour, time.Minute, 24 * time.Hour},
		{c.ResetTTL, &s.ResetTTL, time.Hour, time.Minute, time.Hour},
		{c.Timeout, &s.Timeout, 5 * time.Second, time.Second, 30 * time.Second},
		{c.Lease, &s.Lease, 5 * time.Second, time.Second, 30 * time.Second},
	}
	for _, d := range durations {
		value := d.fallback
		if d.input != "" {
			parsed, err := time.ParseDuration(d.input)
			if err != nil {
				return s, errors.New("invalid recovery duration")
			}

			value = parsed
		}

		if value < d.min || value > d.max || value%time.Microsecond != 0 {
			return s, errors.New("invalid recovery duration")
		}

		*d.target = value
	}

	if s.Lease < s.Timeout {
		return s, errors.New("recovery lease is shorter than operation timeout")
	}

	bounds := []struct {
		input         int
		target        *int
		fallback, max int
	}{
		{c.TokenAttempts, &s.TokenAttempts, 5, 10}, {c.IssuanceAttempts, &s.IssuanceAttempts, 3, 10},
		{c.CompletionAttempts, &s.CompletionAttempts, 10, 20}, {c.CleanupBatch, &s.CleanupBatch, 1000, 1000},
	}
	for _, b := range bounds {
		value := b.input
		if value == 0 {
			value = b.fallback
		}

		if value < 1 || value > b.max {
			return s, errors.New("invalid recovery limit")
		}

		*b.target = value
	}

	return s, nil
}
