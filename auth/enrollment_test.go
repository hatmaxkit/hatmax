// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/protocol"
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
func registrationResponse(t testing.TB, challenge *EnrollmentChallenge) []byte {
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

func alterRegistration(t *testing.T, body []byte, client func(map[string]any), authData func([]byte) []byte) []byte {
	t.Helper()

	var wire map[string]any

	err := json.Unmarshal(body, &wire)
	if err != nil {
		t.Fatal(err)
	}

	response := wire["response"].(map[string]any)
	if client != nil {
		decoded, err := base64.RawURLEncoding.DecodeString(response["clientDataJSON"].(string))
		if err != nil {
			t.Fatal(err)
		}

		var data map[string]any

		err = json.Unmarshal(decoded, &data)
		if err != nil {
			t.Fatal(err)
		}

		client(data)

		encoded, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}

		response["clientDataJSON"] = base64.RawURLEncoding.EncodeToString(encoded)
	}

	if authData != nil {
		decoded, err := base64.RawURLEncoding.DecodeString(response["attestationObject"].(string))
		if err != nil {
			t.Fatal(err)
		}

		var attestation map[string]any

		err = cbor.Unmarshal(decoded, &attestation)
		if err != nil {
			t.Fatal(err)
		}

		attestation["authData"] = authData(attestation["authData"].([]byte))

		encoded, err := cbor.Marshal(attestation)
		if err != nil {
			t.Fatal(err)
		}

		response["attestationObject"] = base64.RawURLEncoding.EncodeToString(encoded)
	}

	encoded, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}

func alterCOSE(body []byte, changes map[int]any) []byte {
	var key map[int]any

	err := cbor.Unmarshal(body[87:], &key)
	if err != nil {
		panic(err)
	}

	for id, value := range changes {
		key[id] = value
	}

	encoded, err := cbor.Marshal(key)
	if err != nil {
		panic(err)
	}

	return append(body[:87], encoded...)
}

// Invalid browser evidence reaches the real verifier and cannot activate a key.
// Admitted malformed/protocol failures spend attempts before verification.
func TestEnrollmentVerification(t *testing.T) {
	tests := []struct {
		name   string
		client func(map[string]any)
		data   func([]byte) []byte
	}{
		{name: "origin", client: func(m map[string]any) { m["origin"] = "https://evil.example.com" }},
		{name: "exact origin", client: func(m map[string]any) { m["origin"] = "https://EXAMPLE.COM" }},
		{name: "challenge", client: func(m map[string]any) { m["challenge"] = "a-wrong-challenge" }},
		{name: "type", client: func(m map[string]any) { m["type"] = "webauthn.get" }},
		{name: "cross origin", client: func(m map[string]any) { m["crossOrigin"] = true }},
		{name: "embedded", client: func(m map[string]any) { m["topOrigin"] = "https://example.com" }},
		{name: "RP", data: func(b []byte) []byte {
			b[0] ^= 1

			return b
		}},
		{name: "presence", data: func(b []byte) []byte {
			b[32] &^= 1

			return b
		}},
		{name: "verification", data: func(b []byte) []byte {
			b[32] &^= 4

			return b
		}},
		{name: "backup flags", data: func(b []byte) []byte {
			b[32] |= 16

			return b
		}},
		{name: "credential ID", data: func(b []byte) []byte {
			b[55] ^= 1

			return b
		}},
		{name: "algorithm", data: func(b []byte) []byte {
			var key map[int]any

			err := cbor.Unmarshal(b[87:], &key)
			if err != nil {
				panic(err)
			}

			key[3] = -35

			encoded, err := cbor.Marshal(key)
			if err != nil {
				panic(err)
			}

			return append(b[:87], encoded...)
		}},
		{name: "curve", data: func(b []byte) []byte { return alterCOSE(b, map[int]any{-1: 2}) }},
		{name: "invalid point", data: func(b []byte) []byte { return alterCOSE(b, map[int]any{-2: make([]byte, 32), -3: make([]byte, 32)}) }},
		{name: "COSE nesting", data: func(b []byte) []byte { return append(b[:87], append(bytes.Repeat([]byte{0x81}, 17), 0)...) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, q, challenge := enrollmentFixture(t)
			body := alterRegistration(t, registrationResponse(t, challenge), test.client, test.data)

			result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, body, enrollmentRequirement())
			if !errors.Is(err, ErrEnrollment) || result != nil || q.confirmed || q.reserved != 1 {
				t.Fatalf("invalid proof admitted: %v", err)
			}
		})
	}

	t.Run("captured subject", func(t *testing.T) {
		svc, q, challenge := enrollmentFixture(t)

		digest, err := parseEnrollmentToken(challenge.Token)
		if err != nil {
			t.Fatal(err)
		}

		p := q.pending[digest]
		p.Handle = bytes.Repeat([]byte{99}, 32)
		q.pending[digest] = p

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, registrationResponse(t, challenge), enrollmentRequirement())
		if !errors.Is(err, ErrEnrollment) || result != nil || q.confirmed || q.reserved != 1 {
			t.Fatalf("foreign handle admitted: %v", err)
		}
	})
	t.Run("backup eligible", func(t *testing.T) {
		svc, _, challenge := enrollmentFixture(t)
		body := alterRegistration(t, registrationResponse(t, challenge), nil, func(b []byte) []byte {
			b[32] |= 8 | 16

			return b
		})

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, body, enrollmentRequirement())
		if err != nil || result == nil || !result.BackupEligible || !result.BackupState {
			t.Fatalf("valid backup flags denied: %v", err)
		}
	})
}

