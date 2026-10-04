// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"hatmax.adrianpk.com/config"
)

var (
	ErrRecoveryUnavailable = errors.New("recovery unavailable")
	ErrRecoveryAttempts    = errors.New("recovery attempt budget exhausted")
	ErrRecoveryBusy        = errors.New("recovery verification busy")
	ErrRecoveryCapacity    = errors.New("recovery capacity exhausted")
)

// RecoveryPurpose cannot confer session or authenticator-management authority.
type RecoveryPurpose uint8

const (
	VerifyMailbox RecoveryPurpose = 1
	ResetPassword RecoveryPurpose = 2
)

// RecoveryToken owns a transient secret. Formatting and serialization omit it.
type RecoveryToken struct {
	id     string
	secret [32]byte
}

func (t RecoveryToken) String() string               { return "[redacted recovery token]" }
func (t RecoveryToken) GoString() string             { return t.String() }
func (t RecoveryToken) MarshalJSON() ([]byte, error) { return json.Marshal(t.String()) }
func (t RecoveryToken) ID() string                   { return t.id }

// Bearer is only for trusted mail dispatch and direct verification submissions.
func (t RecoveryToken) Bearer() string {
	return t.id + "." + base64.RawURLEncoding.EncodeToString(t.secret[:])
}

// ParseRecoveryToken accepts exactly one UUIDv4/base64url representation.
func ParseRecoveryToken(input string) (RecoveryToken, error) {
	var token RecoveryToken
	if len(input) != 80 || input[36] != '.' {
		return token, ErrRecoveryUnavailable
	}

	id, err := uuid.Parse(input[:36])
	if err != nil || id.Version() != 4 || id.Variant() != uuid.RFC4122 || id.String() != input[:36] {
		return token, ErrRecoveryUnavailable
	}

	secret, err := base64.RawURLEncoding.Strict().DecodeString(input[37:])
	if err != nil || len(secret) != 32 || base64.RawURLEncoding.EncodeToString(secret) != input[37:] {
		return token, ErrRecoveryUnavailable
	}

	token.id = id.String()
	copy(token.secret[:], secret)
	clear(secret)

	return token, nil
}

func newRecoveryToken() (RecoveryToken, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return RecoveryToken{}, err
	}

	token := RecoveryToken{id: id.String()}
	_, err = rand.Read(token.secret[:])

	return token, err
}

// RecoveryDigest separates token purposes, subjects and record identities.
func RecoveryDigest(token RecoveryToken, subject string, purpose RecoveryPurpose) [32]byte {
	digest := sha256.New()
	digest.Write([]byte("hatmax:account-token:v1\x00"))
	digest.Write([]byte{byte(purpose)})
	digest.Write([]byte(subject))
	digest.Write([]byte{0})
	digest.Write([]byte(token.id))
	digest.Write([]byte{0})
	digest.Write(token.secret[:])

	var result [32]byte
	copy(result[:], digest.Sum(nil))

	return result
}

// MailboxRecord is an owned snapshot; it never contains a raw bearer.
type MailboxRecord struct {
	ID                                string
	State                             CredentialState
	Purpose                           RecoveryPurpose
	Target, PolicyRevision            string
	Digest                            [32]byte `json:"-"`
	CreatedAt, ExpiresAt              time.Time
	Attempts, AttemptLimit            int
	Revision                          int64
	LeaseUntil, ConsumedAt, RevokedAt time.Time
}

// Check validates stored shape and current eligibility; attempts are checked at admission.
func (p MailboxRecord) Check(now time.Time, revision string, settings config.RecoverySettings) error {
	id, err := uuid.Parse(p.ID)
	if err != nil || id.Version() != 4 || id.Variant() != uuid.RFC4122 || id.String() != p.ID || !boundedID(p.State.UserID, 128) || p.State.Version < 1 || p.State.Version == math.MaxInt64 || p.Purpose != VerifyMailbox || len(p.Target) == 0 || len(p.Target) > 254 || strings.ContainsAny(p.Target, "\x00\r\n") || !validRecoveryRevision(revision) || p.PolicyRevision != revision || p.Digest == ([32]byte{}) || p.Revision < 1 || p.AttemptLimit != settings.TokenAttempts || p.Attempts < 0 || p.Attempts > p.AttemptLimit || p.CreatedAt.After(now) || p.ExpiresAt.Sub(p.CreatedAt) != settings.VerificationTTL || !now.Before(p.ExpiresAt) || !p.ConsumedAt.IsZero() || !p.RevokedAt.IsZero() {
		return ErrRecoveryUnavailable
	}

	return nil
}

func validRecoveryRevision(revision string) bool {
	if len(revision) == 0 || len(revision) > 128 {
		return false
	}

	for _, b := range []byte(revision) {
		if b < 33 || b > 126 {
			return false
		}
	}

	return true
}

// Matches checks the actual token's ID and digest in constant time.
func (t RecoveryToken) Matches(p MailboxRecord) bool {
	digest := RecoveryDigest(t, p.State.UserID, p.Purpose)

	return t.id == p.ID && subtle.ConstantTimeCompare(digest[:], p.Digest[:]) == 1
}

// MailboxIssue is a transient trusted-dispatch result, not a public response.
type MailboxIssue struct {
	Token     RecoveryToken `json:"-"`
	Target    string        `json:"-"`
	ExpiresAt time.Time     `json:"expiresAt"`
}

func (p MailboxIssue) String() string   { return "[redacted mailbox dispatch]" }
func (p MailboxIssue) GoString() string { return p.String() }

// MailboxVerification reports committed ownership metadata without authentication.
type MailboxVerification struct {
	Subject    string    `json:"subject"`
	VerifiedAt time.Time `json:"verifiedAt"`
}
