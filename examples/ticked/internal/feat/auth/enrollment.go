// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

var _ core.AuthenticatorQueries = (*Queries)(nil)

// EnsureWebAuthnHandle creates an opaque stable RP handle under the subject lock.
func (q *Queries) EnsureWebAuthnHandle(ctx context.Context, state core.CredentialState, rp string, proposed []byte) ([]byte, error) {
	if len(proposed) != 32 || len(rp) == 0 || len(rp) > 253 {
		return nil, core.ErrEnrollment
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

	handle, err := dal.New(tx).EnsureWebAuthnHandle(ctx, dal.EnsureWebAuthnHandleParams{UserID: state.UserID, RpID: rp, Handle: proposed})
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return handle, nil
}

func initialEnrollment(ctx context.Context, queries *dal.Queries, userID string, settings config.EnrollmentSettings) error {
	count, err := queries.CountAuthenticators(ctx, userID)
	if err != nil {
		return err
	}

	if count != 0 || count >= int64(settings.MaxAuthenticators) {
		return core.ErrEnrollment
	}

	return nil
}

// CreateEnrollment admits retained pending rows without eviction. Expired rows
// are reclaimed in a finite subject batch before capacity is checked.
func (q *Queries) CreateEnrollment(ctx context.Context, p core.EnrollmentPending, settings config.EnrollmentSettings) error {
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

	err = initialEnrollment(ctx, queries, p.State.UserID, settings)
	if err != nil {
		return err
	}

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

	if p.Attempts != 0 || p.Revision != 0 || !p.LeaseUntil.IsZero() {
		return core.ErrEnrollment
	}
	// A caller-owned pending snapshot cannot substitute a different stored handle.
	handle, err := queries.EnsureWebAuthnHandle(ctx, dal.EnsureWebAuthnHandleParams{UserID: p.State.UserID, RpID: p.RPID, Handle: p.Handle})
	if err != nil {
		return err
	}

	if !bytes.Equal(handle, p.Handle) {
		return core.ErrEnrollment
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return err
	}

	err = p.Check(now, p.Requirement, settings)
	if err != nil {
		return err
	}

	err = queries.CreateEnrollment(ctx, dal.CreateEnrollmentParams{RpBinding: p.RPBinding[:], Digest: p.Digest[:], UserID: p.State.UserID, AuthVersion: p.State.Version, RpID: p.RPID, UserHandle: p.Handle, CeremonyData: p.Ceremony, RequiredProof: int16(p.Requirement.Proof), PolicyRevision: p.Requirement.Revision, MaxAgeUs: p.Requirement.MaxAge.Microseconds(), PasswordAt: p.PasswordAt, CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt})
	if err != nil {
		return err
	}

	return tx.Commit()
}

func enrollmentSnapshot(row dal.AuthPending) core.EnrollmentPending {
	p := core.EnrollmentPending{State: core.CredentialState{UserID: row.UserID, Version: row.AuthVersion}, RPID: row.RpID, Handle: row.UserHandle, Ceremony: row.CeremonyData, Requirement: core.AccessRequirement{Proof: core.RequiredProof(row.RequiredProof), Revision: row.PolicyRevision, MaxAge: time.Duration(row.MaxAgeUs) * time.Microsecond}, PasswordAt: row.PasswordAt, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt, Attempts: int(row.Attempts), Revision: row.Revision}
	copy(p.Digest[:], row.Digest)
	copy(p.RPBinding[:], row.RpBinding)

	if row.LeaseUntil.Valid {
		p.LeaseUntil = row.LeaseUntil.Time
	}

	return p
}

func lockEnrollment(ctx context.Context, queries *dal.Queries, digest core.EnrollmentDigest) (core.EnrollmentPending, error) {
	userID, err := queries.EnrollmentSubject(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return core.EnrollmentPending{}, core.ErrEnrollment
	}

	if err != nil {
		return core.EnrollmentPending{}, err
	}

	user, err := queries.GetUserForAuth(ctx, userID)
	if err != nil {
		return core.EnrollmentPending{}, err
	}

	row, err := queries.LockEnrollment(ctx, digest[:])
	if errors.Is(err, sql.ErrNoRows) {
		return core.EnrollmentPending{}, core.ErrEnrollment
	}

	if err != nil {
		return core.EnrollmentPending{}, err
	}

	if !user.Active || user.AuthVersion != row.AuthVersion {
		return core.EnrollmentPending{}, core.ErrCredentialChanged
	}

	return enrollmentSnapshot(row), nil
}

func chargeFactorAttempt(ctx context.Context, queries *dal.Queries, p core.EnrollmentPending, settings config.EnrollmentSettings, now time.Time) (time.Time, error) {
	err := queries.EnsureFactorBudget(ctx, dal.EnsureFactorBudgetParams{UserID: p.State.UserID, WindowStart: now})
	if err != nil {
		return time.Time{}, err
	}

	budget, err := queries.LockFactorBudget(ctx, p.State.UserID)
	if err != nil {
		return time.Time{}, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return time.Time{}, err
	}

	err = p.Check(now, p.Requirement, settings)
	if err != nil {
		return time.Time{}, err
	}

	if budget.CooldownUntil.Valid && now.Before(budget.CooldownUntil.Time) {
		return time.Time{}, core.ErrEnrollmentAttempts
	}

	if now.Sub(budget.WindowStart) >= settings.BudgetWindow {
		budget.Attempts = 0
		budget.WindowStart = now
		budget.CooldownUntil = sql.NullTime{}
	}

	if budget.Attempts >= int32(settings.SubjectAttempts) {
		return time.Time{}, core.ErrEnrollmentAttempts
	}

	budget.Attempts++
	if budget.Attempts >= int32(settings.SubjectAttempts) {
		budget.CooldownUntil = sql.NullTime{Time: now.Add(settings.Cooldown), Valid: true}
	}

	err = queries.ChargeFactorBudget(ctx, dal.ChargeFactorBudgetParams{UserID: budget.UserID, WindowStart: budget.WindowStart, Attempts: budget.Attempts, CooldownUntil: budget.CooldownUntil})

	return now, err
}

// ReserveEnrollment commits durable attempts before protocol parsing/verification.
// Denied or busy admissions spend nothing; admitted failures never refund.
func (q *Queries) ReserveEnrollment(ctx context.Context, digest core.EnrollmentDigest, required core.AccessRequirement, settings config.EnrollmentSettings) (*core.EnrollmentPending, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	p, err := lockEnrollment(ctx, queries, digest)
	if err != nil {
		return nil, err
	}

	err = initialEnrollment(ctx, queries, p.State.UserID, settings)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return nil, err
	}

	if !p.LeaseUntil.IsZero() && now.Before(p.LeaseUntil) {
		return nil, core.ErrEnrollmentBusy
	}

	if p.Attempts >= settings.PendingAttempts {
		return nil, core.ErrEnrollmentAttempts
	}

	now, err = chargeFactorAttempt(ctx, queries, p, settings, now)
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

	return &p, nil
}

