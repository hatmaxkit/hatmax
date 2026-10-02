// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package book

import (
	"fmt"
	"strconv"
	"strings"
)

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(value string) (version, error) {
	normalized := strings.TrimPrefix(value, "v")

	parts := strings.Split(normalized, ".")
	if len(parts) != 3 {
		return version{}, fmt.Errorf("version %q must use major.minor.patch", value)
	}

	values := make([]int, len(parts))
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return version{}, fmt.Errorf("version %q has an invalid numeric component", value)
		}

		values[index] = parsed
	}

	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func compareVersion(left, right version) int {
	leftParts := [...]int{left.major, left.minor, left.patch}
	rightParts := [...]int{right.major, right.minor, right.patch}

	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1
		}

		if leftParts[index] > rightParts[index] {
			return 1
		}
	}

	return 0
}

// SupportsHatmax reports whether a Hatmax release is inside the Book range.
func (b *Book) SupportsHatmax(value string) (bool, error) {
	requested, err := parseVersion(value)
	if err != nil {
		return false, err
	}

	minimum, err := parseVersion(b.manifest.Hatmax.Minimum)
	if err != nil {
		return false, fmt.Errorf("parse minimum Hatmax version: %w", err)
	}

	maximum, err := parseVersion(b.manifest.Hatmax.MaximumExclusive)
	if err != nil {
		return false, fmt.Errorf("parse maximum Hatmax version: %w", err)
	}

	return compareVersion(requested, minimum) >= 0 && compareVersion(requested, maximum) < 0, nil
}
