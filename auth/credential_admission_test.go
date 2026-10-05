// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
)

type admissionFake struct {
	calls  int
	charge func(context.Context) error
	count  int64
}

func (f *admissionFake) ChargeCredentialAdmission(ctx context.Context, _ string, _ CredentialAdmissionPurpose, _ config.CredentialAdmissionSettings) error {
	f.calls++
	if f.charge != nil {
		return f.charge(ctx)
	}

	return nil
}
func (f *admissionFake) CleanupCredentialAdmission(context.Context, config.CredentialAdmissionSettings) (int64, error) {
	return f.count, nil
}

// Invalid public structure and canceled callers never reach durable admission.
func TestCredentialAdmission(t *testing.T) {
	tests := []struct {
		name, identity string
		purpose        CredentialAdmissionPurpose
		canceled       bool
		want           error
	}{
		{"canonical", "User@example.com", CredentialPasswordProof, false, nil},
		{"unicode", "用户@example.com", CredentialRegistration, false, nil},
		{"empty", "", CredentialPasswordProof, false, ErrCredentialIdentity},
		{"oversized", strings.Repeat("a", 255), CredentialPasswordProof, false, ErrCredentialIdentity},
		{"invalid encoding", string([]byte{255}), CredentialPasswordProof, false, ErrCredentialIdentity},
		{"control", "a\n@example.com", CredentialPasswordProof, false, ErrCredentialIdentity},
		{"purpose", "a@example.com", 0, false, ErrCredentialIdentity},
		{"canceled", "a@example.com", CredentialPasswordProof, true, context.Canceled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := &admissionFake{}

			s, err := NewCredentialAdmission(store, config.CredentialAdmissionConfig{})
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tc.canceled {
				cancel()
			}

			err = s.Admit(ctx, tc.identity, tc.purpose)
			if !errors.Is(err, tc.want) {
				t.Fatalf("admission: %v", err)
			}

			if tc.want != nil && store.calls != 0 || tc.want == nil && store.calls != 1 {
				t.Fatal("unexpected storage work")
			}
		})
	}
}

// Late adapter success cannot admit work after the original caller deadline.
func TestCredentialAdmissionDeadline(t *testing.T) {
	store := &admissionFake{charge: func(ctx context.Context) error {
		<-ctx.Done()

		return nil
	}}

	s, err := NewCredentialAdmission(store, config.CredentialAdmissionConfig{})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Millisecond)
	defer cancel()

	err = s.Admit(ctx, "a@example.com", CredentialPasswordProof)
	if !errors.Is(err, context.DeadlineExceeded) || store.calls != 1 {
		t.Fatalf("late admission: %v", err)
	}

	store.count = 1001

	_, err = s.Cleanup(t.Context())
	if !errors.Is(err, ErrCredentialAdmissionState) {
		t.Fatal("unbounded cleanup result")
	}
}

// Fuzzing verifies rejected identity structure cannot allocate durable records.
func FuzzCredentialIdentity(f *testing.F) {
	for _, v := range []string{"a@example.com", "用户@example.com", "", "a\x00b", string([]byte{255}), strings.Repeat("a", 255)} {
		f.Add(v)
	}

	f.Fuzz(func(t *testing.T, identity string) {
		store := &admissionFake{}

		s, err := NewCredentialAdmission(store, config.CredentialAdmissionConfig{})
		if err != nil {
			t.Fatal(err)
		}

		invalid := CheckCredentialIdentity(identity) != nil

		err = s.Admit(t.Context(), identity, CredentialPasswordProof)
		if invalid && (!errors.Is(err, ErrCredentialIdentity) || store.calls != 0) || !invalid && (err != nil || store.calls != 1) {
			t.Fatal("invalid identity admission boundary")
		}
	})
}
