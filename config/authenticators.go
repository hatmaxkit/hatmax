// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package config

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

// AuthenticatorConfig supplies trusted RP identity and finite enrollment limits.
// No origin or policy is inferred from HTTP request headers.
type AuthenticatorConfig struct {
	RPID                 string   `koanf:"rp_id"`
	RPName               string   `koanf:"rp_name"`
	Origins              []string `koanf:"origins"`
	LocalhostDevelopment bool     `koanf:"localhost_development"`
	PendingTTL           string   `koanf:"pending_ttl"`
	RecentProofAge       string   `koanf:"recent_proof_age"`
	Timeout              string   `koanf:"timeout"`
	Lease                string   `koanf:"lease"`
	BudgetWindow         string   `koanf:"budget_window"`
	Cooldown             string   `koanf:"cooldown"`
	MaxPending           int      `koanf:"max_pending"`
	MaxAuthenticators    int      `koanf:"max_authenticators"`
	PendingAttempts      int      `koanf:"pending_attempts"`
	SubjectAttempts      int      `koanf:"subject_attempts"`
	MaxConcurrent        int      `koanf:"max_concurrent"`
	CleanupBatch         int      `koanf:"cleanup_batch"`
}

// EnrollmentSettings is an owned validated constructor snapshot.
type EnrollmentSettings struct {
	RPBinding                                                                                    [32]byte
	RPID, RPName                                                                                 string
	Origins                                                                                      []string
	PendingTTL, RecentProofAge, Timeout, Lease, BudgetWindow, Cooldown                           time.Duration
	MaxPending, MaxAuthenticators, PendingAttempts, SubjectAttempts, MaxConcurrent, CleanupBatch int
}

// EnrollmentSettings validates explicit identity and applies finite defaults.
func (c AuthenticatorConfig) EnrollmentSettings() (EnrollmentSettings, error) {
	s := EnrollmentSettings{RPID: c.RPID, RPName: c.RPName, Origins: append([]string(nil), c.Origins...)}
	if len(c.RPID) == 0 || len(c.RPID) > 253 || strings.ContainsAny(c.RPID, "/:@?# ") || strings.ToLower(c.RPID) != c.RPID || len(c.RPName) == 0 || len(c.RPName) > 128 || len(c.Origins) == 0 || len(c.Origins) > 8 {
		return s, errors.New("invalid authenticator RP configuration")
	}

	seen := make(map[string]bool)

	for _, origin := range s.Origins {
		u, err := url.Parse(origin)
		if err != nil {
			return s, err
		}

		if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Host == "" || u.String() != origin || strings.ToLower(u.Host) != u.Host || seen[origin] || (u.Hostname() != c.RPID && !strings.HasSuffix(u.Hostname(), "."+c.RPID)) || (u.Scheme != "https" && !(c.LocalhostDevelopment && c.RPID == "localhost" && u.Hostname() == "localhost" && u.Scheme == "http")) {
			return s, errors.New("invalid authenticator origin")
		}

		seen[origin] = true
	}

	var err error

	s, err = c.factorSettings(s)
	if err != nil {
		return s, err
	}

	encoded, err := json.Marshal(struct {
		RPID    string
		Origins []string
		Profile string
	}{s.RPID, s.Origins, "ES256/RK/UV/none/v1"})
	if err != nil {
		return s, err
	}

	s.RPBinding = sha256.Sum256(encoded)

	return s, nil
}

// factorSettings validates shared admission bounds without imposing an RP on OTP.
func (c AuthenticatorConfig) factorSettings(s EnrollmentSettings) (EnrollmentSettings, error) {
	durations := []struct {
		input              string
		target             *time.Duration
		fallback, min, max time.Duration
	}{
		{c.PendingTTL, &s.PendingTTL, 5 * time.Minute, time.Minute, 10 * time.Minute},
		{c.RecentProofAge, &s.RecentProofAge, 5 * time.Minute, time.Second, 10 * time.Minute},
		{c.Timeout, &s.Timeout, 5 * time.Second, time.Millisecond, 30 * time.Second},
		{c.Lease, &s.Lease, 5 * time.Second, time.Millisecond, 30 * time.Second},
		{c.BudgetWindow, &s.BudgetWindow, 15 * time.Minute, time.Minute, time.Hour},
		{c.Cooldown, &s.Cooldown, 15 * time.Minute, time.Minute, time.Hour},
	}
	for _, d := range durations {
		value := d.fallback
		if d.input != "" {
			parsed, err := time.ParseDuration(d.input)
			if err != nil {
				return s, err
			}

			value = parsed
		}

		if value < d.min || value > d.max || value%time.Microsecond != 0 {
			return s, errors.New("invalid authenticator duration")
		}

		*d.target = value
	}

	if s.Lease < s.Timeout {
		return s, errors.New("verification lease is shorter than operation timeout")
	}

	limits := []struct {
		input         int
		target        *int
		fallback, max int
	}{
		{c.MaxPending, &s.MaxPending, 5, 10}, {c.MaxAuthenticators, &s.MaxAuthenticators, 10, 20},
		{c.PendingAttempts, &s.PendingAttempts, 5, 10}, {c.SubjectAttempts, &s.SubjectAttempts, 10, 100},
		{c.MaxConcurrent, &s.MaxConcurrent, 2, 2}, {c.CleanupBatch, &s.CleanupBatch, 1000, 1000},
	}
	for _, l := range limits {
		value := l.input
		if value == 0 {
			value = l.fallback
		}

		if value < 1 || value > l.max {
			return s, errors.New("invalid authenticator capacity")
		}

		*l.target = value
	}

	return s, nil
}