// Non-admitted oversized input, canceled work and policy mismatch cannot consume
// protocol capacity or produce restricted authority from a wrong password.
func TestEnrollmentBounds(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		requirement AccessRequirement
		want        error
		attempts    int
	}{
		{"body bound", bytes.Repeat([]byte{' '}, MaxEnrollmentBody+1), enrollmentRequirement(), ErrEnrollment, 0},
		{"malformed", []byte("{broken"), enrollmentRequirement(), ErrEnrollment, 1},
		{"nesting", []byte(strings.Repeat("[", 17) + "0" + strings.Repeat("]", 17)), enrollmentRequirement(), ErrEnrollment, 1},
		{"policy", []byte("{}"), AccessRequirement{Proof: RequirePassword, Revision: "changed-v2"}, ErrEnrollment, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, q, challenge := enrollmentFixture(t)

			result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, test.body, test.requirement)
			if result != nil || !errors.Is(err, test.want) || q.reserved != test.attempts {
				t.Fatalf("bounds failed: %v attempts %d", err, q.reserved)
			}
		})
	}

	t.Run("admission", func(t *testing.T) {
		svc, q, challenge := enrollmentFixture(t)
		for range cap(svc.slots) {
			svc.slots <- struct{}{}
		}

		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, []byte("{}"), enrollmentRequirement())
		if result != nil || !errors.Is(err, ErrEnrollmentBusy) || q.reserved != 0 {
			t.Fatalf("queued protocol work: %v", err)
		}
	})
	t.Run("actual password required", func(t *testing.T) {
		svc, q, _ := enrollmentFixture(t)

		result, err := svc.BeginWebAuthnEnrollment(t.Context(), "enrollment@example.com", "a wrong safe password", enrollmentRequirement())
		if result != nil || !errors.Is(err, ErrInvalidPassword) || len(q.pending) != 1 {
			t.Fatalf("false password created pending: %v", err)
		}
	})
	t.Run("independent secret namespaces", func(t *testing.T) {
		token, digest, err := enrollmentToken()
		if err != nil {
			t.Fatal(err)
		}

		sameSecret := base64.URLEncoding.EncodeToString(mustDecodeRaw(t, token[8:]))

		session, err := ParseSessionToken(sameSecret)
		if err != nil || EnrollmentDigest(session) == digest {
			t.Fatal("session and enrollment digest domains collide")
		}

		_, err = parseEnrollmentToken(sameSecret)
		if !errors.Is(err, ErrEnrollment) {
			t.Fatal("session bearer parsed as enrollment")
		}
	})
}
func mustDecodeRaw(t *testing.T, s string) []byte {
	t.Helper()

	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}

	return decoded
}

