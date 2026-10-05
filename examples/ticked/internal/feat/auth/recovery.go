// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/model"
)

var _ core.RecoveryQueries = (*Queries)(nil)

func chargeRecoveryWindow(ctx context.Context, q *dal.Queries, user string, kind int16, limit int, now time.Time) error {
	window := now.UTC().Truncate(15 * time.Minute)

	err := q.EnsureRecoveryBudget(ctx, dal.EnsureRecoveryBudgetParams{UserID: user, Kind: kind, WindowStart: window})
	if err != nil {
		return err
	}

	budget, err := q.LockRecoveryBudget(ctx, dal.LockRecoveryBudgetParams{UserID: user, Kind: kind})
	if err != nil {
		return err
	}

	attempts := budget.Attempts
	if budget.WindowStart.After(window) {
		return core.ErrRecoveryUnavailable
	}

	if !budget.WindowStart.Equal(window) {
		attempts = 0
	}

	if attempts < 0 || attempts >= int32(limit) {
		return core.ErrRecoveryAttempts
	}

	return q.ChargeRecoveryBudget(ctx, dal.ChargeRecoveryBudgetParams{UserID: user, Kind: kind, WindowStart: window, Attempts: attempts + 1})
}

// IssueMailbox replaces one purpose slot under the same subject lock as budget admission.
func (q *Queries) IssueMailbox(ctx context.Context, p core.MailboxRecord, settings config.RecoverySettings) error {
	return q.issueMailbox(ctx, p, settings, core.VerifyMailbox)
}
func (q *Queries) issueMailbox(ctx context.Context, p core.MailboxRecord, settings config.RecoverySettings, purpose core.RecoveryPurpose) error {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	user, err := lockCredential(ctx, tx, p.State)
	if err != nil {
		if errors.Is(err, core.ErrCredentialChanged) {
			return core.ErrRecoveryUnavailable
		}

		return err
	}

	queries := dal.New(tx)

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, p.PolicyRevision, settings)
	if err != nil || p.Purpose != purpose || user.Email != p.Target || !eligibleMailbox(user, purpose, now) || p.Attempts != 0 || p.Revision != 1 || !p.LeaseUntil.IsZero() {
		return core.ErrRecoveryUnavailable
	}

	err = chargeRecoveryWindow(ctx, queries, user.ID, int16(p.Purpose), settings.IssuanceAttempts, now)
	if err != nil {
		return err
	}

	err = queries.IssueMailboxToken(ctx, dal.IssueMailboxTokenParams{ID: p.ID, UserID: user.ID, Purpose: int16(p.Purpose), Target: p.Target, AuthVersion: p.State.Version, PolicyRevision: p.PolicyRevision, SecretDigest: p.Digest[:], CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, AttemptLimit: int32(p.AttemptLimit)})
	if err != nil {
		return err
	}

	return tx.Commit()
}

func mailboxSnapshot(row dal.MailboxToken) core.MailboxRecord {
	p := core.MailboxRecord{ID: row.ID, State: core.CredentialState{UserID: row.UserID, Version: row.AuthVersion}, Purpose: core.RecoveryPurpose(row.Purpose), Target: row.Target, PolicyRevision: row.PolicyRevision, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, Attempts: int(row.Attempts), AttemptLimit: int(row.AttemptLimit), Revision: row.Revision, LeaseUntil: row.LeaseUntil.Time, ConsumedAt: row.ConsumedAt.Time, RevokedAt: row.RevokedAt.Time}
	copy(p.Digest[:], row.SecretDigest)

	return p
}

func lockMailbox(ctx context.Context, tx *sql.Tx, id string, purpose core.RecoveryPurpose) (core.MailboxRecord, error) {
	queries := dal.New(tx)

	subject, err := queries.MailboxSubject(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return core.MailboxRecord{}, core.ErrRecoveryUnavailable
	}

	if err != nil {
		return core.MailboxRecord{}, err
	}

	user, err := queries.GetUserForAuth(ctx, subject)
	if errors.Is(err, sql.ErrNoRows) {
		return core.MailboxRecord{}, core.ErrRecoveryUnavailable
	}

	if err != nil {
		return core.MailboxRecord{}, err
	}

	row, err := queries.LockMailboxToken(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return core.MailboxRecord{}, core.ErrRecoveryUnavailable
	}

	if err != nil {
		return core.MailboxRecord{}, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return core.MailboxRecord{}, err
	}

	if !user.Active || core.RecoveryPurpose(row.Purpose) != purpose || user.AuthVersion != row.AuthVersion || user.Email != row.Target || !eligibleMailbox(user, purpose, now) || len(row.SecretDigest) != 32 {
		return core.MailboxRecord{}, core.ErrRecoveryUnavailable
	}

	return mailboxSnapshot(row), nil
}

