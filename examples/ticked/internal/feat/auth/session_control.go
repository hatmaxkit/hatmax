// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"database/sql"
	"errors"
	"hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
	"math"
)

// lockedActor discovers the subject without authorizing, then locks the current
// account before sessions. Multi-record operations lock IDs in stable order.
func lockedActor(ctx context.Context, tx *sql.Tx, digest auth.SessionDigest, all bool) (dal.User, dal.Session, error) {
	queries := dal.New(tx)

	initial, err := queries.GetSessionByDigest(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return dal.User{}, dal.Session{}, auth.ErrSessionNotFound
	}

	if err != nil {
		return dal.User{}, dal.Session{}, err
	}

	user, err := lockCredential(ctx, tx, auth.CredentialState{UserID: initial.UserID, Version: initial.AuthVersion})
	if err != nil {
		return dal.User{}, dal.Session{}, err
	}

	if all {
		ids, err := queries.LockSubjectSessions(ctx, user.ID)
		if err != nil {
			return dal.User{}, dal.Session{}, err
		}

		if len(ids) > 100 {
			return dal.User{}, dal.Session{}, auth.ErrSessionCapacity
		}
	}

	stored, err := queries.GetSessionForUpdate(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return dal.User{}, dal.Session{}, auth.ErrSessionNotFound
	}

	if err != nil {
		return dal.User{}, dal.Session{}, err
	}

	if stored.UserID != user.ID || stored.AuthVersion != user.AuthVersion {
		return dal.User{}, dal.Session{}, auth.ErrCredentialChanged
	}

	return user, stored, nil
}

func evaluateStored(ctx context.Context, queries *dal.Queries, stored dal.Session, requirement auth.AccessRequirement) (*auth.Session, error) {
	metadata, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	err = validateProofFactor(ctx, queries, *metadata)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = requirement.Evaluate(*metadata, now)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// RotateSession rechecks the old live generation and commits one secret change.
func (q *Queries) RotateSession(ctx context.Context, state auth.CredentialState, digest auth.SessionDigest, generation int64, replacement auth.SessionRecord, requirement auth.AccessRequirement) (*auth.Session, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user, stored, err := lockedActor(ctx, tx, digest, false)
	if err != nil {
		return nil, err
	}

	if user.ID != state.UserID || user.AuthVersion != state.Version {
		return nil, auth.ErrCredentialChanged
	}

	if generation < 1 || generation == math.MaxInt64 || stored.Generation != generation || replacement.Generation != generation+1 {
		return nil, auth.ErrSessionGeneration
	}

	if replacement.Proof.Method != auth.PasswordProof || replacement.ID != stored.ID || replacement.UserID != user.ID || replacement.AuthVersion != user.AuthVersion || !replacement.CreatedAt.Equal(stored.CreatedAt) || replacement.Digest == digest || replacement.AuthenticatedAt.Before(stored.AuthenticatedAt) || !replacement.Proof.VerifiedAt.Equal(replacement.AuthenticatedAt) {
		return nil, auth.ErrSessionRecord
	}

	queries := dal.New(tx)
	oldRequirement := requirement
	oldRequirement.MaxAge = 0

	old, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	err = validateProofFactor(ctx, queries, *old)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = oldRequirement.Evaluate(*old, now)
	if err != nil {
		return nil, err
	}

	err = requirement.Evaluate(replacement.Session, now)
	if err != nil {
		return nil, err
	}

	rotated, err := queries.RotateSession(ctx, dal.RotateSessionParams{
		OldDigest: digest[:], ExpectedGeneration: generation, NewDigest: replacement.Digest[:], PolicyRevision: replacement.PolicyRevision,
		ProofMethod: int16(replacement.Proof.Method), ProofVerifiedAt: replacement.Proof.VerifiedAt, AuthenticatedAt: replacement.AuthenticatedAt,
		LastActivityAt: replacement.LastActivityAt, ExpiresAt: replacement.ExpiresAt, InactivityUs: replacement.InactivityTTL.Microseconds(),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, auth.ErrSessionGeneration
	}

	if err != nil {
		return nil, err
	}

	metadata, err := toAuthSession(rotated)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// ListSessions returns only actor-owned safe metadata in bounded ID order.
func (q *Queries) ListSessions(ctx context.Context, digest auth.SessionDigest, requirement auth.AccessRequirement, limit int, cursor string) (*auth.SessionPage, error) {
	if limit < 1 || limit > 100 || requirement.MaxAge <= 0 {
		return nil, auth.ErrAccessRequirement
	}

	after, err := auth.ParseSessionCursor(cursor)
	if err != nil {
		return nil, err
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user, actor, err := lockedActor(ctx, tx, digest, false)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	rows, err := queries.ListSubjectSessions(ctx, dal.ListSubjectSessionsParams{Subject: user.ID, AfterID: after, PageLimit: int32(limit + 1)})
	if err != nil {
		return nil, err
	}

	_, err = evaluateStored(ctx, queries, actor, requirement)
	if err != nil {
		return nil, err
	}

	page := &auth.SessionPage{CurrentID: actor.ID, Sessions: make([]auth.Session, 0, limit)}

	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}

	for _, row := range rows {
		safe, err := toAuthSession(row)
		if err != nil {
			return nil, err
		}

		page.Sessions = append(page.Sessions, *safe)
	}

	if more {
		page.NextCursor, err = auth.SessionCursor(rows[len(rows)-1].ID)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return page, nil
}

// RevokeSessions rechecks recent actor proof after all locks, then deletes only
// selected rows of that subject. No request-supplied subject is accepted.
func (q *Queries) RevokeSessions(ctx context.Context, digest auth.SessionDigest, requirement auth.AccessRequirement, selection auth.SessionSelection) (int64, error) {
	err := selection.Check()
	if err != nil {
		return 0, err
	}

	if requirement.MaxAge <= 0 {
		return 0, auth.ErrAccessRequirement
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	user, actor, err := lockedActor(ctx, tx, digest, true)
	if err != nil {
		return 0, err
	}

	queries := dal.New(tx)

	_, err = evaluateStored(ctx, queries, actor, requirement)
	if err != nil {
		return 0, err
	}

	count, err := queries.RevokeSubjectSessions(ctx, dal.RevokeSubjectSessionsParams{Subject: user.ID, Scope: int32(selection.Scope), ActorID: actor.ID, SelectedID: selection.ID})
	if err != nil {
		return 0, err
	}

	if selection.Scope == auth.SessionSelected && count == 0 {
		return 0, auth.ErrSessionNotFound
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return count, nil
}
