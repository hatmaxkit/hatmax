// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"errors"
	"time"
)

// SecurityObservationConfig bounds synchronous best-effort callbacks.
type SecurityObservationConfig struct {
	Timeout     string `koanf:"timeout"`
	Concurrency int    `koanf:"concurrency"`
}

// SecurityObservationSettings is an immutable validated callback budget.
type SecurityObservationSettings struct {
	Timeout     time.Duration
	Concurrency int
}

// SecurityObservationSettings resolves finite defaults and rejects unlimited work.
func (c SecurityObservationConfig) SecurityObservationSettings() (SecurityObservationSettings, error) {
	s := SecurityObservationSettings{Timeout: 100 * time.Millisecond, Concurrency: 2}
	if c.Timeout != "" {
		d, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return s, errors.New("invalid security observation timeout")
		}

		s.Timeout = d
	}

	if c.Concurrency != 0 {
		s.Concurrency = c.Concurrency
	}

	if s.Timeout < time.Millisecond || s.Timeout > 100*time.Millisecond || s.Concurrency < 1 || s.Concurrency > 16 {
		return s, errors.New("invalid security observation budget")
	}

	return s, nil
}
