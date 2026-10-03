// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// An independent SHA-256 vector pins both the canonical bearer and its domain.
func TestSessionToken(t *testing.T) {
	const token = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="

	digest, err := ParseSessionToken(token)
	if err != nil || hex.EncodeToString(digest[:]) != "918ece5680f4caa9f52d6bab6c99bf4d443137bf8997686866f091a264dd8d96" {
		t.Fatalf("digest vector: %v", err)
	}

	tests := []struct{ name, value string }{
		{"empty", ""}, {"oversized", strings.Repeat("A", 4096)},
		{"unpadded", strings.TrimSuffix(token, "=")}, {"wrong alphabet", "!" + token[1:]},
		{"padding bits", token[:42] + "9="}, {"line break", token[:20] + "\n" + token[21:]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, parseErr := ParseSessionToken(test.value)
			if !errors.Is(parseErr, ErrSessionToken) || got != (SessionDigest{}) {
				t.Fatal("noncanonical token accepted or result leaked")
			}
		})
	}

	first, firstDigest, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}

	second, secondDigest, err := newSessionToken()
	if err != nil || len(first) != 44 || len(second) != 44 || first == second || firstDigest == secondDigest {
		t.Fatalf("issuance shape/independence: %v", err)
	}
}

// Fixed timestamps establish strict equality and corrupt-record rejection.
func TestSessionExpiry(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	base := Session{ID: "session", UserID: "user", AuthVersion: 1, Generation: 1, AuthenticatedAt: now, CreatedAt: now, LastActivityAt: now, ExpiresAt: now.Add(time.Hour), InactivityTTL: 10 * time.Minute}

	tests := []struct {
		name   string
		offset time.Duration
		change func(*Session)
		want   error
	}{
		{name: "live", offset: 9 * time.Minute},
		{name: "idle equality", offset: 10 * time.Minute, want: ErrSessionExpired},
		{name: "absolute equality", offset: time.Hour, change: func(s *Session) { s.InactivityTTL = time.Hour }, want: ErrSessionExpired},
		{name: "future authentication", change: func(s *Session) { s.AuthenticatedAt = now.Add(time.Second) }, want: ErrSessionRecord},
		{name: "future activity", change: func(s *Session) { s.LastActivityAt = now.Add(time.Second) }, want: ErrSessionRecord},
		{name: "missing subject", change: func(s *Session) { s.UserID = "" }, want: ErrSessionRecord},
		{name: "zero version", change: func(s *Session) { s.AuthVersion = 0 }, want: ErrSessionRecord},
		{name: "zero generation", change: func(s *Session) { s.Generation = 0 }, want: ErrSessionRecord},
		{name: "creation after proof", change: func(s *Session) { s.CreatedAt = now.Add(time.Second) }, want: ErrSessionRecord},
		{name: "short absolute", change: func(s *Session) { s.ExpiresAt = now.Add(59 * time.Second) }, want: ErrSessionRecord},
		{name: "long absolute", change: func(s *Session) { s.ExpiresAt = now.Add(31 * 24 * time.Hour) }, want: ErrSessionRecord},
		{name: "idle exceeds absolute", change: func(s *Session) { s.InactivityTTL = 2 * time.Hour }, want: ErrSessionRecord},
		{name: "fractional storage unit", change: func(s *Session) { s.InactivityTTL += time.Nanosecond }, want: ErrSessionRecord},
		{name: "activity cannot extend absolute", offset: time.Hour, change: func(s *Session) { s.LastActivityAt = now.Add(59 * time.Minute) }, want: ErrSessionExpired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := base
			if test.change != nil {
				test.change(&session)
			}

			err := session.Check(now.Add(test.offset))
			if !errors.Is(err, test.want) {
				t.Fatalf("check: got %v, want %v", err, test.want)
			}
		})
	}
}

// Fuzzing never allocates from caller-controlled lengths or invokes KDF/storage.
func FuzzSessionToken(f *testing.F) {
	f.Add("AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=")
	f.Add("")
	f.Add(strings.Repeat("A", 44))
	f.Fuzz(func(t *testing.T, token string) {
		digest, err := ParseSessionToken(token)
		if err != nil {
			if digest != (SessionDigest{}) {
				t.Fatal("failure returned digest")
			}

			return
		}

		decoded, err := base64.URLEncoding.Strict().DecodeString(token)
		if err != nil || len(token) != 44 || len(decoded) != 32 || base64.URLEncoding.EncodeToString(decoded) != token {
			t.Fatal("accepted noncanonical input")
		}
	})
}

type failedEntropy struct{}

func (failedEntropy) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// The package has no parallel tests; restore the actual source before returning.
// A source read failure must leave both token and digest empty.
func TestSessionEntropy(t *testing.T) {
	previous := rand.Reader

	rand.Reader = failedEntropy{}
	defer func() { rand.Reader = previous }()

	token, digest, err := newSessionToken()
	if !errors.Is(err, io.ErrUnexpectedEOF) || token != "" || digest != (SessionDigest{}) {
		t.Fatal("source failure returned bearer material")
	}
}
