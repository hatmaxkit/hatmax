// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

type acceptanceReceipt struct {
	runtimeReceipt
	Dependencies map[string]string `json:"execution_receipts"`
	Native       string            `json:"native_tools_sha256"`
	Blocked      []string          `json:"blocked_rows"`
}

func acceptanceMethod(r row, _ inventory) (string, error) {
	for number := 1; number <= 5; number++ {
		if r.id == fmt.Sprintf("example:README.md#block-%d", number) {
			return "executed", nil
		}
	}

	pages := []string{"CHANGELOG.md", "README.md", "docs/README.md", "docs/explanation/README.md",
		"docs/how-to/README.md", "docs/reference/README.md", "docs/reference/package-map/README.md",
		"docs/reference/terminology/README.md", "docs/tutorials/README.md", "docs/tutorials/user-guide/README.md"}
	if slices.Contains(pages, strings.TrimPrefix(r.id, "page:")) && strings.HasPrefix(r.id, "page:") ||
		r.id == "walkthrough:docs/how-to/README.md" {
		return "source-inspected", nil
	}

	return "", fmt.Errorf("unaccounted integrated surface: %s", r.id)
}

func checkRootBindings(inv inventory) error {
	for number := 1; number <= 3; number++ {
		root, err := sourceBlock(inv.contents, "README.md", number)
		if err != nil {
			return err
		}

		tutorial, err := sourceBlock(inv.contents, "docs/how-to/bootstrap-application/README.md", number)
		if err != nil || root != tutorial {
			return fmt.Errorf("root bootstrap execution context differs: block %d", number)
		}
	}

	root, err := sourceBlock(inv.contents, "README.md", 4)
	if err != nil || strings.TrimSpace(root) != "cd ../hatmax\ngo install ./cmd/hm\ncd ..\nhm" {
		return fmt.Errorf("unaccounted root source-install procedure")
	}

	root, err = sourceBlock(inv.contents, "README.md", 5)
	if err != nil {
		return err
	}

	reference, err := sourceBlock(inv.contents, "docs/reference/generator/README.md", 2)
	if err != nil || root != reference {
		return fmt.Errorf("unaccounted root headless request")
	}

	return nil
}

func closureCoverage(rows []row) error {
	var unresolved []string

	for _, r := range rows {
		if r.page == "missing" || r.receipt == "pending" || !slices.Contains([]string{"source-inspected", "compiled", "rendered", "executed", "package-tests"}, r.status) {
			unresolved = append(unresolved, r.id+": "+r.status)
		}
	}

	if len(unresolved) > 0 {
		return fmt.Errorf("integrated acceptance blocked by %d unresolved coverage rows: %s", len(unresolved), strings.Join(unresolved, "; "))
	}

	return nil
}

