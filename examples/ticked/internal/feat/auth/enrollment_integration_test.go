//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/fxamacker/cbor/v2"
	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
	"testing"
	"time"
)

func enrollmentRequirement() core.AccessRequirement {
	return core.AccessRequirement{Proof: core.RequirePhishingResistantMFA, Revision: "enroll-v1", MaxAge: 5 * time.Minute}
}

func enrollmentFixture(t *testing.T, options config.AuthenticatorConfig) (*sql.DB, *Queries, *core.Service, *core.AuthenticatorService, *core.EnrollmentChallenge) {
	t.Helper()
	db, q, base, _ := sessionFixture(t)
	options.RPID = "example.com"
	options.RPName = "Example"
	options.Origins = []string{"https://example.com"}
	svc, err := core.NewAuthenticatorService(base, q, options)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}
	return db, q, base, svc, challenge
}

// enrollmentResponse generates a real ES256 credential and an attestation-none
// browser response. It does not bypass the production library verifier.
func enrollmentResponse(t *testing.T, challenge *core.EnrollmentChallenge) []byte {
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

// Real registration and PostgreSQL establish atomic activation, invalidation and
// single-use setup without interpreting enrollment as a sign-in proof.
func TestEnrollmentTransactions(t *testing.T) {
	t.Run("confirmation invalidates prior authority", func(t *testing.T) {
		db, _, _, svc, challenge := enrollmentFixture(t, config.AuthenticatorConfig{})
		response := enrollmentResponse(t, challenge)
		result, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if err != nil || result == nil {
			t.Fatalf("registration: %v", err)
		}
		var factors, sessions, pending, version int
		err = db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM authenticators),(SELECT count(*) FROM sessions),(SELECT count(*) FROM auth_pending),auth_version FROM users WHERE email='session@example.com'").Scan(&factors, &sessions, &pending, &version)
		if err != nil || factors != 1 || sessions != 0 || pending != 0 || version != 2 {
			t.Fatalf("non-atomic registration: %d %d %d %d %v", factors, sessions, pending, version, err)
		}
		replay, err := svc.FinishWebAuthnEnrollment(t.Context(), challenge.Token, response, enrollmentRequirement())
		if err == nil || replay != nil {
			t.Fatal("registration replay succeeded")
		}
		setup, err := svc.BeginWebAuthnEnrollment(t.Context(), "session@example.com", "a distinct safe password", enrollmentRequirement())
		if err == nil || setup != nil {
			t.Fatal("password-only established-factor enrollment succeeded")
		}
	})
}
