// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

type passwordCheckerFunc func(context.Context, string) (bool, error)

func (f passwordCheckerFunc) Disallowed(ctx context.Context, password string) (bool, error) {
	return f(ctx, password)
}

// Invalid limits and missing checkers must fail during construction so callers
// cannot silently weaken password-only policy or prevent long Unicode passwords.
func TestPasswordPolicyConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  PasswordPolicyConfig
	}{
		{name: "short single factor", cfg: PasswordPolicyConfig{MinLength: 8}},
		{name: "short multifactor", cfg: PasswordPolicyConfig{MinLength: 7, MFARequired: true}},
		{name: "negative minimum", cfg: PasswordPolicyConfig{MinLength: -1}},
		{name: "inverted lengths", cfg: PasswordPolicyConfig{MinLength: 65, MaxLength: 64}},
		{name: "small maximum", cfg: PasswordPolicyConfig{MaxLength: 63}},
		{name: "large maximum", cfg: PasswordPolicyConfig{MaxLength: 1025}},
		{name: "unicode byte cap", cfg: PasswordPolicyConfig{MaxLength: 64, MaxBytes: 255}},
		{name: "large byte cap", cfg: PasswordPolicyConfig{MaxBytes: 4097}},
		{name: "negative timeout", cfg: PasswordPolicyConfig{CheckTimeout: -time.Second}},
		{name: "large timeout", cfg: PasswordPolicyConfig{CheckTimeout: 31 * time.Second}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := passwordCheckerFunc(func(context.Context, string) (bool, error) {
				t.Fatal("invalid configuration called checker")

				return false, nil
			})

			policy, err := NewPasswordPolicy(tc.cfg, checker)
			if !errors.Is(err, ErrPasswordPolicy) || policy != nil {
				t.Fatalf("construction = %v, %v; want nil and invalid policy", policy, err)
			}
		})
	}

	t.Run("missing checker", func(t *testing.T) {
		policy, err := NewPasswordPolicy(PasswordPolicyConfig{}, nil)
		if !errors.Is(err, ErrPasswordPolicy) || policy != nil {
			t.Fatalf("construction = %v, %v; want nil and invalid policy", policy, err)
		}
	})
}

// Boundaries use normalized code points rather than UTF-8 bytes. The exact
// normalized candidate reaches checking/hashing; whitespace and case survive.
func TestPasswordPolicyInput(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cfg      PasswordPolicyConfig
		password string
		want     string
		wantErr  error
	}{
		{name: "default minimum", password: strings.Repeat("a", 15), want: strings.Repeat("a", 15)},
		{name: "below minimum", password: strings.Repeat("a", 14), wantErr: ErrPasswordTooShort},
		{name: "unicode minimum", password: strings.Repeat("界", 15), want: strings.Repeat("界", 15)},
		{name: "unicode too short", password: strings.Repeat("界", 8), wantErr: ErrPasswordTooShort},
		{name: "long unicode", password: strings.Repeat("😀", 64), want: strings.Repeat("😀", 64)},
		{name: "maximum unicode", password: strings.Repeat("😀", 1024), want: strings.Repeat("😀", 1024)},
		{name: "above character cap", password: strings.Repeat("a", 1025), wantErr: ErrPasswordTooLong},
		{name: "above byte cap", password: strings.Repeat("a", 4097), wantErr: ErrPasswordTooLong},
		{name: "invalid utf8", password: strings.Repeat("a", 15) + "\xff", wantErr: ErrPasswordEncoding},
		{name: "empty", wantErr: ErrPasswordTooShort},
		{name: "multifactor minimum", cfg: PasswordPolicyConfig{MFARequired: true}, password: "abcdefgh", want: "abcdefgh"},
		{name: "below multifactor", cfg: PasswordPolicyConfig{MFARequired: true}, password: "abcdefg", wantErr: ErrPasswordTooShort},
		{name: "composed candidate", password: strings.Repeat("e\u0301", 15), want: strings.Repeat("é", 15)},
		{name: "normalized minimum", password: strings.Repeat("e\u0301", 8), wantErr: ErrPasswordTooShort},
		{name: "normalized expansion", cfg: PasswordPolicyConfig{MaxLength: 64}, password: strings.Repeat("\u0344", 33), wantErr: ErrPasswordTooLong},
		{name: "spaces and case", password: "  Mixed Phrase  ", want: "  Mixed Phrase  "},
		{name: "no composition rule", password: strings.Repeat(" ", 15), want: strings.Repeat(" ", 15)},
		{name: "configured maximum", cfg: PasswordPolicyConfig{MaxLength: 64, MaxBytes: 256}, password: strings.Repeat("😀", 64), want: strings.Repeat("😀", 64)},
		{name: "configured minimum", cfg: PasswordPolicyConfig{MinLength: 20}, password: strings.Repeat("a", 19), wantErr: ErrPasswordTooShort},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			checker := passwordCheckerFunc(func(ctx context.Context, password string) (bool, error) {
				called = true

				if password != tc.want {
					t.Fatal("checker received a different candidate")
				}

				_, ok := ctx.Deadline()
				if !ok {
					t.Fatal("checker has no deadline")
				}

				return false, nil
			})

			policy, err := NewPasswordPolicy(tc.cfg, checker)
			if err != nil {
				t.Fatal(err)
			}

			got, err := policy.Prepare(context.Background(), tc.password)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("preparation mismatch: error = %v; want %v", err, tc.wantErr)
			}

			if called != (tc.wantErr == nil) {
				t.Fatal("invalid candidate reached checker, or valid candidate bypassed it")
			}
		})
	}
}

