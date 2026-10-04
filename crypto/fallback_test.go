// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package crypto

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// A returned integer step lets the durable consumer reject replay atomically.
func TestMatchedStep(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	now := time.Unix(1800000000, 0)

	code, err := totp.GenerateCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}

	step, err := MatchTOTPCode(secret, code, now, 1)
	if err != nil || step != now.Unix()/30 {
		t.Fatalf("matched step %d: %v", step, err)
	}
}

// Backup parsing retains canonical wire bytes and binds verifier input to owner.
func TestBackupSecret(t *testing.T) {
	code, err := NewBackupSecret()
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseBackupCode(code.Wire)
	if err != nil || parsed != code {
		t.Fatalf("parse: %v", err)
	}

	first, err := BackupVerifierInput("first", code)
	if err != nil {
		t.Fatal(err)
	}

	second, err := BackupVerifierInput("second", code)
	if err != nil || first == second {
		t.Fatal("subject not bound")
	}
}