// ReserveMailbox durably commits invalid-secret attempts; no error path refunds them.
func (q *Queries) ReserveMailbox(ctx context.Context, token core.RecoveryToken, policy string, settings config.RecoverySettings) (*core.MailboxRecord, error) {
	return q.reserveMailbox(ctx, token, policy, settings, core.VerifyMailbox)
}
func (q *Queries) reserveMailbox(ctx context.Context, token core.RecoveryToken, policy string, settings config.RecoverySettings, purpose core.RecoveryPurpose) (*core.MailboxRecord, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockMailbox(ctx, tx, token.ID(), purpose)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, policy, settings)
	if err != nil {
		return nil, err
	}

	if p.Revision >= math.MaxInt64-1 {
		return nil, core.ErrRecoveryUnavailable
	}

	if p.Attempts >= p.AttemptLimit {
		return nil, core.ErrRecoveryAttempts
	}

	if now.Before(p.LeaseUntil) {
		return nil, core.ErrRecoveryBusy
	}

	err = chargeRecoveryWindow(ctx, queries, p.State.UserID, 3, settings.CompletionAttempts, now)
	if err != nil {
		return nil, err
	}

	err = queries.ChargeMailboxToken(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	p.Attempts++

	p.Revision++
	if !token.Matches(p) {
		err = tx.Commit()
		if err != nil {
			return nil, err
		}

		return nil, core.ErrRecoveryUnavailable
	}

	p.LeaseUntil = now.Add(settings.Lease)

	err = queries.LeaseMailboxToken(ctx, dal.LeaseMailboxTokenParams{ID: p.ID, LeaseUntil: sql.NullTime{Time: p.LeaseUntil, Valid: true}, Revision: p.Revision})
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// ConfirmMailbox serializes the current version and commits every final effect together.
func (q *Queries) ConfirmMailbox(ctx context.Context, pending core.MailboxRecord, policy string, settings config.RecoverySettings) (*core.MailboxVerification, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockMailbox(ctx, tx, pending.ID, core.VerifyMailbox)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, policy, settings)
	if err != nil || p.Attempts < 1 || p.Revision != pending.Revision || p.State != pending.State || p.Digest != pending.Digest || !p.LeaseUntil.Equal(pending.LeaseUntil) || !now.Before(p.LeaseUntil) {
		return nil, core.ErrRecoveryUnavailable
	}

	_, err = queries.ReclaimRecoveryNotices(ctx, p.State.UserID)
	if err != nil {
		return nil, err
	}

	count, err := queries.CountRecoveryNotices(ctx, p.State.UserID)
	if err != nil {
		return nil, err
	}

	if count >= 20 {
		return nil, core.ErrRecoveryCapacity
	}

	err = queries.VerifyCurrentMailbox(ctx, dal.VerifyCurrentMailboxParams{ID: p.State.UserID, MailboxVerifiedAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return nil, err
	}

	err = queries.ConsumeMailboxToken(ctx, dal.ConsumeMailboxTokenParams{ID: p.ID, ConsumedAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return nil, err
	}

	err = queries.DeleteUserSessions(ctx, p.State.UserID)
	if err != nil {
		return nil, err
	}

	err = queries.DeleteSubjectEnrollments(ctx, p.State.UserID)
	if err != nil {
		return nil, err
	}

	err = queries.RevokeSubjectMailboxTokens(ctx, dal.RevokeSubjectMailboxTokensParams{UserID: p.State.UserID, RevokedAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return nil, err
	}

	err = queries.CreateRecoveryNotice(ctx, dal.CreateRecoveryNoticeParams{ID: model.NewID(), UserID: p.State.UserID, Kind: 1, Destination: p.Target, CreatedAt: now})
	if err != nil {
		return nil, err
	}
	// A lock wait or slow write cannot make a stale reservation valid at commit.
	finalTime, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	if !finalTime.Before(p.ExpiresAt) || !finalTime.Before(p.LeaseUntil) {
		return nil, core.ErrRecoveryUnavailable
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &core.MailboxVerification{Subject: p.State.UserID, VerifiedAt: now}, nil
}

// ReleaseMailbox only touches the exact child revision and takes no subsequent subject lock.
func (q *Queries) ReleaseMailbox(ctx context.Context, id string, revision int64) error {
	if q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return core.ErrRecoveryUnavailable
	}

	return dal.New(q.dbProvider.GetDB()).ReleaseMailboxToken(ctx, dal.ReleaseMailboxTokenParams{ID: id, Revision: revision})
}

// DeleteExpiredMailboxTokens uses a finite retained-only batch without a lock inversion.
func (q *Queries) DeleteExpiredMailboxTokens(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 || q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return 0, core.ErrRecoveryUnavailable
	}

	return dal.New(q.dbProvider.GetDB()).DeleteExpiredMailboxTokens(ctx, int32(limit))
}

// Eligibility is current-address ownership, never activation or MFA proof.
func eligibleMailbox(user dal.User, purpose core.RecoveryPurpose, now time.Time) bool {
	if purpose == core.VerifyMailbox {
		return !user.MailboxVerifiedAt.Valid
	}

	return purpose == core.ResetPassword && user.MailboxVerifiedAt.Valid && !user.MailboxVerifiedAt.Time.IsZero() && !user.MailboxVerifiedAt.Time.After(now)
}
