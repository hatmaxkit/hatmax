// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/examples/ticked/internal/dal"
)

// CredentialAdmissionStore owns private durable counters, not authentication.
// Replicas must supply the same immutable namespace and application-owned key.
type CredentialAdmissionStore struct {
	queries   *Queries
	namespace string
	key       []byte
	binding   []byte
}

// NewCredentialAdmissionStore copies a bounded key and starts no worker.
// Changing a key for an existing namespace is rejected by the durable binding.
func NewCredentialAdmissionStore(queries *Queries, namespace string, key []byte) (*CredentialAdmissionStore, error) {
	if queries == nil || queries.dbProvider == nil || len(namespace) < 1 || len(namespace) > 64 || len(key) < 32 || len(key) > 64 {
		return nil, core.ErrCredentialAdmissionState
	}

	for _, r := range namespace {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._-", r) {
			return nil, core.ErrCredentialAdmissionState
		}
	}

	s := &CredentialAdmissionStore{queries: queries, namespace: namespace, key: append([]byte(nil), key...)}
	s.binding = s.digest("key", "")

	return s, nil
}

func (s *CredentialAdmissionStore) digest(domain, identity string) []byte {
	mac := hmac.New(sha256.New, s.key)
	_, _ = mac.Write([]byte("hatmax/credential-admission/" + domain + "/v1\x00" + s.namespace + "\x00" + identity))

	return mac.Sum(nil)
}

func credentialRule(p core.CredentialAdmissionPurpose, settings config.CredentialAdmissionSettings) (int, time.Duration, time.Duration, error) {
	switch p {
	case core.CredentialRegistration:
		return settings.RegistrationAttempts, settings.RegistrationWindow, settings.RegistrationCooldown, nil
	case core.CredentialPasswordProof:
		return settings.PasswordAttempts, settings.PasswordWindow, settings.PasswordCooldown, nil
	default:
		return 0, 0, 0, core.ErrCredentialIdentity
	}
}

func (s *CredentialAdmissionStore) checkCapacity(row dal.CredentialAdmissionCapacity) error {
	if row.Namespace != s.namespace || !hmac.Equal(row.KeyBinding, s.binding) || row.Records < 0 || row.Records > 100000 || row.MaxRecords < 100 || row.MaxRecords > 100000 {
		return core.ErrCredentialAdmissionState
	}

	return nil
}

func credentialRetirement(row dal.CredentialAdmission) time.Time {
	end := row.WindowEnd
	if row.CooldownUntil.Valid && row.CooldownUntil.Time.After(end) {
		end = row.CooldownUntil.Time
	}

	return end
}

func checkCredentialRecord(row dal.CredentialAdmission, now time.Time) error {
	span := row.WindowEnd.Sub(row.WindowStart)
	if row.WindowStart.IsZero() || row.WindowStart.After(now) || span < time.Minute || span > time.Hour || row.CooldownUs < time.Minute.Microseconds() || row.CooldownUs > time.Hour.Microseconds() || row.AttemptLimit < 1 || row.AttemptLimit > 20 || row.Purpose == int16(core.CredentialRegistration) && row.AttemptLimit > 10 || row.Attempts < 1 || row.Attempts > row.AttemptLimit {
		return core.ErrCredentialAdmissionState
	}

	if row.CooldownUntil.Valid && (!row.CooldownUntil.Time.After(row.WindowStart) || row.CooldownUntil.Time.After(row.WindowEnd.Add(time.Duration(row.CooldownUs)*time.Microsecond))) {
		return core.ErrCredentialAdmissionState
	}

	return nil
}

func advanceCredentialRecord(row dal.CredentialAdmission, now time.Time, limit int, window, cooldown time.Duration) (dal.CredentialAdmission, error) {
	if !row.WindowStart.IsZero() {
		err := checkCredentialRecord(row, now)
		if err != nil {
			return row, err
		}
	}

	if row.WindowStart.IsZero() || !now.Before(credentialRetirement(row)) {
		row.WindowStart = now
		row.WindowEnd = now.Add(window)
		row.CooldownUs = cooldown.Microseconds()
		row.AttemptLimit = int32(limit)
		row.Attempts = 0
		row.CooldownUntil = sql.NullTime{}
	}

	effective := min(row.AttemptLimit, int32(limit))
	if now.Before(row.WindowStart) || row.CooldownUntil.Valid && now.Before(row.CooldownUntil.Time) || row.Attempts >= effective || !now.Before(row.WindowEnd) {
		return row, &core.CredentialAdmissionDenial{RetryAt: credentialRetirement(row)}
	}

	row.Attempts++
	if row.Attempts == effective {
		row.CooldownUntil = sql.NullTime{Time: now.Add(time.Duration(row.CooldownUs) * time.Microsecond), Valid: true}
	}

	return row, nil
}

