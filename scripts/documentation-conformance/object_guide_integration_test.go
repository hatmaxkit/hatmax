//go:build integration && documentation

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The guide root requires the real remote executable and owned fixture cleanup.
func TestObjectStorageGuide(t *testing.T) {
	if os.Getenv("HATMAX_STORAGE_MINIO") == "" || os.Getenv("HATMAX_STORAGE_FIXTURE_RECEIPT") == "" {
		t.Fatal("owned identified fixture is required; use the object storage slice runner")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 540*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "test", "-json", "-race", "-tags=integration,documentation", "-count=1", "-p=1", "-parallel=2", "-timeout=360s", "-run", "^TestDocumentedStorage$", "./storage/s3")
	command.Dir = filepath.Join("..", "..")
	command.WaitDelay = 5 * time.Second

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("owned documented remote workflow failed: %v", err)
	}

	passed := false

	for line := range strings.SplitSeq(string(output), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event struct{ Action, Package, Test string }

		err := json.Unmarshal([]byte(line), &event)
		if err != nil {
			t.Fatal("remote root produced malformed result evidence")
		}

		if event.Action == "skip" {
			t.Fatal("required documented workflow skipped")
		}

		if event.Package == "hatmax.adrianpk.com/storage/s3" && event.Test == "TestDocumentedStorage" && event.Action == "pass" {
			passed = true
		}
	}

	if !passed {
		t.Fatal("documented remote workflow root did not execute")
	}

	receipt, err := os.ReadFile(os.Getenv("HATMAX_STORAGE_FIXTURE_RECEIPT"))
	if err != nil {
		t.Fatal(err)
	}

	for _, required := range []string{"guide: signed POST and exact final readback passed", "guide: spool empty", "objects-after-cleanup: 0", "bucket: removed", "server: stopped", "temporary-root: removed"} {
		if !strings.Contains(string(receipt), required+"\n") {
			t.Fatalf("documented workflow cleanup evidence missing: %s", required)
		}
	}
}
