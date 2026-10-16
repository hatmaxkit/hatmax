// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reconcile the committed OS ledger against current build-tag-selected source.
func TestObjectAcceptance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "run", "./scripts/object-storage", "contract")
	command.Dir = filepath.Join("..", "..")
	command.WaitDelay = 5 * time.Second

	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "OS01-OS10 ledger:") {
		t.Fatalf("object acceptance root/criterion drift: %v: %s", err, output)
	}
}
