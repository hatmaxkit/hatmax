// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import "errors"

// FallbackConfig has no RP dependency. Limits use the shared factor admission bounds.
// Skew is fixed to one by default; StrictStep disables adjacent-step acceptance.
type FallbackConfig struct {
	Limits      AuthenticatorConfig
	Issuer      string
	StrictStep  bool
	BackupCodes int
}
type FallbackSettings struct {
	EnrollmentSettings
	Issuer      string
	Skew        uint
	BackupCodes int
}

func (c FallbackConfig) Settings() (FallbackSettings, error) {
	limits, err := c.Limits.factorSettings(EnrollmentSettings{})
	if err != nil {
		return FallbackSettings{}, err
	}

	s := FallbackSettings{EnrollmentSettings: limits, Issuer: c.Issuer, Skew: 1, BackupCodes: c.BackupCodes}
	if len(s.Issuer) == 0 || len(s.Issuer) > 128 {
		return s, errors.New("invalid TOTP issuer")
	}

	for i := range len(s.Issuer) {
		if s.Issuer[i] < 32 || s.Issuer[i] > 126 {
			return s, errors.New("invalid TOTP issuer")
		}
	}

	if c.StrictStep {
		s.Skew = 0
	}

	if s.BackupCodes == 0 {
		s.BackupCodes = 8
	}

	if s.BackupCodes < 1 || s.BackupCodes > 10 {
		return s, errors.New("invalid backup code count")
	}

	return s, nil
}
