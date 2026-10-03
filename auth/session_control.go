// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"hatmax.adrianpk.com/model"
	"math"
	"time"
)

var (
	ErrSessionCapacity   = errors.New("session capacity exhausted")
	ErrSessionSelection  = errors.New("invalid session selection")
	ErrSessionCursor     = errors.New("invalid session cursor")
	ErrSessionGeneration = errors.New("session generation changed or exhausted")
)

// SessionScope selects only sessions owned by the revalidated actor.
type SessionScope uint8

const (
	SessionCurrent SessionScope = iota + 1
	SessionSelected
	SessionOthers
	SessionAll
)

// SessionSelection carries no subject or administrative authority.
type SessionSelection struct {
	Scope SessionScope
	ID    string
}

func (s SessionSelection) Check() error {
	if s.Scope < SessionCurrent || s.Scope > SessionAll {
		return ErrSessionSelection
	}

	if s.Scope == SessionSelected {
		if !boundedID(s.ID, 128) {
			return ErrSessionSelection
		}
	} else if s.ID != "" {
		return ErrSessionSelection
	}

	return nil
}

// SessionPage contains safe metadata only; the cursor is not an authentication secret.
type SessionPage struct {
	Sessions   []Session
	CurrentID  string
	NextCursor string
}

func boundedID(id string, max int) bool {
	if len(id) == 0 || len(id) > max {
		return false
	}

	for i := range len(id) {
		if id[i] < 32 || id[i] > 126 {
			return false
		}
	}

	return true
}

// ParseSessionCursor validates a bounded canonical keyset cursor before storage.
func ParseSessionCursor(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}

	if len(cursor) > 128 {
		return "", ErrSessionCursor
	}

	id, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
	if err != nil || !boundedID(string(id), 96) || base64.RawURLEncoding.EncodeToString(id) != cursor {
		return "", ErrSessionCursor
	}

	return string(id), nil
}

// SessionCursor encodes a bounded record ID for subject-scoped keyset pagination.
func SessionCursor(id string) (string, error) {
	if !boundedID(id, 96) {
		return "", ErrSessionCursor
	}

	return base64.RawURLEncoding.EncodeToString([]byte(id)), nil
}

// Reauthenticate repeats actual password verification and atomically rotates a
// live session. Current proof may be old, but current policy and expiry still hold.
func (s *Service) Reauthenticate(ctx context.Context, token, password string, requirement AccessRequirement) (*AuthenticationResult, error) {
	err := requirement.Check(s.sessions.TTL)
	if err != nil {
		return nil, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.passwordTimeout)
	defer cancel()

	currentRequirement := requirement
	currentRequirement.MaxAge = 0

	current, err := s.ValidateSession(workCtx, token, currentRequirement, NoActivity)
	if err != nil {
		return nil, err
	}

	if requirement.Proof != RequirePassword {
		return unmetRequirement(current.User, requirement.Proof), nil
	}

	if current.Session.Generation == math.MaxInt64 {
		return nil, ErrSessionGeneration
	}

	err = s.verifier.Verify(workCtx, current.User.PasswordHash, password)
	if errors.Is(err, model.ErrPasswordMismatch) {
		return nil, ErrInvalidPassword
	}

	if err != nil {
		return nil, fmt.Errorf("cannot verify password: %w", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	replacementToken, replacementDigest, err := s.sessionToken()
	if err != nil {
		return nil, fmt.Errorf("cannot generate session secret: %w", err)
	}

	oldDigest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	record := SessionRecord{Session: current.Session, Digest: replacementDigest}
	record.Generation++
	record.PolicyRevision = requirement.Revision
	record.Proof = VerifiedProof{Method: PasswordProof, VerifiedAt: now}
	record.AuthenticatedAt = now
	record.LastActivityAt = now
	record.ExpiresAt = now.Add(s.sessions.TTL)
	record.InactivityTTL = s.sessions.InactivityTTL

	sessionCtx, sessionCancel := context.WithTimeout(workCtx, s.sessions.Timeout)
	defer sessionCancel()

	rotated, err := s.queries.RotateSession(sessionCtx, CredentialState{UserID: current.User.ID, Version: current.User.AuthVersion}, oldDigest, current.Session.Generation, record, requirement)
	if err != nil {
		return nil, fmt.Errorf("cannot rotate session: %w", err)
	}

	err = sessionCtx.Err()
	if err != nil {
		return nil, err
	}

	return &AuthenticationResult{Outcome: AuthenticationCompleted, Reason: AuthenticationSatisfied, Issued: &IssuedSession{Session: *rotated, Token: replacementToken}}, nil
}

func (s *Service) managementRequirement(requirement AccessRequirement) (AccessRequirement, error) {
	err := requirement.Check(s.sessions.TTL)
	if err != nil {
		return AccessRequirement{}, err
	}

	if requirement.MaxAge == 0 || requirement.MaxAge > s.sessions.RecentProofAge {
		requirement.MaxAge = s.sessions.RecentProofAge
	}

	return requirement, nil
}

// ListSessions derives ownership from the bearer and requires current recent proof.
func (s *Service) ListSessions(ctx context.Context, token string, requirement AccessRequirement, cursor string) (*SessionPage, error) {
	requirement, err := s.managementRequirement(requirement)
	if err != nil {
		return nil, err
	}

	_, err = ParseSessionCursor(cursor)
	if err != nil {
		return nil, err
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	page, err := s.queries.ListSessions(workCtx, digest, requirement, s.sessions.PageSize, cursor)
	if err != nil {
		return nil, fmt.Errorf("cannot list sessions: %w", err)
	}

	err = workCtx.Err()
	if err != nil {
		return nil, err
	}

	return page, nil
}

// RevokeSessions atomically revalidates the actor and deletes only its selected sessions.
func (s *Service) RevokeSessions(ctx context.Context, token string, requirement AccessRequirement, selection SessionSelection) (int64, error) {
	requirement, err := s.managementRequirement(requirement)
	if err != nil {
		return 0, err
	}

	err = selection.Check()
	if err != nil {
		return 0, err
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return 0, err
	}

	workCtx, cancel := context.WithTimeout(ctx, s.sessions.Timeout)
	defer cancel()

	count, err := s.queries.RevokeSessions(workCtx, digest, requirement, selection)
	if err != nil {
		return 0, fmt.Errorf("cannot revoke sessions: %w", err)
	}

	err = workCtx.Err()
	if err != nil {
		return 0, err
	}

	return count, nil
}
