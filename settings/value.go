// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package settings

import "strconv"

// ParseBool parses a string value as boolean.
func ParseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}

// ParseInt parses a string value as integer.
func ParseInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}

	return strconv.Atoi(value)
}

// FormatBool formats a boolean as string.
func FormatBool(v bool) string {
	return strconv.FormatBool(v)
}

// FormatInt formats an integer as string.
func FormatInt(v int) string {
	return strconv.Itoa(v)
}
