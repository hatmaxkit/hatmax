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
	"math"
	"reflect"
	"slices"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

var _ core.FactorQueries = (*Queries)(nil)

// factorActor runs only after the subject has been locked. Current factor bindings
// and recent proof are checked again after all target locks and after writes.
func factorActor(ctx context.Context, queries *dal.Queries, digest core.SessionDigest, policy core.FactorPolicy, target core.FactorSelection, settings config.EnrollmentSettings) (*core.Session, error) {
	if policy.Check(settings) != nil {
		return nil, core.ErrFactorChange
	}

	row, err := queries.GetSessionForUpdate(ctx, digest[:])
	if err != nil {
		return nil, core.ErrSessionNotFound
	}

	actor, err := toAuthSession(row)
	if err != nil {
		return nil, err
	}

	if actor.AuthVersion == math.MaxInt64 || actor.Generation == math.MaxInt64 {
		return nil, core.ErrFactorChange
	}

	err = lockManagedFactors(ctx, queries, *actor, target)
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

	err = policy.Management.Evaluate(*actor, now)

	return actor, err
}
func factorSubject(ctx context.Context, tx *sql.Tx, digest core.SessionDigest) (core.CredentialState, error) {
	row, err := dal.New(tx).GetSessionByDigest(ctx, digest[:])
	if err != nil {
		return core.CredentialState{}, core.ErrSessionNotFound
	}

	state := core.CredentialState{UserID: row.UserID, Version: row.AuthVersion}
	_, err = lockCredential(ctx, tx, state)

	return state, err
}
func safeFactors(ctx context.Context, queries *dal.Queries, user string) ([]core.Factor, error) {
	rows, err := queries.SafeFactors(ctx, user)
	if err != nil {
		return nil, err
	}

	if len(rows) > 20 {
		return nil, core.ErrEnrollmentCapacity
	}

	result := make([]core.Factor, 0, len(rows))
	for _, row := range rows {
		f := core.Factor{FactorSelection: core.FactorSelection{ID: row.ID, Kind: core.FactorKind(row.Kind), Revision: row.Revision}, CreatedAt: row.CreatedAt, BackupEligible: row.BackupEligible, BackupState: row.BackupState}
		if f.Check() != nil {
			return nil, core.ErrFactorChange
		}

		result = append(result, f)
	}

	return result, nil
}
func (q *Queries) ListFactors(ctx context.Context, digest core.SessionDigest, policy core.FactorPolicy, settings config.EnrollmentSettings) ([]core.Factor, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	state, err := factorSubject(ctx, tx, digest)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	actor, err := factorActor(ctx, queries, digest, policy, core.FactorSelection{}, settings)
	if err != nil {
		return nil, err
	}

	factors, err := safeFactors(ctx, queries, state.UserID)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = policy.Management.Evaluate(*actor, now)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()

	return factors, err
}

