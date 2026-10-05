// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/model"
)

var (
	ErrEnrollment         = errors.New("invalid or unavailable enrollment")
	ErrEnrollmentCapacity = errors.New("enrollment capacity reached")
	ErrEnrollmentBusy     = errors.New("enrollment verification busy")
	ErrEnrollmentAttempts = errors.New("enrollment attempt budget exhausted")
)

// EnrollmentDigest cannot be resolved by an ordinary session store.
type EnrollmentDigest [32]byte

// EnrollmentPending is a caller-owned storage snapshot, never a session.
type EnrollmentPending struct {
	RPBinding                        [32]byte
	Digest                           EnrollmentDigest
	State                            CredentialState
	Requirement                      AccessRequirement
	RPID                             string
	Handle, Ceremony                 []byte
	PasswordAt, CreatedAt, ExpiresAt time.Time
	Attempts                         int
	Revision                         int64
	LeaseUntil                       time.Time
}

// Check binds restricted password authority to current trusted policy and time.
func (p EnrollmentPending) Check(now time.Time, requirement AccessRequirement, settings config.EnrollmentSettings) error {
	if p.State.UserID == "" || p.State.Version < 1 || p.Digest == (EnrollmentDigest{}) || p.RPBinding != settings.RPBinding || p.RPID != settings.RPID || len(p.Handle) != 32 || len(p.Ceremony) == 0 || len(p.Ceremony) > 16384 || p.PasswordAt.IsZero() || p.PasswordAt.After(p.CreatedAt) || p.CreatedAt.After(now) || !now.Before(p.ExpiresAt) || p.ExpiresAt.Sub(p.CreatedAt) != settings.PendingTTL || p.Requirement != requirement || p.Attempts < 0 || p.Attempts > settings.PendingAttempts || p.Revision < 0 {
		return ErrEnrollment
	}

	age := settings.RecentProofAge
	if requirement.MaxAge != 0 && requirement.MaxAge < age {
		age = requirement.MaxAge
	}

	if now.Sub(p.PasswordAt) >= age {
		return ErrEnrollment
	}

	return requirement.Check(settings.PendingTTL)
}

// Authenticator is safe confirmed enrollment metadata. It confers no authority.
type Authenticator struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	BackupEligible bool      `json:"backupEligible"`
	BackupState    bool      `json:"backupState"`
}

// RegistrationRecord carries bounded actual library output to trusted storage.
// It is not accepted by any service method as an authentication proof.
type RegistrationRecord struct {
	Authenticator
	CredentialID, PublicKey, Data []byte
	Counter                       uint32
	VerifiedAt                    time.Time
}

// AuthenticatorQueries is mandatory storage for the executable enrollment flow.
// Every state mutation locks subject first and rechecks active version and time.
// Reserve commits both budgets and a lease before returning; Release never refunds.
// Confirm consumes pending, inserts the factor, increments version and revokes
// sessions/pending in one commit. It returns no result before that commit.
type AuthenticatorQueries interface {
	GetUserByEmail(context.Context, string) (*User, error)
	EnsureWebAuthnHandle(context.Context, CredentialState, string, []byte) ([]byte, error)
	CreateEnrollment(context.Context, EnrollmentPending, config.EnrollmentSettings) error
	ReserveEnrollment(context.Context, EnrollmentDigest, AccessRequirement, config.EnrollmentSettings) (*EnrollmentPending, error)
	ReleaseEnrollment(context.Context, EnrollmentDigest, int64) error
	ConfirmEnrollment(context.Context, EnrollmentPending, RegistrationRecord, AccessRequirement, config.EnrollmentSettings) (*Authenticator, error)
	DeleteExpiredEnrollments(context.Context, int) (int64, error)
}

// EnrollmentChallenge contains only a restricted transient bearer and browser options.
type EnrollmentChallenge struct {
	Token   string                       `json:"token"`
	Options *protocol.CredentialCreation `json:"options"`
}

// AuthenticatorService owns initial registration through the real protocol verifier.
type AuthenticatorService struct {
	credentials *Service
	queries     AuthenticatorQueries
	settings    config.EnrollmentSettings
	protocol    *webauthn.WebAuthn
	slots       chan struct{}
}

