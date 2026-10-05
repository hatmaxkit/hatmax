// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import "context"

// Signup applies candidate policy and creates a salted, encoded credential.
// The storage write enforces uniqueness even for simultaneous signup requests.
// Observations run after operation resources are released.
func (s *Service) Signup(ctx context.Context, email, password string) (*User, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.signup(work, email, password)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else {
			facts.subject = securityReference(result.ID)
			facts.record = facts.subject
		}
	}

	s.observations.finish(ctx, SecurityRegistration, SecurityCommitted, facts, err)

	return result, err
}

// Signin verifies a password and applies trusted policy before issuance.
// Unmet requirements return non-authorizing outcomes with no secret or row.
// Observations run after operation resources are released.
func (s *Service) Signin(ctx context.Context, email, password string, requirement AccessRequirement) (*AuthenticationResult, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.signin(work, email, password, requirement)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else {
			switch result.Outcome {
			case AuthenticationCompleted:
				facts.session(result.Issued)
			case AuthenticationPendingEnrollment:
				facts.outcome = SecurityPendingEnrollment
			case AuthenticationPendingProof:
				facts.outcome = SecurityPendingProof
			case AuthenticationDenied:
				facts.outcome = SecurityDenied
			default:
				facts.outcome = SecurityOperatingUnknown
			}
		}
	}

	s.observations.finish(ctx, SecuritySignin, SecurityAuthenticated, facts, err)

	return result, err
}

// Signout destroys a session.
// Observations run after operation resources are released.
func (s *Service) Signout(ctx context.Context, token string) error {
	work, facts := securityAttempt(ctx)
	err := s.signout(work, token)
	s.observations.finish(ctx, SecuritySignout, SecurityCommitted, facts, err)

	return err
}

// Reauthenticate repeats actual password verification and atomically rotates a
// live session. Current proof may be old, but current policy and expiry still hold.
// Observations run after operation resources are released.
func (s *Service) Reauthenticate(ctx context.Context, token, password string, requirement AccessRequirement) (*AuthenticationResult, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.reauthenticate(work, token, password, requirement)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else {
			switch result.Outcome {
			case AuthenticationCompleted:
				facts.session(result.Issued)
			case AuthenticationPendingEnrollment:
				facts.outcome = SecurityPendingEnrollment
			case AuthenticationPendingProof:
				facts.outcome = SecurityPendingProof
			case AuthenticationDenied:
				facts.outcome = SecurityDenied
			default:
				facts.outcome = SecurityOperatingUnknown
			}
		}
	}

	s.observations.finish(ctx, SecurityReauthentication, SecurityAuthenticated, facts, err)

	return result, err
}

// RevokeSessions atomically revalidates the actor and deletes only its selected sessions.
// Observations run after operation resources are released.
func (s *Service) RevokeSessions(ctx context.Context, token string, requirement AccessRequirement, selection SessionSelection) (int64, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.revokeSessions(work, token, requirement, selection)
	s.observations.finish(ctx, SecuritySessionRevocation, SecurityCommitted, facts, err)

	return result, err
}

// BeginWebAuthnEnrollment verifies an actual fresh password for initial setup.
// Established factors cannot be replaced using this restricted operation.
// Observations run after operation resources are released.
func (s *AuthenticatorService) BeginWebAuthnEnrollment(ctx context.Context, email, password string, requirement AccessRequirement) (*EnrollmentChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginWebAuthnEnrollment(work, email, password, requirement)
	s.credentials.observations.finish(ctx, SecurityEnrollmentBegin, SecurityPending, facts, err)

	return result, err
}

// FinishWebAuthnEnrollment verifies bounded browser registration data after a
// durable reservation. Registration never issues a strong or ordinary session.
// Observations run after operation resources are released.
func (s *AuthenticatorService) FinishWebAuthnEnrollment(ctx context.Context, token string, body []byte, requirement AccessRequirement) (*Authenticator, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.finishWebAuthnEnrollment(work, token, body, requirement)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else {
			facts.record = securityReference(result.ID)
		}
	}

	s.credentials.observations.finish(ctx, SecurityEnrollmentFinish, SecurityCommitted, facts, err)

	return result, err
}

// BeginWebAuthnAuthentication is subject-first and does not grant access.
// Observations run after operation resources are released.
func (s *WebAuthnService) BeginWebAuthnAuthentication(ctx context.Context, email string, required AccessRequirement) (*AssertionChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginWebAuthnAuthentication(work, email, required)
	s.credentials.observations.finish(ctx, SecurityWebAuthnBegin, SecurityPending, facts, err)

	return result, err
}

