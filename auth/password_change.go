// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"fmt"
	"math"
	"time"
)

// PasswordChangePolicy is trusted server policy. Factor proof defaults to
// phishing-resistant MFA; RequireMFA is an explicit lower-assurance choice.
// AllowPassword applies only when no primary factor is established.
type PasswordChangePolicy struct {
	Requirement   AccessRequirement
	AllowPassword bool
}

func (p PasswordChangePolicy) Check() error {
	if p.Requirement.Proof != RequireMFA && p.Requirement.Proof != RequirePhishingResistantMFA || p.Requirement.MaxAge <= 0 {
		return ErrAccessRequirement
	}

	return p.Requirement.Check(30 * 24 * time.Hour)
}

// PasswordChangeAuthorization is an owned adapter snapshot, never caller proof.
// Factor bindings are current primary-factor identities/revisions, bounded to 20.
// Storage revalidates the bearer, complete snapshot and policy at commit.
type PasswordChangeAuthorization struct {
	Actor       Session
	ActorDigest SessionDigest `json:"-"`
	Factors     []FactorSelection
	Policy      PasswordChangePolicy
}

func (p PasswordChangeAuthorization) String() string   { return "PasswordChangeAuthorization{redacted}" }
func (p PasswordChangeAuthorization) GoString() string { return p.String() }

// Check enforces the factor-dependent minimum and current recent proof.
func (p PasswordChangeAuthorization) Check(now time.Time, policy PasswordChangePolicy) error {
	if p.Policy != policy || policy.Check() != nil || p.ActorDigest == (SessionDigest{}) || p.Actor.AuthVersion == math.MaxInt64 || p.Actor.Generation == math.MaxInt64 || len(p.Factors) > 20 {
		return ErrRecoveryUnavailable
	}

	previous := ""
	for _, f := range p.Factors {
		if f.Check() != nil || f.ID <= previous {
			return ErrRecoveryUnavailable
		}

		previous = f.ID
	}

	requirement := policy.Requirement
	if len(p.Factors) == 0 && policy.AllowPassword {
		requirement.Proof = RequirePassword
	}

	return requirement.Evaluate(p.Actor, now)
}

// PasswordChanged contains no bearer or credential record. Every old session,
// including the actor, has been revoked; the user must authenticate again.
type PasswordChanged struct {
	Subject   string    `json:"subject"`
	ChangedAt time.Time `json:"changedAt"`
}

// ChangePassword derives ownership exclusively from the real session bearer.
// Admission commits before the shared checker/KDF, which run outside DB locks.
// Failure never refunds admission; ambiguous commits are not retried.
func (s *RecoveryService) ChangePassword(ctx context.Context, bearer, password string, policy PasswordChangePolicy) (*PasswordChanged, error) {
	if policy.Requirement.Proof == 0 {
		policy.Requirement.Proof = RequirePhishingResistantMFA
	}

	requirement, err := s.base.managementRequirement(policy.Requirement)
	if err != nil {
		return nil, err
	}

	policy.Requirement = requirement
	if policy.Check() != nil {
		return nil, ErrAccessRequirement
	}
	// Reject oversized input before storage, preserving the existing Unicode policy.
	if len(password) > s.base.policy.config.MaxBytes {
		return nil, ErrPasswordTooLong
	}

	digest, err := ParseSessionToken(bearer)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	pending, err := s.queries.AuthorizePasswordChange(work, digest, policy, s.settings)
	if err != nil {
		return nil, err
	}

	if pending == nil || pending.ActorDigest != digest {
		return nil, ErrRecoveryUnavailable
	}

	err = pending.Check(time.Now().UTC(), policy)
	if err != nil {
		return nil, err
	}

	passwordWork, passwordCancel := context.WithTimeout(work, s.base.passwordTimeout)
	defer passwordCancel()

	candidate, err := s.base.policy.Prepare(passwordWork, password)
	if err != nil {
		return nil, err
	}

	encoded, err := s.base.verifier.Hash(passwordWork, candidate)
	if err != nil {
		return nil, fmt.Errorf("cannot hash changed password: %w", err)
	}

	result, err := s.queries.CommitPasswordChange(passwordWork, *pending, encoded, policy, s.settings)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Subject != pending.Actor.UserID || result.ChangedAt.IsZero() {
		return nil, ErrRecoveryUnavailable
	}

	return result, nil
}
