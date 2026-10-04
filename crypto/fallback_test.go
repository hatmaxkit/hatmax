// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package crypto

import (
	"strings"
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

// Step equality, adjacent periods and code shape are decided by supplied time.
func TestTOTPWindow(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
	now := time.Unix(1800000000, 0)

	tests := []struct {
		name   string
		offset int64
		skew   uint
		valid  bool
	}{
		{"current", 0, 0, true}, {"previous", -1, 1, true}, {"next", 1, 1, true},
		{"previous strict", -1, 0, false}, {"next strict", 1, 0, false},
		{"too old", -2, 1, false}, {"too new", 2, 1, false}, {"excessive skew", 0, 2, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			at := now.Add(time.Duration(test.offset) * 30 * time.Second)

			code, err := totp.GenerateCode(secret, at)
			if err != nil {
				t.Fatal(err)
			}

			step, err := MatchTOTPCode(secret, code, now, test.skew)
			if (err == nil) != test.valid || test.valid && step != at.Unix()/30 {
				t.Fatalf("step %d: %v", step, err)
			}
		})
	}

	code, err := totp.GenerateCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}

	step := now.Unix() / 30

	testsTime := []struct {
		name  string
		at    time.Time
		skew  uint
		valid bool
	}{
		{"end of same period", now.Add(30*time.Second - time.Nanosecond), 0, true},
		{"next period equality", now.Add(30 * time.Second), 0, false},
		{"skew last instant", now.Add(60*time.Second - time.Nanosecond), 1, true},
		{"skew expiry equality", now.Add(60 * time.Second), 1, false},
	}
	for _, test := range testsTime {
		t.Run(test.name, func(t *testing.T) {
			if TOTPInWindow(step, test.at, test.skew) != test.valid {
				t.Fatal("wrong commit window")
			}
		})
	}

	for _, bad := range []string{code + " ", " " + code, code[:5], "12345a", "１２３４５６"} {
		_, err = MatchTOTPCode(secret, bad, now, 1)
		if err == nil {
			t.Fatal("noncanonical OTP accepted")
		}
	}
}

// Canonical parsing rejects aliases before lookup/KDF and never normalizes secrets.
func TestBackupCodeShape(t *testing.T) {
	code, err := NewBackupSecret()
	if err != nil {
		t.Fatal(err)
	}

	for _, bad := range []string{"", code.Wire + " ", " " + code.Wire, code.Wire + "=", strings.ToUpper(code.Wire), strings.Replace(code.Wire, "backup1.", "backup2.", 1), code.Wire[:len(code.Wire)-1]} {
		_, err = ParseBackupCode(bad)
		if err == nil {
			t.Fatalf("noncanonical backup accepted: length %d", len(bad))
		}
	}

	other, err := NewBackupSecret()
	if err != nil {
		t.Fatal(err)
	}

	one, err := BackupVerifierInput("subject", code)
	if err != nil {
		t.Fatal(err)
	}

	two, err := BackupVerifierInput("subject", other)
	if err != nil || one == two {
		t.Fatal("code ID/material not bound")
	}
}

// The parser target has fixed small allocations and invokes neither KDF nor DB.
func FuzzBackupCode(f *testing.F) {
	f.Add("backup1.00000000-0000-0000-0000-000000000001.AAAAAAAAAAAAAAAAAAAAAA")
	f.Add("")
	f.Add("backup1.invalid.123456")
	f.Fuzz(func(t *testing.T, wire string) {
		code, err := ParseBackupCode(wire)
		if err == nil && (len(wire) != 67 || code.Wire != wire || code.ID == "" || len(code.Secret) != 22) {
			t.Fatal("accepted noncanonical shape")
		}
	})
}
