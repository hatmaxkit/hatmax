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
	"math"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/model"
)

var ErrWebAuthn = errors.New("invalid or unavailable WebAuthn authentication")

type AssertionPurpose uint8

const (
	AssertionSignin AssertionPurpose = 2
	AssertionStepUp AssertionPurpose = 3
)

type AssertionDigest [32]byte

// FactorBinding captures security identity, independent of replay revision.
type FactorBinding struct {
	ID       string
	Revision int64
}

// WebAuthnFactor is an owned internal verifier snapshot, never a proof receipt.
type WebAuthnFactor struct {
	FactorBinding
	ReplayRevision                int64
	CredentialID, PublicKey, Data []byte
	Counter                       uint32
	BackupEligible, BackupState   bool
}
type WebAuthnSubject struct {
	State   CredentialState
	Handle  []byte
	Factors []WebAuthnFactor
}

// AssertionPending contains exact server-owned ceremony and actor bindings.
type AssertionPending struct {
	Digest                           AssertionDigest
	Purpose                          AssertionPurpose
	State                            CredentialState
	Requirement                      AccessRequirement
	RPBinding                        [32]byte
	RPID                             string
	Handle, Ceremony                 []byte
	Factors                          []FactorBinding
	ActorID                          string
	ActorDigest                      SessionDigest
	ActorGeneration                  int64
	CreatedAt, ExpiresAt, LeaseUntil time.Time
	Attempts                         int
	Revision                         int64
}

func (p AssertionPending) Check(now time.Time, requirement AccessRequirement, settings config.EnrollmentSettings) error {
	if p.Digest == (AssertionDigest{}) || (p.Purpose != AssertionSignin && p.Purpose != AssertionStepUp) || p.State.UserID == "" || p.State.Version < 1 || p.RPBinding != settings.RPBinding || p.RPID != settings.RPID || len(p.Handle) != 32 || len(p.Ceremony) == 0 || len(p.Ceremony) > 16384 || len(p.Factors) == 0 || len(p.Factors) > settings.MaxAuthenticators || p.Requirement != requirement || p.CreatedAt.IsZero() || p.CreatedAt.After(now) || !now.Before(p.ExpiresAt) || p.ExpiresAt.Sub(p.CreatedAt) != settings.PendingTTL || p.Attempts < 0 || p.Attempts > settings.PendingAttempts || p.Revision < 0 {
		return ErrWebAuthn
	}

	if p.Purpose == AssertionSignin && (p.ActorID != "" || p.ActorDigest != (SessionDigest{}) || p.ActorGeneration != 0) {
		return ErrWebAuthn
	}

	if p.Purpose == AssertionStepUp && (!boundedID(p.ActorID, 128) || p.ActorDigest == (SessionDigest{}) || p.ActorGeneration < 1 || p.ActorGeneration == math.MaxInt64) {
		return ErrWebAuthn
	}

	seen := make(map[string]bool, len(p.Factors))
	for _, f := range p.Factors {
		if !boundedID(f.ID, 128) || f.Revision < 1 || seen[f.ID] {
			return ErrWebAuthn
		}

		seen[f.ID] = true
	}

	return requirement.Check(settings.PendingTTL)
}

type AssertionReservation struct {
	Pending AssertionPending
	Factors []WebAuthnFactor
	Actor   *Session
}

type AssertionCompletion struct {
	Factor      WebAuthnFactor
	Counter     uint32
	BackupState bool
	Data        []byte
	Session     SessionRecord
}

