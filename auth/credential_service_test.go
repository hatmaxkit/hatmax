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

	"hatmax.adrianpk.com/config"
	"hatmax.adrianpk.com/log"
	"hatmax.adrianpk.com/model"
)

// Construction fails before any operation when policy, resource settings or the
// required checker are missing. An always-MFA minimum cannot weaken this service.
func TestCredentialServiceConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		checker PasswordChecker
		change  func(*config.Config)
	}{
		{name: "missing checker"},
		{name: "weak password minimum", checker: passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), change: func(c *config.Config) { c.Auth.PasswordMinLen = 8 }},
		{name: "invalid memory budget", checker: passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), change: func(c *config.Config) { c.Auth.PasswordMaxConcurrent = 9 }},
		{name: "invalid timeout", checker: passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), change: func(c *config.Config) { c.Auth.PasswordTimeout = "invalid" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.New()
			if tc.change != nil {
				tc.change(cfg)
			}

			svc, err := NewService(newMockQueries(), cfg, tc.checker, newAdmissionForTest(t), log.NewTestLogger("error"))
			if err == nil || svc != nil {
				t.Fatal("invalid credential setup was accepted")
			}
		})
	}
}

// Signup enforces complete normalized candidate checking before persistence and
// produces a supported record. Verification does not rerun creation-only policy.
func TestCredentialServicePolicy(t *testing.T) {
	providerErr := errors.New("secret checker diagnostic")
	for _, tc := range []struct {
		name, password        string
		disallowed            bool
		checkerErr, errorWant error
	}{
		{name: "long Unicode", password: strings.Repeat("😀", 64)},
		{name: "normalized candidate", password: strings.Repeat("e\u0301", 15)},
		{name: "short Unicode", password: strings.Repeat("界", 14), errorWant: ErrPasswordTooShort},
		{name: "disallowed candidate", password: "disallowed password phrase", disallowed: true, errorWant: ErrPasswordDisallowed},
		{name: "unavailable checker", password: "candidate password phrase", checkerErr: providerErr, errorWant: ErrPasswordCheckFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.New()
			cfg.Auth.ArgonMemoryKiB = 19456
			cfg.Auth.ArgonIterations = 2
			cfg.Auth.ArgonParallelism = 1
			checked := ""
			calls := 0
			checker := passwordCheckerFunc(func(ctx context.Context, password string) (bool, error) {
				checked = password
				calls++

				return tc.disallowed, tc.checkerErr
			})
			q := newMockQueries()

			svc, err := NewService(q, cfg, checker, newAdmissionForTest(t), log.NewTestLogger("error"))
			if err != nil {
				t.Fatal(err)
			}

			user, err := svc.Signup(context.Background(), "candidate@example.com", tc.password)
			if !errors.Is(err, tc.errorWant) {
				t.Fatalf("signup = %v; want %v", err, tc.errorWant)
			}

			if tc.errorWant != nil {
				if user != nil || len(q.users) != 0 {
					t.Fatal("rejected password persisted")
				}

				if tc.checkerErr != nil && strings.Contains(err.Error(), tc.checkerErr.Error()) {
					t.Fatal("checker diagnostic leaked")
				}

				return
			}

			if user.AuthVersion != 1 || !strings.HasPrefix(user.PasswordHash, "$argon2id$v=19$") {
				t.Fatal("signup stored unsupported credential")
			}

			if tc.name == "normalized candidate" && checked != strings.Repeat("é", 15) {
				t.Fatal("checker saw an unnormalized candidate")
			}

			session, err := testSignin(svc, context.Background(), user.Email, tc.password)
			if err != nil || session == nil || calls != 1 {
				t.Fatalf("sign-in reran policy or failed: %v; calls %d", err, calls)
			}
		})
	}
}

// Malformed/unsupported stored data and canceled verification are
// operating outcomes, distinct from an ordinary mismatch; none can create sessions.
func TestCredentialServiceFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		record   string
		password string
		canceled bool
		want     error
	}{
		{name: "legacy record", record: "$2a$12$legacy", password: "correct-password-123", want: model.ErrPasswordRecord},
		{name: "malformed record", record: "invalid", password: "correct-password-123", want: model.ErrPasswordRecord},
		{name: "invalid encoding", password: "\xff", want: model.ErrPasswordInput},
		{name: "wrong proof", password: "wrong password phrase", want: ErrInvalidPassword},
		{name: "canceled request", password: "correct-password-123", canceled: true, want: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := newMockQueries()
			svc := newServiceForTest(t, q, config.New(), log.NewTestLogger("error"))

			user, err := svc.Signup(context.Background(), "proof@example.com", "correct-password-123")
			if err != nil {
				t.Fatal(err)
			}

			if tc.record != "" {
				q.users[user.ID].PasswordHash = tc.record
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tc.canceled {
				cancel()
			}

			session, err := testSignin(svc, ctx, user.Email, tc.password)
			if session != nil || !errors.Is(err, tc.want) || len(q.sessions) != 0 {
				t.Fatalf("failed proof issued a session: %v; want %v", err, tc.want)
			}
		})
	}
}

// Mandatory admission prevents accidental construction of an unguarded service.
func TestCredentialAdmissionRequired(t *testing.T) {
	for _, tc := range []struct {
		name      string
		admission *CredentialAdmission
	}{
		{"missing", nil}, {"uninitialized", &CredentialAdmission{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := NewService(newMockQueries(), config.New(), passwordCheckerFunc(func(context.Context, string) (bool, error) { return false, nil }), tc.admission, log.NewTestLogger("error"))
			if err == nil || svc != nil {
				t.Fatal("unguarded construction accepted")
			}
		})
	}
}
