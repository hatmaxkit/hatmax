// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Command object-storage binds the delivery's criteria, roots and exact evidence.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 1 && args[0] == "contract" {
		roots, err := discoverRoots()
		if err != nil {
			return err
		}

		err = checkLedger(roots)
		if err == nil {
			fmt.Printf("OS01-OS10 ledger: %d discovered roots; acceptance still requires exact-candidate execution.\n", len(roots))
		}

		return err
	}

	return fmt.Errorf("usage: object-storage contract")
}

type listedPackage struct {
	ImportPath, Dir           string
	TestGoFiles, XTestGoFiles []string
}

func listedRoots(tags string, packages []string) ([]root, error) {
	arguments := []string{"list", "-json"}
	if tags != "" {
		arguments = append(arguments, "-tags="+tags)
	}

	arguments = append(arguments, packages...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "go", arguments...)
	command.WaitDelay = 5 * time.Second

	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("discover tagged package identity: %w", err)
	}

	var result []root

	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var listed listedPackage

		err := decoder.Decode(&listed)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		for _, file := range append(listed.TestGoFiles, listed.XTestGoFiles...) {
			source, err := parser.ParseFile(token.NewFileSet(), filepath.Join(listed.Dir, file), nil, 0)
			if err != nil {
				return nil, err
			}

			for _, declaration := range source.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if ok && function.Recv == nil && strings.HasPrefix(function.Name.Name, "Test") && function.Name.Name != "TestMain" {
					result = append(result, root{Package: strings.TrimPrefix(listed.ImportPath, "hatmax.adrianpk.com/"), Name: function.Name.Name})
				}
			}
		}
	}

	return result, nil
}

func discoverRoots() ([]root, error) {
	groups := []struct {
		name, tags string
		packages   []string
	}{
		{"ordinary", "", []string{"./storage/...", "./image/...", "./scripts/documentation-conformance", "./scripts/object-storage"}},
		{"remote", "integration", []string{"./storage/s3"}},
		{"guide", "integration,documentation", []string{"./scripts/documentation-conformance", "./storage/s3"}},
	}
	seen := make(map[string]bool)

	var result []root

	for _, group := range groups {
		roots, err := listedRoots(group.tags, group.packages)
		if err != nil {
			return nil, err
		}

		for _, r := range roots {
			identity := r.Package + ":" + r.Name
			if seen[identity] {
				continue
			}

			seen[identity] = true

			r.Group = group.name
			if group.name == "guide" && r.Package == "storage/s3" {
				r.Group = "guide-child"
			}

			result = append(result, r)
		}
	}

	slices.SortFunc(result, func(a, b root) int { return strings.Compare(a.key(), b.key()) })

	return result, nil
}

func checkLedger(roots []root) error {
	known := make(map[string]bool)
	for _, r := range roots {
		if known[r.key()] {
			return fmt.Errorf("duplicate discovered root: %s", r.key())
		}

		known[r.key()] = true
	}

	if len(criteria) != 10 {
		return fmt.Errorf("OS01-OS10 ledger is incomplete")
	}

	for index, criterion := range criteria {
		if criterion.ID != fmt.Sprintf("OS%02d", index+1) || len(criterion.Roots) == 0 {
			return fmt.Errorf("invalid criterion identity or empty root mapping")
		}

		for _, r := range criterion.Roots {
			if !known[r.key()] {
				return fmt.Errorf("criterion %s requires missing tagged root: %s", criterion.ID, r.key())
			}
		}
	}

	return nil
}