// WebAuthnQueries is required typed storage for actual assertion completion.
// Reservation commits admission first. Completion rechecks locked state and
// atomically consumes replay/pending state with session insertion or rotation.
type WebAuthnQueries interface {
	AuthenticatorQueries
	LoadWebAuthnSubject(context.Context, CredentialState, string, int) (*WebAuthnSubject, error)
	CreateAssertion(context.Context, AssertionPending, config.EnrollmentSettings) error
	ReserveAssertion(context.Context, AssertionDigest, AssertionPurpose, AccessRequirement, config.EnrollmentSettings) (*AssertionReservation, error)
	ReleaseAssertion(context.Context, AssertionDigest, int64) error
	CompleteAssertion(context.Context, AssertionReservation, AssertionCompletion, AccessRequirement, config.EnrollmentSettings, int) (*Session, error)
}

type AssertionChallenge struct {
	Token   string                        `json:"token"`
	Options *protocol.CredentialAssertion `json:"options"`
}

// WebAuthnService extends enrollment with required assertion storage.
type WebAuthnService struct {
	*AuthenticatorService
	queries WebAuthnQueries
}

func NewWebAuthnService(credentials *Service, queries WebAuthnQueries, cfg config.AuthenticatorConfig) (*WebAuthnService, error) {
	common, err := NewAuthenticatorService(credentials, queries, cfg)
	if err != nil {
		return nil, err
	}

	return &WebAuthnService{AuthenticatorService: common, queries: queries}, nil
}

func assertionPrefix(p AssertionPurpose) string {
	if p == AssertionSignin {
		return "assert1."
	}

	if p == AssertionStepUp {
		return "stepup1."
	}

	return ""
}
func parseAssertionToken(token string) (AssertionDigest, AssertionPurpose, error) {
	if len(token) != 51 {
		return AssertionDigest{}, 0, ErrWebAuthn
	}

	purpose := AssertionSignin
	if token[:8] == "stepup1." {
		purpose = AssertionStepUp
	}

	if token[:8] != assertionPrefix(purpose) {
		return AssertionDigest{}, 0, ErrWebAuthn
	}

	secret, err := base64.RawURLEncoding.Strict().DecodeString(token[8:])
	if err != nil || len(secret) != 32 {
		return AssertionDigest{}, 0, ErrWebAuthn
	}

	return sha256.Sum256(append([]byte("hatmax/pending/"+assertionPrefix(purpose)+"/v1\x00"), secret...)), purpose, nil
}

type assertionUser struct {
	handle      []byte
	credentials []webauthn.Credential
}

