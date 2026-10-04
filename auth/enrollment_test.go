// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
)

type enrollmentMemory struct {
	*mockQueries
	mu        sync.Mutex
	pending   map[EnrollmentDigest]EnrollmentPending
	handle    []byte
	confirmed bool
	reserved  int
}

func (q *enrollmentMemory) EnsureWebAuthnHandle(_ context.Context, _ CredentialState, _ string, handle []byte) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.handle == nil {
		q.handle = append([]byte(nil), handle...)
	}

	return append([]byte(nil), q.handle...), nil
}
func (q *enrollmentMemory) CreateEnrollment(_ context.Context, p EnrollmentPending, s config.EnrollmentSettings) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.confirmed {
		return ErrEnrollment
	}

	if len(q.pending) >= s.MaxPending {
		return ErrEnrollmentCapacity
	}

	q.pending[p.Digest] = p

	return nil
}
func (q *enrollmentMemory) ReserveEnrollment(_ context.Context, d EnrollmentDigest, r AccessRequirement, s config.EnrollmentSettings) (*EnrollmentPending, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	p, ok := q.pending[d]
	if !ok {
		return nil, ErrEnrollment
	}

	err := p.Check(time.Now(), r, s)
	if err != nil {
		return nil, err
	}

	if time.Now().Before(p.LeaseUntil) {
		return nil, ErrEnrollmentBusy
	}

	if p.Attempts >= s.PendingAttempts {
		return nil, ErrEnrollmentAttempts
	}

	p.Revision++
	p.Attempts++
	p.LeaseUntil = time.Now().UTC().Truncate(time.Microsecond).Add(s.Lease)
	q.reserved++
	q.pending[d] = p

	return &p, nil
}
func (q *enrollmentMemory) ReleaseEnrollment(_ context.Context, d EnrollmentDigest, revision int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	p, ok := q.pending[d]
	if ok && p.Revision == revision {
		p.LeaseUntil = time.Time{}
		q.pending[d] = p
	}

	return nil
}
func (q *enrollmentMemory) ConfirmEnrollment(_ context.Context, p EnrollmentPending, r RegistrationRecord, _ AccessRequirement, _ config.EnrollmentSettings) (*Authenticator, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.confirmed || q.pending[p.Digest].Revision != p.Revision {
		return nil, ErrEnrollment
	}

	q.confirmed = true
	q.pending = make(map[EnrollmentDigest]EnrollmentPending)

	return &r.Authenticator, nil
}
func (q *enrollmentMemory) DeleteExpiredEnrollments(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

func enrollmentFixture(t *testing.T) (*AuthenticatorService, *enrollmentMemory, *EnrollmentChallenge) {
	t.Helper()

	q := &enrollmentMemory{mockQueries: newMockQueries(), pending: make(map[EnrollmentDigest]EnrollmentPending)}
	base := newServiceForTest(t, q.mockQueries, config.New(), log.NewTestLogger("error"))

	_, err := base.Signup(t.Context(), "enrollment@example.com", "a distinct safe password")
	if err != nil {
		t.Fatal(err)
	}

	svc, err := NewAuthenticatorService(base, q, config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}})
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := svc.BeginWebAuthnEnrollment(t.Context(), "enrollment@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return svc, q, challenge
}
func enrollmentRequirement() AccessRequirement {
	return AccessRequirement{Proof: RequirePhishingResistantMFA, Revision: "enroll-v1", MaxAge: 5 * time.Minute}
}

// registrationResponse generates a real ES256 credential and an attestation-none
// browser response. It does not bypass the production library verifier.
func registrationResponse(t *testing.T, challenge *EnrollmentChallenge) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	cose, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: key.X.FillBytes(make([]byte, 32)), -3: key.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}

	id := make([]byte, 32)

	_, err = rand.Read(id)
	if err != nil {
		t.Fatal(err)
	}

	rp := sha256.Sum256([]byte("example.com"))
	authData := append(rp[:], 0x45, 0, 0, 0, 0)
	authData = append(authData, make([]byte, 16)...)
	authData = append(authData, 0, 32)
	authData = append(authData, id...)
	authData = append(authData, cose...)

	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		t.Fatal(err)
	}

	client, err := json.Marshal(map[string]any{"type": "webauthn.create", "challenge": base64.RawURLEncoding.EncodeToString(challenge.Options.Response.Challenge), "origin": "https://example.com", "crossOrigin": false})
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{"id": base64.RawURLEncoding.EncodeToString(id), "rawId": base64.RawURLEncoding.EncodeToString(id), "type": "public-key", "response": map[string]any{"clientDataJSON": base64.RawURLEncoding.EncodeToString(client), "attestationObject": base64.RawURLEncoding.EncodeToString(attestation)}, "clientExtensionResults": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}

	return body
}

// Registration confirms a real library-verified key without producing a session
// or allowing the restricted bearer to enter ordinary session validation.
func TestEnrollment(t *testing.T) {
	svc, q, challenge := enrollmentFixture(t)

	_, err := ParseSessionToken(challenge.Token)
	if !errors.Is(err, ErrSessionToken) {
		t.Fatal("enrollment token parsed as session")
	}

	result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, registrationResponse(t, challenge), enrollmentRequirement())
	if err != nil || result == nil || result.ID == "" || q.reserved != 1 || !q.confirmed || len(q.sessions) != 0 {
		t.Fatalf("registration failed or issued a session: %v", err)
	}
}