func updateCredentialCapacity(ctx context.Context, q *dal.Queries, namespace string, records int32, maximum int) error {
	changed, err := q.UpdateCredentialAdmissionCapacity(ctx, dal.UpdateCredentialAdmissionCapacityParams{Namespace: namespace, Records: records, MaxRecords: int32(maximum)})
	if err != nil {
		return err
	}

	if changed != 1 {
		return core.ErrCredentialAdmissionState
	}

	return nil
}

// ChargeCredentialAdmission serializes capacity before identity, commits once,
// and never acquires a subject lock or performs credential work.
func (s *CredentialAdmissionStore) ChargeCredentialAdmission(ctx context.Context, identity string, purpose core.CredentialAdmissionPurpose, settings config.CredentialAdmissionSettings) error {
	err := core.CheckCredentialIdentity(identity)
	if err != nil {
		return err
	}

	err = settings.Check()
	if err != nil {
		return err
	}

	limit, window, cooldown, err := credentialRule(purpose, settings)
	if err != nil {
		return err
	}

	work, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	tx, err := s.queries.beginCredentialTx(work)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := dal.New(tx)

	err = q.EnsureCredentialAdmissionCapacity(work, dal.EnsureCredentialAdmissionCapacityParams{Namespace: s.namespace, KeyBinding: s.binding, MaxRecords: int32(settings.MaxIdentities)})
	if err != nil {
		return err
	}

	capacity, err := q.LockCredentialAdmissionCapacity(work, s.namespace)
	if err != nil {
		return err
	}

	err = s.checkCapacity(capacity)
	if err != nil {
		return err
	}

	key := s.digest("identity", identity)
	row, err := q.LockCredentialAdmission(work, dal.LockCredentialAdmissionParams{Namespace: s.namespace, Purpose: int16(purpose), IdentityKey: key})

	fresh := errors.Is(err, sql.ErrNoRows)
	if err != nil && !fresh {
		return err
	}

	if fresh {
		if capacity.Records >= int32(settings.MaxIdentities) {
			return core.ErrCredentialAdmissionCapacity
		}

		row = dal.CredentialAdmission{Namespace: s.namespace, Purpose: int16(purpose), IdentityKey: key}
	} else if capacity.Records < 1 {
		return core.ErrCredentialAdmissionState
	}

	now, err := q.SessionClock(work)
	if err != nil {
		return err
	}

	row, err = advanceCredentialRecord(row, now, limit, window, cooldown)
	if err != nil {
		return err
	}

	if fresh {
		err = q.InsertCredentialAdmission(work, dal.InsertCredentialAdmissionParams(row))
		if err != nil {
			return err
		}

		capacity.Records++
	} else {
		changed, updateErr := q.UpdateCredentialAdmission(work, dal.UpdateCredentialAdmissionParams(row))
		if updateErr != nil {
			return updateErr
		}

		if changed != 1 {
			return core.ErrCredentialAdmissionState
		}
	}

	err = updateCredentialCapacity(work, q, s.namespace, capacity.Records, settings.MaxIdentities)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// CleanupCredentialAdmission rechecks complete retirement and atomically releases
// at most one configured batch. Live windows and security authority are untouched.
func (s *CredentialAdmissionStore) CleanupCredentialAdmission(ctx context.Context, settings config.CredentialAdmissionSettings) (int64, error) {
	err := settings.Check()
	if err != nil {
		return 0, err
	}

	work, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	tx, err := s.queries.beginCredentialTx(work)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := dal.New(tx)

	capacity, err := q.LockCredentialAdmissionCapacity(work, s.namespace)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	err = s.checkCapacity(capacity)
	if err != nil {
		return 0, err
	}

	now, err := q.SessionClock(work)
	if err != nil {
		return 0, err
	}

	count, err := q.DeleteRetiredCredentialAdmissions(work, dal.DeleteRetiredCredentialAdmissionsParams{Namespace: s.namespace, Now: now, CleanupBatch: int32(settings.CleanupBatch)})
	if err != nil {
		return 0, err
	}

	if count < 0 || count > int64(settings.CleanupBatch) || count > int64(capacity.Records) {
		return 0, core.ErrCredentialAdmissionState
	}

	err = updateCredentialCapacity(work, q, s.namespace, capacity.Records-int32(count), settings.MaxIdentities)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return count, nil
}