func (u assertionUser) WebAuthnID() []byte                         { return u.handle }
func (u assertionUser) WebAuthnName() string                       { return "" }
func (u assertionUser) WebAuthnDisplayName() string                { return "" }
func (u assertionUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func assertionCredentials(factors []WebAuthnFactor) ([]webauthn.Credential, error) {
	result := make([]webauthn.Credential, 0, len(factors))
	for _, f := range factors {
		var c webauthn.Credential

		if len(f.Data) == 0 || len(f.Data) > 65536 || len(f.CredentialID) == 0 || len(f.CredentialID) > 1024 || len(f.PublicKey) == 0 || len(f.PublicKey) > 4096 || f.Revision < 1 || f.ReplayRevision < 1 {
			return nil, ErrWebAuthn
		}

		err := json.Unmarshal(f.Data, &c)
		if err != nil || !bytes.Equal(c.ID, f.CredentialID) || !bytes.Equal(c.PublicKey, f.PublicKey) || c.Authenticator.SignCount != f.Counter || c.Flags.BackupEligible != f.BackupEligible || c.Flags.BackupState != f.BackupState || !c.Flags.UserVerified {
			return nil, ErrWebAuthn
		}

		key, err := webauthncose.ParsePublicKey(c.PublicKey)

		parsed, ok := key.(webauthncose.EC2PublicKeyData)
		if err != nil || !ok || parsed.Algorithm != int64(webauthncose.AlgES256) {
			return nil, ErrWebAuthn
		}

		result = append(result, c)
	}

	return result, nil
}

func (s *WebAuthnService) beginWebAuthnAuthentication(ctx context.Context, email string, required AccessRequirement) (*AssertionChallenge, error) {
	if len(email) == 0 || len(email) > 254 {
		return nil, ErrWebAuthn
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	user, err := s.queries.GetUserByEmail(work, email)
	if err != nil {
		outcome := SecurityOperatingUnknown
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrUserNotFound) {
			outcome = SecurityMissing
		}

		securityClassification(work, outcome)

		return nil, ErrWebAuthn
	}

	if user == nil {
		securityClassification(work, SecurityOperatingUnknown)

		return nil, ErrWebAuthn
	}

	securitySubject(work, user.ID)

	if !user.Active {
		securityClassification(work, SecurityInactive)

		return nil, ErrWebAuthn
	}

	return s.beginAssertion(work, CredentialState{UserID: user.ID, Version: user.AuthVersion}, nil, SessionDigest{}, required)
}

func (s *WebAuthnService) beginWebAuthnStepUp(ctx context.Context, token string, required AccessRequirement) (*AssertionChallenge, error) {
	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	entry := required
	entry.Proof = RequirePassword
	entry.MaxAge = 0

	actor, err := s.credentials.ValidateSession(work, token, entry, NoActivity)
	if err != nil {
		return nil, err
	}

	securityActor(work, actor.Session)

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	return s.beginAssertion(work, CredentialState{UserID: actor.User.ID, Version: actor.User.AuthVersion}, &actor.Session, digest, required)
}

func (s *WebAuthnService) beginAssertion(ctx context.Context, state CredentialState, actor *Session, actorDigest SessionDigest, required AccessRequirement) (*AssertionChallenge, error) {
	err := required.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, err
	}

	subject, err := s.queries.LoadWebAuthnSubject(ctx, state, s.settings.RPID, s.settings.MaxAuthenticators)
	if err != nil {
		return nil, err
	}

	if subject == nil || subject.State != state || len(subject.Handle) != 32 || len(subject.Factors) == 0 || len(subject.Factors) > s.settings.MaxAuthenticators {
		return nil, ErrWebAuthn
	}

	credentials, err := assertionCredentials(subject.Factors)
	if err != nil {
		return nil, err
	}

	options, ceremony, err := s.protocol.BeginLogin(assertionUser{handle: subject.Handle, credentials: credentials}, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, ErrWebAuthn
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	ceremony.Expires = now.Add(s.settings.PendingTTL)

	encoded, err := json.Marshal(ceremony)
	if err != nil {
		return nil, err
	}

	secret := make([]byte, 32)

	_, err = rand.Read(secret)
	if err != nil {
		return nil, err
	}

	p := AssertionPending{Purpose: AssertionSignin, State: state, Requirement: required, RPBinding: s.settings.RPBinding, RPID: s.settings.RPID, Handle: subject.Handle, Ceremony: encoded, CreatedAt: now, ExpiresAt: ceremony.Expires}
	for _, f := range subject.Factors {
		p.Factors = append(p.Factors, f.FactorBinding)
	}

	if actor != nil {
		p.Purpose = AssertionStepUp
		p.ActorID = actor.ID
		p.ActorDigest = actorDigest
		p.ActorGeneration = actor.Generation
	}

	token := assertionPrefix(p.Purpose) + base64.RawURLEncoding.EncodeToString(secret)

	p.Digest, _, err = parseAssertionToken(token)
	if err != nil {
		return nil, err
	}

	err = s.queries.CreateAssertion(ctx, p, s.settings)
	if err != nil {
		return nil, err
	}

	return &AssertionChallenge{Token: token, Options: options}, nil
}

// ValidAssertionCounter permits counterless authenticators, otherwise requires
// strictly increasing counters including rejection of nonzero-to-zero changes.
func ValidAssertionCounter(previous, current uint32) bool {
	return previous == 0 && current == 0 || current > previous
}

func (s *WebAuthnService) finishWebAuthn(ctx context.Context, token string, body []byte, required AccessRequirement) (*IssuedSession, error) {
	if len(body) == 0 || len(body) > MaxEnrollmentBody {
		securityClassification(ctx, SecurityInvalidProof)

		return nil, ErrWebAuthn
	}

	digest, purpose, err := parseAssertionToken(token)
	if err != nil {
		return nil, err
	}

	err = required.Check(s.settings.PendingTTL)
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

	reserved, err := s.queries.ReserveAssertion(work, digest, purpose, required, s.settings)
	if err != nil {
		return nil, err
	}

	defer func() { _ = s.queries.ReleaseAssertion(work, digest, reserved.Pending.Revision) }()

	p := reserved.Pending

	err = p.Check(time.Now().UTC(), required, s.settings)
	if err != nil {
		return nil, err
	}

	securitySubject(work, p.State.UserID)

	parsed, err := parseAssertionResponse(body)
	if err != nil || !slices.Contains(s.settings.Origins, parsed.Response.CollectedClientData.Origin) {
		securityClassification(work, SecurityInvalidProof)

		return nil, ErrWebAuthn
	}

	var ceremony webauthn.SessionData

	err = json.Unmarshal(p.Ceremony, &ceremony)
	if err != nil || ceremony.RelyingPartyID != s.settings.RPID || !bytes.Equal(ceremony.UserID, p.Handle) || !ceremony.Expires.Equal(p.ExpiresAt) || ceremony.UserVerification != protocol.VerificationRequired || len(ceremony.AllowedCredentialIDs) == 0 || len(ceremony.AllowedCredentialIDs) != len(p.Factors) || ceremony.Mediation == protocol.MediationConditional {
		return nil, ErrWebAuthn
	}

	credentials, err := assertionCredentials(reserved.Factors)
	if err != nil {
		return nil, err
	}

	credential, err := s.protocol.ValidateLogin(assertionUser{handle: p.Handle, credentials: credentials}, ceremony, parsed)
	if err != nil || !parsed.Response.AuthenticatorData.Flags.UserPresent() || !parsed.Response.AuthenticatorData.Flags.UserVerified() {
		securityClassification(work, SecurityInvalidProof)

		return nil, ErrWebAuthn
	}

	var selected *WebAuthnFactor

	for i := range reserved.Factors {
		if bytes.Equal(reserved.Factors[i].CredentialID, parsed.RawID) {
			selected = &reserved.Factors[i]

			break
		}
	}

	if selected == nil || !slices.Contains(p.Factors, selected.FactorBinding) || !ValidAssertionCounter(selected.Counter, parsed.Response.AuthenticatorData.Counter) || credential.Flags.BackupEligible != selected.BackupEligible || credential.Flags.BackupState && !credential.Flags.BackupEligible {
		return nil, ErrWebAuthn
	}

	securityProof(work, WebAuthnProof)

	data, err := json.Marshal(credential)
	if err != nil || len(data) > 65536 {
		return nil, ErrWebAuthn
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	session := Session{ID: model.NewID(), UserID: p.State.UserID, AuthVersion: p.State.Version, Generation: 1, CreatedAt: now}
	if p.Purpose == AssertionStepUp {
		if reserved.Actor == nil || reserved.Actor.Generation == math.MaxInt64 {
			return nil, ErrWebAuthn
		}

		session = *reserved.Actor
		session.Generation++
	}

	session.PolicyRevision = required.Revision
	session.Proof = VerifiedProof{Method: WebAuthnProof, VerifiedAt: now, FactorID: selected.ID, FactorRevision: selected.Revision}
	session.AuthenticatedAt = now
	session.LastActivityAt = now
	session.ExpiresAt = now.Add(s.credentials.sessions.TTL)
	session.InactivityTTL = s.credentials.sessions.InactivityTTL

	bearer, newDigest, err := s.credentials.sessionToken()
	if err != nil {
		return nil, err
	}

	command := AssertionCompletion{Factor: *selected, Counter: parsed.Response.AuthenticatorData.Counter, BackupState: credential.Flags.BackupState, Data: data, Session: SessionRecord{Session: session, Digest: newDigest}}

	completed, err := s.queries.CompleteAssertion(work, *reserved, command, required, s.settings, s.credentials.sessions.MaxPerSubject)
	if err != nil {
		return nil, err
	}

	err = work.Err()
	if err != nil {
		return nil, err
	}

	return &IssuedSession{Session: *completed, Token: bearer}, nil
}

// Matches denies any substitution of the durable reservation snapshot.
func (p AssertionPending) Matches(other AssertionPending) bool {
	return p.Digest == other.Digest && p.Purpose == other.Purpose && p.State == other.State && p.Requirement == other.Requirement && p.RPBinding == other.RPBinding && p.RPID == other.RPID && bytes.Equal(p.Handle, other.Handle) && bytes.Equal(p.Ceremony, other.Ceremony) && slices.Equal(p.Factors, other.Factors) && p.ActorID == other.ActorID && p.ActorDigest == other.ActorDigest && p.ActorGeneration == other.ActorGeneration && p.CreatedAt.Equal(other.CreatedAt) && p.ExpiresAt.Equal(other.ExpiresAt) && p.LeaseUntil.Equal(other.LeaseUntil) && p.Attempts == other.Attempts && p.Revision == other.Revision
}

// Check validates the core-produced completion against current trusted time.
// The adapter additionally matches the locked factor's replay/security snapshot.
func (c AssertionCompletion) Check(p AssertionPending, actor *Session, now time.Time, required AccessRequirement, settings config.EnrollmentSettings) error {
	err := p.Check(now, required, settings)
	if err != nil {
		return err
	}

	s := c.Session
	if p.Attempts < 1 || p.Revision < 1 || p.LeaseUntil.IsZero() || !now.Before(p.LeaseUntil) || s.Digest == (SessionDigest{}) || s.UserID != p.State.UserID || s.AuthVersion != p.State.Version || s.Proof.Method != WebAuthnProof || s.Proof.FactorID != c.Factor.ID || s.Proof.FactorRevision != c.Factor.Revision || !slices.Contains(p.Factors, c.Factor.FactorBinding) || !ValidAssertionCounter(c.Factor.Counter, c.Counter) || c.BackupState && !c.Factor.BackupEligible || len(c.Data) == 0 || len(c.Data) > 65536 || !s.Proof.VerifiedAt.Equal(s.AuthenticatedAt) || s.Proof.VerifiedAt.Before(p.LeaseUntil.Add(-settings.Lease)) || s.Proof.VerifiedAt.After(now) || s.Proof.VerifiedAt.Before(p.CreatedAt) {
		return ErrWebAuthn
	}

	age := settings.RecentProofAge
	if required.MaxAge != 0 && required.MaxAge < age {
		age = required.MaxAge
	}

	if now.Sub(s.Proof.VerifiedAt) >= age {
		return ErrSessionProofExpired
	}

	if p.Purpose == AssertionSignin {
		if actor != nil || s.Generation != 1 || !s.CreatedAt.Equal(s.AuthenticatedAt) {
			return ErrWebAuthn
		}
	} else {
		if actor == nil || actor.ID != p.ActorID || actor.Generation != p.ActorGeneration || actor.UserID != p.State.UserID || s.ID != actor.ID || s.Generation != actor.Generation+1 || !s.CreatedAt.Equal(actor.CreatedAt) || s.AuthenticatedAt.Before(actor.AuthenticatedAt) || s.Digest == p.ActorDigest {
			return ErrWebAuthn
		}

		entry := required
		entry.Proof = RequirePassword
		entry.MaxAge = 0

		err = entry.Evaluate(*actor, now)
		if err != nil {
			return err
		}
	}

	return required.Evaluate(s.Session, now)
}
