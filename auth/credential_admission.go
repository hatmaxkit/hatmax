// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"time"
	"unicode"
	"unicode/utf8"

	"hatmax.adrianpk.com/config"
)

var (
	ErrCredentialIdentity          = errors.New("invalid credential identity")
	ErrCredentialAdmissionAttempts = errors.New("credential admission attempts exhausted")
	ErrCredentialAdmissionCapacity = errors.New("credential admission capacity exhausted")
	ErrCredentialAdmissionState    = errors.New("invalid credential admission state")
)

// CredentialAdmissionPurpose separates registration from actual password proof.
type CredentialAdmissionPurpose uint8

const (
	CredentialRegistration  CredentialAdmissionPurpose = 1
	CredentialPasswordProof CredentialAdmissionPurpose = 2
)

// CredentialAdmissionDenial exposes trusted retry time only to internal adapters.
// Public unauthenticated responses must not disclose this identity-specific time.
type CredentialAdmissionDenial struct{ RetryAt time.Time }

func (d *CredentialAdmissionDenial) Error() string { return ErrCredentialAdmissionAttempts.Error() }
func (d *CredentialAdmissionDenial) Unwrap() error { return ErrCredentialAdmissionAttempts }

// CheckCredentialIdentity checks structure only, never normalizing account identity.
func CheckCredentialIdentity(identity string) error {
	if len(identity) == 0 || len(identity) > 254 || !utf8.ValidString(identity) {
		return ErrCredentialIdentity
	}

	for _, r := range identity {
		if unicode.IsControl(r) {
			return ErrCredentialIdentity
		}
	}

	return nil
}

// CredentialAdmissionQueries commits charges independently of authentication.
// It owns private identity keys, finite capacity, trusted time and row locks.
// Denied work changes no counter; admitted work has no release/refund operation.
type CredentialAdmissionQueries interface {
	ChargeCredentialAdmission(context.Context, string, CredentialAdmissionPurpose, config.CredentialAdmissionSettings) error
	CleanupCredentialAdmission(context.Context, config.CredentialAdmissionSettings) (int64, error)
}

// CredentialAdmission bounds work using shared caller-owned durable state.
// It creates no worker and grants no identity, proof or session authority.
type CredentialAdmission struct {
	queries  CredentialAdmissionQueries
	settings config.CredentialAdmissionSettings
}

// NewCredentialAdmission validates a finite snapshot and requires durable storage.
func NewCredentialAdmission(queries CredentialAdmissionQueries, cfg config.CredentialAdmissionConfig) (*CredentialAdmission, error) {
	if queries == nil {
		return nil, errors.New("credential admission storage is required")
	}

	settings, err := cfg.CredentialAdmissionSettings()
	if err != nil {
		return nil, err
	}

	return &CredentialAdmission{queries: queries, settings: settings}, nil
}

// Admit commits one charge before account lookup, checker or credential work.
// Earlier caller deadlines apply; an ambiguous commit must not be retried.
func (s *CredentialAdmission) Admit(ctx context.Context, identity string, purpose CredentialAdmissionPurpose) error {
	err := CheckCredentialIdentity(identity)
	if err != nil {
		return err
	}

	if purpose != CredentialRegistration && purpose != CredentialPasswordProof {
		return ErrCredentialIdentity
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	err = work.Err()
	if err != nil {
		return err
	}

	err = s.queries.ChargeCredentialAdmission(work, identity, purpose, s.settings)
	if err != nil {
		return err
	}

	return work.Err()
}

// Cleanup removes at most the configured batch whose complete denial ended.
func (s *CredentialAdmission) Cleanup(ctx context.Context) (int64, error) {
	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	err := work.Err()
	if err != nil {
		return 0, err
	}

	count, err := s.queries.CleanupCredentialAdmission(work, s.settings)
	if err != nil {
		return 0, err
	}

	if count < 0 || count > int64(s.settings.CleanupBatch) {
		return 0, ErrCredentialAdmissionState
	}

	err = work.Err()
	if err != nil {
		return 0, err
	}

	return count, nil
}
