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
	"encoding/base64"
	"errors"
	"math"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/crypto"
	"hatmax.adrianpk.com/model"
)

var ErrFallback = errors.New("invalid or unavailable fallback authentication")

type FallbackMethod uint8

const (
	FallbackTOTP   FallbackMethod = 1
	FallbackBackup FallbackMethod = 2
)

type FallbackPurpose uint8

const (
	FallbackSetup  FallbackPurpose = 4
	FallbackSignin FallbackPurpose = 5
	FallbackStepUp FallbackPurpose = 6
)

type FallbackDigest [32]byte

// FallbackFactor is storage material, never public session metadata.
type FallbackFactor struct {
	FactorBinding
	ReplayRevision int64
	KeyID          string
	Envelope       []byte
	AcceptedStep   int64
}

// FallbackMaterial is the closed, bounded server ceremony payload.
type FallbackMaterial struct {
	Method   FallbackMethod
	Skew     uint
	Factor   FactorBinding
	KeyID    string
	Envelope []byte
}
type FallbackPending struct {
	Digest                                       FallbackDigest
	Purpose                                      FallbackPurpose
	State                                        CredentialState
	Requirement                                  AccessRequirement
	Material                                     FallbackMaterial
	ActorID                                      string
	ActorDigest                                  SessionDigest
	ActorGeneration                              int64
	PasswordAt, CreatedAt, ExpiresAt, LeaseUntil time.Time
	Attempts                                     int
	Revision                                     int64
}

func (p FallbackPending) Check(now time.Time, required AccessRequirement, settings config.FallbackSettings) error {
	if p.Digest == (FallbackDigest{}) || p.Purpose < FallbackSetup || p.Purpose > FallbackStepUp || !boundedID(p.State.UserID, 128) || p.State.Version < 1 || p.State.Version == math.MaxInt64 || p.Requirement != required || required.Proof == RequirePhishingResistantMFA || p.PasswordAt.IsZero() || p.PasswordAt.After(p.CreatedAt) || p.CreatedAt.After(now) || !now.Before(p.ExpiresAt) || p.ExpiresAt.Sub(p.CreatedAt) != settings.PendingTTL || p.Attempts < 0 || p.Attempts > settings.PendingAttempts || p.Revision < 0 || p.Revision == math.MaxInt64 || !boundedID(p.Material.Factor.ID, 128) || p.Material.Factor.Revision < 1 {
		return ErrFallback
	}

	age := settings.RecentProofAge
	if required.MaxAge != 0 && required.MaxAge < age {
		age = required.MaxAge
	}

	if now.Sub(p.PasswordAt) >= age {
		return ErrSessionProofExpired
	}

	if p.Material.Method != FallbackTOTP && p.Material.Method != FallbackBackup || p.Material.Skew != settings.Skew {
		return ErrFallback
	}

	if p.Purpose == FallbackSetup {
		if p.Material.Method != FallbackTOTP || p.Material.Factor.Revision != 1 || !boundedID(p.Material.KeyID, 64) || len(p.Material.Envelope) != 61 {
			return ErrFallback
		}
	} else if p.Material.KeyID != "" || len(p.Material.Envelope) != 0 {
		return ErrFallback
	}

	if p.Purpose == FallbackStepUp {
		if !boundedID(p.ActorID, 128) || p.ActorDigest == (SessionDigest{}) || p.ActorGeneration < 1 || p.ActorGeneration == math.MaxInt64 {
			return ErrFallback
		}
	} else if p.ActorID != "" || p.ActorDigest != (SessionDigest{}) || p.ActorGeneration != 0 {
		return ErrFallback
	}

	return required.Check(settings.PendingTTL)
}
func (p FallbackPending) Matches(b FallbackPending) bool {
	return p.Digest == b.Digest && p.Purpose == b.Purpose && p.State == b.State && p.Requirement == b.Requirement && p.Material.Method == b.Material.Method && p.Material.Skew == b.Material.Skew && p.Material.Factor == b.Material.Factor && p.Material.KeyID == b.Material.KeyID && bytes.Equal(p.Material.Envelope, b.Material.Envelope) && p.ActorID == b.ActorID && p.ActorDigest == b.ActorDigest && p.ActorGeneration == b.ActorGeneration && p.PasswordAt.Equal(b.PasswordAt) && p.CreatedAt.Equal(b.CreatedAt) && p.ExpiresAt.Equal(b.ExpiresAt) && p.LeaseUntil.Equal(b.LeaseUntil) && p.Attempts == b.Attempts && p.Revision == b.Revision
}