// Fuzzing stays within the parser boundary: no KDF, database or completion work.
func FuzzEnrollmentResponse(f *testing.F) {
	f.Add([]byte("{}"))
	f.Add([]byte("{broken"))

	challenge := &EnrollmentChallenge{Options: &protocol.CredentialCreation{Response: protocol.PublicKeyCredentialCreationOptions{Challenge: []byte("a bounded challenge for parser fuzz")}}}
	f.Add(registrationResponse(f, challenge))
	f.Fuzz(func(_ *testing.T, body []byte) { _, _ = parseEnrollmentResponse(body) })
}

// Exact expiry/freshness and configuration/subject bindings cannot be weakened
// by a valid protocol response or by reusing stored ceremony data elsewhere.
func TestEnrollmentSnapshot(t *testing.T) {
	svc, q, challenge := enrollmentFixture(t)

	digest, err := parseEnrollmentToken(challenge.Token)
	if err != nil {
		t.Fatal(err)
	}

	original := q.pending[digest]
	settings := svc.settings
	settings.RecentProofAge = time.Minute

	tests := []struct {
		name   string
		change func(*EnrollmentPending)
		now    time.Time
	}{
		{"expiry equality", func(*EnrollmentPending) {}, original.ExpiresAt},
		{"proof equality", func(*EnrollmentPending) {}, original.PasswordAt.Add(settings.RecentProofAge)},
		{"future password", func(p *EnrollmentPending) { p.PasswordAt = p.CreatedAt.Add(time.Second) }, original.CreatedAt},
		{"version", func(p *EnrollmentPending) { p.State.Version = 0 }, original.CreatedAt},
		{"RP binding", func(p *EnrollmentPending) { p.RPBinding[0] ^= 1 }, original.CreatedAt},
		{"handle length", func(p *EnrollmentPending) { p.Handle = nil }, original.CreatedAt},
		{"ceremony bound", func(p *EnrollmentPending) { p.Ceremony = make([]byte, 16385) }, original.CreatedAt},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pending := original
			test.change(&pending)

			err := pending.Check(test.now, enrollmentRequirement(), settings)
			if !errors.Is(err, ErrEnrollment) {
				t.Fatal("invalid captured authority accepted")
			}
		})
	}
}

// Service construction rejects missing/uninitialized credential dependencies,
// and later caller mutations cannot rewrite the service's accepted origin set.
func TestEnrollmentConstruction(t *testing.T) {
	q := &enrollmentMemory{mockQueries: newMockQueries(), pending: make(map[EnrollmentDigest]EnrollmentPending)}
	base := newServiceForTest(t, q.mockQueries, config.New(), log.NewTestLogger("error"))
	cfg := config.AuthenticatorConfig{RPID: "example.com", RPName: "Example", Origins: []string{"https://example.com"}}

	tests := []struct {
		name    string
		base    *Service
		queries AuthenticatorQueries
	}{
		{"missing credentials", nil, q},
		{"uninitialized credentials", &Service{}, q},
		{"missing storage", base, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := NewAuthenticatorService(test.base, test.queries, cfg)
			if err == nil || result != nil {
				t.Fatal("invalid mandatory dependencies accepted")
			}
		})
	}

	svc, err := NewAuthenticatorService(base, q, cfg)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Origins[0] = "https://evil.example.com"
	if svc.settings.Origins[0] != "https://example.com" {
		t.Fatal("caller changed service's trusted origins")
	}
}