// NewAuthenticatorService validates and snapshots trusted configuration. Password
// verification shares the credential service's existing KDF admission budget.
func NewAuthenticatorService(credentials *Service, queries AuthenticatorQueries, cfg config.AuthenticatorConfig) (*AuthenticatorService, error) {
	if credentials == nil || credentials.verifier == nil || credentials.passwordTimeout <= 0 || queries == nil {
		return nil, errors.New("authenticator dependencies are required")
	}

	settings, err := cfg.EnrollmentSettings()
	if err != nil {
		return nil, err
	}

	rp, err := webauthn.New(&webauthn.Config{
		RPID: settings.RPID, RPDisplayName: settings.RPName, RPOrigins: settings.Origins,
		AttestationPreference:  protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired},
	})
	if err != nil {
		return nil, err
	}

	return &AuthenticatorService{credentials: credentials, queries: queries, settings: settings, protocol: rp, slots: make(chan struct{}, settings.MaxConcurrent)}, nil
}

func enrollmentToken() (string, EnrollmentDigest, error) {
	secret := make([]byte, 32)

	_, err := rand.Read(secret)
	if err != nil {
		return "", EnrollmentDigest{}, err
	}

	token := "enroll1." + base64.RawURLEncoding.EncodeToString(secret)
	digest, err := parseEnrollmentToken(token)

	return token, digest, err
}

func parseEnrollmentToken(token string) (EnrollmentDigest, error) {
	if len(token) != 51 || token[:8] != "enroll1." {
		return EnrollmentDigest{}, ErrEnrollment
	}

	secret, err := base64.RawURLEncoding.Strict().DecodeString(token[8:])
	if err != nil || len(secret) != 32 {
		return EnrollmentDigest{}, ErrEnrollment
	}

	return sha256.Sum256(append([]byte("hatmax/pending/enrollment/v1\x00"), secret...)), nil
}

type registrationUser struct {
	handle []byte
	name   string
}

func (u registrationUser) WebAuthnID() []byte                         { return u.handle }
func (u registrationUser) WebAuthnName() string                       { return u.name }
func (u registrationUser) WebAuthnDisplayName() string                { return u.name }
func (u registrationUser) WebAuthnCredentials() []webauthn.Credential { return nil }

func (s *AuthenticatorService) beginWebAuthnEnrollment(ctx context.Context, email, password string, requirement AccessRequirement) (*EnrollmentChallenge, error) {
	err := requirement.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, min(s.settings.Timeout, s.credentials.passwordTimeout))
	defer cancel()

	err = s.credentials.admitCredential(work, email, password, CredentialPasswordProof)
	if err != nil {
		return nil, err
	}

	user, err := s.queries.GetUserByEmail(work, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}

	if err != nil {
		return nil, err
	}

	securitySubject(work, user.ID)

	if !user.Active {
		return nil, errors.Join(ErrEnrollment, ErrUserInactive)
	}

	if user.TOTPEnabled {
		securityClassification(work, SecurityPolicyRejected)

		return nil, ErrEnrollment
	}

	state := CredentialState{UserID: user.ID, Version: user.AuthVersion}

	err = s.credentials.verifier.Verify(work, user.PasswordHash, password)
	if errors.Is(err, model.ErrPasswordMismatch) {
		return nil, ErrInvalidPassword
	}

	if err != nil {
		return nil, err
	}

	securityProof(work, PasswordProof)

	now := time.Now().UTC().Truncate(time.Microsecond)
	handle := make([]byte, 32)

	_, err = rand.Read(handle)
	if err != nil {
		return nil, err
	}

	handle, err = s.queries.EnsureWebAuthnHandle(work, state, s.settings.RPID, handle)
	if err != nil {
		return nil, err
	}

	options, ceremony, err := s.protocol.BeginRegistration(registrationUser{handle: handle, name: email}, webauthn.WithCredentialParameters([]protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}}))
	if err != nil {
		return nil, err
	}

	ceremony.Expires = now.Add(s.settings.PendingTTL)

	encoded, err := json.Marshal(ceremony)
	if err != nil {
		return nil, err
	}

	token, digest, err := enrollmentToken()
	if err != nil {
		return nil, err
	}

	pending := EnrollmentPending{RPBinding: s.settings.RPBinding, Digest: digest, State: state, Requirement: requirement, RPID: s.settings.RPID, Handle: handle, Ceremony: encoded, PasswordAt: now, CreatedAt: now, ExpiresAt: ceremony.Expires}

	err = s.queries.CreateEnrollment(work, pending, s.settings)
	if err != nil {
		return nil, err
	}

	return &EnrollmentChallenge{Token: token, Options: options}, nil
}

