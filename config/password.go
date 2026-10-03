// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"fmt"
	"time"

	"hatmax.adrianpk.com/model"
)

// PasswordSettings is a validated snapshot of credential startup settings.
type PasswordSettings struct {
	MinLength    int
	MaxLength    int
	MaxBytes     int
	CheckTimeout time.Duration
	Timeout      time.Duration
	Verifier     model.PasswordVerifierConfig
}

// PasswordSettings resolves optional zero maxima/timeouts and validates policy
// and KDF budgets. Password-only access always requires at least 15 code points.
func (c AuthConfig) PasswordSettings() (PasswordSettings, error) {
	if c.PasswordMaxLen == 0 {
		c.PasswordMaxLen = 1024
	}

	if c.PasswordMaxBytes == 0 {
		c.PasswordMaxBytes = 4096
	}

	if c.PasswordCheckTimeout == "" {
		c.PasswordCheckTimeout = "2s"
	}

	if c.PasswordTimeout == "" {
		c.PasswordTimeout = "5s"
	}

	if c.PasswordMinLen < 15 || c.PasswordMinLen > c.PasswordMaxLen {
		return PasswordSettings{}, fmt.Errorf("auth.password_min_len must be between 15 and password_max_len")
	}

	if c.PasswordMaxLen < 64 || c.PasswordMaxLen > 1024 {
		return PasswordSettings{}, fmt.Errorf("auth.password_max_len must be between 64 and 1024")
	}

	if c.PasswordMaxBytes < c.PasswordMaxLen*4 || c.PasswordMaxBytes > 4096 {
		return PasswordSettings{}, fmt.Errorf("auth.password_max_bytes must cover password_max_len and not exceed 4096")
	}

	check, err := time.ParseDuration(c.PasswordCheckTimeout)
	if err != nil || check <= 0 || check > 30*time.Second {
		return PasswordSettings{}, fmt.Errorf("auth.password_check_timeout must be positive and at most 30s")
	}

	timeout, err := time.ParseDuration(c.PasswordTimeout)
	if err != nil || timeout <= 0 || timeout > 30*time.Second {
		return PasswordSettings{}, fmt.Errorf("auth.password_timeout must be positive and at most 30s")
	}

	if c.ArgonMemoryKiB > 262144 || c.ArgonMaxMemoryKiB > 262144 || c.ArgonIterations > 10 || c.ArgonMaxIterations > 10 || c.ArgonParallelism > 8 || c.ArgonMaxParallelism > 8 {
		return PasswordSettings{}, fmt.Errorf("auth Argon2 parameters exceed supported limits")
	}

	verifier := model.PasswordVerifierConfig{MemoryKiB: uint32(c.ArgonMemoryKiB), Iterations: uint32(c.ArgonIterations), Parallelism: uint8(c.ArgonParallelism), MaxMemoryKiB: uint32(c.ArgonMaxMemoryKiB), MaxIterations: uint32(c.ArgonMaxIterations), MaxParallelism: uint8(c.ArgonMaxParallelism), MaxConcurrent: c.PasswordMaxConcurrent}

	_, err = model.NewPasswordVerifier(verifier)
	if err != nil {
		return PasswordSettings{}, fmt.Errorf("auth password verifier: %w", err)
	}

	return PasswordSettings{MinLength: c.PasswordMinLen, MaxLength: c.PasswordMaxLen, MaxBytes: c.PasswordMaxBytes, CheckTimeout: check, Timeout: timeout, Verifier: verifier}, nil
}
