// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"errors"
	"time"
)

// CredentialAdmissionConfig bounds durable credential work across replicas.
type CredentialAdmissionConfig struct {
	PasswordAttempts     int    `koanf:"password_attempts"`
	PasswordWindow       string `koanf:"password_window"`
	PasswordCooldown     string `koanf:"password_cooldown"`
	RegistrationAttempts int    `koanf:"registration_attempts"`
	RegistrationWindow   string `koanf:"registration_window"`
	RegistrationCooldown string `koanf:"registration_cooldown"`
	MaxIdentities        int    `koanf:"max_identities"`
	Timeout              string `koanf:"timeout"`
	CleanupBatch         int    `koanf:"cleanup_batch"`
}

// CredentialAdmissionSettings is an owned validated operation snapshot.
type CredentialAdmissionSettings struct {
	PasswordAttempts, RegistrationAttempts, MaxIdentities, CleanupBatch                 int
	PasswordWindow, PasswordCooldown, RegistrationWindow, RegistrationCooldown, Timeout time.Duration
}

// CredentialAdmissionSettings resolves zero defaults and rejects unlimited work.
func (c CredentialAdmissionConfig) CredentialAdmissionSettings() (CredentialAdmissionSettings, error) {
	s := CredentialAdmissionSettings{}

	durations := []struct {
		value              string
		target             *time.Duration
		fallback, min, max time.Duration
	}{
		{c.PasswordWindow, &s.PasswordWindow, 10 * time.Minute, time.Minute, time.Hour},
		{c.PasswordCooldown, &s.PasswordCooldown, 10 * time.Minute, time.Minute, time.Hour},
		{c.RegistrationWindow, &s.RegistrationWindow, time.Hour, time.Minute, time.Hour},
		{c.RegistrationCooldown, &s.RegistrationCooldown, time.Hour, time.Minute, time.Hour},
		{c.Timeout, &s.Timeout, time.Second, 100 * time.Millisecond, 5 * time.Second},
	}
	for _, d := range durations {
		v := d.fallback
		if d.value != "" {
			parsed, err := time.ParseDuration(d.value)
			if err != nil {
				return s, errors.New("invalid credential admission duration")
			}

			v = parsed
		}

		if v < d.min || v > d.max || v%time.Microsecond != 0 {
			return s, errors.New("invalid credential admission duration")
		}

		*d.target = v
	}

	limits := []struct {
		value              int
		target             *int
		fallback, min, max int
	}{
		{c.PasswordAttempts, &s.PasswordAttempts, 10, 1, 20},
		{c.RegistrationAttempts, &s.RegistrationAttempts, 3, 1, 10},
		{c.MaxIdentities, &s.MaxIdentities, 10000, 100, 100000},
		{c.CleanupBatch, &s.CleanupBatch, 1000, 1, 1000},
	}
	for _, b := range limits {
		v := b.value
		if v == 0 {
			v = b.fallback
		}

		if v < b.min || v > b.max {
			return s, errors.New("invalid credential admission limit")
		}

		*b.target = v
	}

	return s, nil
}

// Check rejects malformed snapshots supplied directly to an adapter.
func (s CredentialAdmissionSettings) Check() error {
	c := CredentialAdmissionConfig{PasswordAttempts: s.PasswordAttempts, RegistrationAttempts: s.RegistrationAttempts, MaxIdentities: s.MaxIdentities, CleanupBatch: s.CleanupBatch, PasswordWindow: s.PasswordWindow.String(), PasswordCooldown: s.PasswordCooldown.String(), RegistrationWindow: s.RegistrationWindow.String(), RegistrationCooldown: s.RegistrationCooldown.String(), Timeout: s.Timeout.String()}

	resolved, err := c.CredentialAdmissionSettings()
	if err != nil {
		return err
	}

	if resolved != s {
		return errors.New("invalid credential admission snapshot")
	}

	return nil
}
