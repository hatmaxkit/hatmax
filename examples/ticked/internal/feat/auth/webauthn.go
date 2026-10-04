// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

var _ core.WebAuthnQueries = (*Queries)(nil)

func factorSnapshot(f dal.Authenticator) core.WebAuthnFactor {
	return core.WebAuthnFactor{FactorBinding: core.FactorBinding{ID: f.ID, Revision: f.Revision}, ReplayRevision: f.ReplayRevision, CredentialID: f.CredentialID, PublicKey: f.PublicKey, Data: f.CredentialData, Counter: uint32(f.SignCount), BackupEligible: f.BackupEligible, BackupState: f.BackupState}
}

// LoadWebAuthnSubject reads a finite owned verifier snapshot under subject lock.
func (q *Queries) LoadWebAuthnSubject(ctx context.Context, state core.CredentialState, rp string, limit int) (*core.WebAuthnSubject, error) {
	if limit < 1 || limit > 20 {
		return nil, core.ErrWebAuthn
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, state)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	handle, err := queries.WebAuthnHandle(ctx, dal.WebAuthnHandleParams{UserID: state.UserID, RpID: rp})
	if err != nil {
		return nil, core.ErrWebAuthn
	}

	rows, err := queries.SubjectAuthenticators(ctx, dal.SubjectAuthenticatorsParams{UserID: state.UserID, RpID: rp, Column3: int32(limit + 1)})
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 || len(rows) > limit {
		return nil, core.ErrWebAuthn
	}

	result := &core.WebAuthnSubject{State: state, Handle: handle}
	for _, row := range rows {
		result.Factors = append(result.Factors, factorSnapshot(row))
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func assertionSnapshot(row dal.AuthPending) (core.AssertionPending, error) {
	p := core.AssertionPending{Purpose: core.AssertionPurpose(row.Purpose), State: core.CredentialState{UserID: row.UserID, Version: row.AuthVersion}, RPID: row.RpID, Handle: row.UserHandle, Ceremony: row.CeremonyData, Requirement: core.AccessRequirement{Proof: core.RequiredProof(row.RequiredProof), Revision: row.PolicyRevision, MaxAge: time.Duration(row.MaxAgeUs) * time.Microsecond}, ActorID: row.ActorID, ActorGeneration: row.ActorGeneration, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, Attempts: int(row.Attempts), Revision: row.Revision}
	if len(row.Digest) != 32 || len(row.RpBinding) != 32 || len(row.FactorBindings) == 0 || len(row.FactorBindings) > 4096 || row.PasswordAt.Valid {
		return p, core.ErrWebAuthn
	}

	copy(p.Digest[:], row.Digest)
	copy(p.RPBinding[:], row.RpBinding)
	copy(p.ActorDigest[:], row.ActorDigest)

	if row.LeaseUntil.Valid {
		p.LeaseUntil = row.LeaseUntil.Time
	}

	err := json.Unmarshal(row.FactorBindings, &p.Factors)

	return p, err
}

func lockAssertion(ctx context.Context, tx *sql.Tx, digest core.AssertionDigest, purpose core.AssertionPurpose) (core.AssertionPending, error) {
	queries := dal.New(tx)

	userID, err := queries.AssertionSubject(ctx, dal.AssertionSubjectParams{Digest: digest[:], Purpose: int16(purpose)})
	if err != nil {
		return core.AssertionPending{}, core.ErrWebAuthn
	}

	user, err := queries.GetUserForAuth(ctx, userID)
	if err != nil {
		return core.AssertionPending{}, err
	}

	row, err := queries.LockAssertion(ctx, dal.LockAssertionParams{Digest: digest[:], Purpose: int16(purpose)})
	if err != nil {
		return core.AssertionPending{}, core.ErrWebAuthn
	}

	if !user.Active || user.AuthVersion != row.AuthVersion {
		return core.AssertionPending{}, core.ErrCredentialChanged
	}

	return assertionSnapshot(row)
}

func assertionActor(ctx context.Context, queries *dal.Queries, p core.AssertionPending) (*core.Session, error) {
	if p.Purpose == core.AssertionSignin {
		return nil, nil
	}

	stored, err := queries.GetSessionForUpdate(ctx, p.ActorDigest[:])
	if err != nil {
		return nil, core.ErrSessionNotFound
	}

	if stored.ID != p.ActorID || stored.Generation != p.ActorGeneration || stored.UserID != p.State.UserID || stored.AuthVersion != p.State.Version {
		return nil, core.ErrSessionGeneration
	}

	return toAuthSession(stored)
}

// CreateAssertion uses the shared pending cap; reissue cannot reset attempts.
func (q *Queries) CreateAssertion(ctx context.Context, p core.AssertionPending, settings config.EnrollmentSettings) error {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, p.State)
	if err != nil {
		return err
	}

	queries := dal.New(tx)

	_, err = queries.ReclaimSubjectEnrollments(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	count, err := queries.CountEnrollments(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	if count >= int64(settings.MaxPending) {
		return core.ErrEnrollmentCapacity
	}

	actor, err := assertionActor(ctx, queries, p)
	if err != nil {
		return err
	}

	for _, binding := range p.Factors {
		row, err := queries.LockAuthenticator(ctx, binding.ID)
		if err != nil || row.UserID != p.State.UserID || row.RpID != p.RPID || row.Revision != binding.Revision {
			return core.ErrWebAuthn
		}
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, p.Requirement, settings)
	if err != nil {
		return err
	}

	if p.Revision != 0 || p.Attempts != 0 || !p.LeaseUntil.IsZero() {
		return core.ErrWebAuthn
	}

	if actor != nil {
		entry := p.Requirement
		entry.Proof = core.RequirePassword
		entry.MaxAge = 0

		err = entry.Evaluate(*actor, now)
		if err != nil {
			return err
		}
	}

	bindings, err := json.Marshal(p.Factors)
	if err != nil {
		return err
	}

	digest := p.ActorDigest[:]
	if p.Purpose == core.AssertionSignin {
		digest = []byte{}
	}

	err = queries.CreateAssertion(ctx, dal.CreateAssertionParams{Digest: p.Digest[:], Purpose: int16(p.Purpose), UserID: p.State.UserID, AuthVersion: p.State.Version, RpID: p.RPID, UserHandle: p.Handle, RpBinding: p.RPBinding[:], CeremonyData: p.Ceremony, RequiredProof: int16(p.Requirement.Proof), PolicyRevision: p.Requirement.Revision, MaxAgeUs: p.Requirement.MaxAge.Microseconds(), CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, FactorBindings: bindings, ActorID: p.ActorID, ActorDigest: digest, ActorGeneration: p.ActorGeneration})
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReserveAssertion persists shared budget and finite lease before signatures.
func (q *Queries) ReserveAssertion(ctx context.Context, digest core.AssertionDigest, purpose core.AssertionPurpose, required core.AccessRequirement, settings config.EnrollmentSettings) (*core.AssertionReservation, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockAssertion(ctx, tx, digest, purpose)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	actor, err := assertionActor(ctx, queries, p)
	if err != nil {
		return nil, err
	}

	result := &core.AssertionReservation{Actor: actor}

	for _, binding := range p.Factors {
		row, err := queries.LockAuthenticator(ctx, binding.ID)
		if err != nil || row.UserID != p.State.UserID || row.RpID != p.RPID || row.Revision != binding.Revision {
			return nil, core.ErrWebAuthn
		}

		result.Factors = append(result.Factors, factorSnapshot(row))
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return nil, err
	}

	if now.Before(p.LeaseUntil) {
		return nil, core.ErrEnrollmentBusy
	}

	if p.Attempts >= settings.PendingAttempts {
		return nil, core.ErrEnrollmentAttempts
	}

	if actor != nil {
		err = validateProofFactor(ctx, queries, *actor)
		if err != nil {
			return nil, err
		}

		entry := required
		entry.Proof = core.RequirePassword
		entry.MaxAge = 0

		now, err = queries.SessionClock(ctx)
		if err != nil {
			return nil, err
		}

		err = entry.Evaluate(*actor, now)
		if err != nil {
			return nil, err
		}
	}

	now, err = chargeSubjectFactorAttempt(ctx, queries, p.State.UserID, settings, now)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return nil, err
	}

	p.Attempts++
	p.Revision++
	p.LeaseUntil = now.Add(settings.Lease)

	err = queries.LeaseEnrollment(ctx, dal.LeaseEnrollmentParams{Digest: digest[:], LeaseUntil: sql.NullTime{Time: p.LeaseUntil, Valid: true}})
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	result.Pending = p

	return result, nil
}

func (q *Queries) ReleaseAssertion(ctx context.Context, digest core.AssertionDigest, revision int64) error {
	if q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return core.ErrWebAuthn
	}

	return dal.New(q.dbProvider.GetDB()).ReleaseEnrollment(ctx, dal.ReleaseEnrollmentParams{Digest: digest[:], Revision: revision})
}

func sameFactor(a, b core.WebAuthnFactor) bool {
	return a.FactorBinding == b.FactorBinding && a.ReplayRevision == b.ReplayRevision && a.Counter == b.Counter && a.BackupEligible == b.BackupEligible && a.BackupState == b.BackupState && bytes.Equal(a.CredentialID, b.CredentialID) && bytes.Equal(a.PublicKey, b.PublicKey) && bytes.Equal(a.Data, b.Data)
}

// validateProofFactor is used after subject/session locks before sampling time.
func validateProofFactor(ctx context.Context, queries *dal.Queries, session core.Session) error {
	if session.Proof.Method == core.PasswordProof {
		return nil
	}

	row, err := queries.LockAuthenticator(ctx, session.Proof.FactorID)
	if err != nil || row.UserID != session.UserID || row.Revision != session.Proof.FactorRevision || row.Kind != 1 || row.RecordVersion != 1 || !row.UserVerified {
		return core.ErrSessionProof
	}

	return nil
}

// CompleteAssertion commits replay state, pending consumption and access together.
func (q *Queries) CompleteAssertion(ctx context.Context, reserved core.AssertionReservation, c core.AssertionCompletion, required core.AccessRequirement, settings config.EnrollmentSettings, limit int) (*core.Session, error) {
	if limit < 1 || limit > 100 {
		return nil, core.ErrSessionCapacity
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockAssertion(ctx, tx, reserved.Pending.Digest, reserved.Pending.Purpose)
	if err != nil {
		return nil, err
	}

	if !p.Matches(reserved.Pending) {
		return nil, core.ErrWebAuthn
	}

	queries := dal.New(tx)

	actor, err := assertionActor(ctx, queries, p)
	if err != nil {
		return nil, err
	}

	if p.Purpose == core.AssertionSignin {
		_, err = queries.ReclaimSubjectSessions(ctx, p.State.UserID)
		if err != nil {
			return nil, err
		}

		count, err := queries.CountSubjectSessions(ctx, p.State.UserID)
		if err != nil {
			return nil, err
		}

		if count >= int64(limit) {
			return nil, core.ErrSessionCapacity
		}
	}

	ids := []string{c.Factor.ID}
	if actor != nil && actor.Proof.Method == core.WebAuthnProof {
		ids = append(ids, actor.Proof.FactorID)
	}

	slices.Sort(ids)

	for _, id := range slices.Compact(ids) {
		row, err := queries.LockAuthenticator(ctx, id)
		if err != nil || row.UserID != p.State.UserID {
			return nil, core.ErrWebAuthn
		}

		if id == c.Factor.ID && (row.RpID != p.RPID || !sameFactor(factorSnapshot(row), c.Factor)) {
			return nil, core.ErrWebAuthn
		}

		if actor != nil && id == actor.Proof.FactorID && row.Revision != actor.Proof.FactorRevision {
			return nil, core.ErrSessionProof
		}
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Check(p, actor, now, required, settings)
	if err != nil {
		return nil, err
	}

	err = queries.AcceptAssertionCounter(ctx, dal.AcceptAssertionCounterParams{ID: c.Factor.ID, SignCount: int64(c.Counter), BackupState: c.BackupState, CredentialData: c.Data})
	if err != nil {
		return nil, err
	}

	count, err := queries.ConsumeAssertion(ctx, dal.ConsumeAssertionParams{Digest: p.Digest[:], Purpose: int16(p.Purpose), Revision: p.Revision})
	if err != nil {
		return nil, err
	}

	if count != 1 {
		return nil, core.ErrWebAuthn
	}

	record := c.Session

	var stored dal.Session
	if p.Purpose == core.AssertionSignin {
		stored, err = queries.CreateSession(ctx, sessionInsert(record))
	} else {
		stored, err = queries.RotateSession(ctx, sessionRotation(record, p.ActorDigest, p.ActorGeneration))
	}

	if err != nil {
		return nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Check(p, actor, now, required, settings)
	if err != nil {
		return nil, err
	}

	result, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func sessionInsert(s core.SessionRecord) dal.CreateSessionParams {
	return dal.CreateSessionParams{ID: s.ID, UserID: s.UserID, TokenDigest: s.Digest[:], AuthVersion: s.AuthVersion, PolicyRevision: s.PolicyRevision, Generation: s.Generation, ProofMethod: int16(s.Proof.Method), ProofVerifiedAt: s.Proof.VerifiedAt, ProofFactorID: s.Proof.FactorID, ProofFactorRevision: s.Proof.FactorRevision, AuthenticatedAt: s.AuthenticatedAt, CreatedAt: s.CreatedAt, LastActivityAt: s.LastActivityAt, ExpiresAt: s.ExpiresAt, InactivityUs: s.InactivityTTL.Microseconds()}
}
func sessionRotation(s core.SessionRecord, old core.SessionDigest, generation int64) dal.RotateSessionParams {
	return dal.RotateSessionParams{OldDigest: old[:], ExpectedGeneration: generation, NewDigest: s.Digest[:], PolicyRevision: s.PolicyRevision, ProofMethod: int16(s.Proof.Method), ProofVerifiedAt: s.Proof.VerifiedAt, ProofFactorID: s.Proof.FactorID, ProofFactorRevision: s.Proof.FactorRevision, AuthenticatedAt: s.AuthenticatedAt, LastActivityAt: s.LastActivityAt, ExpiresAt: s.ExpiresAt, InactivityUs: s.InactivityTTL.Microseconds()}
}
