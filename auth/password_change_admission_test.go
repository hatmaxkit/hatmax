// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

type changeAdmissionStore struct {
	RecoveryQueries
	actor      Session
	admissions atomic.Int32
	commits    atomic.Int32
}

func (s *changeAdmissionStore) AuthorizePasswordChange(_ context.Context, d SessionDigest, p PasswordChangePolicy, _ config.RecoverySettings) (*PasswordChangeAuthorization, error) {
	s.admissions.Add(1)

	return &PasswordChangeAuthorization{Actor: s.actor, ActorDigest: d, Policy: p}, nil
}
func (s *changeAdmissionStore) CommitPasswordChange(_ context.Context, a PasswordChangeAuthorization, _ string, _ PasswordChangePolicy, _ config.RecoverySettings) (*PasswordChanged, error) {
	s.commits.Add(1)

	return &PasswordChanged{Subject: a.Actor.UserID, ChangedAt: time.Now().UTC()}, nil
}

// A checker barrier releases admitted requests together into the actual single
// shared KDF slot. Busy work spends authorization and cannot reach persistence.
func TestPasswordChangeAdmission(t *testing.T) {
	const workers = 8

	entered := make(chan struct{}, workers)
	release := make(chan struct{})
	cfg := config.New()
	cfg.Auth.ArgonMemoryKiB = 19456
	cfg.Auth.ArgonIterations = 2
	cfg.Auth.ArgonParallelism = 1
	cfg.Auth.PasswordMaxConcurrent = 1

	base, err := NewService(newMockQueries(), cfg, passwordCheckerFunc(func(ctx context.Context, _ string) (bool, error) {
		entered <- struct{}{}

		select {
		case <-release:
			return false, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}), newAdmissionForTest(t), log.NewTestLogger("error"))
	if err != nil {
		t.Fatal(err)
	}

	store := &changeAdmissionStore{actor: changeActor(time.Now().UTC())}

	s, err := NewRecoveryService(base, store, config.RecoveryConfig{}, "mailbox-v1")
	if err != nil {
		t.Fatal(err)
	}

	token, _, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	results := make(chan error, workers)
	p := PasswordChangePolicy{Requirement: AccessRequirement{Revision: "test-v1"}, AllowPassword: true}

	for range workers {
		go func() { _, e := s.ChangePassword(ctx, token, "a complete new password", p); results <- e }()
	}

	for range workers {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}

	close(release)

	busy, success := 0, 0

	for range workers {
		switch err := <-results; {
		case err == nil:
			success++
		case errors.Is(err, model.ErrPasswordVerifierBusy):
			busy++
		default:
			t.Fatalf("KDF result: %v", err)
		}
	}

	if busy == 0 || success == 0 || store.admissions.Load() != workers || store.commits.Load() != int32(success) {
		t.Fatal("shared admission or failure boundary violated")
	}
}
