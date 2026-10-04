// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package crypto

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTPKey creates a new TOTP key for the given issuer and account.
func GenerateTOTPKey(issuer, account string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
}

// MatchTOTPCode returns a trusted-time matched integer step, preferring the
// current step. Shape is canonical six ASCII digits; skew is bounded to one.
func MatchTOTPCode(secret, code string, now time.Time, skew uint) (int64, error) {
	if len(secret) != 32 || len(code) != 6 || skew > 1 || now.Unix() < 30 {
		return 0, ErrTOTPCode
	}

	for i := range len(code) {
		if code[i] < '0' || code[i] > '9' {
			return 0, ErrTOTPCode
		}
	}

	step := now.Unix() / 30

	offsets := []int64{0}
	if skew == 1 {
		offsets = append(offsets, -1, 1)
	}

	for _, offset := range offsets {
		candidate := step + offset

		valid, err := totp.ValidateCustom(code, secret, time.Unix(candidate*30, 0), totp.ValidateOpts{Period: 30, Skew: 0, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err != nil {
			return 0, ErrTOTPCode
		}

		if valid {
			return candidate, nil
		}
	}

	return 0, ErrTOTPCode
}

// TOTPInWindow is also evaluated with fresh post-lock time at completion.
func TOTPInWindow(step int64, now time.Time, skew uint) bool {
	current := now.Unix() / 30

	return skew <= 1 && step >= 1 && step >= current-int64(skew) && step <= current+int64(skew)
}

var ErrTOTPCode = errors.New("invalid TOTP code")

// GenerateQRCodePNG generates a PNG image of the QR code for the TOTP key.
func GenerateQRCodePNG(key *otp.Key, size int) ([]byte, error) {
	img, err := key.Image(size, size)
	if err != nil {
		return nil, fmt.Errorf("cannot generate QR code image: %w", err)
	}

	var buf bytes.Buffer

	encodeErr := png.Encode(&buf, img)
	if encodeErr != nil {
		return nil, fmt.Errorf("cannot encode QR code as PNG: %w", encodeErr)
	}

	return buf.Bytes(), nil
}
