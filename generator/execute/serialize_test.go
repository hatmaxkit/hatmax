// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarshalYAMLIsStableAndOrdered(t *testing.T) {
	value := validTestManifest()

	first, err := MarshalYAML(value)
	if err != nil {
		t.Fatalf("MarshalYAML() error = %v", err)
	}

	second, err := MarshalYAML(value)
	if err != nil {
		t.Fatalf("second MarshalYAML() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf("MarshalYAML() output is not stable:\n%s\n---\n%s", first, second)
	}

	text := string(first)
	identityIndex := strings.Index(text, "plan_digest:")

	editsIndex := strings.Index(text, "edits:")
	if identityIndex < 0 || editsIndex <= identityIndex {
		t.Errorf("MarshalYAML() fields are not in contract order:\n%s", text)
	}
}
