// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"fmt"
	"time"
)

// PasswordReset reports safe committed state, never a bearer or MFA authority.
type PasswordReset struct {
	Subject string    `json:"subject"`
	ResetAt time.Time `json:"resetAt"`
}

// RequestPasswordReset returns a transient token only to trusted application mail
// dispatch, after issuance commits against the previously verified current mailbox.
func (s *RecoveryService) RequestPasswordReset(ctx context.Context, mailbox string) (*MailboxIssue, error) {
	return s.requestMailbox(ctx, mailbox, ResetPassword)
}

// ResetPassword proves the actual one-use mailbox secret before candidate work.
// It cannot activate an account, change MFA or issue an authentication session.
func (s *RecoveryService) ResetPassword(ctx context.Context, bearer, password string) (*PasswordReset, error) {
	if len(password) > s.base.policy.config.MaxBytes {
		return nil, ErrPasswordTooLong
	}

	token, err := ParseRecoveryToken(bearer)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	pending, err := s.queries.ReservePasswordReset(work, token, s.policyRevision, s.settings)
	if err != nil {
		return nil, err
	}

	if pending == nil || pending.Purpose != ResetPassword {
		return nil, ErrRecoveryUnavailable
	}
	defer func() { _ = s.queries.ReleaseMailbox(work, pending.ID, pending.Revision) }()

	now := time.Now().UTC().Truncate(time.Microsecond)

	err = pending.Check(now, s.policyRevision, s.settings)
	if err != nil || !token.Matches(*pending) || pending.Attempts < 1 || !now.Before(pending.LeaseUntil) {
		return nil, ErrRecoveryUnavailable
	}

	passwordWork, passwordCancel := context.WithTimeout(work, s.base.passwordTimeout)
	defer passwordCancel()

	candidate, err := s.base.policy.Prepare(passwordWork, password)
	if err != nil {
		return nil, err
	}

	encoded, err := s.base.verifier.Hash(passwordWork, candidate)
	if err != nil {
		return nil, fmt.Errorf("cannot hash reset password: %w", err)
	}

	result, err := s.queries.CompletePasswordReset(passwordWork, *pending, encoded, s.policyRevision, s.settings)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Subject != pending.State.UserID || result.ResetAt.IsZero() {
		return nil, ErrRecoveryUnavailable
	}

	return result, nil
}
