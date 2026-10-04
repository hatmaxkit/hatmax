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
	"errors"
	"math"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

var _ core.FallbackQueries = (*Queries)(nil)

func totpSnapshot(row dal.TotpAuthenticator) core.FallbackFactor {
	return core.FallbackFactor{FactorBinding: core.FactorBinding{ID: row.ID, Revision: row.Revision}, ReplayRevision: row.ReplayRevision, KeyID: row.KeyID, Envelope: row.Envelope, AcceptedStep: row.AcceptedStep}
}
func fallbackFactor(ctx context.Context, queries *dal.Queries, state core.CredentialState, method core.FallbackMethod) (core.FallbackFactor, error) {
	if method == core.FallbackTOTP {
		row, err := queries.LockTOTP(ctx, state.UserID)
		if err != nil || row.RecordVersion != 1 || row.ReplayRevision == math.MaxInt64 {
			return core.FallbackFactor{}, core.ErrFallback
		}

		return totpSnapshot(row), nil
	}

	if method == core.FallbackBackup {
		row, err := queries.LockBackupSet(ctx, state.UserID)
		if err != nil {
			return core.FallbackFactor{}, core.ErrFallback
		}

		return core.FallbackFactor{FactorBinding: core.FactorBinding{ID: row.ID, Revision: row.Revision}}, nil
	}

	return core.FallbackFactor{}, core.ErrFallback
}
func (q *Queries) LoadFallbackFactor(ctx context.Context, state core.CredentialState, method core.FallbackMethod) (core.FactorBinding, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return core.FactorBinding{}, err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, state)
	if err != nil {
		return core.FactorBinding{}, err
	}

	f, err := fallbackFactor(ctx, dal.New(tx), state, method)
	if err != nil {
		return core.FactorBinding{}, err
	}

	err = tx.Commit()

	return f.FactorBinding, err
}
func fallbackSnapshot(row dal.AuthPending) (core.FallbackPending, error) {
	p := core.FallbackPending{Purpose: core.FallbackPurpose(row.Purpose), State: core.CredentialState{UserID: row.UserID, Version: row.AuthVersion}, Requirement: core.AccessRequirement{Proof: core.RequiredProof(row.RequiredProof), Revision: row.PolicyRevision, MaxAge: time.Duration(row.MaxAgeUs) * time.Microsecond}, ActorID: row.ActorID, ActorGeneration: row.ActorGeneration, PasswordAt: row.PasswordAt.Time, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, Attempts: int(row.Attempts), Revision: row.Revision}
	if len(row.Digest) != 32 || row.RpID.Valid || len(row.UserHandle) != 0 || len(row.RpBinding) != 32 || !bytes.Equal(row.RpBinding, make([]byte, 32)) || len(row.FactorBindings) != 0 || !row.PasswordAt.Valid || len(row.CeremonyData) == 0 || len(row.CeremonyData) > 1024 {
		return p, core.ErrFallback
	}

	copy(p.Digest[:], row.Digest)
	copy(p.ActorDigest[:], row.ActorDigest)

	if row.LeaseUntil.Valid {
		p.LeaseUntil = row.LeaseUntil.Time
	}

	decoder := json.NewDecoder(bytes.NewReader(row.CeremonyData))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&p.Material)

	return p, err
}
func lockFallback(ctx context.Context, tx *sql.Tx, digest core.FallbackDigest, purpose core.FallbackPurpose) (core.FallbackPending, error) {
	if purpose < core.FallbackSetup || purpose > core.FallbackStepUp {
		return core.FallbackPending{}, core.ErrFallback
	}

	queries := dal.New(tx)

	id, err := queries.AssertionSubject(ctx, dal.AssertionSubjectParams{Digest: digest[:], Purpose: int16(purpose)})
	if err != nil {
		return core.FallbackPending{}, core.ErrFallback
	}

	user, err := queries.GetUserForAuth(ctx, id)
	if err != nil {
		return core.FallbackPending{}, err
	}

	row, err := queries.LockAssertion(ctx, dal.LockAssertionParams{Digest: digest[:], Purpose: int16(purpose)})
	if err != nil {
		return core.FallbackPending{}, core.ErrFallback
	}

	if !user.Active || user.AuthVersion != row.AuthVersion {
		return core.FallbackPending{}, core.ErrCredentialChanged
	}

	return fallbackSnapshot(row)
}
func fallbackActor(ctx context.Context, queries *dal.Queries, p core.FallbackPending) (*core.Session, error) {
	if p.Purpose != core.FallbackStepUp {
		return nil, nil
	}

	row, err := queries.GetSessionForUpdate(ctx, p.ActorDigest[:])
	if err != nil || row.ID != p.ActorID || row.Generation != p.ActorGeneration || row.UserID != p.State.UserID || row.AuthVersion != p.State.Version {
		return nil, core.ErrSessionGeneration
	}

	return toAuthSession(row)
}
func checkFallbackActor(ctx context.Context, queries *dal.Queries, actor *core.Session, required core.AccessRequirement) error {
	if actor == nil {
		return nil
	}

	err := validateProofFactor(ctx, queries, *actor)
	if err != nil {
		return err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	entry := required
	entry.Proof = core.RequirePassword
	entry.MaxAge = 0

	return entry.Evaluate(*actor, now)
}
func (q *Queries) CreateFallback(ctx context.Context, p core.FallbackPending, settings config.FallbackSettings) error {
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

	actor, err := fallbackActor(ctx, queries, p)
	if err != nil {
		return err
	}

	err = checkFallbackActor(ctx, queries, actor, p.Requirement)
	if err != nil {
		return err
	}

	if p.Purpose == core.FallbackSetup {
		setups, err := queries.CountFactorSetups(ctx, p.State.UserID)
		if err != nil {
			return err
		}

		if setups >= int64(settings.MaxAuthenticators) {
			return core.ErrEnrollmentCapacity
		}

		count, err := queries.CountAuthenticators(ctx, p.State.UserID)
		if err != nil {
			return err
		}

		if count != 0 {
			return core.ErrFallback
		}

		_, err = queries.LockBackupSet(ctx, p.State.UserID)
		if !errors.Is(err, sql.ErrNoRows) {
			return core.ErrFallback
		}
	} else {
		f, err := fallbackFactor(ctx, queries, p.State, p.Material.Method)
		if err != nil || f.FactorBinding != p.Material.Factor {
			return core.ErrFallback
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
		return core.ErrFallback
	}

	payload, err := json.Marshal(p.Material)
	if err != nil {
		return err
	}

	actorDigest := []byte{}
	if actor != nil {
		actorDigest = p.ActorDigest[:]
	}

	err = queries.CreateFallback(ctx, dal.CreateFallbackParams{Digest: p.Digest[:], Purpose: int16(p.Purpose), UserID: p.State.UserID, AuthVersion: p.State.Version, CeremonyData: payload, RequiredProof: int16(p.Requirement.Proof), PolicyRevision: p.Requirement.Revision, MaxAgeUs: p.Requirement.MaxAge.Microseconds(), PasswordAt: sql.NullTime{Time: p.PasswordAt, Valid: true}, CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, ActorID: p.ActorID, ActorDigest: actorDigest, ActorGeneration: p.ActorGeneration})
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ReserveFallback spends the same durable subject budget as WebAuthn. A
// canonical nonexistent backup ID still consumes attempts before returning no proof.
func (q *Queries) ReserveFallback(ctx context.Context, digest core.FallbackDigest, purpose core.FallbackPurpose, codeID string, required core.AccessRequirement, settings config.FallbackSettings) (*core.FallbackReservation, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockFallback(ctx, tx, digest, purpose)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	actor, err := fallbackActor(ctx, queries, p)
	if err != nil {
		return nil, err
	}

	err = checkFallbackActor(ctx, queries, actor, required)
	if err != nil {
		return nil, err
	}

	result := &core.FallbackReservation{Actor: actor}

	if purpose != core.FallbackSetup {
		f, err := fallbackFactor(ctx, queries, p.State, p.Material.Method)
		if err != nil || f.FactorBinding != p.Material.Factor {
			return nil, core.ErrFallback
		}

		result.Factor = f
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

	now, err = chargeSubjectFactorAttempt(ctx, queries, p.State.UserID, settings.EnrollmentSettings, now)
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

	if p.Material.Method == core.FallbackBackup {
		row, err := queries.LockBackupCode(ctx, dal.LockBackupCodeParams{UserID: p.State.UserID, ID: codeID})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		if err == nil {
			result.Backup = core.BackupVerifier{ID: row.ID, SetID: row.SetID, Record: row.Verifier, Revision: row.Revision}
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	result.Pending = p

	return result, nil
}
func (q *Queries) ReleaseFallback(ctx context.Context, digest core.FallbackDigest, revision int64) error {
	return dal.New(q.dbProvider.GetDB()).ReleaseEnrollment(ctx, dal.ReleaseEnrollmentParams{Digest: digest[:], Revision: revision})
}
func sameTOTP(a, b core.FallbackFactor) bool {
	return a.FactorBinding == b.FactorBinding && a.ReplayRevision == b.ReplayRevision && a.AcceptedStep == b.AcceptedStep && a.KeyID == b.KeyID && bytes.Equal(a.Envelope, b.Envelope)
}
func (q *Queries) CompleteTOTPSetup(ctx context.Context, r core.FallbackReservation, step int64, required core.AccessRequirement, settings config.FallbackSettings) error {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	p, err := lockFallback(ctx, tx, r.Pending.Digest, core.FallbackSetup)
	if err != nil {
		return err
	}

	if !p.Matches(r.Pending) {
		return core.ErrFallback
	}

	queries := dal.New(tx)

	count, err := queries.CountAuthenticators(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	if count != 0 {
		return core.ErrFallback
	}

	_, err = queries.LockBackupSet(ctx, p.State.UserID)
	if !errors.Is(err, sql.ErrNoRows) {
		return core.ErrFallback
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return err
	}

	if p.Attempts < 1 || p.Revision < 1 || !now.Before(p.LeaseUntil) || !crypto.TOTPInWindow(step, now, settings.Skew) {
		return core.ErrFallback
	}

	err = queries.CreateTOTP(ctx, dal.CreateTOTPParams{ID: p.Material.Factor.ID, UserID: p.State.UserID, KeyID: p.Material.KeyID, Envelope: p.Material.Envelope, AcceptedStep: step, CreatedAt: now})
	if err != nil {
		return err
	}

	err = queries.AdvanceAuthenticatorVersion(ctx, dal.AdvanceAuthenticatorVersionParams{ID: p.State.UserID, UpdatedAt: now})
	if err != nil {
		return err
	}

	err = queries.DeleteUserSessions(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	err = queries.DeleteSubjectEnrollments(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return err
	}

	if !now.Before(p.LeaseUntil) || !crypto.TOTPInWindow(step, now, settings.Skew) {
		return core.ErrFallback
	}

	return tx.Commit()
}
func (q *Queries) CompleteFallback(ctx context.Context, r core.FallbackReservation, c core.FallbackCompletion, required core.AccessRequirement, settings config.FallbackSettings, limit int) (*core.Session, error) {
	if limit < 1 || limit > 100 {
		return nil, core.ErrSessionCapacity
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	p, err := lockFallback(ctx, tx, r.Pending.Digest, r.Pending.Purpose)
	if err != nil {
		return nil, err
	}

	if !p.Matches(r.Pending) {
		return nil, core.ErrFallback
	}

	queries := dal.New(tx)

	actor, err := fallbackActor(ctx, queries, p)
	if err != nil {
		return nil, err
	}

	err = checkFallbackActor(ctx, queries, actor, required)
	if err != nil {
		return nil, err
	}

	if p.Purpose == core.FallbackSignin {
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

	f, err := fallbackFactor(ctx, queries, p.State, p.Material.Method)
	if err != nil || f.FactorBinding != p.Material.Factor {
		return nil, core.ErrFallback
	}

	if p.Material.Method == core.FallbackTOTP {
		if !sameTOTP(f, r.Factor) {
			return nil, core.ErrFallback
		}
	} else {
		row, err := queries.LockBackupCode(ctx, dal.LockBackupCodeParams{UserID: p.State.UserID, ID: r.Backup.ID})
		if err != nil || row.SetID != p.Material.Factor.ID || (core.BackupVerifier{ID: row.ID, SetID: row.SetID, Record: row.Verifier, Revision: row.Revision}) != r.Backup {
			return nil, core.ErrFallback
		}
	}

	current := r
	current.Pending = p
	current.Factor = f
	current.Actor = actor

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Check(current, now, required, settings)
	if err != nil {
		return nil, err
	}

	var count int64
	if p.Material.Method == core.FallbackTOTP {
		count, err = queries.AcceptTOTPStep(ctx, dal.AcceptTOTPStepParams{ID: f.ID, AcceptedStep: c.Step})
	} else {
		count, err = queries.ConsumeBackupCode(ctx, dal.ConsumeBackupCodeParams{ID: r.Backup.ID, SetID: r.Backup.SetID})
	}

	if err != nil {
		return nil, err
	}

	if count != 1 {
		return nil, core.ErrFallback
	}

	count, err = queries.ConsumeAssertion(ctx, dal.ConsumeAssertionParams{Digest: p.Digest[:], Purpose: int16(p.Purpose), Revision: p.Revision})
	if err != nil {
		return nil, err
	}

	if count != 1 {
		return nil, core.ErrFallback
	}

	var stored dal.Session
	if p.Purpose == core.FallbackSignin {
		stored, err = queries.CreateSession(ctx, sessionInsert(c.Session))
	} else {
		stored, err = queries.RotateSession(ctx, sessionRotation(c.Session, p.ActorDigest, p.ActorGeneration))
	}

	if err != nil {
		return nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = c.Check(current, now, required, settings)
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
func (q *Queries) ReserveBackupIssue(ctx context.Context, digest core.SessionDigest, required core.AccessRequirement, settings config.FallbackSettings) (*core.BackupIssueReservation, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	lookup, err := queries.GetSessionByDigest(ctx, digest[:])
	if err != nil {
		return nil, core.ErrSessionNotFound
	}

	state := core.CredentialState{UserID: lookup.UserID, Version: lookup.AuthVersion}

	_, err = lockCredential(ctx, tx, state)
	if err != nil {
		return nil, err
	}

	stored, err := queries.GetSessionForUpdate(ctx, digest[:])
	if err != nil {
		return nil, core.ErrSessionNotFound
	}

	actor, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	err = validateProofFactor(ctx, queries, *actor)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = required.Evaluate(*actor, now)
	if err != nil {
		return nil, err
	}

	if required.Proof == core.RequirePassword || required.MaxAge == 0 || required.MaxAge > settings.RecentProofAge || actor.Proof.Method == core.PasswordBackupProof || actor.Generation == math.MaxInt64 || state.Version == math.MaxInt64 {
		return nil, core.ErrSessionProof
	}

	now, err = chargeSubjectFactorAttempt(ctx, queries, state.UserID, settings.EnrollmentSettings, now)
	if err != nil {
		return nil, err
	}

	err = required.Evaluate(*actor, now)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &core.BackupIssueReservation{State: state, Actor: *actor, Digest: digest}, nil
}
func (q *Queries) ReplaceBackupSet(ctx context.Context, r core.BackupIssueReservation, set core.BackupSet, replacement core.SessionRecord, required core.AccessRequirement, settings config.FallbackSettings) (*core.Session, error) {
	if len(set.Codes) != settings.BackupCodes || len(set.ID) == 0 || len(set.ID) > 128 || required.Proof == core.RequirePassword || required.MaxAge == 0 || required.MaxAge > settings.RecentProofAge {
		return nil, core.ErrFallback
	}

	expected := r.Actor
	expected.AuthVersion++

	expected.Generation++
	if replacement.Session != expected || replacement.Digest == (core.SessionDigest{}) || replacement.Digest == r.Digest || r.Actor.Proof.Method == core.PasswordBackupProof {
		return nil, core.ErrFallback
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = lockCredential(ctx, tx, r.State)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	stored, err := queries.GetSessionForUpdate(ctx, r.Digest[:])
	if err != nil {
		return nil, core.ErrSessionNotFound
	}

	actor, err := toAuthSession(stored)
	if err != nil {
		return nil, err
	}

	if *actor != r.Actor || actor.UserID != r.State.UserID || actor.AuthVersion != r.State.Version {
		return nil, core.ErrSessionGeneration
	}

	err = validateProofFactor(ctx, queries, *actor)
	if err != nil {
		return nil, err
	}

	_, err = queries.LockBackupSet(ctx, r.State.UserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = required.Evaluate(*actor, now)
	if err != nil {
		return nil, err
	}

	err = queries.DeleteBackupSet(ctx, r.State.UserID)
	if err != nil {
		return nil, err
	}

	err = queries.CreateBackupSet(ctx, dal.CreateBackupSetParams{ID: set.ID, UserID: r.State.UserID, CreatedAt: now})
	if err != nil {
		return nil, err
	}

	for _, code := range set.Codes {
		if code.SetID != set.ID || code.Revision != 1 {
			return nil, core.ErrFallback
		}

		err = queries.CreateBackupCode(ctx, dal.CreateBackupCodeParams{ID: code.ID, SetID: code.SetID, Verifier: code.Record})
		if err != nil {
			return nil, err
		}
	}

	err = queries.AdvanceAuthenticatorVersion(ctx, dal.AdvanceAuthenticatorVersionParams{ID: r.State.UserID, UpdatedAt: now})
	if err != nil {
		return nil, err
	}

	err = queries.DeleteUserSessions(ctx, r.State.UserID)
	if err != nil {
		return nil, err
	}

	err = queries.DeleteSubjectEnrollments(ctx, r.State.UserID)
	if err != nil {
		return nil, err
	}

	stored, err = queries.CreateSession(ctx, sessionInsert(replacement))
	if err != nil {
		return nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = required.Evaluate(*actor, now)
	if err != nil {
		return nil, err
	}

	err = required.Evaluate(replacement.Session, now)
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
