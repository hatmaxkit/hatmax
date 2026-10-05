// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/model"
)

// SecurityOperation is a closed core classification, never request metadata.
type SecurityOperation string

const (
	SecurityRegistration         SecurityOperation = "registration"
	SecuritySignin               SecurityOperation = "signin"
	SecurityReauthentication     SecurityOperation = "reauthentication"
	SecuritySignout              SecurityOperation = "signout"
	SecuritySessionRevocation    SecurityOperation = "session_revocation"
	SecurityEnrollmentBegin      SecurityOperation = "enrollment_begin"
	SecurityEnrollmentFinish     SecurityOperation = "enrollment_finish"
	SecurityWebAuthnBegin        SecurityOperation = "webauthn_begin"
	SecurityWebAuthnStepUp       SecurityOperation = "webauthn_step_up"
	SecurityWebAuthnFinish       SecurityOperation = "webauthn_finish"
	SecurityTOTPSetupBegin       SecurityOperation = "totp_setup_begin"
	SecurityTOTPSetupFinish      SecurityOperation = "totp_setup_finish"
	SecurityFallbackBegin        SecurityOperation = "fallback_begin"
	SecurityFallbackStepUp       SecurityOperation = "fallback_step_up"
	SecurityFallbackFinish       SecurityOperation = "fallback_finish"
	SecurityBackupIssue          SecurityOperation = "backup_issue"
	SecurityWebAuthnChangeBegin  SecurityOperation = "webauthn_change_begin"
	SecurityTOTPChangeBegin      SecurityOperation = "totp_change_begin"
	SecurityWebAuthnChangeFinish SecurityOperation = "webauthn_change_finish"
	SecurityTOTPChangeFinish     SecurityOperation = "totp_change_finish"
	SecurityFactorRemoval        SecurityOperation = "factor_removal"
	SecurityMailboxIssue         SecurityOperation = "mailbox_issue"
	SecurityMailboxVerification  SecurityOperation = "mailbox_verification"
	SecurityPasswordResetIssue   SecurityOperation = "password_reset_issue"
	SecurityPasswordReset        SecurityOperation = "password_reset"
	SecurityPasswordChange       SecurityOperation = "password_change"
)

// SecurityOutcome is a closed core classification, never request metadata.
type SecurityOutcome string

const (
	SecurityCommitted         SecurityOutcome = "committed"
	SecurityAuthenticated     SecurityOutcome = "authenticated"
	SecurityPending           SecurityOutcome = "pending"
	SecurityPendingEnrollment SecurityOutcome = "pending_enrollment"
	SecurityPendingProof      SecurityOutcome = "pending_proof"
	SecurityDenied            SecurityOutcome = "denied"
	SecurityMissing           SecurityOutcome = "missing"
	SecurityInactive          SecurityOutcome = "inactive"
	SecurityInvalidProof      SecurityOutcome = "invalid_proof"
	SecurityUnavailableState  SecurityOutcome = "expired_or_replayed"
	SecurityPolicyRejected    SecurityOutcome = "policy_rejected"
	SecurityAttemptsExhausted SecurityOutcome = "attempts_exhausted"
	SecurityCapacityExhausted SecurityOutcome = "capacity_exhausted"
	SecurityStale             SecurityOutcome = "stale"
	SecurityOperatingUnknown  SecurityOutcome = "operating_unknown"
)

var (
	ErrSecurityEvent               = errors.New("invalid security event")
	ErrSecurityObservationRejected = errors.New("security observation rejected")
)

