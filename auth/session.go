// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"hatmax.adrianpk.com/crypto"
)

var (
	ErrSessionToken    = errors.New("invalid session token")
	ErrSessionRecord   = errors.New("invalid session record")
	ErrSessionActivity = errors.New("invalid session activity")
)

// SessionDigest is a purpose-separated digest of a 32-byte random bearer.
// Storage and metadata never need the reusable bearer value.
type SessionDigest [sha256.Size]byte

// SessionRecord is the owned storage representation; Digest is not public metadata.
type SessionRecord struct {
	Session
	Digest SessionDigest
}

// IssuedSession carries the raw bearer only in the immediate issuance response.
type IssuedSession struct {
	Session
	Token string
}

// ValidatedSession contains owned current user and safe session snapshots.
type ValidatedSession struct {
	User    *User
	Session Session
}

// SessionActivity is a trusted server decision about relevant subscriber activity.
// Background polling must use NoActivity rather than extending inactivity expiry.
type SessionActivity uint8

const (
	NoActivity SessionActivity = iota
	RelevantActivity
)

// ParseSessionToken rejects noncanonical input before deriving a lookup digest.
// The fixed domain keeps a future pending secret in a separate lookup namespace.
func ParseSessionToken(token string) (SessionDigest, error) {
	if len(token) != 44 {
		return SessionDigest{}, ErrSessionToken
	}

	decoded, err := base64.URLEncoding.Strict().DecodeString(token)
	if err != nil || len(decoded) != 32 || base64.URLEncoding.EncodeToString(decoded) != token {
		return SessionDigest{}, ErrSessionToken
	}

	return sha256.Sum256(append([]byte("hatmax/session/v1\x00"), decoded...)), nil
}

func newSessionToken() (string, SessionDigest, error) {
	token, err := crypto.GenerateSecureToken(32)
	if err != nil {
		return "", SessionDigest{}, err
	}

	digest, err := ParseSessionToken(token)
	if err != nil {
		return "", SessionDigest{}, err
	}

	return token, digest, nil
}

// Check validates stored shape and exact expiry using trusted time evaluated
// after storage lock waits. Activity cannot renew absolute expiry or proof time.
func (s Session) Check(now time.Time) error {
	lifetime := s.ExpiresAt.Sub(s.AuthenticatedAt)
	if s.ID == "" || s.UserID == "" || s.AuthVersion < 1 || s.Generation < 1 || s.AuthenticatedAt.IsZero() || s.CreatedAt.IsZero() || s.LastActivityAt.IsZero() || s.CreatedAt.After(s.AuthenticatedAt) || s.AuthenticatedAt.After(s.LastActivityAt) || s.LastActivityAt.After(now) || lifetime < time.Minute || lifetime > 30*24*time.Hour || s.InactivityTTL < time.Minute || s.InactivityTTL > lifetime || s.InactivityTTL%time.Microsecond != 0 {
		return ErrSessionRecord
	}

	if !now.Before(s.ExpiresAt) || !now.Before(s.LastActivityAt.Add(s.InactivityTTL)) {
		return ErrSessionExpired
	}

	return nil
}
