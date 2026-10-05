// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"hatmax.adrianpk.com/config"
)

// RecoveryQueries owns mailbox lifecycle transactions. Subject locks precede tokens.
// Wrong-secret attempts commit independently of final verification effects.
// Confirm consumes, verifies, advances version, invalidates all continuations and
// sessions, and persists a notification intent in one transaction.
type RecoveryQueries interface {
	IssuePasswordReset(context.Context, MailboxRecord, config.RecoverySettings) error
	ReservePasswordReset(context.Context, RecoveryToken, string, config.RecoverySettings) (*MailboxRecord, error)
	CompletePasswordReset(context.Context, MailboxRecord, string, string, config.RecoverySettings) (*PasswordReset, error)
	AuthorizePasswordChange(context.Context, SessionDigest, PasswordChangePolicy, config.RecoverySettings) (*PasswordChangeAuthorization, error)
	CommitPasswordChange(context.Context, PasswordChangeAuthorization, string, PasswordChangePolicy, config.RecoverySettings) (*PasswordChanged, error)
	IssueMailbox(context.Context, MailboxRecord, config.RecoverySettings) error
	ReserveMailbox(context.Context, RecoveryToken, string, config.RecoverySettings) (*MailboxRecord, error)
	ConfirmMailbox(context.Context, MailboxRecord, string, config.RecoverySettings) (*MailboxVerification, error)
	ReleaseMailbox(context.Context, string, int64) error
	DeleteExpiredMailboxTokens(context.Context, int) (int64, error)
}

// RecoveryService extends the existing credential engine with restricted mailbox tokens.
type RecoveryService struct {
	base           *Service
	queries        RecoveryQueries
	settings       config.RecoverySettings
	policyRevision string
}

func NewRecoveryService(base *Service, queries RecoveryQueries, cfg config.RecoveryConfig, policyRevision string) (*RecoveryService, error) {
	if base == nil || queries == nil || !validRecoveryRevision(policyRevision) {
		return nil, errors.New("recovery dependencies and trusted policy are required")
	}

	settings, err := cfg.RecoverySettings()
	if err != nil {
		return nil, err
	}

	return &RecoveryService{base: base, queries: queries, settings: settings, policyRevision: policyRevision}, nil
}

func (s *RecoveryService) requestMailboxVerification(ctx context.Context, mailbox string) (*MailboxIssue, error) {
	return s.requestMailbox(ctx, mailbox, VerifyMailbox)
}

func (s *RecoveryService) requestMailbox(ctx context.Context, mailbox string, purpose RecoveryPurpose) (*MailboxIssue, error) {
	if len(mailbox) == 0 || len(mailbox) > 254 || strings.ContainsAny(mailbox, "\x00\r\n") {
		return nil, ErrRecoveryUnavailable
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	user, err := s.base.queries.GetUserByEmail(work, mailbox)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrUserNotFound) {
		securityClassification(work, SecurityMissing)

		return nil, ErrRecoveryUnavailable
	}

	if err != nil {
		return nil, fmt.Errorf("cannot resolve recovery subject: %w", err)
	}

	if user == nil || !user.Active || user.Email != mailbox || purpose == ResetPassword && (user.MailboxVerifiedAt == nil || user.MailboxVerifiedAt.IsZero() || user.MailboxVerifiedAt.After(time.Now().UTC())) {
		outcome := SecurityPolicyRejected
		if user == nil || user.Email != mailbox {
			outcome = SecurityOperatingUnknown
		} else if !user.Active {
			outcome = SecurityInactive
		}

		securityClassification(work, outcome)

		return nil, ErrRecoveryUnavailable
	}

	securitySubject(work, user.ID)

	token, err := newRecoveryToken()
	if err != nil {
		return nil, errors.New("cannot generate recovery secret")
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	ttl := s.settings.VerificationTTL
	if purpose == ResetPassword {
		ttl = s.settings.ResetTTL
	}

	record := MailboxRecord{ID: token.ID(), State: CredentialState{UserID: user.ID, Version: user.AuthVersion}, Purpose: purpose, Target: user.Email, PolicyRevision: s.policyRevision, Digest: RecoveryDigest(token, user.ID, purpose), CreatedAt: now, ExpiresAt: now.Add(ttl), AttemptLimit: s.settings.TokenAttempts, Revision: 1}

	err = record.Check(now, s.policyRevision, s.settings)
	if err != nil {
		return nil, err
	}

	if purpose == VerifyMailbox {
		err = s.queries.IssueMailbox(work, record, s.settings)
	} else {
		err = s.queries.IssuePasswordReset(work, record, s.settings)
	}

	if err != nil {
		return nil, err
	}

	if work.Err() != nil {
		return nil, work.Err()
	}

	securityRecord(work, record.ID)

	return &MailboxIssue{Token: token, Target: record.Target, ExpiresAt: record.ExpiresAt}, nil
}

func (s *RecoveryService) confirmMailboxVerification(ctx context.Context, bearer string) (*MailboxVerification, error) {
	token, err := ParseRecoveryToken(bearer)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	pending, err := s.queries.ReserveMailbox(work, token, s.policyRevision, s.settings)
	if err != nil {
		return nil, err
	}

	if pending == nil || pending.Purpose != VerifyMailbox {
		return nil, ErrRecoveryUnavailable
	}
	// A cancelled release leaves only the finite lease; it cannot extend request time.
	defer func() { _ = s.queries.ReleaseMailbox(work, pending.ID, pending.Revision) }()

	now := time.Now().UTC().Truncate(time.Microsecond)

	err = pending.Check(now, s.policyRevision, s.settings)
	if err != nil || !token.Matches(*pending) || pending.Attempts < 1 || !now.Before(pending.LeaseUntil) {
		return nil, ErrRecoveryUnavailable
	}

	securitySubject(work, pending.State.UserID)
	securityRecord(work, pending.ID)

	result, err := s.queries.ConfirmMailbox(work, *pending, s.policyRevision, s.settings)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Subject != pending.State.UserID || result.VerifiedAt.IsZero() {
		securityClassification(work, SecurityOperatingUnknown)

		return nil, ErrRecoveryUnavailable
	}

	if work.Err() != nil {
		return nil, work.Err()
	}

	return result, nil
}

// CleanupMailboxTokens deletes one finite retained-expired batch; it owns no worker.
func (s *RecoveryService) CleanupMailboxTokens(ctx context.Context) (int64, error) {
	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	return s.queries.DeleteExpiredMailboxTokens(work, s.settings.CleanupBatch)
}
