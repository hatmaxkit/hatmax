// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrPasswordPolicy      = errors.New("invalid password policy")
	ErrPasswordTooLong     = errors.New("password too long")
	ErrPasswordEncoding    = errors.New("invalid password encoding")
	ErrPasswordDisallowed  = errors.New("password is disallowed")
	ErrPasswordCheckFailed = errors.New("password check failed")
)

// PasswordChecker checks the complete, NFC-normalized candidate against a
// caller-owned source of common, compromised or application-specific values.
// It must honor ctx, bound its own work and be safe for concurrent calls.
// A true result means the candidate is disallowed. It must not retain passwords.
type PasswordChecker interface {
	Disallowed(ctx context.Context, password string) (bool, error)
}

// PasswordPolicyConfig defines trusted server-side credential policy.
// Zero limits use defaults: 15 characters, maximum 1024 characters/4096 bytes
// and a two-second checker timeout. MFARequired permits an eight-character
// minimum only when the application always requires MFA for password access.
// The flag does not implement or prove MFA.
type PasswordPolicyConfig struct {
	MinLength    int
	MaxLength    int
	MaxBytes     int
	CheckTimeout time.Duration
	MFARequired  bool
}

// PasswordPolicy prepares candidates for new credentials. Its configuration is
// immutable; concurrent safety also depends on the supplied checker.
type PasswordPolicy struct {
	config  PasswordPolicyConfig
	checker PasswordChecker
}

// NewPasswordPolicy validates configuration and requires a password checker.
// MaxLength is between 64 and 1024; MaxBytes supports every Unicode code point
// at that length and cannot exceed 4096. Checker timeouts cannot exceed 30 seconds.
func NewPasswordPolicy(cfg PasswordPolicyConfig, checker PasswordChecker) (*PasswordPolicy, error) {
	minimum := 15
	if cfg.MFARequired {
		minimum = 8
	}

	if cfg.MinLength == 0 {
		cfg.MinLength = minimum
	}

	if cfg.MaxLength == 0 {
		cfg.MaxLength = 1024
	}

	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 4096
	}

	if cfg.CheckTimeout == 0 {
		cfg.CheckTimeout = 2 * time.Second
	}

	if cfg.MinLength < minimum || cfg.MinLength > cfg.MaxLength {
		return nil, fmt.Errorf("%w: minimum length", ErrPasswordPolicy)
	}

	if cfg.MaxLength < 64 || cfg.MaxLength > 1024 {
		return nil, fmt.Errorf("%w: maximum length", ErrPasswordPolicy)
	}

	if cfg.MaxBytes < cfg.MaxLength*utf8.UTFMax || cfg.MaxBytes > 4096 {
		return nil, fmt.Errorf("%w: maximum bytes", ErrPasswordPolicy)
	}

	if cfg.CheckTimeout <= 0 || cfg.CheckTimeout > 30*time.Second {
		return nil, fmt.Errorf("%w: checker timeout", ErrPasswordPolicy)
	}

	if checker == nil {
		return nil, fmt.Errorf("%w: password checker is required", ErrPasswordPolicy)
	}

	return &PasswordPolicy{config: cfg, checker: checker}, nil
}

// Prepare validates and NFC-normalizes a new password without trimming, folding
// case or truncating. Use its returned value for hashing, and apply NFC before
// verification of that credential format. Verification does not run this
// new-password policy again. Failures return an empty candidate.
// Checker cancellation is cooperative: no detached goroutine is started to hide
// a checker that ignores its context. Late successful results are rejected.
func (p *PasswordPolicy) Prepare(ctx context.Context, password string) (string, error) {
	if p == nil || p.checker == nil {
		return "", ErrPasswordPolicy
	}

	err := ctx.Err()
	if err != nil {
		return "", err
	}

	if len(password) > p.config.MaxBytes {
		return "", ErrPasswordTooLong
	}

	if !utf8.ValidString(password) {
		return "", ErrPasswordEncoding
	}

	password = norm.NFC.String(password)
	length := utf8.RuneCountInString(password)

	if length < p.config.MinLength {
		return "", ErrPasswordTooShort
	}

	if length > p.config.MaxLength || len(password) > p.config.MaxBytes {
		return "", ErrPasswordTooLong
	}

	checkCtx, cancel := context.WithTimeout(ctx, p.config.CheckTimeout)
	defer cancel()

	err = checkCtx.Err()
	if err != nil {
		return "", err
	}

	disallowed, checkErr := p.checker.Disallowed(checkCtx, password)

	err = checkCtx.Err()
	if err != nil {
		return "", err
	}

	if checkErr != nil {
		return "", &passwordCheckError{cause: checkErr}
	}

	if disallowed {
		return "", ErrPasswordDisallowed
	}

	return password, nil
}

// Error text is deliberately independent of checker errors, which can contain
// provider diagnostics. The original cause remains available to errors.Is/As.
type passwordCheckError struct {
	cause error
}

func (e *passwordCheckError) Error() string {
	return ErrPasswordCheckFailed.Error()
}

func (e *passwordCheckError) Unwrap() error {
	return e.cause
}

func (e *passwordCheckError) Is(target error) bool {
	return target == ErrPasswordCheckFailed
}
