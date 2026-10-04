// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"hatmax.adrianpk.com/model"
)

// passwordChangeActor requires the subject lock acquired by lockedActor. Every
// supported factor mutation shares that lock, so the bounded profile is stable.
func passwordChangeActor(ctx context.Context, queries *dal.Queries, row dal.Session, digest core.SessionDigest, policy core.PasswordChangePolicy) (*core.PasswordChangeAuthorization, error) {
	actor, err := toAuthSession(row)
	if err != nil {
		return nil, err
	}

	rows, err := queries.RecoveryPasswordFactors(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	if len(rows) > 20 {
		return nil, core.ErrRecoveryCapacity
	}

	factors := make([]core.FactorSelection, 0, len(rows))
	for _, f := range rows {
		factors = append(factors, core.FactorSelection{Kind: core.FactorKind(f.Kind), ID: f.ID, Revision: f.Revision})
	}

	err = validateProofFactor(ctx, queries, *actor)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	pending := &core.PasswordChangeAuthorization{Actor: *actor, ActorDigest: digest, Factors: factors, Policy: policy}
	err = pending.Check(now, policy)

	return pending, err
}

// AuthorizePasswordChange returns no raw bearer and durably spends a shared
// completion/change attempt before password policy or cryptographic work.
func (q *Queries) AuthorizePasswordChange(ctx context.Context, digest core.SessionDigest, policy core.PasswordChangePolicy, settings config.RecoverySettings) (*core.PasswordChangeAuthorization, error) {
	if policy.Check() != nil {
		return nil, core.ErrAccessRequirement
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, row, err := lockedActor(ctx, tx, digest, false)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	pending, err := passwordChangeActor(ctx, queries, row, digest, policy)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = chargeRecoveryWindow(ctx, queries, row.UserID, 3, settings.CompletionAttempts, now)
	if err != nil {
		return nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = pending.Check(now, policy)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return pending, nil
}

// CommitPasswordChange rechecks current ownership, policy, proof and factor
// revisions, then commits the complete credential/revocation/notice together.
func (q *Queries) CommitPasswordChange(ctx context.Context, pending core.PasswordChangeAuthorization, encoded string, policy core.PasswordChangePolicy, settings config.RecoverySettings) (*core.PasswordChanged, error) {
	// The trusted core supplies the complete verifier output, never a transport hash.
	if len(encoded) > 128 || !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		return nil, core.ErrRecoveryUnavailable
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user, row, err := lockedActor(ctx, tx, pending.ActorDigest, true)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	current, err := passwordChangeActor(ctx, queries, row, pending.ActorDigest, policy)
	if err != nil {
		return nil, err
	}

	if current.Actor != pending.Actor || current.Policy != pending.Policy || !slices.Equal(current.Factors, pending.Factors) {
		return nil, core.ErrRecoveryUnavailable
	}

	_, err = queries.ReclaimRecoveryNotices(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	count, err := queries.CountRecoveryNotices(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if count >= 20 {
		return nil, core.ErrRecoveryCapacity
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = current.Check(now, policy)
	if err != nil {
		return nil, err
	}

	_, err = queries.ReplacePassword(ctx, dal.ReplacePasswordParams{ID: user.ID, PasswordHash: encoded, UpdatedAt: now})
	if err != nil {
		return nil, err
	}

	err = queries.DeleteUserSessions(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	err = queries.DeleteSubjectEnrollments(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	err = queries.RevokeSubjectMailboxTokens(ctx, dal.RevokeSubjectMailboxTokensParams{UserID: user.ID, RevokedAt: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		return nil, err
	}

	err = queries.CreateRecoveryNotice(ctx, dal.CreateRecoveryNoticeParams{ID: model.NewID(), UserID: user.ID, Kind: 2, Destination: user.Email, CreatedAt: now})
	if err != nil {
		return nil, err
	}
	// Time is sampled after all possible lock waits/writes; failure rolls back
	// final effects, leaving the separately committed admission spent.
	finalTime, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = current.Check(finalTime, policy)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &core.PasswordChanged{Subject: user.ID, ChangedAt: now}, nil
}

// ChangePassword performs security mutation before bounded notification delivery.
func (d *MailboxDelivery) ChangePassword(ctx context.Context, bearer, password string, policy core.PasswordChangePolicy) error {
	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := d.service.ChangePassword(work, bearer, password, policy)
	if err != nil {
		return err
	}

	err = d.DispatchMailboxNotices(work, result.Subject, 5)
	if err != nil {
		d.logger.Error("Password notification dispatch failed; committed security state is unchanged")
	}

	return nil
}
