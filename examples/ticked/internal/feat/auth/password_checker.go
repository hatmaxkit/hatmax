// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package auth

import "context"

// PasswordChecker is a bounded demonstration source of common and application-
// specific passwords. It is safe for concurrent use and retains no candidates.
// Its curated list is not a compromised-password corpus or a production coverage
// claim. Production applications must choose and document their checking source.
type PasswordChecker struct {
	disallowed map[string]struct{}
}

// NewPasswordChecker creates the example's finite, exact-match policy source.
// Values are curated common phrases and Ticked-specific names, with no external
// provider, network operation or unbounded lookup work.
func NewPasswordChecker() *PasswordChecker {
	values := []string{"password", "password123456789", "123456789012345", "correcthorsebatterystaple", "ticked", "ticked-password", "ticked-password123"}

	disallowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		disallowed[value] = struct{}{}
	}

	return &PasswordChecker{disallowed: disallowed}
}

// Disallowed checks the complete NFC candidate and observes caller cancellation.
func (c *PasswordChecker) Disallowed(ctx context.Context, password string) (bool, error) {
	err := ctx.Err()
	if err != nil {
		return false, err
	}

	_, found := c.disallowed[password]

	return found, nil
}
