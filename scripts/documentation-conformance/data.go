// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func newDataVerification(fixture string) (*verification, inventory, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, inventory{}, err
	}

	fixture, err = filepath.Abs(fixture)
	if err != nil {
		return nil, inventory{}, err
	}

	relative, err := filepath.Rel(root, fixture)
	if err != nil || !strings.HasPrefix(filepath.ToSlash(relative), ".tmp/documentation-conformance/slice-3.") {
		return nil, inventory{}, fmt.Errorf("data checks require owned slice-3 fixture storage")
	}

	_, err = command("git", "check-ignore", filepath.Join(fixture, "data-receipts.json"))
	if err != nil {
		return nil, inventory{}, err
	}

	discovered, err := discover()
	if err != nil {
		return nil, inventory{}, err
	}

	head, err := command("git", "rev-parse", "HEAD")
	if err != nil {
		return nil, inventory{}, err
	}

	version, err := command("go", "version")
	if err != nil {
		return nil, inventory{}, err
	}

	compiler, err := command("go", "env", "-json", "GOOS", "GOARCH", "CGO_ENABLED", "GOTOOLCHAIN", "GOFLAGS", "GOWORK")
	if err != nil {
		return nil, inventory{}, err
	}

	inputs, err := executionInputs(discovered)
	if err != nil {
		return nil, inventory{}, err
	}

	v := &verification{root: root, fixture: fixture, receipt: runtimeReceipt{Head: strings.TrimSpace(head), Go: strings.TrimSpace(version), Compiler: compiler, Inputs: inputs}}

	return v, discovered, nil
}

func runDataContexts(fixture string) error {
	v, discovered, err := newDataVerification(fixture)
	if err != nil {
		return err
	}

	err = compileDataContexts(v, discovered)
	if err != nil {
		return err
	}

	fmt.Printf("Data/configuration contextual compilation passed: %d exact Go fragments; execution receipts remain pending.\n", len(v.receipt.Proofs))

	return nil
}

func runDataWorkbench(fixture string) error {
	v, discovered, err := newDataVerification(fixture)
	if err != nil {
		return err
	}

	if os.Getenv("HATMAX_DOC_PGDATA") != filepath.Join(v.fixture, "postgres-data") || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("workbench requires the invocation-owned PostgreSQL fixture")
	}

	err = v.dataWorkbench(discovered)
	if err != nil {
		return err
	}

	fmt.Println("Data/configuration workbench passed: real validators, SQLC queries, migration history, settings and seeder tracking; full Slice 3 receipts remain pending.")

	return nil
}