type BackupVerifier struct {
	ID, SetID, Record string
	Revision          int64
}
type FallbackReservation struct {
	Pending FallbackPending
	Factor  FallbackFactor
	Backup  BackupVerifier
	Actor   *Session
}
type FallbackCompletion struct {
	Step     int64
	FactorAt time.Time
	Session  SessionRecord
}
type BackupIssueReservation struct {
	State  CredentialState
	Actor  Session
	Digest SessionDigest
}
type BackupSet struct {
	ID    string
	Codes []BackupVerifier
}

// FallbackQueries is mandatory typed storage. Reservations durably charge the
// shared budget before cryptography; completion consumes replay and access in
// one transaction, conditionally matching all captured revisions and policy.
type FallbackQueries interface {
	GetUserByEmail(context.Context, string) (*User, error)
	DeleteExpiredEnrollments(context.Context, int) (int64, error)
	LoadFallbackFactor(context.Context, CredentialState, FallbackMethod) (FactorBinding, error)
	CreateFallback(context.Context, FallbackPending, config.FallbackSettings) error
	ReserveFallback(context.Context, FallbackDigest, FallbackPurpose, string, AccessRequirement, config.FallbackSettings) (*FallbackReservation, error)
	ReleaseFallback(context.Context, FallbackDigest, int64) error
	CompleteTOTPSetup(context.Context, FallbackReservation, int64, AccessRequirement, config.FallbackSettings) error
	CompleteFallback(context.Context, FallbackReservation, FallbackCompletion, AccessRequirement, config.FallbackSettings, int) (*Session, error)
	ReserveBackupIssue(context.Context, SessionDigest, AccessRequirement, config.FallbackSettings) (*BackupIssueReservation, error)
	ReplaceBackupSet(context.Context, BackupIssueReservation, BackupSet, SessionRecord, AccessRequirement, config.FallbackSettings) (*Session, error)
}
type FallbackService struct {
	credentials *Service
	queries     FallbackQueries
	settings    config.FallbackSettings
	seeds       *seedCipher
	slots       chan struct{}
}

func NewFallbackService(credentials *Service, queries FallbackQueries, cfg config.FallbackConfig, keys SeedKeys) (*FallbackService, error) {
	if credentials == nil || credentials.verifier == nil || credentials.passwordTimeout <= 0 || queries == nil {
		return nil, ErrFallback
	}

	settings, err := cfg.Settings()
	if err != nil {
		return nil, err
	}

	seeds, err := newSeedCipher(keys)
	if err != nil {
		return nil, err
	}

	return &FallbackService{credentials: credentials, queries: queries, settings: settings, seeds: seeds, slots: make(chan struct{}, settings.MaxConcurrent)}, nil
}
func fallbackPrefix(p FallbackPurpose) string {
	switch p {
	case FallbackSetup:
		return "totpset1."
	case FallbackSignin:
		return "fallback1."
	case FallbackStepUp:
		return "fallstep1."
	}

	return ""
}
func ParseFallbackToken(token string) (FallbackDigest, FallbackPurpose, error) {
	for _, p := range []FallbackPurpose{FallbackSetup, FallbackSignin, FallbackStepUp} {
		prefix := fallbackPrefix(p)
		if len(token) != len(prefix)+43 || token[:len(prefix)] != prefix {
			continue
		}

		raw, err := base64.RawURLEncoding.Strict().DecodeString(token[len(prefix):])
		if err != nil || len(raw) != 32 {
			return FallbackDigest{}, 0, ErrFallback
		}

		return sha256.Sum256(append([]byte("hatmax/pending/"+prefix+"/v1\x00"), raw...)), p, nil
	}

	return FallbackDigest{}, 0, ErrFallback
}