// ReleaseEnrollment releases only the named reservation revision, without refund.
func (q *Queries) ReleaseEnrollment(ctx context.Context, digest core.EnrollmentDigest, revision int64) error {
	if q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return errors.New("database connection not available")
	}

	return dal.New(q.dbProvider.GetDB()).ReleaseEnrollment(ctx, dal.ReleaseEnrollmentParams{Digest: digest[:], Revision: revision})
}

func sameEnrollment(a, b core.EnrollmentPending) bool {
	return a.RPBinding == b.RPBinding && a.Digest == b.Digest && a.State == b.State && a.Requirement == b.Requirement && a.RPID == b.RPID && a.Revision == b.Revision && a.Attempts == b.Attempts && a.PasswordAt.Equal(b.PasswordAt) && a.CreatedAt.Equal(b.CreatedAt) && a.ExpiresAt.Equal(b.ExpiresAt) && a.LeaseUntil.Equal(b.LeaseUntil) && bytes.Equal(a.Handle, b.Handle) && bytes.Equal(a.Ceremony, b.Ceremony)
}

// ConfirmEnrollment atomically activates actual registration and invalidates all
// previous sessions and pending ceremonies. A failed write returns no success.
func (q *Queries) ConfirmEnrollment(ctx context.Context, reserved core.EnrollmentPending, record core.RegistrationRecord, required core.AccessRequirement, settings config.EnrollmentSettings) (*core.Authenticator, error) {
	tx, err := q.beginCredentialTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queries := dal.New(tx)

	p, err := lockEnrollment(ctx, queries, reserved.Digest)
	if err != nil {
		return nil, err
	}

	err = initialEnrollment(ctx, queries, p.State.UserID, settings)
	if err != nil {
		return nil, err
	}

	now, err := queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, required, settings)
	if err != nil {
		return nil, err
	}

	if !sameEnrollment(p, reserved) || p.Revision < 1 || p.Attempts < 1 || p.LeaseUntil.IsZero() || !now.Before(p.LeaseUntil) || record.VerifiedAt.Before(p.LeaseUntil.Add(-settings.Lease)) || record.VerifiedAt.After(now) || record.CreatedAt.Before(p.CreatedAt) || record.CreatedAt.After(now) || record.ID == "" || len(record.CredentialID) == 0 || len(record.CredentialID) > 1024 || len(record.PublicKey) == 0 || len(record.PublicKey) > 4096 || len(record.Data) == 0 || len(record.Data) > 65536 || (record.BackupState && !record.BackupEligible) {
		return nil, core.ErrEnrollment
	}

	record.CreatedAt = now

	err = queries.ConfirmAuthenticator(ctx, dal.ConfirmAuthenticatorParams{ID: record.ID, UserID: p.State.UserID, RpID: p.RPID, CredentialID: record.CredentialID, PublicKey: record.PublicKey, CredentialData: record.Data, SignCount: int64(record.Counter), BackupEligible: record.BackupEligible, BackupState: record.BackupState, CreatedAt: now})
	if err != nil {
		return nil, err
	}

	now, err = queries.SessionClock(ctx)
	if err != nil {
		return nil, err
	}

	err = p.Check(now, required, settings)
	if err != nil || !now.Before(p.LeaseUntil) {
		return nil, core.ErrEnrollment
	}

	err = queries.AdvanceAuthenticatorVersion(ctx, dal.AdvanceAuthenticatorVersionParams{ID: p.State.UserID, UpdatedAt: now})
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

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &record.Authenticator, nil
}

// DeleteExpiredEnrollments uses a finite expired-only batch and takes no later
// subject lock, preserving the mutation lock order under cleanup races.
func (q *Queries) DeleteExpiredEnrollments(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 || q.dbProvider == nil || q.dbProvider.GetDB() == nil {
		return 0, core.ErrEnrollment
	}

	return dal.New(q.dbProvider.GetDB()).DeleteExpiredEnrollments(ctx, int32(limit))
}
