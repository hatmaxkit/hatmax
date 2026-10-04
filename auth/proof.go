// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"errors"
	"time"
)

var (
	ErrAccessRequirement   = errors.New("invalid access requirement")
	ErrSessionPolicy       = errors.New("session policy changed")
	ErrSessionProof        = errors.New("insufficient session proof")
	ErrSessionProofExpired = errors.New("session proof is not recent")
)

// RequiredProof describes trusted server policy, never verified method facts.
type RequiredProof uint8

const (
	RequirePassword RequiredProof = iota + 1
	RequireMFA
	RequirePhishingResistantMFA
)

// AccessRequirement is an immutable operation policy supplied by server code.
// MaxAge zero disables freshness; a nonzero age expires at exact equality.
type AccessRequirement struct {
	Proof    RequiredProof
	Revision string
	MaxAge   time.Duration
}

func validPolicyRevision(revision string) bool {
	if len(revision) == 0 || len(revision) > 128 {
		return false
	}

	for i := range len(revision) {
		if revision[i] < 32 || revision[i] > 126 {
			return false
		}
	}

	return true
}

// Check rejects unknown profiles, revisions and invalid freshness bounds.
func (r AccessRequirement) Check(lifetime time.Duration) error {
	if lifetime < time.Minute || lifetime > 30*24*time.Hour || r.Proof < RequirePassword || r.Proof > RequirePhishingResistantMFA || !validPolicyRevision(r.Revision) || r.MaxAge < 0 || (r.MaxAge != 0 && (r.MaxAge < time.Second || r.MaxAge > lifetime || r.MaxAge%time.Microsecond != 0)) {
		return ErrAccessRequirement
	}

	return nil
}

// ProofMethod is a closed representation of supported verifier output.
// Enrollment and unimplemented methods cannot create a valid stored fact.
type ProofMethod uint8

const (
	PasswordProof ProofMethod = 1
	WebAuthnProof ProofMethod = 2
)

// VerifiedProof is read-only metadata about actual completed core verification.
// Public metadata does not supply a completion or proof-assertion API.
type VerifiedProof struct {
	Method         ProofMethod
	VerifiedAt     time.Time
	FactorID       string
	FactorRevision int64
}

// Check validates the supported proof and its relation to completed authentication.
func (p VerifiedProof) Check(session Session) error {
	if (p.Method != PasswordProof && p.Method != WebAuthnProof) || p.VerifiedAt.IsZero() || p.VerifiedAt.Before(session.CreatedAt) || p.VerifiedAt.After(session.AuthenticatedAt) {
		return ErrSessionRecord
	}

	if p.Method == PasswordProof && (p.FactorID != "" || p.FactorRevision != 0) {
		return ErrSessionRecord
	}

	if p.Method == WebAuthnProof && (!boundedID(p.FactorID, 128) || p.FactorRevision < 1) {
		return ErrSessionRecord
	}

	return nil
}

// Evaluate matches stored facts to current policy at the locked trusted time.
// It cannot manufacture stronger method properties or change proof time.
func (r AccessRequirement) Evaluate(session Session, now time.Time) error {
	err := session.Check(now)
	if err != nil {
		return err
	}

	err = r.Check(session.ExpiresAt.Sub(session.AuthenticatedAt))
	if err != nil {
		return err
	}

	if r.Revision != session.PolicyRevision {
		return ErrSessionPolicy
	}

	if r.Proof != RequirePassword && session.Proof.Method != WebAuthnProof {
		return ErrSessionProof
	}

	if r.MaxAge != 0 && now.Sub(session.Proof.VerifiedAt) >= r.MaxAge {
		return ErrSessionProofExpired
	}

	return nil
}

// AuthenticationOutcome distinguishes access from non-authorizing next steps.
type AuthenticationOutcome uint8

const (
	AuthenticationCompleted AuthenticationOutcome = iota + 1
	AuthenticationPendingEnrollment
	AuthenticationPendingProof
	AuthenticationDenied
)

type AuthenticationReason uint8

const (
	AuthenticationSatisfied AuthenticationReason = iota
	AuthenticationMethodUnavailable
)

// AuthenticationResult is a response, never a proof receipt accepted by core.
// Only a completed core result carries an issued session. Other outcomes have no
// bearer, continuation or persistent pending state.
type AuthenticationResult struct {
	Outcome AuthenticationOutcome
	Reason  AuthenticationReason
	Issued  *IssuedSession
}

// CompletedSession prevents transports from interpreting a pending result as access.
func (r *AuthenticationResult) CompletedSession() (*IssuedSession, bool) {
	if r == nil || r.Outcome != AuthenticationCompleted || r.Reason != AuthenticationSatisfied || r.Issued == nil {
		return nil, false
	}

	return r.Issued, true
}

func unmetRequirement(user *User, required RequiredProof) *AuthenticationResult {
	outcome := AuthenticationDenied
	if required == RequireMFA {
		outcome = AuthenticationPendingEnrollment
		if user.TOTPEnabled {
			outcome = AuthenticationPendingProof
		}
	}

	return &AuthenticationResult{Outcome: outcome, Reason: AuthenticationMethodUnavailable}
}