type FallbackChallenge struct {
	Token string `json:"token"`
}
type TOTPSetup struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func (s *FallbackService) verifyPassword(ctx context.Context, email, password string) (*User, time.Time, error) {
	if len(email) == 0 || len(email) > 254 {
		return nil, time.Time{}, ErrFallback
	}

	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil || user == nil || !user.Active || user.AuthVersion < 1 {
		return nil, time.Time{}, ErrFallback
	}

	err = s.credentials.verifier.Verify(ctx, user.PasswordHash, password)
	if err != nil {
		return nil, time.Time{}, err
	}

	return user, time.Now().UTC().Truncate(time.Microsecond), nil
}
func (s *FallbackService) makePending(state CredentialState, at time.Time, purpose FallbackPurpose, material FallbackMaterial, required AccessRequirement) (FallbackPending, string, error) {
	var secret [32]byte

	_, err := rand.Read(secret[:])
	if err != nil {
		return FallbackPending{}, "", err
	}

	material.Skew = s.settings.Skew

	token := fallbackPrefix(purpose) + base64.RawURLEncoding.EncodeToString(secret[:])
	digest, _, err := ParseFallbackToken(token)
	now := time.Now().UTC().Truncate(time.Microsecond)

	return FallbackPending{Digest: digest, Purpose: purpose, State: state, Requirement: required, Material: material, PasswordAt: at, CreatedAt: now, ExpiresAt: now.Add(s.settings.PendingTTL)}, token, err
}

// BeginTOTPSetup supports only password-authorized initial setup. Confirmation
// verifies a real code, consumes that step and invalidates sessions; no access is issued.
func (s *FallbackService) BeginTOTPSetup(ctx context.Context, email, password string, required AccessRequirement) (*TOTPSetup, error) {
	if required.Proof != RequireMFA && required.Proof != RequirePassword {
		return nil, ErrSessionProof
	}

	err := required.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	user, at, err := s.verifyPassword(work, email, password)
	if err != nil {
		return nil, err
	}

	key, err := crypto.GenerateTOTPKey(s.settings.Issuer, user.Email)
	if err != nil {
		return nil, err
	}

	factor := model.NewID()

	keyID, envelope, err := s.seeds.seal(user.ID, factor, key.Secret())
	if err != nil {
		return nil, err
	}

	p, token, err := s.makePending(CredentialState{UserID: user.ID, Version: user.AuthVersion}, at, FallbackSetup, FallbackMaterial{Method: FallbackTOTP, Factor: FactorBinding{ID: factor, Revision: 1}, KeyID: keyID, Envelope: envelope}, required)
	if err != nil {
		return nil, err
	}

	err = s.queries.CreateFallback(work, p, s.settings)
	if err != nil {
		return nil, err
	}

	return &TOTPSetup{Token: token, URL: key.URL()}, nil
}