// Checker failures cannot approve a password or expose provider diagnostics.
// Cancellation remains observable even if a checker returns a late success.
func TestPasswordPolicyChecker(t *testing.T) {
	providerErr := errors.New("provider diagnostic with candidate material")

	for _, tc := range []struct {
		name    string
		checker passwordCheckerFunc
		wantErr error
	}{
		{
			name: "common or breached",
			checker: func(context.Context, string) (bool, error) {
				return true, nil
			},
			wantErr: ErrPasswordDisallowed,
		},
		{
			name: "provider failure",
			checker: func(context.Context, string) (bool, error) {
				return false, providerErr
			},
			wantErr: ErrPasswordCheckFailed,
		},
		{
			name: "provider timeout",
			checker: func(ctx context.Context, _ string) (bool, error) {
				<-ctx.Done()

				return false, ctx.Err()
			},
			wantErr: context.DeadlineExceeded,
		},
		{
			name: "late success",
			checker: func(ctx context.Context, _ string) (bool, error) {
				<-ctx.Done()

				return false, nil
			},
			wantErr: context.DeadlineExceeded,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				policy, err := NewPasswordPolicy(PasswordPolicyConfig{CheckTimeout: time.Millisecond}, tc.checker)
				if err != nil {
					t.Fatal(err)
				}

				got, err := policy.Prepare(context.Background(), "correct-password")
				if got != "" || !errors.Is(err, tc.wantErr) {
					t.Fatalf("preparation = nonempty:%t, %v; want rejection %v", got != "", err, tc.wantErr)
				}

				if tc.wantErr == ErrPasswordCheckFailed {
					if !errors.Is(err, providerErr) || strings.Contains(err.Error(), providerErr.Error()) {
						t.Fatal("provider cause was lost or exposed in error text")
					}
				}
			})
		})
	}
}

// Canceled requests and uninitialized policies cannot invoke a password source.
// The parent's earlier deadline must constrain the checker rather than extend it.
func TestPasswordPolicyContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := passwordCheckerFunc(func(context.Context, string) (bool, error) {
		t.Fatal("canceled request called checker")

		return false, nil
	})

	policy, err := NewPasswordPolicy(PasswordPolicyConfig{}, checker)
	if err != nil {
		t.Fatal(err)
	}

	got, err := policy.Prepare(ctx, "correct-password")
	if got != "" || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation did not reject preparation")
	}

	for _, invalid := range []*PasswordPolicy{nil, {}} {
		got, err = invalid.Prepare(context.Background(), "correct-password")
		if got != "" || !errors.Is(err, ErrPasswordPolicy) {
			t.Fatal("uninitialized policy did not reject preparation")
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wantDeadline, _ := ctx.Deadline()
	checker = passwordCheckerFunc(func(checkCtx context.Context, _ string) (bool, error) {
		deadline, ok := checkCtx.Deadline()
		if !ok || !deadline.Equal(wantDeadline) {
			t.Fatal("checker extended the parent's deadline")
		}

		return false, nil
	})

	policy, err = NewPasswordPolicy(PasswordPolicyConfig{}, checker)
	if err != nil {
		t.Fatal(err)
	}

	_, err = policy.Prepare(ctx, "correct-password")
	if err != nil {
		t.Fatal(err)
	}
}

// Sharing a policy must not mix candidate data between requests. This exercises
// the same immutable instance with a concurrency-safe checker under the race gate.
func TestPasswordPolicyConcurrent(t *testing.T) {
	checker := passwordCheckerFunc(func(_ context.Context, password string) (bool, error) {
		return password == "blocked-password", nil
	})

	policy, err := NewPasswordPolicy(PasswordPolicyConfig{}, checker)
	if err != nil {
		t.Fatal(err)
	}

	var workers sync.WaitGroup
	for i := range 16 {
		workers.Go(func() {
			password := "correct-password"

			var wantErr error

			if i%2 == 0 {
				password = "blocked-password"
				wantErr = ErrPasswordDisallowed
			}

			got, err := policy.Prepare(context.Background(), password)
			if !errors.Is(err, wantErr) || (err == nil && got != password) || (err != nil && got != "") {
				t.Error("concurrent policy result changed")
			}
		})
	}

	workers.Wait()
}
