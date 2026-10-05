// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/model"
)

func (q *Queries) IssuePasswordReset(ctx context.Context, p core.MailboxRecord, settings config.RecoverySettings) error {
	return q.issueMailbox(ctx, p, settings, core.ResetPassword)
}
func (q *Queries) ReservePasswordReset(ctx context.Context, t core.RecoveryToken, policy string, settings config.RecoverySettings) (*core.MailboxRecord, error) {
	return q.reserveMailbox(ctx, t, policy, settings, core.ResetPassword)
}

// sameResetReservation binds the complete owned security snapshot, including
// time equality independent of timestamp location representation.
func sameResetReservation(p, reserved core.MailboxRecord) bool {
	return p.ID == reserved.ID && p.State == reserved.State && p.Purpose == core.ResetPassword && reserved.Purpose == core.ResetPassword &&
		p.Target == reserved.Target && p.PolicyRevision == reserved.PolicyRevision && p.Digest == reserved.Digest &&
		p.Attempts == reserved.Attempts && p.AttemptLimit == reserved.AttemptLimit && p.Revision == reserved.Revision &&
		p.CreatedAt.Equal(reserved.CreatedAt) && p.ExpiresAt.Equal(reserved.ExpiresAt) && p.LeaseUntil.Equal(reserved.LeaseUntil)
}

// CompletePasswordReset commits token consumption and complete password effects
// together under the subject lock. Admission is a prior independent transaction.
func (q *Queries) CompletePasswordReset(ctx context.Context, reserved core.MailboxRecord, encoded, policy string, settings config.RecoverySettings) (*core.PasswordReset, error) {
	if len(encoded) > 128 || !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		return nil, core.ErrRecoveryUnavailable
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockMailbox(ctx, tx, reserved.ID, core.ResetPassword)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, policy, settings)
	if err != nil || p.Attempts < 1 || !sameResetReservation(p, reserved) || !now.Before(p.LeaseUntil) {
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

	_, err = queries.ReplacePassword(ctx, dal.ReplacePasswordParams{ID: p.State.UserID, PasswordHash: encoded, UpdatedAt: now})
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

	err = queries.CreateRecoveryNotice(ctx, dal.CreateRecoveryNoticeParams{ID: model.NewID(), UserID: p.State.UserID, Kind: 3, Destination: p.Target, CreatedAt: now})
	if err != nil {
		return nil, err
	}
	// Reclaim/child-row waits and slow writes must not extend token authority.
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

	return &core.PasswordReset{Subject: p.State.UserID, ResetAt: now}, nil
}

// RequestPasswordReset never returns a token or candidate to public/operator UI.
func (d *MailboxDelivery) RequestPasswordReset(ctx context.Context, mailbox string) error {
	return d.requestMailbox(ctx, mailbox, core.ResetPassword)
}
func (d *MailboxDelivery) ResetPassword(ctx context.Context, bearer, password string) error {
	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := d.service.ResetPassword(work, bearer, password)
	if err != nil {
		return err
	}

	err = d.DispatchMailboxNotices(work, result.Subject, 5)
	if err != nil {
		d.logger.Error("Password reset notification dispatch failed; committed security state is unchanged")
	}

	return nil
}
