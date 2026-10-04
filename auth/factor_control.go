// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/model"
)

var ErrFactorChange = errors.New("invalid or unavailable authenticator change")

// FactorKind identifies a confirmed primary authenticator, never proof authority.
type FactorKind uint8

const (
	FactorWebAuthn FactorKind = 1
	FactorTOTP     FactorKind = 2
)

// FactorSelection is bounded lookup metadata. Storage must recheck subject ownership.
type FactorSelection struct {
	Kind     FactorKind `json:"kind"`
	ID       string     `json:"id"`
	Revision int64      `json:"revision"`
}

func (f FactorSelection) Check() error {
	if f.Kind < FactorWebAuthn || f.Kind > FactorTOTP || !boundedID(f.ID, 128) || f.Revision < 1 {
		return ErrFactorChange
	}

	return nil
}

// Factor is safe metadata; no credential, seed, digest or verifier is returned.
type Factor struct {
	FactorSelection
	CreatedAt      time.Time `json:"createdAt"`
	BackupEligible bool      `json:"backupEligible"`
	BackupState    bool      `json:"backupState"`
}

// FactorPolicy captures trusted recent management and the current usable-factor profile.
// Management defaults belong to the application: phishing-resistant MFA is recommended.
type FactorPolicy struct {
	Management AccessRequirement
	Access     RequiredProof
	// TOTPKeyIDs is filled by FactorService from its actual owned key ring.
	// Caller values are ignored at service entry; storage uses this finite snapshot.
	TOTPKeyIDs [8]string
}

func (p FactorPolicy) Check(settings config.EnrollmentSettings) error {
	if (p.Management.Proof != RequireMFA && p.Management.Proof != RequirePhishingResistantMFA) || p.Management.MaxAge < time.Second || p.Management.MaxAge > settings.RecentProofAge || p.Access < RequirePassword || p.Access > RequirePhishingResistantMFA {
		return ErrSessionProof
	}

	previous := ""

	for i, id := range p.TOTPKeyIDs {
		if id == "" {
			for _, remaining := range p.TOTPKeyIDs[i:] {
				if remaining != "" {
					return ErrFactorChange
				}
			}

			break
		}

		if !boundedID(id, 64) || id <= previous {
			return ErrFactorChange
		}

		previous = id
	}

	return p.Management.Check(settings.PendingTTL)
}

// FactorChangePending is a restricted, purpose-separated, owned storage snapshot.
// Target zero means addition; replacement binds an existing exact owned revision.
type FactorChangePending struct {
	Digest               EnrollmentDigest
	State                CredentialState
	Actor                Session
	ActorDigest          SessionDigest
	Policy               FactorPolicy
	Kind                 FactorKind
	Target               FactorSelection
	RPID                 string
	RPBinding            [32]byte
	Handle, Ceremony     []byte
	Material             FallbackMaterial
	CreatedAt, ExpiresAt time.Time
	Attempts             int
	Revision             int64
	LeaseUntil           time.Time
}

func (p FactorChangePending) Check(now time.Time, policy FactorPolicy, settings config.EnrollmentSettings) error {
	if policy != p.Policy || policy.Check(settings) != nil || p.Digest == (EnrollmentDigest{}) || p.ActorDigest == (SessionDigest{}) || p.State.UserID == "" || p.State.Version < 1 || p.State.Version == math.MaxInt64 || p.Actor.UserID != p.State.UserID || p.Actor.AuthVersion != p.State.Version || p.Actor.Generation == math.MaxInt64 || p.CreatedAt.After(now) || !now.Before(p.ExpiresAt) || p.ExpiresAt.Sub(p.CreatedAt) != settings.PendingTTL || p.Attempts < 0 || p.Attempts > settings.PendingAttempts || p.Revision < 0 {
		return ErrFactorChange
	}

	if p.Target != (FactorSelection{}) && p.Target.Check() != nil {
		return ErrFactorChange
	}

	if p.Kind == FactorWebAuthn {
		if p.RPID != settings.RPID || p.RPBinding != settings.RPBinding || len(p.Handle) != 32 || len(p.Ceremony) == 0 || len(p.Ceremony) > 16384 || (p.Material.Method != 0 || p.Material.Skew != 0 || p.Material.Factor != (FactorBinding{}) || p.Material.KeyID != "" || len(p.Material.Envelope) != 0) {
			return ErrFactorChange
		}
	} else if p.Kind == FactorTOTP {
		if p.RPID != "" || p.RPBinding != ([32]byte{}) || len(p.Handle) != 0 || len(p.Ceremony) != 0 || p.Material.Method != FallbackTOTP || p.Material.Factor.Revision != 1 || !boundedID(p.Material.Factor.ID, 128) || !boundedID(p.Material.KeyID, 64) || len(p.Material.Envelope) != 61 || p.Material.Skew > 1 {
			return ErrFactorChange
		}
	} else {
		return ErrFactorChange
	}

	return policy.Management.Evaluate(p.Actor, now)
}

