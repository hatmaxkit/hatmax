// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package hatmaxcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpaqueProjectIdentityIsStableAndDoesNotExposePath(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "project")

	err := os.Mkdir(nested, 0o755)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	first, err := opaqueProjectIdentity(nested)
	if err != nil {
		t.Fatalf("opaqueProjectIdentity() error = %v", err)
	}

	second, err := opaqueProjectIdentity(filepath.Join(nested, "."))
	if err != nil {
		t.Fatalf("opaqueProjectIdentity() repeated error = %v", err)
	}

	if first != second || !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("identities = %q and %q, want stable SHA-256 identity", first, second)
	}

	if strings.Contains(first, nested) {
		t.Fatalf("opaque identity %q exposes project path", first)
	}
}