// BeginWebAuthnStepUp binds the current live actor's digest and generation.
// Observations run after operation resources are released.
func (s *WebAuthnService) BeginWebAuthnStepUp(ctx context.Context, token string, required AccessRequirement) (*AssertionChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginWebAuthnStepUp(work, token, required)
	s.credentials.observations.finish(ctx, SecurityWebAuthnStepUp, SecurityPending, facts, err)

	return result, err
}

// FinishWebAuthn verifies an actual signed assertion after durable admission.
// No issued bearer leaves core before the final storage commit succeeds.
// Observations run after operation resources are released.
func (s *WebAuthnService) FinishWebAuthn(ctx context.Context, token string, body []byte, required AccessRequirement) (*IssuedSession, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.finishWebAuthn(work, token, body, required)
	if err == nil {
		facts.session(result)
	}

	s.credentials.observations.finish(ctx, SecurityWebAuthnFinish, SecurityAuthenticated, facts, err)

	return result, err
}

// BeginTOTPSetup supports only password-authorized initial setup. Confirmation
// verifies a real code, consumes that step and invalidates sessions; no access is issued.
// Observations run after operation resources are released.
func (s *FallbackService) BeginTOTPSetup(ctx context.Context, email, password string, required AccessRequirement) (*TOTPSetup, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginTOTPSetup(work, email, password, required)
	s.credentials.observations.finish(ctx, SecurityTOTPSetupBegin, SecurityPending, facts, err)

	return result, err
}

// BeginFallbackAuthentication verifies an actual password before capturing one
// owned current factor/set. Method and requirement must be selected by server code.
// Observations run after operation resources are released.
func (s *FallbackService) BeginFallbackAuthentication(ctx context.Context, email, password string, method FallbackMethod, required AccessRequirement) (*FallbackChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginFallbackAuthentication(work, email, password, method, required)
	s.credentials.observations.finish(ctx, SecurityFallbackBegin, SecurityPending, facts, err)

	return result, err
}

// BeginFallbackStepUp observes one terminal outcome after operation resources are released.
func (s *FallbackService) BeginFallbackStepUp(ctx context.Context, token, password string, method FallbackMethod, required AccessRequirement) (*FallbackChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginFallbackStepUp(work, token, password, method, required)
	s.credentials.observations.finish(ctx, SecurityFallbackStepUp, SecurityPending, facts, err)

	return result, err
}

// ConfirmTOTPSetup observes one terminal outcome after operation resources are released.
func (s *FallbackService) ConfirmTOTPSetup(ctx context.Context, token, code string, required AccessRequirement) error {
	work, facts := securityAttempt(ctx)
	err := s.confirmTOTPSetup(work, token, code, required)
	s.credentials.observations.finish(ctx, SecurityTOTPSetupFinish, SecurityCommitted, facts, err)

	return err
}

// FinishFallback observes one terminal outcome after operation resources are released.
func (s *FallbackService) FinishFallback(ctx context.Context, token, code string, required AccessRequirement) (*IssuedSession, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.finishFallback(work, token, code, required)
	if err == nil {
		facts.session(result)
	}

	s.credentials.observations.finish(ctx, SecurityFallbackFinish, SecurityAuthenticated, facts, err)

	return result, err
}

// IssueBackupCodes requires current recent management authority. Default policy
// is phishing-resistant MFA; an explicit server policy may allow actual MFA.
// The old set is replaced and the actor rotated atomically; plaintexts return once.
// Observations run after operation resources are released.
func (s *FallbackService) IssueBackupCodes(ctx context.Context, token string, required AccessRequirement) ([]string, *IssuedSession, error) {
	work, facts := securityAttempt(ctx)

	codes, session, err := s.issueBackupCodes(work, token, required)
	if err == nil {
		facts.session(session)
	}

	s.credentials.observations.finish(ctx, SecurityBackupIssue, SecurityCommitted, facts, err)

	return codes, session, err
}

// BeginWebAuthnChange adds or replaces a factor under actual recent management proof.
// Observations run after operation resources are released.
func (s *FactorService) BeginWebAuthnChange(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*EnrollmentChallenge, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginWebAuthnChange(work, token, target, policy)
	s.enrollment.credentials.observations.finish(ctx, SecurityWebAuthnChangeBegin, SecurityPending, facts, err)

	return result, err
}