// FactorChangeRecord contains actual verifier output for the reserved operation.
type FactorChangeRecord struct {
	Registration *RegistrationRecord
	Step         int64
	VerifiedAt   time.Time
}

// FactorChangeResult omits the transient bearer from JSON; transports set a cookie.
type FactorChangeResult struct {
	Factor *Factor        `json:"factor,omitempty"`
	Issued *IssuedSession `json:"-"`
}

// FactorQueries is required typed storage. All mutations lock subject first,
// revalidate current actor/target/proof/time, and commit version, revocation and
// optional actor rotation together. Reserve durably charges the shared budgets.
type FactorQueries interface {
	EnsureWebAuthnHandle(context.Context, CredentialState, string, []byte) ([]byte, error)
	ListFactors(context.Context, SessionDigest, FactorPolicy, config.EnrollmentSettings) ([]Factor, error)
	CreateFactorChange(context.Context, FactorChangePending, config.EnrollmentSettings) error
	ReserveFactorChange(context.Context, EnrollmentDigest, SessionDigest, FactorPolicy, config.EnrollmentSettings) (*FactorChangePending, error)
	ReleaseEnrollment(context.Context, EnrollmentDigest, int64) error
	CompleteFactorChange(context.Context, FactorChangePending, FactorChangeRecord, SessionRecord, FactorPolicy, config.EnrollmentSettings) (*Factor, *Session, error)
	RemoveFactor(context.Context, SessionDigest, FactorSelection, SessionRecord, FactorPolicy, config.EnrollmentSettings) (*Session, error)
}

// FactorService owns established-factor changes. A nil fallback explicitly disables
// TOTP change operations; the mandatory storage contract is unchanged.
type FactorService struct {
	enrollment *AuthenticatorService
	queries    FactorQueries
	fallback   *FallbackService
}