// lockFactor checks an exact owned security revision, never a client-selected owner.
func lockFactor(ctx context.Context, queries *dal.Queries, user string, target core.FactorSelection) error {
	if target.Check() != nil {
		return core.ErrFactorChange
	}

	if target.Kind == core.FactorWebAuthn {
		row, err := queries.LockManagedAuthenticator(ctx, dal.LockManagedAuthenticatorParams{UserID: user, ID: target.ID})
		if err != nil {
			return core.ErrFactorChange
		}

		if row.UserID != user || row.Revision != target.Revision || row.Kind != 1 || row.RecordVersion != 1 || !row.UserVerified {
			return core.ErrFactorChange
		}
	} else {
		row, err := queries.LockTOTP(ctx, user)
		if err != nil {
			return core.ErrFactorChange
		}

		if row.ID != target.ID || row.Revision != target.Revision || row.RecordVersion != 1 {
			return core.ErrFactorChange
		}
	}

	return nil
}
func factorCapacity(ctx context.Context, queries *dal.Queries, p core.FactorChangePending, settings config.EnrollmentSettings) error {
	factors, err := safeFactors(ctx, queries, p.State.UserID)
	if err != nil {
		return err
	}

	if len(factors) == 0 {
		return core.ErrFactorChange
	}

	if p.Target != (core.FactorSelection{}) {
		err = lockFactor(ctx, queries, p.State.UserID, p.Target)
		if err != nil {
			return err
		}
	}

	if p.Kind == core.FactorTOTP {
		for _, f := range factors {
			if f.Kind == core.FactorTOTP && f.FactorSelection != p.Target {
				return core.ErrFactorChange
			}
		}
	}

	setups, err := queries.CountFactorSetups(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	if int64(len(factors))+setups >= int64(settings.MaxAuthenticators) {
		return core.ErrEnrollmentCapacity
	}

	return nil
}
func (q *Queries) CreateFactorChange(ctx context.Context, p core.FactorChangePending, settings config.EnrollmentSettings) error {
	if p.Attempts != 0 || p.Revision != 0 || !p.LeaseUntil.IsZero() {
		return core.ErrFactorChange
	}

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

	actor, err := factorActor(ctx, queries, p.ActorDigest, p.Policy, p.Target, settings)
	if err != nil {
		return err
	}

	if *actor != p.Actor {
		return core.ErrSessionGeneration
	}

	_, err = queries.ReclaimSubjectEnrollments(ctx, p.State.UserID)
	if err != nil {
		return err
	}

	err = factorCapacity(ctx, queries, p, settings)
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

	if p.Kind == core.FactorWebAuthn {
		handle, handleErr := queries.WebAuthnHandle(ctx, dal.WebAuthnHandleParams{UserID: p.State.UserID, RpID: p.RPID})
		if handleErr != nil || !bytes.Equal(handle, p.Handle) {
			return core.ErrFactorChange
		}
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, p.Policy, settings)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}

	if len(encoded) > 16384 {
		return core.ErrFactorChange
	}

	err = queries.CreateFallback(ctx, dal.CreateFallbackParams{Digest: p.Digest[:], Purpose: 7, UserID: p.State.UserID, AuthVersion: p.State.Version, CeremonyData: encoded, RequiredProof: int16(p.Policy.Management.Proof), PolicyRevision: p.Policy.Management.Revision, MaxAgeUs: p.Policy.Management.MaxAge.Microseconds(), CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt, ActorID: p.Actor.ID, ActorDigest: p.ActorDigest[:], ActorGeneration: p.Actor.Generation})
	if err != nil {
		return err
	}

	return tx.Commit()
}
func lockFactorChange(ctx context.Context, queries *dal.Queries, digest core.EnrollmentDigest) (*core.FactorChangePending, error) {
	user, err := queries.AssertionSubject(ctx, dal.AssertionSubjectParams{Digest: digest[:], Purpose: 7})
	if err != nil {
		return nil, core.ErrFactorChange
	}

	rowUser, err := queries.GetUserForAuth(ctx, user)
	if err != nil {
		return nil, err
	}

	row, err := queries.LockAssertion(ctx, dal.LockAssertionParams{Digest: digest[:], Purpose: 7})
	if err != nil {
		return nil, core.ErrFactorChange
	}

	if !rowUser.Active || rowUser.AuthVersion != row.AuthVersion {
		return nil, core.ErrCredentialChanged
	}

	var p core.FactorChangePending

	decoder := json.NewDecoder(bytes.NewReader(row.CeremonyData))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(&p)
	if err != nil {
		return nil, core.ErrFactorChange
	}

	if p.Digest != digest || p.State.UserID != user || p.State.Version != row.AuthVersion || p.Actor.ID != row.ActorID || p.Actor.Generation != row.ActorGeneration || !bytes.Equal(p.ActorDigest[:], row.ActorDigest) || p.Policy.Management.Proof != core.RequiredProof(row.RequiredProof) || p.Policy.Management.Revision != row.PolicyRevision || p.Policy.Management.MaxAge.Microseconds() != row.MaxAgeUs || !p.CreatedAt.Equal(row.CreatedAt) || !p.ExpiresAt.Equal(row.ExpiresAt) || p.Attempts != 0 || p.Revision != 0 || !p.LeaseUntil.IsZero() {
		return nil, core.ErrFactorChange
	}

	p.Attempts = int(row.Attempts)
	p.Revision = row.Revision
	p.LeaseUntil = row.LeaseUntil.Time

	return &p, nil
}
func currentChange(ctx context.Context, queries *dal.Queries, p core.FactorChangePending, policy core.FactorPolicy, settings config.EnrollmentSettings) error {
	actor, err := factorActor(ctx, queries, p.ActorDigest, policy, p.Target, settings)
	if err != nil {
		return err
	}

	if *actor != p.Actor {
		return core.ErrSessionGeneration
	}

	if p.Target != (core.FactorSelection{}) {
		err = lockFactor(ctx, queries, p.State.UserID, p.Target)
		if err != nil {
			return err
		}
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	return p.Check(now, policy, settings)
}
func (q *Queries) ReserveFactorChange(ctx context.Context, digest core.EnrollmentDigest, actorDigest core.SessionDigest, policy core.FactorPolicy, settings config.EnrollmentSettings) (*core.FactorChangePending, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	p, err := lockFactorChange(ctx, queries, digest)
	if err != nil {
		return nil, err
	}

	if actorDigest != p.ActorDigest {
		return nil, core.ErrFactorChange
	}

	err = currentChange(ctx, queries, *p, policy, settings)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	if !p.LeaseUntil.IsZero() && now.Before(p.LeaseUntil) {
		return nil, core.ErrEnrollmentBusy
	}

	if p.Attempts >= settings.PendingAttempts {
		return nil, core.ErrEnrollmentAttempts
	}

	now, err = chargeSubjectFactorAttempt(ctx, queries, p.State.UserID, settings, now)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, policy, settings)
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

	return p, err
}
func deleteFactor(ctx context.Context, queries *dal.Queries, user string, target core.FactorSelection) error {
	var (
		count int64
		err   error
	)
	if target.Kind == core.FactorWebAuthn {
		count, err = queries.RemoveWebAuthnFactor(ctx, dal.RemoveWebAuthnFactorParams{UserID: user, ID: target.ID, Revision: target.Revision})
	} else {
		count, err = queries.RemoveTOTPFactor(ctx, dal.RemoveTOTPFactorParams{UserID: user, ID: target.ID, Revision: target.Revision})
	}

	if err != nil {
		return err
	}

	if count != 1 {
		return core.ErrFactorChange
	}

	return nil
}
func remainingFactors(ctx context.Context, queries *dal.Queries, user string, policy core.FactorPolicy) error {
	factors, err := safeFactors(ctx, queries, user)
	if err != nil {
		return err
	}

	if len(factors) == 0 {
		return core.ErrFactorChange
	}

	if policy.Access == core.RequirePhishingResistantMFA {
		for _, f := range factors {
			if f.Kind == core.FactorWebAuthn {
				return nil
			}
		}

		return core.ErrSessionProof
	}

	return nil
}
func finishFactorMutation(ctx context.Context, queries *dal.Queries, actor core.Session, digest core.SessionDigest, target core.FactorSelection, replacement core.SessionRecord, policy core.FactorPolicy, settings config.EnrollmentSettings) (*core.Session, error) {
	expected := actor
	expected.AuthVersion++

	expected.Generation++
	if replacement.Session != expected || replacement.Digest == (core.SessionDigest{}) || replacement.Digest == digest {
		return nil, core.ErrSessionGeneration
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = policy.Management.Evaluate(actor, now)
	if err != nil {
		return nil, err
	}

	err = remainingFactors(ctx, queries, actor.UserID, policy)
	if err != nil {
		return nil, err
	}

	access := policy.Management
	access.Proof = policy.Access
	access.MaxAge = 0

	retain := target.ID != actor.Proof.FactorID && access.Evaluate(actor, now) == nil
	if retain {
		err = validateProofFactor(ctx, queries, actor)
		if err != nil {
			return nil, err
		}
	}

	err = queries.AdvanceAuthenticatorVersion(ctx, dal.AdvanceAuthenticatorVersionParams{ID: actor.UserID, UpdatedAt: now})
	if err != nil {
		return nil, err
	}

	err = queries.DeleteUserSessions(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	err = queries.DeleteSubjectEnrollments(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}

	var result *core.Session

	if retain {
		stored, insertErr := queries.CreateSession(ctx, sessionInsert(replacement))
		if insertErr != nil {
			return nil, insertErr
		}

		result, err = toAuthSession(stored)
		if err != nil {
			return nil, err
		}
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = policy.Management.Evaluate(actor, now)
	if err != nil {
		return nil, err
	}

	if result != nil {
		err = policy.Management.Evaluate(*result, now)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
func (q *Queries) CompleteFactorChange(ctx context.Context, reserved core.FactorChangePending, record core.FactorChangeRecord, replacement core.SessionRecord, policy core.FactorPolicy, settings config.EnrollmentSettings) (*core.Factor, *core.Session, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	p, err := lockFactorChange(ctx, queries, reserved.Digest)
	if err != nil {
		return nil, nil, err
	}

	if !reflect.DeepEqual(*p, reserved) || p.Revision < 1 || p.Attempts < 1 {
		return nil, nil, core.ErrFactorChange
	}

	err = currentChange(ctx, queries, *p, policy, settings)
	if err != nil {
		return nil, nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, nil, err
	}

	if p.LeaseUntil.IsZero() || !now.Before(p.LeaseUntil) || record.VerifiedAt.Before(p.LeaseUntil.Add(-settings.Lease)) || record.VerifiedAt.After(now) {
		return nil, nil, core.ErrFactorChange
	}
	// The target is deleted in the same transaction as verified activation. Any
	// uniqueness, policy, rotation or final time failure restores the original factor.
	if p.Target != (core.FactorSelection{}) {
		err = deleteFactor(ctx, queries, p.State.UserID, p.Target)
		if err != nil {
			return nil, nil, err
		}
	}

	factor := &core.Factor{FactorSelection: core.FactorSelection{Kind: p.Kind, Revision: 1}, CreatedAt: now}
	if p.Kind == core.FactorWebAuthn {
		r := record.Registration
		if r == nil || !r.VerifiedAt.Equal(record.VerifiedAt) || r.ID == "" || len(r.CredentialID) == 0 || len(r.CredentialID) > 1024 || len(r.PublicKey) == 0 || len(r.PublicKey) > 4096 || len(r.Data) == 0 || len(r.Data) > 65536 || r.BackupState && !r.BackupEligible || record.Step != 0 {
			return nil, nil, core.ErrFactorChange
		}

		err = queries.ConfirmAuthenticator(ctx, dal.ConfirmAuthenticatorParams{ID: r.ID, UserID: p.State.UserID, RpID: p.RPID, CredentialID: r.CredentialID, PublicKey: r.PublicKey, CredentialData: r.Data, SignCount: int64(r.Counter), BackupEligible: r.BackupEligible, BackupState: r.BackupState, CreatedAt: now})
		factor.ID = r.ID
		factor.BackupEligible = r.BackupEligible
		factor.BackupState = r.BackupState
	} else {
		if record.Registration != nil || record.Step < 1 || !crypto.TOTPInWindow(record.Step, now, p.Material.Skew) {
			return nil, nil, core.ErrFactorChange
		}

		err = queries.CreateTOTP(ctx, dal.CreateTOTPParams{ID: p.Material.Factor.ID, UserID: p.State.UserID, KeyID: p.Material.KeyID, Envelope: p.Material.Envelope, AcceptedStep: record.Step, CreatedAt: now})
		factor.ID = p.Material.Factor.ID
	}

	if err != nil {
		return nil, nil, err
	}

	result, err := finishFactorMutation(ctx, queries, p.Actor, p.ActorDigest, p.Target, replacement, policy, settings)
	if err != nil {
		return nil, nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, nil, err
	}

	err = p.Check(now, policy, settings)
	if err != nil || !now.Before(p.LeaseUntil) || p.Kind == core.FactorTOTP && !crypto.TOTPInWindow(record.Step, now, p.Material.Skew) {
		return nil, nil, core.ErrFactorChange
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, err
	}

	return factor, result, nil
}
func (q *Queries) RemoveFactor(ctx context.Context, digest core.SessionDigest, target core.FactorSelection, replacement core.SessionRecord, policy core.FactorPolicy, settings config.EnrollmentSettings) (*core.Session, error) {
	if target.Check() != nil {
		return nil, core.ErrFactorChange
	}

	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	state, err := factorSubject(ctx, tx, digest)
	if err != nil {
		return nil, err
	}

	queries := dal.New(tx)

	actor, err := factorActor(ctx, queries, digest, policy, target, settings)
	if err != nil {
		return nil, err
	}

	err = lockFactor(ctx, queries, state.UserID, target)
	if err != nil {
		return nil, err
	}

	err = deleteFactor(ctx, queries, state.UserID, target)
	if err != nil {
		return nil, err
	}

	result, err := finishFactorMutation(ctx, queries, *actor, digest, target, replacement, policy, settings)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Initial acquisition uses stable IDs even when actor and target have different
// kinds. Owner-constrained SQL cannot lock a foreign subject's factor.
func lockManagedFactors(ctx context.Context, queries *dal.Queries, actor core.Session, target core.FactorSelection) error {
	selections := make([]core.FactorSelection, 0, 2)

	switch actor.Proof.Method {
	case core.WebAuthnProof:
		selections = append(selections, core.FactorSelection{Kind: core.FactorWebAuthn, ID: actor.Proof.FactorID, Revision: actor.Proof.FactorRevision})
	case core.PasswordTOTPProof:
		selections = append(selections, core.FactorSelection{Kind: core.FactorTOTP, ID: actor.Proof.FactorID, Revision: actor.Proof.FactorRevision})
	case core.PasswordBackupProof:
		selections = append(selections, core.FactorSelection{Kind: 3, ID: actor.Proof.FactorID, Revision: actor.Proof.FactorRevision})
	}

	if target != (core.FactorSelection{}) {
		selections = append(selections, target)
	}

	slices.SortFunc(selections, func(a, b core.FactorSelection) int {
		if a.ID < b.ID {
			return -1
		}

		if a.ID > b.ID {
			return 1
		}

		return 0
	})

	for _, selection := range selections {
		if selection.Kind == 3 {
			row, err := queries.LockBackupSet(ctx, actor.UserID)
			if err != nil || row.ID != selection.ID || row.Revision != selection.Revision {
				return core.ErrFactorChange
			}
		} else {
			err := lockFactor(ctx, queries, actor.UserID, selection)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