// SecurityEvent contains bounded trusted references and actual verified methods.
// It is an observation, never a proof receipt or an authorization input.
type SecurityEvent struct {
	ID         string            `json:"id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Operation  SecurityOperation `json:"operation"`
	Outcome    SecurityOutcome   `json:"outcome"`
	Subject    string            `json:"subject,omitempty"`
	Record     string            `json:"record,omitempty"`
	Proof      ProofMethod       `json:"proof,omitempty"`
}

func securityReference(id string) string {
	if len(id) == 0 || len(id) > 128 {
		return ""
	}

	for i := range len(id) {
		c := id[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return ""
		}
	}

	return id
}

// Check validates the closed shape and its 1024-byte serialized envelope.
func (e SecurityEvent) Check() error {
	switch e.Operation {
	case SecurityRegistration, SecuritySignin, SecurityReauthentication, SecuritySignout, SecuritySessionRevocation, SecurityEnrollmentBegin, SecurityEnrollmentFinish, SecurityWebAuthnBegin, SecurityWebAuthnStepUp, SecurityWebAuthnFinish, SecurityTOTPSetupBegin, SecurityTOTPSetupFinish, SecurityFallbackBegin, SecurityFallbackStepUp, SecurityFallbackFinish, SecurityBackupIssue, SecurityWebAuthnChangeBegin, SecurityTOTPChangeBegin, SecurityWebAuthnChangeFinish, SecurityTOTPChangeFinish, SecurityFactorRemoval, SecurityMailboxIssue, SecurityMailboxVerification, SecurityPasswordResetIssue, SecurityPasswordReset, SecurityPasswordChange:
	default:
		return ErrSecurityEvent
	}

	switch e.Outcome {
	case SecurityCommitted, SecurityAuthenticated, SecurityPending, SecurityPendingEnrollment, SecurityPendingProof, SecurityDenied, SecurityMissing, SecurityInactive, SecurityInvalidProof, SecurityUnavailableState, SecurityPolicyRejected, SecurityAttemptsExhausted, SecurityCapacityExhausted, SecurityStale, SecurityOperatingUnknown:
	default:
		return ErrSecurityEvent
	}

	if e.Outcome == SecurityAuthenticated && e.Proof == 0 {
		return ErrSecurityEvent
	}

	id, idErr := model.ParseID(e.ID)
	if idErr != nil || id.String() != e.ID || len(e.ID) != 36 || securityReference(e.ID) != e.ID || e.OccurredAt.IsZero() || e.OccurredAt.Location() != time.UTC || e.Proof > PasswordBackupProof || (e.Subject != "" && securityReference(e.Subject) != e.Subject) || (e.Record != "" && securityReference(e.Record) != e.Record) {
		return ErrSecurityEvent
	}

	data, err := json.Marshal(e)
	if err != nil || len(data) > 1024 {
		return ErrSecurityEvent
	}

	return nil
}

// SecurityObserver is caller-owned, concurrent-safe and cooperatively context-aware.
// Observe must return within ctx's deadline; core creates no worker or retry.
type SecurityObserver interface {
	Observe(context.Context, SecurityEvent) error
}

// DiscardSecurityObserver explicitly discards events, without audit assurance.
type DiscardSecurityObserver struct{}

func (DiscardSecurityObserver) Observe(context.Context, SecurityEvent) error { return nil }

// SecurityObservationDiagnostics has fixed, identity-free delivery counters.
// Delivered means a timely callback, not durable external persistence.
type SecurityObservationDiagnostics struct {
	Delivered, Saturated, Rejected, Canceled, Deadline, Operating uint64
}

// SecurityObservations shares finite non-waiting callback capacity across services.
// Authentication results remain independent of callback delivery.
type SecurityObservations struct {
	observer                                                      SecurityObserver
	settings                                                      config.SecurityObservationSettings
	slots                                                         chan struct{}
	delivered, saturated, rejected, canceled, deadline, operating atomic.Uint64
}

func NewSecurityObservations(observer SecurityObserver, cfg config.SecurityObservationConfig) (*SecurityObservations, error) {
	if observer == nil {
		return nil, errors.New("security observer is required")
	}

	settings, err := cfg.SecurityObservationSettings()
	if err != nil {
		return nil, err
	}

	return &SecurityObservations{observer: observer, settings: settings, slots: make(chan struct{}, settings.Concurrency)}, nil
}

// Diagnostics returns independently loaded race-safe counters.
func (s *SecurityObservations) Diagnostics() SecurityObservationDiagnostics {
	return SecurityObservationDiagnostics{Delivered: s.delivered.Load(), Saturated: s.saturated.Load(), Rejected: s.rejected.Load(), Canceled: s.canceled.Load(), Deadline: s.deadline.Load(), Operating: s.operating.Load()}
}

func (s *SecurityObservations) observe(ctx context.Context, event SecurityEvent) {
	if event.Check() != nil {
		s.operating.Add(1)

		return
	}

	if ctx.Err() != nil {
		s.deliveryError(ctx.Err())

		return
	}

	select {
	case s.slots <- struct{}{}:
	default:
		s.saturated.Add(1)

		return
	}

	defer func() { <-s.slots }()

	limit := time.Now().Add(s.settings.Timeout)
	if callerDeadline, ok := ctx.Deadline(); ok && !limit.Before(callerDeadline) {
		limit = callerDeadline.Add(-time.Nanosecond)
	}

	callback, cancel := context.WithDeadline(ctx, limit)
	defer cancel()

	if callback.Err() != nil {
		s.deliveryError(callback.Err())

		return
	}

	err := invokeSecurityObserver(s.observer, callback, event)
	// Cancellation wins even if a non-cooperative observer returns nil late.
	if callback.Err() != nil {
		s.deliveryError(callback.Err())

		return
	}

	if !time.Now().Before(limit) {
		s.deadline.Add(1)

		return
	}

	if err != nil {
		s.deliveryError(err)

		return
	}

	s.delivered.Add(1)
}

func invokeSecurityObserver(observer SecurityObserver, ctx context.Context, event SecurityEvent) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("security observer failed")
		}
	}()

	return observer.Observe(ctx, event)
}

func (s *SecurityObservations) deliveryError(err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		s.deadline.Add(1)
	case errors.Is(err, context.Canceled):
		s.canceled.Add(1)
	case errors.Is(err, ErrSecurityObservationRejected):
		s.rejected.Add(1)
	default:
		s.operating.Add(1)
	}
}

// securityFacts contains only package-owned trusted operation metadata.
type securityFacts struct {
	subject, record string
	proof           ProofMethod
	outcome         SecurityOutcome
}

type securityFactsKey struct{}

func securityAttempt(ctx context.Context) (context.Context, *securityFacts) {
	facts := &securityFacts{}

	return context.WithValue(ctx, securityFactsKey{}, facts), facts
}

func securitySubject(ctx context.Context, subject string) {
	if f, ok := ctx.Value(securityFactsKey{}).(*securityFacts); ok {
		f.subject = securityReference(subject)
	}
}
func securityProof(ctx context.Context, method ProofMethod) {
	if f, ok := ctx.Value(securityFactsKey{}).(*securityFacts); ok && method >= PasswordProof && method <= PasswordBackupProof {
		f.proof = method
	}
}
func securityClassification(ctx context.Context, outcome SecurityOutcome) {
	if f, ok := ctx.Value(securityFactsKey{}).(*securityFacts); ok {
		f.outcome = outcome
	}
}
func securityRecord(ctx context.Context, record string) {
	if f, ok := ctx.Value(securityFactsKey{}).(*securityFacts); ok {
		f.record = securityReference(record)
	}
}
func (f *securityFacts) session(s *IssuedSession) {
	if s == nil || s.Session.Proof.Check(s.Session) != nil {
		f.outcome = SecurityOperatingUnknown

		return
	}

	f.subject = securityReference(s.Session.UserID)
	f.record = securityReference(s.Session.ID)
	f.proof = s.Session.Proof.Method
}
func securityActor(ctx context.Context, actor Session) {
	securitySubject(ctx, actor.UserID)
	securityRecord(ctx, actor.ID)

	if actor.Proof.Check(actor) == nil {
		securityProof(ctx, actor.Proof.Method)
	}
}
func (s *SecurityObservations) finish(ctx context.Context, operation SecurityOperation, success SecurityOutcome, facts *securityFacts, err error) {
	outcome := success
	if err != nil {
		outcome = securityErrorOutcome(err)
	}

	if facts.outcome != "" {
		outcome = facts.outcome
	}

	s.observe(ctx, SecurityEvent{ID: model.NewID(), OccurredAt: time.Now().UTC(), Operation: operation, Outcome: outcome, Subject: facts.subject, Record: facts.record, Proof: facts.proof})
}
func securityErrorOutcome(err error) SecurityOutcome {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.Is(err, ErrPasswordCheckFailed), errors.Is(err, model.ErrPasswordRecord), errors.Is(err, ErrCredentialAdmissionState):
		return SecurityOperatingUnknown
	case errors.Is(err, ErrCredentialAdmissionAttempts), errors.Is(err, ErrEnrollmentAttempts), errors.Is(err, ErrRecoveryAttempts):
		return SecurityAttemptsExhausted
	case errors.Is(err, ErrCredentialAdmissionCapacity), errors.Is(err, ErrSessionCapacity), errors.Is(err, ErrEnrollmentCapacity), errors.Is(err, ErrEnrollmentBusy), errors.Is(err, ErrRecoveryCapacity), errors.Is(err, ErrRecoveryBusy), errors.Is(err, model.ErrPasswordVerifierBusy):
		return SecurityCapacityExhausted
	case errors.Is(err, ErrUserNotFound):
		return SecurityMissing
	case errors.Is(err, ErrUserInactive):
		return SecurityInactive
	case errors.Is(err, ErrCredentialChanged), errors.Is(err, ErrSessionGeneration), errors.Is(err, ErrSessionPolicy):
		return SecurityStale
	case errors.Is(err, crypto.ErrTOTPCode), errors.Is(err, crypto.ErrBackupCode), errors.Is(err, ErrInvalidPassword), errors.Is(err, model.ErrPasswordMismatch), errors.Is(err, model.ErrPasswordInput), errors.Is(err, ErrSessionToken):
		return SecurityInvalidProof
	case errors.Is(err, ErrSessionExpired), errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrSessionProofExpired), errors.Is(err, ErrRecoveryUnavailable), errors.Is(err, ErrEnrollment), errors.Is(err, ErrWebAuthn), errors.Is(err, ErrFallback), errors.Is(err, ErrFactorChange):
		return SecurityUnavailableState
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrPasswordTooShort), errors.Is(err, ErrPasswordTooLong), errors.Is(err, ErrPasswordEncoding), errors.Is(err, ErrPasswordDisallowed), errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrCredentialIdentity), errors.Is(err, ErrAccessRequirement), errors.Is(err, ErrSessionSelection), errors.Is(err, ErrSessionProof):
		return SecurityPolicyRejected
	default:
		return SecurityOperatingUnknown
	}
}
