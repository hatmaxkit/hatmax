//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	core "hatmax.adrianpk.com/auth"
	"hatmax.adrianpk.com/config"
)

func webAuthnRequirement() core.AccessRequirement {
	r := PasswordRequirement()
	r.Proof = core.RequirePhishingResistantMFA
	r.MaxAge = 5 * time.Minute

	return r
}

type webAuthnFixture struct {
	db     *sql.DB
	q      *Queries
	base   *core.Service
	svc    *core.WebAuthnService
	key    *ecdsa.PrivateKey
	id     []byte
	factor *core.Authenticator
}

func newWebAuthnFixture(t *testing.T, options config.AuthenticatorConfig) webAuthnFixture {
	t.Helper()
	db, q, base, enrollment, challenge := enrollmentFixture(t, options)
	body, key, id := enrollmentKeyResponse(t, challenge)

	factor, err := enrollment.FinishWebAuthnEnrollment(t.Context(), challenge.Token, body, enrollmentRequirement())
	if err != nil {
		t.Fatal(err)
	}

	options.RPID = "example.com"
	options.RPName = "Example"
	options.Origins = []string{"https://example.com"}

	svc, err := core.NewWebAuthnService(base, q, options)
	if err != nil {
		t.Fatal(err)
	}

	return webAuthnFixture{db: db, q: q, base: base, svc: svc, key: key, id: id, factor: factor}
}

// signedAssertion uses a real ES256 key enrolled by the production verifier.
// Mutations are made before signing to test semantic checks independently.
func signedAssertion(t *testing.T, f webAuthnFixture, challenge *core.AssertionChallenge, counter uint32, flags byte, change func(map[string]any, []byte)) []byte {
	t.Helper()

	rp := sha256.Sum256([]byte("example.com"))
	data := append(rp[:], flags, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(data[33:], counter)

	client := map[string]any{"type": "webauthn.get", "challenge": base64.RawURLEncoding.EncodeToString(challenge.Options.Response.Challenge), "origin": "https://example.com", "crossOrigin": false}
	if change != nil {
		change(client, data)
	}

	encoded, err := json.Marshal(client)
	if err != nil {
		t.Fatal(err)
	}

	clientHash := sha256.Sum256(encoded)
	signed := sha256.Sum256(append(append([]byte(nil), data...), clientHash[:]...))

	signature, err := ecdsa.SignASN1(rand.Reader, f.key, signed[:])
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]any{"id": base64.RawURLEncoding.EncodeToString(f.id), "rawId": base64.RawURLEncoding.EncodeToString(f.id), "type": "public-key", "response": map[string]any{"clientDataJSON": base64.RawURLEncoding.EncodeToString(encoded), "authenticatorData": base64.RawURLEncoding.EncodeToString(data), "signature": base64.RawURLEncoding.EncodeToString(signature)}, "clientExtensionResults": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}

	return body
}
func beginAssertion(t *testing.T, f webAuthnFixture) *core.AssertionChallenge {
	t.Helper()

	challenge, err := f.svc.BeginWebAuthnAuthentication(t.Context(), "session@example.com", webAuthnRequirement())
	if err != nil {
		t.Fatal(err)
	}

	return challenge
}
func finishAssertion(t *testing.T, f webAuthnFixture, challenge *core.AssertionChallenge, counter uint32) *core.IssuedSession {
	t.Helper()

	issued, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, signedAssertion(t, f, challenge, counter, 5, nil), webAuthnRequirement())
	if err != nil || issued == nil {
		t.Fatalf("assertion completion: %v", err)
	}

	return issued
}

// Real signatures and PostgreSQL establish completed proof and bound rotation.
func TestWebAuthnTransactions(t *testing.T) {
	t.Run("signed authentication", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})
		challenge := beginAssertion(t, f)

		issued := finishAssertion(t, f, challenge, 1)
		if issued.Proof.Method != core.WebAuthnProof || issued.Proof.FactorID != f.factor.ID || issued.Proof.FactorRevision != 1 {
			t.Fatal("missing actual constituent")
		}

		for _, method := range []core.RequiredProof{core.RequirePassword, core.RequireMFA, core.RequirePhishingResistantMFA} {
			requirement := webAuthnRequirement()
			requirement.Proof = method

			current, err := f.base.ValidateSession(t.Context(), issued.Token, requirement, core.NoActivity)
			if err != nil || current == nil {
				t.Fatalf("proof %d: %v", method, err)
			}
		}

		replay, err := f.svc.FinishWebAuthn(t.Context(), challenge.Token, signedAssertion(t, f, challenge, 2, 5, nil), webAuthnRequirement())
		if err == nil || replay != nil {
			t.Fatal("pending replay issued access")
		}

		finishAssertion(t, f, beginAssertion(t, f), 2)

		_, err = f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.NoActivity)
		if err != nil {
			t.Fatalf("routine counter update invalidated prior proof: %v", err)
		}
	})
	t.Run("password actor step-up", func(t *testing.T) {
		f := newWebAuthnFixture(t, config.AuthenticatorConfig{})

		result, err := f.base.Signin(t.Context(), "session@example.com", "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		old, _ := result.CompletedSession()

		_, err = f.base.ValidateSession(t.Context(), old.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("password satisfies stronger route")
		}

		challenge, err := f.svc.BeginWebAuthnStepUp(t.Context(), old.Token, webAuthnRequirement())
		if err != nil {
			t.Fatal(err)
		}

		issued := finishAssertion(t, f, challenge, 1)
		if issued.ID != old.ID || issued.Generation != old.Generation+1 || issued.Token == old.Token || !issued.CreatedAt.Equal(old.CreatedAt) {
			t.Fatal("step-up did not rotate actor")
		}

		_, err = f.base.ValidateSession(t.Context(), old.Token, PasswordRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("old actor bearer survived")
		}

		_, err = f.base.ValidateSession(t.Context(), issued.Token, webAuthnRequirement(), core.NoActivity)
		if err != nil {
			t.Fatal(err)
		}

		weaker, err := f.base.Reauthenticate(t.Context(), issued.Token, "a distinct safe password", PasswordRequirement())
		if err != nil {
			t.Fatal(err)
		}

		pwd, _ := weaker.CompletedSession()
		if pwd.Proof.Method != core.PasswordProof || pwd.Proof.FactorID != "" {
			t.Fatal("password refreshed WebAuthn proof")
		}

		_, err = f.base.ValidateSession(t.Context(), pwd.Token, webAuthnRequirement(), core.NoActivity)
		if err == nil {
			t.Fatal("password reauth issued stronger access")
		}
	})
}