// BeginTOTPChange observes one terminal outcome after operation resources are released.
func (s *FactorService) BeginTOTPChange(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*TOTPSetup, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.beginTOTPChange(work, token, target, policy)
	s.enrollment.credentials.observations.finish(ctx, SecurityTOTPChangeBegin, SecurityPending, facts, err)

	return result, err
}

// FinishWebAuthnChange observes one terminal outcome after operation resources are released.
func (s *FactorService) FinishWebAuthnChange(ctx context.Context, actorToken, token string, body []byte, policy FactorPolicy) (*FactorChangeResult, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.finishWebAuthnChange(work, actorToken, token, body, policy)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else if result.Factor != nil {
			facts.record = securityReference(result.Factor.ID)
		}
	}

	s.enrollment.credentials.observations.finish(ctx, SecurityWebAuthnChangeFinish, SecurityCommitted, facts, err)

	return result, err
}

// FinishTOTPChange observes one terminal outcome after operation resources are released.
func (s *FactorService) FinishTOTPChange(ctx context.Context, actorToken, token, code string, policy FactorPolicy) (*FactorChangeResult, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.finishTOTPChange(work, actorToken, token, code, policy)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else if result.Factor != nil {
			facts.record = securityReference(result.Factor.ID)
		}
	}

	s.enrollment.credentials.observations.finish(ctx, SecurityTOTPChangeFinish, SecurityCommitted, facts, err)

	return result, err
}

// Remove observes one terminal outcome after operation resources are released.
func (s *FactorService) Remove(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*FactorChangeResult, error) {
	work, facts := securityAttempt(ctx)

	result, err := s.remove(work, token, target, policy)
	if err == nil {
		if result == nil {
			facts.outcome = SecurityOperatingUnknown
		} else if result.Factor != nil {
			facts.record = securityReference(result.Factor.ID)
		}
	}

	s.enrollment.credentials.observations.finish(ctx, SecurityFactorRemoval, SecurityCommitted, facts, err)

	return result, err
}

// RequestMailboxVerification resolves only the caller's canonical current address.
// Applications map unavailable/budget outcomes to the same public acknowledgment.
// Observations run after operation resources are released.
func (s *RecoveryService) RequestMailboxVerification(ctx context.Context, mailbox string) (*MailboxIssue, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.requestMailboxVerification(work, mailbox)
	s.base.observations.finish(ctx, SecurityMailboxIssue, SecurityPending, facts, err)

	return result, err
}

// ConfirmMailboxVerification never creates a session or authorizes factor management.
// Observations run after operation resources are released.
func (s *RecoveryService) ConfirmMailboxVerification(ctx context.Context, bearer string) (*MailboxVerification, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.confirmMailboxVerification(work, bearer)
	s.base.observations.finish(ctx, SecurityMailboxVerification, SecurityCommitted, facts, err)

	return result, err
}

// RequestPasswordReset returns a transient token only to trusted application mail
// dispatch, after issuance commits against the previously verified current mailbox.
// Observations run after operation resources are released.
func (s *RecoveryService) RequestPasswordReset(ctx context.Context, mailbox string) (*MailboxIssue, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.requestPasswordReset(work, mailbox)
	s.base.observations.finish(ctx, SecurityPasswordResetIssue, SecurityPending, facts, err)

	return result, err
}

// ResetPassword proves the actual one-use mailbox secret before candidate work.
// It cannot activate an account, change MFA or issue an authentication session.
// Observations run after operation resources are released.
func (s *RecoveryService) ResetPassword(ctx context.Context, bearer, password string) (*PasswordReset, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.resetPassword(work, bearer, password)
	s.base.observations.finish(ctx, SecurityPasswordReset, SecurityCommitted, facts, err)

	return result, err
}

// ChangePassword derives ownership exclusively from the real session bearer.
// Admission commits before the shared checker/KDF, which run outside DB locks.
// Failure never refunds admission; ambiguous commits are not retried.
// Observations run after operation resources are released.
func (s *RecoveryService) ChangePassword(ctx context.Context, bearer, password string, policy PasswordChangePolicy) (*PasswordChanged, error) {
	work, facts := securityAttempt(ctx)
	result, err := s.changePassword(work, bearer, password, policy)
	s.base.observations.finish(ctx, SecurityPasswordChange, SecurityCommitted, facts, err)

	return result, err
}