func NewFactorService(credentials *Service, queries FactorQueries, enrollment *AuthenticatorService, fallback *FallbackService) (*FactorService, error) {
	if credentials == nil || queries == nil || enrollment == nil || enrollment.credentials != credentials || fallback != nil && fallback.credentials != credentials {
		return nil, ErrFactorChange
	}

	return &FactorService{enrollment: enrollment, queries: queries, fallback: fallback}, nil
}
func factorChangeToken() (string, EnrollmentDigest, error) {
	secret := make([]byte, 32)

	_, err := rand.Read(secret)
	if err != nil {
		return "", EnrollmentDigest{}, err
	}

	token := "change1." + base64.RawURLEncoding.EncodeToString(secret)
	digest, err := parseFactorChangeToken(token)

	return token, digest, err
}
func parseFactorChangeToken(token string) (EnrollmentDigest, error) {
	if len(token) != 51 || token[:8] != "change1." {
		return EnrollmentDigest{}, ErrFactorChange
	}

	secret, err := base64.RawURLEncoding.Strict().DecodeString(token[8:])
	if err != nil || len(secret) != 32 {
		return EnrollmentDigest{}, ErrFactorChange
	}

	return sha256.Sum256(append([]byte("hatmax/pending/factor-change/v1\x00"), secret...)), nil
}
func (s *FactorService) policy(policy FactorPolicy) FactorPolicy {
	policy.TOTPKeyIDs = [8]string{}

	if s.fallback != nil {
		identities := make([]string, 0, len(s.fallback.seeds.keys))
		for id := range s.fallback.seeds.keys {
			identities = append(identities, id)
		}

		slices.Sort(identities)
		copy(policy.TOTPKeyIDs[:], identities)
	}

	return policy
}
func (s *FactorService) actor(ctx context.Context, token string, policy FactorPolicy) (*ValidatedSession, SessionDigest, error) {
	err := policy.Check(s.enrollment.settings)
	if err != nil {
		return nil, SessionDigest{}, err
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, SessionDigest{}, err
	}

	actor, err := s.enrollment.credentials.ValidateSession(ctx, token, policy.Management, NoActivity)

	return actor, digest, err
}
func (s *FactorService) List(ctx context.Context, token string, policy FactorPolicy) ([]Factor, error) {
	policy = s.policy(policy)

	err := policy.Check(s.enrollment.settings)
	if err != nil {
		return nil, err
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.enrollment.settings.Timeout)
	defer cancel()

	return s.queries.ListFactors(work, digest, policy, s.enrollment.settings)
}
func (s *FactorService) pending(ctx context.Context, token string, kind FactorKind, target FactorSelection, policy FactorPolicy) (*FactorChangePending, string, string, error) {
	if target != (FactorSelection{}) && target.Check() != nil {
		return nil, "", "", ErrFactorChange
	}

	actor, digest, err := s.actor(ctx, token, policy)
	if err != nil {
		return nil, "", "", err
	}

	wire, pdigest, err := factorChangeToken()
	if err != nil {
		return nil, "", "", err
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	p := &FactorChangePending{Digest: pdigest, State: CredentialState{UserID: actor.Session.UserID, Version: actor.Session.AuthVersion}, Actor: actor.Session, ActorDigest: digest, Policy: policy, Kind: kind, Target: target, CreatedAt: now, ExpiresAt: now.Add(s.enrollment.settings.PendingTTL)}

	return p, wire, actor.User.Email, nil
}

// BeginWebAuthnChange adds or replaces a factor under actual recent management proof.
func (s *FactorService) BeginWebAuthnChange(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*EnrollmentChallenge, error) {
	policy = s.policy(policy)

	work, cancel := context.WithTimeout(ctx, s.enrollment.settings.Timeout)
	defer cancel()

	p, wire, email, err := s.pending(work, token, FactorWebAuthn, target, policy)
	if err != nil {
		return nil, err
	}

	handle := make([]byte, 32)

	_, err = rand.Read(handle)
	if err != nil {
		return nil, err
	}

	p.RPID = s.enrollment.settings.RPID
	p.RPBinding = s.enrollment.settings.RPBinding

	p.Handle, err = s.queries.EnsureWebAuthnHandle(work, p.State, p.RPID, handle)
	if err != nil {
		return nil, err
	}

	options, ceremony, err := s.enrollment.protocol.BeginRegistration(registrationUser{handle: p.Handle, name: email}, webauthn.WithCredentialParameters([]protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}}))
	if err != nil {
		return nil, err
	}

	ceremony.Expires = p.ExpiresAt

	p.Ceremony, err = json.Marshal(ceremony)
	if err != nil {
		return nil, err
	}

	err = s.queries.CreateFactorChange(work, *p, s.enrollment.settings)
	if err != nil {
		return nil, err
	}

	return &EnrollmentChallenge{Token: wire, Options: options}, nil
}
func (s *FactorService) BeginTOTPChange(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*TOTPSetup, error) {
	policy = s.policy(policy)
	if s.fallback == nil {
		return nil, ErrFactorChange
	}

	work, cancel := context.WithTimeout(ctx, s.enrollment.settings.Timeout)
	defer cancel()

	p, wire, email, err := s.pending(work, token, FactorTOTP, target, policy)
	if err != nil {
		return nil, err
	}

	key, err := crypto.GenerateTOTPKey(s.fallback.settings.Issuer, email)
	if err != nil {
		return nil, err
	}

	id := model.NewID()

	keyID, envelope, err := s.fallback.seeds.seal(p.State.UserID, id, key.Secret())
	if err != nil {
		return nil, err
	}

	p.Material = FallbackMaterial{Method: FallbackTOTP, Skew: s.fallback.settings.Skew, Factor: FactorBinding{ID: id, Revision: 1}, KeyID: keyID, Envelope: envelope}

	err = s.queries.CreateFactorChange(work, *p, s.enrollment.settings)
	if err != nil {
		return nil, err
	}

	return &TOTPSetup{Token: wire, URL: key.URL()}, nil
}
func (s *FactorService) rotation(actor Session) (SessionRecord, string, error) {
	if actor.AuthVersion == math.MaxInt64 || actor.Generation == math.MaxInt64 {
		return SessionRecord{}, "", ErrFactorChange
	}

	token, digest, err := s.enrollment.credentials.sessionToken()
	actor.AuthVersion++
	actor.Generation++

	return SessionRecord{Session: actor, Digest: digest}, token, err
}
func (s *FactorService) finish(ctx context.Context, actorToken, token string, kind FactorKind, body []byte, code string, policy FactorPolicy) (*FactorChangeResult, error) {
	policy = s.policy(policy)

	settings := s.enrollment.settings
	if policy.Check(settings) != nil || kind == FactorTOTP && s.fallback == nil || kind == FactorWebAuthn && (len(body) == 0 || len(body) > MaxEnrollmentBody) {
		return nil, ErrFactorChange
	}

	digest, err := parseFactorChangeToken(token)
	if err != nil {
		return nil, err
	}

	actorDigest, err := ParseSessionToken(actorToken)
	if err != nil {
		return nil, err
	}

	select {
	case s.enrollment.slots <- struct{}{}:
	default:
		return nil, ErrEnrollmentBusy
	}

	defer func() { <-s.enrollment.slots }()

	work, cancel := context.WithTimeout(ctx, settings.Timeout)
	defer cancel()

	p, err := s.queries.ReserveFactorChange(work, digest, actorDigest, policy, settings)
	if err != nil {
		return nil, err
	}

	defer func() { _ = s.queries.ReleaseEnrollment(work, digest, p.Revision) }()

	if p.Kind != kind || p.ActorDigest != actorDigest {
		return nil, ErrFactorChange
	}

	err = p.Check(time.Now().UTC(), policy, settings)
	if err != nil {
		return nil, err
	}

	record := FactorChangeRecord{}
	if kind == FactorWebAuthn {
		record.Registration, err = s.enrollment.registration(EnrollmentPending{RPID: p.RPID, Handle: p.Handle, Ceremony: p.Ceremony, CreatedAt: p.CreatedAt, ExpiresAt: p.ExpiresAt}, body)
		if err != nil {
			return nil, err
		}

		record.VerifiedAt = record.Registration.VerifiedAt
	} else {
		if p.Material.Skew != s.fallback.settings.Skew {
			return nil, ErrFactorChange
		}

		secret, openErr := s.fallback.seeds.open(p.State.UserID, p.Material.Factor.ID, p.Material.KeyID, p.Material.Envelope)
		if openErr != nil {
			return nil, openErr
		}

		record.VerifiedAt = time.Now().UTC().Truncate(time.Microsecond)

		record.Step, err = crypto.MatchTOTPCode(secret, code, record.VerifiedAt, p.Material.Skew)
		if err != nil {
			return nil, err
		}
	}

	replacement, bearer, err := s.rotation(p.Actor)
	if err != nil {
		return nil, err
	}

	factor, session, err := s.queries.CompleteFactorChange(work, *p, record, replacement, policy, settings)
	if err != nil {
		return nil, err
	}

	if work.Err() != nil {
		return nil, work.Err()
	}

	result := &FactorChangeResult{Factor: factor}
	if session != nil {
		result.Issued = &IssuedSession{Session: *session, Token: bearer}
	}

	return result, nil
}
func (s *FactorService) FinishWebAuthnChange(ctx context.Context, actorToken, token string, body []byte, policy FactorPolicy) (*FactorChangeResult, error) {
	return s.finish(ctx, actorToken, token, FactorWebAuthn, body, "", policy)
}
func (s *FactorService) FinishTOTPChange(ctx context.Context, actorToken, token, code string, policy FactorPolicy) (*FactorChangeResult, error) {
	if len(code) != 6 {
		return nil, ErrFactorChange
	}

	return s.finish(ctx, actorToken, token, FactorTOTP, nil, code, policy)
}
func (s *FactorService) Remove(ctx context.Context, token string, target FactorSelection, policy FactorPolicy) (*FactorChangeResult, error) {
	policy = s.policy(policy)

	if target.Check() != nil {
		return nil, ErrFactorChange
	}

	work, cancel := context.WithTimeout(ctx, s.enrollment.settings.Timeout)
	defer cancel()

	actor, digest, err := s.actor(work, token, policy)
	if err != nil {
		return nil, err
	}

	replacement, bearer, err := s.rotation(actor.Session)
	if err != nil {
		return nil, err
	}

	session, err := s.queries.RemoveFactor(work, digest, target, replacement, policy, s.enrollment.settings)
	if err != nil {
		return nil, err
	}

	if work.Err() != nil {
		return nil, work.Err()
	}

	result := &FactorChangeResult{}
	if session != nil {
		result.Issued = &IssuedSession{Session: *session, Token: bearer}
	}

	return result, nil
}