func (s *AuthenticatorService) finishWebAuthnEnrollment(ctx context.Context, token string, body []byte, requirement AccessRequirement) (*Authenticator, error) {
	if len(body) == 0 || len(body) > MaxEnrollmentBody {
		return nil, ErrEnrollment
	}

	err := requirement.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, err
	}

	digest, err := parseEnrollmentToken(token)
	if err != nil {
		return nil, err
	}

	select {
	case s.slots <- struct{}{}:
	default:
		return nil, ErrEnrollmentBusy
	}

	defer func() { <-s.slots }()

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	pending, err := s.queries.ReserveEnrollment(work, digest, requirement, s.settings)
	if err != nil {
		return nil, err
	}

	defer func() { _ = s.queries.ReleaseEnrollment(work, digest, pending.Revision) }()

	err = pending.Check(time.Now().UTC(), requirement, s.settings)
	if err != nil {
		return nil, err
	}

	securitySubject(work, pending.State.UserID)

	record, err := s.registration(*pending, body)
	if err != nil {
		securityClassification(work, SecurityInvalidProof)

		return nil, err
	}

	result, err := s.queries.ConfirmEnrollment(work, *pending, *record, requirement, s.settings)
	if err == nil && work.Err() != nil {
		return nil, work.Err()
	}

	return result, err
}

// CleanupEnrollments removes at most the configured expired-row batch. No owned
// goroutine, clock override or background cleanup loop is introduced.
func (s *AuthenticatorService) CleanupEnrollments(ctx context.Context) (int64, error) {
	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	return s.queries.DeleteExpiredEnrollments(work, s.settings.CleanupBatch)
}

// registration verifies the same protocol profile for initial and established enrollment.
func (s *AuthenticatorService) registration(pending EnrollmentPending, body []byte) (*RegistrationRecord, error) {
	parsed, err := parseEnrollmentResponse(body)
	if err != nil {
		return nil, ErrEnrollment
	}

	if !slices.Contains(s.settings.Origins, parsed.Response.CollectedClientData.Origin) {
		return nil, ErrEnrollment
	}

	var ceremony webauthn.SessionData

	err = json.Unmarshal(pending.Ceremony, &ceremony)
	if err != nil || ceremony.RelyingPartyID != s.settings.RPID || !bytes.Equal(ceremony.UserID, pending.Handle) || !ceremony.Expires.Equal(pending.ExpiresAt) || ceremony.UserVerification != protocol.VerificationRequired || len(ceremony.CredParams) != 1 || ceremony.CredParams[0].Algorithm != webauthncose.AlgES256 || ceremony.Mediation == protocol.MediationConditional {
		return nil, ErrEnrollment
	}

	credential, err := s.protocol.CreateCredential(registrationUser{handle: pending.Handle}, ceremony, parsed)
	if err != nil {
		return nil, ErrEnrollment
	}

	if !bytes.Equal(parsed.RawID, credential.ID) || !credential.Flags.UserPresent || !credential.Flags.UserVerified || len(credential.ID) == 0 || len(credential.ID) > 1024 || len(credential.PublicKey) == 0 || len(credential.PublicKey) > 4096 {
		return nil, ErrEnrollment
	}

	// Attestation-none checks the advertised algorithm but has no attestation
	// signature that would otherwise force public-key parsing/validation.
	parsedKey, err := webauthncose.ParsePublicKey(credential.PublicKey)
	if err != nil {
		return nil, ErrEnrollment
	}

	key, ok := parsedKey.(webauthncose.EC2PublicKeyData)
	if !ok || key.Algorithm != int64(webauthncose.AlgES256) {
		return nil, ErrEnrollment
	}

	data, err := json.Marshal(credential)
	if err != nil || len(data) > 65536 {
		return nil, ErrEnrollment
	}

	record := RegistrationRecord{Authenticator: Authenticator{ID: model.NewID(), CreatedAt: time.Now().UTC().Truncate(time.Microsecond), BackupEligible: credential.Flags.BackupEligible, BackupState: credential.Flags.BackupState}, CredentialID: credential.ID, PublicKey: credential.PublicKey, Data: data, Counter: credential.Authenticator.SignCount, VerifiedAt: time.Now().UTC().Truncate(time.Microsecond)}

	return &record, nil
}
