// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

type generatorReceipt struct {
	dataReceipt
	Fixtures map[string]string `json:"fixture_inputs"`
}

func generatorMethod(r row, discovered inventory) (string, error) {
	if strings.HasPrefix(r.id, "package:") || strings.HasPrefix(r.id, "implementation:") {
		return "package-tests", nil
	}

	if strings.HasPrefix(r.id, "command:") || strings.HasPrefix(r.id, "help:") || strings.HasPrefix(r.id, "walkthrough:") {
		return "executed", nil
	}

	if !strings.HasPrefix(r.id, "example:") {
		return "source-inspected", nil
	}

	// Every current generator fence is a published command or request. New ones
	// need a deliberate workflow binding; source review cannot fill that gap.
	for page, count := range map[string]int{"docs/reference/generator/README.md": 3, generatorGuide: 9} {
		for number := 1; number <= count; number++ {
			if r.id == fmt.Sprintf("example:%s#block-%d", page, number) {
				return "executed", nil
			}
		}
	}

	_, err := rowBlock(discovered, r)
	if err != nil {
		return "", err
	}

	return "", fmt.Errorf("unaccounted executable generator example: %s", r.id)
}

func generatorFixtureInputs(fixture string) (map[string]string, error) {
	inputs := make(map[string]string)

	for _, name := range []string{"go.mod", "go.sum", "workflow_test.go", "hm", "hatmax"} {
		file := filepath.Join("generator-workbench", name)

		contents, err := os.ReadFile(filepath.Join(fixture, file))
		if err != nil {
			return nil, err
		}

		inputs[filepath.ToSlash(file)] = digest(string(contents))
	}

	// The copied canonical project is executable context, even though product
	// discovery deliberately excludes testdata from published surface inventory.
	source := "generator/project/testdata/supported"

	err := filepath.WalkDir(source, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		contents, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		inputs["source-fixture/"+filepath.ToSlash(file)] = digest(string(contents))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return inputs, nil
}

func runGenerator(fixture string) error {
	v, discovered, err := newGroupVerification(fixture, 6)
	if err != nil {
		return err
	}

	cluster := filepath.Join(v.fixture, "postgres-data")
	if os.Getenv("HATMAX_DOC_PGDATA") != cluster || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("generator checks require invocation-owned PostgreSQL")
	}

	_, err = os.Stat(filepath.Join(v.fixture, "generator-receipts.json"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("fixture already contains generator evidence")
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	for _, check := range []func(inventory) error{
		func(inv inventory) error { return v.packageGroup(inv, 6) },
		v.generatorWorkbench,
	} {
		err = check(discovered)
		if err != nil {
			return err
		}
	}

	v.receipt.Observations = append(v.receipt.Observations,
		"fresh invocation-owned PostgreSQL cluster and native Go/shell project commands; generated files and conversation state isolated in owned temporary storage",
		"exact published requests passed through fixture interpretation into production coordinators, persistence, planning, approval, rendering, conformance and generated-project validation",
		"real hm/hatmax entrypoints, source install, owned PTY help/quit, conversation selection and no-run playground preparation; no inference request or resident daemon mutation",
		"Book positive/negative illustrative files and backend prerequisites compared to current source; fixture interpretation and backend protocol suites do not establish authenticated model acceptance",
		"bare and initial-feature scaffolds complete feature evolution; required timestamps and minimum-length validation pass real generated-project checks",
	)

	for _, r := range discovered.rows {
		if r.slice != 6 || strings.HasPrefix(r.id, "package:") || strings.HasPrefix(r.id, "implementation:") {
			continue
		}

		method, err := generatorMethod(r, discovered)
		if err != nil {
			return err
		}

		outcome := "T6.1 current source, selected Book contracts and provider-prerequisite comparison; illustrative positive/negative Book fragments are not standalone programs"
		if method == "executed" {
			outcome = "named native command and generated workflow tests passed: exact published prompts, owned source install/state/playground, PTY help/quit and production-kernel approval/resume/mutation controls; interpreter fixture, no model acceptance claim"
		}

		v.bindData(r, method, outcome)
	}

	err = verifyGroupProofs(discovered, v.receipt, 6, generatorMethod)
	if err != nil {
		return err
	}

	err = verifyGeneratorCommands(v.receipt, v.fixture)
	if err != nil {
		return err
	}

	inputs, err := generatorFixtureInputs(v.fixture)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(generatorReceipt{dataReceipt: dataReceipt{runtimeReceipt: v.receipt, Tools: tools, Cluster: cluster}, Fixtures: inputs}, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(v.fixture, "generator-receipts.json"), string(output)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Generator receipts reconciled: %d bindings with completed native command and generated-project evidence.\n", len(v.receipt.Proofs))

	return nil
}

func verifyGeneratorCommands(receipt runtimeReceipt, fixture string) error {
	workflow := false

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid generator diagnostic path")
		}

		output, err := os.ReadFile(filepath.Join(fixture, cmd.Log))
		if err != nil || digest(string(output)) != cmd.LogDigest {
			return fmt.Errorf("generator diagnostic identity changed")
		}

		if !slices.Equal(cmd.Arguments, []string{"go", "test", "-json", "-count=1", "-timeout=9m", "./..."}) || !strings.HasSuffix(cmd.Directory, "/generator-workbench") {
			continue
		}

		for _, name := range []string{"TestBuilder", "TestHeadless", "TestCommands"} {
			if !namedTestPassed(output, name) {
				return fmt.Errorf("missing executed generator workflow: %s", name)
			}
		}

		if !strings.Contains(string(output), "fresh-scaffold evolution completed: resumed conversation and project checks passed") {
			return fmt.Errorf("fresh-scaffold completion evidence missing")
		}

		if !strings.Contains(string(output), "timestamp evolution completed: SQLC and generated project checks passed") {
			return fmt.Errorf("timestamp completion evidence missing")
		}

		if !strings.Contains(string(output), "validation evolution completed: generated model tests and project lint passed") {
			return fmt.Errorf("validation completion evidence missing")
		}

		workflow = true
	}

	if !workflow {
		return fmt.Errorf("generator workflow execution missing")
	}

	return nil
}

func checkGeneratorEvidence(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var receipt generatorReceipt

	err = json.Unmarshal(data, &receipt)
	if err != nil {
		return err
	}

	v, discovered, err := newGroupVerification(filepath.Dir(file), 6)
	if err != nil {
		return err
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	inputs, err := generatorFixtureInputs(v.fixture)
	if err != nil {
		return err
	}

	if receipt.Head != v.receipt.Head || receipt.Inputs != v.receipt.Inputs || receipt.Go != v.receipt.Go || receipt.Compiler != v.receipt.Compiler || !reflect.DeepEqual(receipt.Tools, tools) || !reflect.DeepEqual(receipt.Fixtures, inputs) {
		return fmt.Errorf("generator source/head/tool/fixture identity mismatch")
	}

	err = verifyGroupProofs(discovered, receipt.runtimeReceipt, 6, generatorMethod)
	if err != nil {
		return err
	}

	err = verifyGeneratorCommands(receipt.runtimeReceipt, v.fixture)
	if err != nil {
		return err
	}

	coverage, err := os.ReadFile(coveragePath)
	if err != nil {
		return err
	}

	rows, err := parseRows(string(coverage))
	if err != nil {
		return err
	}

	for _, r := range rows {
		if r.slice != 6 {
			continue
		}

		method, err := generatorMethod(r, discovered)
		if err != nil {
			return err
		}

		if r.status != method || r.receipt != "slice-6/generator-receipts.json" {
			return fmt.Errorf("coverage disagrees with generator verification: %s", r.id)
		}
	}

	if receipt.Cluster != filepath.Join(v.fixture, "postgres-data") {
		return fmt.Errorf("generator cluster ownership mismatch")
	}

	_, err = os.Stat(filepath.Join(receipt.Cluster, "postmaster.pid"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("owned generator cluster has not stopped")
	}

	stopped, err := os.ReadFile(filepath.Join(v.fixture, "postgres-stop.log"))
	if err != nil || !strings.Contains(string(stopped), "server stopped") {
		return fmt.Errorf("owned generator shutdown evidence missing")
	}

	fmt.Printf("Exact-head generator evidence reconciled: %d completed bindings; owned cluster stopped.\n", len(receipt.Proofs))

	return nil
}
