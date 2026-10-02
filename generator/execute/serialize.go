// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// MarshalYAML returns a stable user-visible representation of a valid
// execution manifest.
func MarshalYAML(value Manifest) ([]byte, error) {
	err := validateForSerialization(value)
	if err != nil {
		return nil, fmt.Errorf("validate execution manifest: %w", err)
	}

	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal execution manifest YAML: %w", err)
	}

	return data, nil
}

func validateForSerialization(value Manifest) error {
	if value.Digest != "" {
		return VerifyDigest(value)
	}

	return Validate(value)
}