// BeginFallbackAuthentication verifies an actual password before capturing one
// owned current factor/set. Method and requirement must be selected by server code.
func (s *FallbackService) BeginFallbackAuthentication(ctx context.Context, email, password string, method FallbackMethod, required AccessRequirement) (*FallbackChallenge, error) {
	return s.begin(ctx, email, password, "", method, required)
}
func (s *FallbackService) BeginFallbackStepUp(ctx context.Context, token, password string, method FallbackMethod, required AccessRequirement) (*FallbackChallenge, error) {
	entry := required
	entry.Proof = RequirePassword
	entry.MaxAge = 0

	actor, err := s.credentials.ValidateSession(ctx, token, entry, NoActivity)
	if err != nil {
		return nil, err
	}

	return s.begin(ctx, actor.User.Email, password, token, method, required)
}
func (s *FallbackService) begin(ctx context.Context, email, password, actorToken string, method FallbackMethod, required AccessRequirement) (*FallbackChallenge, error) {
	if required.Proof == RequirePhishingResistantMFA || method < FallbackTOTP || method > FallbackBackup {
		return nil, ErrSessionProof
	}

	err := required.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	user, at, err := s.verifyPassword(work, email, password)
	if err != nil {
		return nil, err
	}

	state := CredentialState{UserID: user.ID, Version: user.AuthVersion}

	binding, err := s.queries.LoadFallbackFactor(work, state, method)
	if err != nil {
		return nil, err
	}

	purpose := FallbackSignin
	if actorToken != "" {
		purpose = FallbackStepUp
	}

	p, token, err := s.makePending(state, at, purpose, FallbackMaterial{Method: method, Factor: binding}, required)
	if err != nil {
		return nil, err
	}

	if actorToken != "" {
		entry := required
		entry.Proof = RequirePassword
		entry.MaxAge = 0

		actor, err := s.credentials.ValidateSession(work, actorToken, entry, NoActivity)
		if err != nil || actor.User.ID != state.UserID || actor.User.AuthVersion != state.Version {
			return nil, ErrFallback
		}

		p.ActorID = actor.Session.ID
		p.ActorGeneration = actor.Session.Generation

		p.ActorDigest, err = ParseSessionToken(actorToken)
		if err != nil {
			return nil, err
		}
	}

	err = s.queries.CreateFallback(work, p, s.settings)
	if err != nil {
		return nil, err
	}

	return &FallbackChallenge{Token: token}, nil
}
func (s *FallbackService) reserve(ctx context.Context, token, code string, required AccessRequirement) (*FallbackReservation, error) {
	if len(code) == 0 || len(code) > 128 || required.Proof == RequirePhishingResistantMFA {
		return nil, ErrFallback
	}

	digest, purpose, err := ParseFallbackToken(token)
	if err != nil {
		return nil, err
	}

	identifier := ""

	if purpose != FallbackSetup && len(code) != 6 {
		parsed, err := crypto.ParseBackupCode(code)
		if err != nil {
			return nil, err
		}

		identifier = parsed.ID
	}

	return s.queries.ReserveFallback(ctx, digest, purpose, identifier, required, s.settings)
}
func (s *FallbackService) verify(ctx context.Context, r FallbackReservation, code string) (int64, error) {
	p := r.Pending
	if p.Material.Method == FallbackBackup {
		parsed, err := crypto.ParseBackupCode(code)
		if err != nil || parsed.ID != r.Backup.ID || r.Backup.SetID != p.Material.Factor.ID || r.Backup.Revision != p.Material.Factor.Revision {
			return 0, ErrFallback
		}

		input, err := crypto.BackupVerifierInput(p.State.UserID, parsed)
		if err != nil {
			return 0, err
		}

		return 0, s.credentials.verifier.Verify(ctx, r.Backup.Record, input)
	}

	factor := r.Factor
	if p.Purpose == FallbackSetup {
		factor = FallbackFactor{FactorBinding: p.Material.Factor, KeyID: p.Material.KeyID, Envelope: p.Material.Envelope, AcceptedStep: -1}
	}

	secret, err := s.seeds.open(p.State.UserID, factor.ID, factor.KeyID, factor.Envelope)
	if err != nil {
		return 0, err
	}

	step, err := crypto.MatchTOTPCode(secret, code, time.Now().UTC(), s.settings.Skew)
	if err != nil || step <= factor.AcceptedStep {
		return 0, ErrFallback
	}

	return step, nil
}
func (s *FallbackService) acquire() error {
	select {
	case s.slots <- struct{}{}:
		return nil
	default:
		return ErrEnrollmentBusy
	}
}
func (s *FallbackService) ConfirmTOTPSetup(ctx context.Context, token, code string, required AccessRequirement) error {
	_, purpose, err := ParseFallbackToken(token)
	if err != nil || purpose != FallbackSetup {
		return ErrFallback
	}

	err = s.acquire()
	if err != nil {
		return err
	}

	defer func() { <-s.slots }()

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	r, err := s.reserve(work, token, code, required)
	if err != nil {
		return err
	}

	defer func() { _ = s.queries.ReleaseFallback(work, r.Pending.Digest, r.Pending.Revision) }()

	step, err := s.verify(work, *r, code)
	if err != nil {
		return err
	}

	return s.queries.CompleteTOTPSetup(work, *r, step, required, s.settings)
}
func (s *FallbackService) FinishFallback(ctx context.Context, token, code string, required AccessRequirement) (*IssuedSession, error) {
	_, purpose, err := ParseFallbackToken(token)
	if err != nil || purpose == FallbackSetup {
		return nil, ErrFallback
	}

	err = s.acquire()
	if err != nil {
		return nil, err
	}

	defer func() { <-s.slots }()

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	r, err := s.reserve(work, token, code, required)
	if err != nil {
		return nil, err
	}

	defer func() { _ = s.queries.ReleaseFallback(work, r.Pending.Digest, r.Pending.Revision) }()

	step, err := s.verify(work, *r, code)
	if err != nil {
		return nil, err
	}

	p := r.Pending
	now := time.Now().UTC().Truncate(time.Microsecond)

	session := Session{ID: model.NewID(), UserID: p.State.UserID, AuthVersion: p.State.Version, Generation: 1, CreatedAt: p.PasswordAt}
	if purpose == FallbackStepUp {
		if r.Actor == nil || r.Actor.Generation == math.MaxInt64 {
			return nil, ErrFallback
		}

		session = *r.Actor
		session.Generation++
	}

	method := PasswordTOTPProof
	if p.Material.Method == FallbackBackup {
		method = PasswordBackupProof
	}

	session.PolicyRevision = required.Revision
	session.Proof = VerifiedProof{Method: method, VerifiedAt: p.PasswordAt, FactorAt: now, FactorID: p.Material.Factor.ID, FactorRevision: p.Material.Factor.Revision}
	session.AuthenticatedAt = now
	session.LastActivityAt = now
	session.ExpiresAt = now.Add(s.credentials.sessions.TTL)
	session.InactivityTTL = s.credentials.sessions.InactivityTTL

	bearer, digest, err := s.credentials.sessionToken()
	if err != nil {
		return nil, err
	}

	c := FallbackCompletion{Step: step, FactorAt: now, Session: SessionRecord{Session: session, Digest: digest}}

	completed, err := s.queries.CompleteFallback(work, *r, c, required, s.settings, s.credentials.sessions.MaxPerSubject)
	if err != nil {
		return nil, err
	}

	err = work.Err()
	if err != nil {
		return nil, err
	}

	return &IssuedSession{Session: *completed, Token: bearer}, nil
}
func (c FallbackCompletion) Check(r FallbackReservation, now time.Time, required AccessRequirement, settings config.FallbackSettings) error {
	p := r.Pending

	err := p.Check(now, required, settings)
	if err != nil {
		return err
	}

	s := c.Session

	method := PasswordTOTPProof
	if p.Material.Method == FallbackBackup {
		method = PasswordBackupProof
	}

	if p.Purpose == FallbackSetup || p.Attempts < 1 || p.Revision < 1 || !now.Before(p.LeaseUntil) || s.Digest == (SessionDigest{}) || s.UserID != p.State.UserID || s.AuthVersion != p.State.Version || s.Proof.Method != method || s.Proof.FactorID != p.Material.Factor.ID || s.Proof.FactorRevision != p.Material.Factor.Revision || !s.Proof.VerifiedAt.Equal(p.PasswordAt) || !s.Proof.FactorAt.Equal(c.FactorAt) || c.FactorAt.Before(p.LeaseUntil.Add(-settings.Lease)) || c.FactorAt.After(now) {
		return ErrFallback
	}

	if method == PasswordTOTPProof && (c.Step <= r.Factor.AcceptedStep || !crypto.TOTPInWindow(c.Step, now, settings.Skew)) || method == PasswordBackupProof && c.Step != 0 {
		return ErrFallback
	}

	if p.Purpose == FallbackSignin {
		if r.Actor != nil || s.Generation != 1 || !s.CreatedAt.Equal(p.PasswordAt) {
			return ErrFallback
		}
	} else {
		actor := r.Actor
		if actor == nil || actor.ID != p.ActorID || actor.Generation != p.ActorGeneration || actor.UserID != p.State.UserID || s.ID != actor.ID || s.Generation != actor.Generation+1 || !s.CreatedAt.Equal(actor.CreatedAt) || s.Digest == p.ActorDigest || s.AuthenticatedAt.Before(actor.AuthenticatedAt) {
			return ErrFallback
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

// IssueBackupCodes requires current recent management authority. Default policy
// is phishing-resistant MFA; an explicit server policy may allow actual MFA.
// The old set is replaced and the actor rotated atomically; plaintexts return once.
func (s *FallbackService) IssueBackupCodes(ctx context.Context, token string, required AccessRequirement) ([]string, *IssuedSession, error) {
	if required.Proof != RequireMFA && required.Proof != RequirePhishingResistantMFA || required.MaxAge == 0 || required.MaxAge > s.settings.RecentProofAge {
		return nil, nil, ErrSessionProof
	}

	err := required.Check(s.settings.PendingTTL)
	if err != nil {
		return nil, nil, err
	}

	err = s.acquire()
	if err != nil {
		return nil, nil, err
	}

	defer func() { <-s.slots }()

	digest, err := ParseSessionToken(token)
	if err != nil {
		return nil, nil, err
	}

	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	r, err := s.queries.ReserveBackupIssue(work, digest, required, s.settings)
	if err != nil {
		return nil, nil, err
	}

	set := BackupSet{ID: model.NewID()}

	plain := make([]string, 0, s.settings.BackupCodes)
	for range s.settings.BackupCodes {
		code, err := crypto.NewBackupSecret()
		if err != nil {
			return nil, nil, err
		}

		input, err := crypto.BackupVerifierInput(r.State.UserID, code)
		if err != nil {
			return nil, nil, err
		}

		record, err := s.credentials.verifier.Hash(work, input)
		if err != nil {
			return nil, nil, err
		}

		set.Codes = append(set.Codes, BackupVerifier{ID: code.ID, SetID: set.ID, Record: record, Revision: 1})
		plain = append(plain, code.Wire)
	}

	actor := r.Actor
	actor.AuthVersion++
	actor.Generation++

	bearer, newDigest, err := s.credentials.sessionToken()
	if err != nil {
		return nil, nil, err
	}

	completed, err := s.queries.ReplaceBackupSet(work, *r, set, SessionRecord{Session: actor, Digest: newDigest}, required, s.settings)
	if err != nil {
		return nil, nil, err
	}

	err = work.Err()
	if err != nil {
		return nil, nil, err
	}

	return plain, &IssuedSession{Session: *completed, Token: bearer}, nil
}

// CleanupPending removes a finite expired-only batch from the shared collection.
// It never deletes live owned work or refunds the separate durable budget.
func (s *FallbackService) CleanupPending(ctx context.Context) (int64, error) {
	work, cancel := context.WithTimeout(ctx, s.settings.Timeout)
	defer cancel()

	return s.queries.DeleteExpiredEnrollments(work, s.settings.CleanupBatch)
}
