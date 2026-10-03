// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package model

import (
	"context"
	"errors"
	"testing"
)

func newTestPasswordVerifier(t *testing.T) *PasswordVerifier {
	t.Helper()

	verifier, err := NewPasswordVerifier(PasswordVerifierConfig{MemoryKiB: 19 * 1024, Iterations: 2, Parallelism: 1})
	if err != nil {
		t.Fatal(err)
	}

	return verifier
}

// A record from the reference libargon2 implementation proves PHC interoperability
// and that verification uses the stored work parameters, without trimming input.
func TestPasswordVerifierReference(t *testing.T) {
	v, err := NewPasswordVerifier(PasswordVerifierConfig{})
	if err != nil {
		t.Fatal(err)
	}

	record := "$argon2id$v=19$m=19456,t=2,p=1$MDEyMzQ1Njc4OWFiY2RlZg$U/rRdJ+r8eQh4ahASAsF3FI6+cldlUPcE+LNiGPEUwM"

	for _, tc := range []struct {
		name     string
		password string
		want     error
	}{
		{name: "reference match", password: " independent test password "},
		{name: "trimmed mismatch", password: "independent test password", want: ErrPasswordMismatch},
		{name: "case mismatch", password: " Independent test password ", want: ErrPasswordMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := v.Verify(context.Background(), record, tc.password)
			if !errors.Is(err, tc.want) {
				t.Fatalf("verification = %v; want %v", err, tc.want)
			}
		})
	}
}

// Hashing and verification share NFC processing. Repeated creation must produce
// independent salts and records, without relying on a legacy reader.
func TestPasswordVerifierRoundTrip(t *testing.T) {
	v := newTestPasswordVerifier(t)
	ctx := context.Background()

	first, err := v.Hash(ctx, "  cafe\u0301 password  ")
	if err != nil {
		t.Fatal(err)
	}

	second, err := v.Hash(ctx, "  café password  ")
	if err != nil {
		t.Fatal(err)
	}

	a, err := parsePasswordRecord(first, v.config)
	if err != nil {
		t.Fatal(err)
	}

	b, err := parsePasswordRecord(second, v.config)
	if err != nil {
		t.Fatal(err)
	}

	if string(a.salt) == string(b.salt) || first == second {
		t.Fatal("credential creation reused a salt")
	}

	for _, record := range []string{first, second} {
		err = v.Verify(ctx, record, "  café password  ")
		if err != nil {
			t.Fatal(err)
		}
	}
}
