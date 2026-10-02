// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Seal validates a manifest and binds its content to a deterministic digest.
// A previously sealed manifest is verified rather than silently resealed.
func Seal(value Manifest) (Manifest, error) {
	if value.Digest != "" {
		err := VerifyDigest(value)
		if err != nil {
			return Manifest{}, err
		}

		return value, nil
	}

	err := Validate(value)
	if err != nil {
		return Manifest{}, err
	}

	digest, err := computeDigest(value)
	if err != nil {
		return Manifest{}, err
	}

	value.Digest = digest

	return value, nil
}

// VerifyDigest confirms that a sealed manifest still matches its semantic
// content.
func VerifyDigest(value Manifest) error {
	if value.Digest == "" {
		return executionError("execution_digest_missing", "digest", "sealed manifest digest is required")
	}

	err := Validate(value)
	if err != nil {
		return err
	}

	want := value.Digest
	value.Digest = ""

	actual, err := computeDigest(value)
	if err != nil {
		return err
	}

	if actual != want {
		return executionError("execution_digest_mismatch", "digest", "manifest content does not match digest")
	}

	return nil
}

func computeDigest(value Manifest) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal canonical execution manifest: %w", err)
	}

	digest := sha256.Sum256(data)

	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