func runAcceptance(args []string) error {
	if len(args) == 2 && args[1] == "--complete" {
		return completeAcceptance(args[0])
	}

	if len(args) != 6 {
		return fmt.Errorf("usage: acceptance OWNED-FIXTURE RUNTIME DATA IDENTITY INFRASTRUCTURE GENERATOR | acceptance OWNED-FIXTURE --complete")
	}

	fixture := args[0]

	v, inv, err := newGroupVerification(fixture, 7)
	if err != nil {
		return err
	}

	fixture = v.fixture

	err = checkNativeTools(filepath.Join(fixture, "native-tools.tsv"))
	if err != nil {
		return err
	}

	err = checkRootBindings(inv)
	if err != nil {
		return err
	}

	err = checkNavigation(inv.contents)
	if err != nil {
		return err
	}

	target := filepath.Join(fixture, "acceptance-receipts.json")

	_, err = os.Stat(target)
	if !os.IsNotExist(err) {
		return fmt.Errorf("fixture already contains acceptance evidence")
	}

	validators := []func(string) error{checkEvidence, checkDataEvidence, checkIdentityEvidence, checkInfrastructureEvidence, checkGeneratorEvidence}
	dependencies := make(map[string]string)

	for i, file := range args[1:] {
		err = validators[i](file)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		dependencies[file] = digest(string(data))

		var parent runtimeReceipt

		err = json.Unmarshal(data, &parent)
		if err != nil {
			return err
		}

		if i != 0 && i != 4 {
			continue
		}

		if i == 4 {
			passed := false

			for _, cmd := range parent.Commands {
				log, err := os.ReadFile(filepath.Join(filepath.Dir(file), cmd.Log))
				if err != nil {
					return err
				}

				if namedTestPassed(log, "TestRootCommands") && namedTestPassed(log, "TestHeadless") {
					passed = true
				}
			}

			if !passed {
				return fmt.Errorf("root procedure and headless execution evidence missing")
			}
		}
		// Parent execution has just been verified; copy safe logs into this receipt's
		// owned namespace so ordinary proof checks retain exact command identities.
		for _, cmd := range parent.Commands {
			log, err := os.ReadFile(filepath.Join(filepath.Dir(file), cmd.Log))
			if err != nil {
				return err
			}

			name := fmt.Sprintf("parent-%d-%s", i, cmd.Log)

			err = writeFile(filepath.Join(fixture, name), string(log))
			if err != nil {
				return err
			}

			cmd.Log = name
			v.receipt.Commands = append(v.receipt.Commands, cmd)
		}
	}

	v.receipt.Observations = []string{
		"Root Go/YAML/module procedure bytes match the fresh executed bootstrap; real config validation, HTTP health and coordinated shutdown, with immutable local clone and owned port.",
		"Exact root install/TUI shell executed in a native PTY; root headless request equals the production CLI fixture request; bounded interpreter, no authenticated model acceptance.",
		"All quadrant anchors and index reachability checked; T7.1 entrypoint intent/package/terminology comparison; index walkthrough is navigation, not a runnable application procedure.",
	}

	for _, r := range inv.rows {
		if r.slice != 7 {
			continue
		}

		method, err := acceptanceMethod(r, inv)
		if err != nil {
			return err
		}

		outcome := v.receipt.Observations[2]
		if method == "executed" {
			outcome = v.receipt.Observations[0] + " " + v.receipt.Observations[1]
		}

		v.bindData(r, method, outcome)
	}

	err = verifyGroupProofs(inv, v.receipt, 7, acceptanceMethod)
	if err != nil {
		return err
	}

	rows, err := coverageRows()
	if err != nil {
		return err
	}

	err = reconcile(inv.rows, rows)
	if err != nil {
		return err
	}

	var blocked []string

	for _, r := range rows {
		if r.status == "blocked" {
			blocked = append(blocked, r.id)
		}

		if r.slice == 7 {
			method, err := acceptanceMethod(r, inv)
			if err != nil {
				return err
			}

			if r.status != method || r.receipt != "slice-7/acceptance-receipts.json" {
				return fmt.Errorf("coverage disagrees with acceptance verification: %s", r.id)
			}
		}
	}

	manifest, err := os.ReadFile(filepath.Join(fixture, "native-tools.tsv"))
	if err != nil {
		return err
	}

	receipt := acceptanceReceipt{runtimeReceipt: v.receipt, Dependencies: dependencies, Native: digest(string(manifest)), Blocked: blocked}

	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(target, string(data)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Integrated evidence reconciled: %d rows, %d entrypoint bindings; %d blocked rows retain unresolved outcomes.\n", len(rows), len(receipt.Proofs), len(blocked))

	return nil
}

func coverageRows() ([]row, error) {
	data, err := os.ReadFile(coveragePath)
	if err != nil {
		return nil, err
	}

	return parseRows(string(data))
}

func completeAcceptance(fixture string) error {
	v, inv, err := newGroupVerification(fixture, 7)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(filepath.Join(v.fixture, "acceptance-receipts.json"))
	if err != nil {
		return err
	}

	var receipt acceptanceReceipt

	err = json.Unmarshal(data, &receipt)
	if err != nil {
		return err
	}

	if receipt.Head != v.receipt.Head || receipt.Inputs != v.receipt.Inputs || receipt.Go != v.receipt.Go || receipt.Compiler != v.receipt.Compiler {
		return fmt.Errorf("integrated evidence source/head/tool identity mismatch")
	}

	err = checkNativeTools(filepath.Join(v.fixture, "native-tools.tsv"))
	if err != nil {
		return err
	}

	manifest, err := os.ReadFile(filepath.Join(v.fixture, "native-tools.tsv"))
	if err != nil || receipt.Native != digest(string(manifest)) {
		return fmt.Errorf("native manifest identity changed")
	}

	err = verifyGroupProofs(inv, receipt.runtimeReceipt, 7, acceptanceMethod)
	if err != nil {
		return err
	}

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid integrated diagnostic path")
		}

		log, err := os.ReadFile(filepath.Join(v.fixture, cmd.Log))
		if err != nil || digest(string(log)) != cmd.LogDigest {
			return fmt.Errorf("integrated command diagnostic identity changed")
		}
	}

	if len(receipt.Dependencies) != 5 {
		return fmt.Errorf("incomplete integrated execution dependencies")
	}

	for file, want := range receipt.Dependencies {
		data, err := os.ReadFile(file)
		if err != nil || digest(string(data)) != want {
			return fmt.Errorf("integrated execution dependency changed")
		}
	}

	rows, err := coverageRows()
	if err != nil {
		return err
	}

	err = reconcile(inv.rows, rows)
	if err != nil {
		return err
	}

	var blocked []string

	for _, r := range rows {
		if r.status == "blocked" {
			blocked = append(blocked, r.id)
		}
	}

	if !reflect.DeepEqual(blocked, receipt.Blocked) {
		return fmt.Errorf("integrated blocker classification changed")
	}

	clean, err := command("git", "status", "--porcelain")
	if err != nil || clean != "" {
		return fmt.Errorf("integrated acceptance requires a clean candidate")
	}

	head, err := command("git", "rev-parse", "origin/dev")
	if err != nil || strings.TrimSpace(head) != receipt.Head {
		return fmt.Errorf("integrated acceptance requires canonical origin/dev candidate")
	}

	return closureCoverage(rows)
}
