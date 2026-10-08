// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const nativeToolNames = "go make golangci-lint sqlc initdb pg_ctl postgres psql createuser createdb chromium bash sh git awk sed rg find sort realpath dirname basename readlink head tail grep cut xargs tr cmp cp mv rm rmdir mkdir ln cat chmod touch date sleep env printf sha256sum mktemp id uname wc tee timeout kill gcc cc g++ c++ as ld ar objcopy objdump ranlib nm strip ldd getent install locale gofmt codex ps lsof nohup"

func nativeImage(data []byte) bool {
	if bytes.HasPrefix(data, []byte{0x7f, 'E', 'L', 'F'}) {
		return true
	}

	line, _, _ := strings.Cut(string(data), "\n")

	return slices.Contains([]string{"#!/bin/sh", "#!/bin/bash", "#!/usr/bin/sh", "#!/usr/bin/bash", "#!/usr/bin/env bash", "#!/usr/bin/env sh"}, strings.TrimSpace(line))
}

func checkNativeTools(manifest string) error {
	bin := os.Getenv("HATMAX_DOC_NATIVE_BIN")
	// go run prepends its own GOROOT/bin. That directory is the required native
	// toolchain, and contains only the same Go and gofmt images bound below.
	toolchainBin := filepath.Join(runtime.GOROOT(), "bin")

	lookup := os.Getenv("PATH")
	if !filepath.IsAbs(bin) || (lookup != bin && lookup != toolchainBin+string(os.PathListSeparator)+bin) || os.Getenv("GOTOOLCHAIN") != "local" {
		return fmt.Errorf("native toolchain requires an allowlist-only PATH and local Go toolchain")
	}

	if lookup != bin {
		entries, err := os.ReadDir(toolchainBin)
		if err != nil || len(entries) != 2 || entries[0].Name() != "go" || entries[1].Name() != "gofmt" {
			return fmt.Errorf("unapproved Go toolchain lookup directory")
		}
	}

	data, err := os.ReadFile(manifest)
	if err != nil {
		return err
	}

	names := strings.Fields(nativeToolNames)
	seen := make(map[string]bool)

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 || !slices.Contains(names, fields[0]) || seen[fields[0]] {
			return fmt.Errorf("unapproved or duplicate native tool")
		}

		name := fields[0]
		seen[name] = true

		target, err := filepath.EvalSymlinks(filepath.Join(bin, name))
		if err != nil {
			return err
		}

		image, err := os.ReadFile(target)
		if err != nil {
			return err
		}

		if target != fields[1] || digest(string(image)) != fields[2] || !nativeImage(image) {
			return fmt.Errorf("native executable identity mismatch: %s", name)
		}
	}

	entries, err := os.ReadDir(bin)
	if err != nil {
		return err
	}

	if len(seen) != len(names) || len(entries) != len(names) {
		return fmt.Errorf("native toolchain is missing required tools or includes unapproved tools")
	}

	for _, entry := range entries {
		if !seen[entry.Name()] {
			return fmt.Errorf("unapproved tool in native PATH: %s", entry.Name())
		}
	}

	fmt.Printf("Required native toolchain verified: %d executable identities; no inherited PATH directories.\n", len(names))

	return nil
}
