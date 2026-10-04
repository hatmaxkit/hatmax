// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
)

// Closed selectors and server policies reject malformed targets and weak authority.
func TestFactorPolicy(t *testing.T) {
	settings := config.EnrollmentSettings{PendingTTL: 5 * time.Minute, RecentProofAge: 5 * time.Minute}
	valid := FactorPolicy{Management: AccessRequirement{Proof: RequirePhishingResistantMFA, Revision: "server-v1", MaxAge: time.Minute}, Access: RequireMFA}

	tests := []struct {
		name    string
		proof   RequiredProof
		age     time.Duration
		access  RequiredProof
		allowed bool
	}{
		{"default", RequirePhishingResistantMFA, time.Minute, RequirePhishingResistantMFA, true},
		{"explicit MFA", RequireMFA, time.Minute, RequireMFA, true},
		{"password", RequirePassword, time.Minute, RequireMFA, false},
		{"unknown management", 9, time.Minute, RequireMFA, false},
		{"no freshness", RequireMFA, 0, RequireMFA, false},
		{"expired bound", RequireMFA, 6 * time.Minute, RequireMFA, false},
		{"unknown access", RequireMFA, time.Minute, 9, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := valid
			p.Management.Proof = test.proof
			p.Management.MaxAge = test.age

			p.Access = test.access
			if (p.Check(settings) == nil) != test.allowed {
				t.Fatal("policy acceptance mismatch")
			}
		})
	}
}

// The independent continuation cannot resolve as session, initial setup or fallback.
func TestFactorToken(t *testing.T) {
	token, digest, err := factorChangeToken()
	if err != nil || digest == (EnrollmentDigest{}) {
		t.Fatal("missing token")
	}

	parsed, err := parseFactorChangeToken(token)
	if err != nil || parsed != digest {
		t.Fatal("canonical token did not round trip")
	}

	for _, input := range []string{"", token + "=", strings.ToUpper(token), "enroll1." + token[8:], "fallback1." + token[8:]} {
		_, err := parseFactorChangeToken(input)
		if err == nil {
			t.Fatal("noncanonical token accepted")
		}
	}

	_, err = ParseSessionToken(token)
	if err == nil {
		t.Fatal("continuation resolved as session")
	}

	_, err = parseEnrollmentToken(token)
	if err == nil {
		t.Fatal("continuation resolved as initial enrollment")
	}

	_, _, err = ParseFallbackToken(token)
	if err == nil {
		t.Fatal("continuation resolved as fallback")
	}

	encoded, err := json.Marshal(FactorChangeResult{Issued: &IssuedSession{Token: "private-bearer"}})
	if err != nil || strings.Contains(string(encoded), "private-bearer") {
		t.Fatal("bearer exposed as JSON")
	}
}

// Untrusted selector JSON stays finite and cannot turn metadata into authority.
func FuzzFactorSelection(f *testing.F) {
	f.Add(uint8(1), "factor", int64(1))
	f.Add(uint8(2), "", int64(0))
	f.Fuzz(func(t *testing.T, kind uint8, id string, revision int64) {
		selection := FactorSelection{Kind: FactorKind(kind), ID: id, Revision: revision}

		err := selection.Check()
		if err == nil && (kind < 1 || kind > 2 || len(id) == 0 || len(id) > 128 || revision < 1) {
			t.Fatal("invalid selection admitted")
		}

		if err == nil {
			for _, c := range []byte(id) {
				if c < 32 || c > 126 {
					t.Fatal("noncanonical identifier")
				}
			}
		}
	})
}
