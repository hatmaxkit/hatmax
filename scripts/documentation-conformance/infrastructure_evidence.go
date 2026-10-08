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

type infrastructureReceipt struct {
	dataReceipt
	Fixtures map[string]string `json:"fixture_inputs"`
}

func infrastructureMethod(r row, discovered inventory) (string, error) {
	if _, ok := infrastructureBindings[r.id]; ok {
		return "executed", nil
	}

	if strings.HasPrefix(r.id, "package:") || strings.HasPrefix(r.id, "implementation:") {
		return "package-tests", nil
	}

	if !strings.HasPrefix(r.id, "example:") {
		return "source-inspected", nil
	}

	block, err := rowBlock(discovered, r)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(block, "```go\n") {
		return "compiled", nil
	}

	if blockMethod(block) == "published command procedure" {
		return "", fmt.Errorf("unaccounted executable infrastructure procedure: %s", r.id)
	}

	return "source-inspected", nil
}

func infrastructureFixtureInputs(fixture string) (map[string]string, error) {
	inputs := make(map[string]string)

	for _, name := range []string{"infrastructure-contexts", "infrastructure-workbench", "published-helper"} {
		err := filepath.WalkDir(filepath.Join(fixture, name), func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				return nil
			}

			if !slices.Contains([]string{".go", ".mod", ".sum", ".yaml"}, filepath.Ext(file)) {
				return fmt.Errorf("unexpected infrastructure fixture input: %s", file)
			}

			relative, err := filepath.Rel(fixture, file)
			if err != nil {
				return err
			}

			contents, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			inputs[filepath.ToSlash(relative)] = digest(string(contents))

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return inputs, nil
}

func runInfrastructure(fixture string) error {
	v, discovered, err := newGroupVerification(fixture, 5)
	if err != nil {
		return err
	}

	cluster := filepath.Join(v.fixture, "postgres-data")
	if os.Getenv("HATMAX_DOC_PGDATA") != cluster || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("infrastructure checks require the invocation-owned PostgreSQL fixture")
	}

	_, err = os.Stat(filepath.Join(v.fixture, "infrastructure-receipts.json"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("fixture already contains infrastructure evidence")
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	for _, check := range []func(inventory) error{
		func(inv inventory) error { return v.packageGroup(inv, 5) },
		func(inv inventory) error { return compileInfrastructureContexts(v, inv) },
		v.infrastructureWorkbench,
	} {
		err = check(discovered)
		if err != nil {
			return err
		}
	}

	v.receipt.Observations = append(v.receipt.Observations,
		"fresh owned PostgreSQL Unix-socket cluster, invocation-private schemas and fixture-only data; no production service or external account",
		"exact published programs and local fragments executed with real mail capture, PNG readback, PostgreSQL acknowledgement/slot/retry state and cooperative shutdown; provider-only guidance remains source-reviewed",
		"Go-owned sinks and pools closed in bounded tests; testhelper cleanup inspected through a separate PostgreSQL command and no generated test schema remained",
	)

	for _, r := range discovered.rows {
		if r.slice != 5 {
			continue
		}

		method, err := infrastructureMethod(r, discovered)
		if err != nil {
			return err
		}

		if method == "source-inspected" {
			v.bindData(r, method, "T5.1 current source/API comparison; conceptual notation, reference or provider guidance; actual local execution bindings remain separate")
		}
	}

	err = verifyGroupProofs(discovered, v.receipt, 5, infrastructureMethod)
	if err != nil {
		return err
	}

	err = verifyInfrastructureCommands(v.receipt, v.fixture, discovered)
	if err != nil {
		return err
	}

	inputs, err := infrastructureFixtureInputs(v.fixture)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(infrastructureReceipt{dataReceipt: dataReceipt{runtimeReceipt: v.receipt, Tools: tools, Cluster: cluster}, Fixtures: inputs}, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(v.fixture, "infrastructure-receipts.json"), string(output)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Infrastructure/helper receipts passed: %d bindings; owned mail, images, PostgreSQL events/jobs, telemetry and helper workflows.\n", len(v.receipt.Proofs))

	return nil
}

func verifyInfrastructureCommands(receipt runtimeReceipt, fixture string, discovered inventory) error {
	var workflow, cleanup bool

	programs := make(map[string]bool)
	helperCommands := make(map[string]bool)

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid infrastructure diagnostic path")
		}

		output, err := os.ReadFile(filepath.Join(fixture, cmd.Log))
		if err != nil || digest(string(output)) != cmd.LogDigest {
			return fmt.Errorf("infrastructure diagnostic identity changed")
		}

		if slices.Equal(cmd.Arguments, []string{"go", "test", "-json", "-count=1", "-timeout=90s", "./..."}) && strings.HasSuffix(cmd.Directory, "/infrastructure-workbench") {
			for _, name := range []string{"TestWelcome", "TestDatabase", "TestReadback", "TestPolling", "TestMail", "TestImages", "TestEvents", "TestJobs", "TestStop", "TestClock", "TestConfiguration"} {
				if !namedTestPassed(output, name) {
					return fmt.Errorf("missing executed infrastructure workflow: %s", name)
				}
			}

			workflow = true
		}

		if len(cmd.Arguments) == 3 && slices.Equal(cmd.Arguments[:2], []string{"go", "run"}) && strings.HasSuffix(cmd.Directory, "/infrastructure-workbench") {
			owner := strings.TrimPrefix(cmd.Arguments[2], "./")
			if expected, ok := infrastructureProgramOutputs[owner]; ok && !strings.Contains(string(output), expected) {
				return fmt.Errorf("published program outcome differs: %s", owner)
			}

			programs[cmd.Arguments[2]] = true
		}

		if len(cmd.Arguments) == 4 && slices.Equal(cmd.Arguments[:3], []string{"bash", "-eu", "-c"}) && strings.HasSuffix(cmd.Directory, "/published-helper") && strings.Contains(string(output), "ok ") && !strings.Contains(string(output), "no tests to run") {
			helperCommands[cmd.Arguments[3]] = true
		}

		if len(cmd.Arguments) > 0 && cmd.Arguments[0] == "psql" && cmd.Arguments[len(cmd.Arguments)-1] == "SELECT COUNT(*) FROM pg_namespace WHERE nspname LIKE 'test_%'" && strings.TrimSpace(string(output)) == "0" {
			cleanup = true
		}
	}

	for _, owner := range []string{"mailer", "pubsub", "telemetry", "slug"} {
		if !programs["./"+owner] {
			return fmt.Errorf("missing published program execution: %s", owner)
		}
	}

	for _, number := range []int{2, 3} {
		body, err := sourceBlock(discovered.contents, "docs/how-to/test-with-postgres/README.md", number)
		if err != nil || !helperCommands[body] {
			return fmt.Errorf("missing exact published helper command: block %d", number)
		}
	}

	if !workflow || !cleanup {
		return fmt.Errorf("infrastructure workflow or schema cleanup evidence missing")
	}

	return nil
}

func checkInfrastructureEvidence(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var receipt infrastructureReceipt

	err = json.Unmarshal(data, &receipt)
	if err != nil {
		return err
	}

	v, discovered, err := newGroupVerification(filepath.Dir(file), 5)
	if err != nil {
		return err
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	inputs, err := infrastructureFixtureInputs(v.fixture)
	if err != nil {
		return err
	}

	if receipt.Head != v.receipt.Head || receipt.Inputs != v.receipt.Inputs || receipt.Go != v.receipt.Go || receipt.Compiler != v.receipt.Compiler || !reflect.DeepEqual(receipt.Tools, tools) || !reflect.DeepEqual(receipt.Fixtures, inputs) {
		return fmt.Errorf("infrastructure source/head/tool/fixture identity mismatch")
	}

	err = verifyGroupProofs(discovered, receipt.runtimeReceipt, 5, infrastructureMethod)
	if err != nil {
		return err
	}

	err = verifyInfrastructureCommands(receipt.runtimeReceipt, v.fixture, discovered)
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
		if r.slice != 5 {
			continue
		}

		method, err := infrastructureMethod(r, discovered)
		if err != nil {
			return err
		}

		if r.status != method || r.receipt != "slice-5/infrastructure-receipts.json" {
			return fmt.Errorf("coverage disagrees with actual infrastructure verification: %s", r.id)
		}
	}

	if receipt.Cluster != filepath.Join(v.fixture, "postgres-data") {
		return fmt.Errorf("infrastructure cluster ownership mismatch")
	}

	_, err = os.Stat(filepath.Join(receipt.Cluster, "postmaster.pid"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("owned infrastructure cluster has not stopped")
	}

	stopped, err := os.ReadFile(filepath.Join(v.fixture, "postgres-stop.log"))
	if err != nil || !strings.Contains(string(stopped), "server stopped") {
		return fmt.Errorf("owned infrastructure shutdown evidence missing")
	}

	fmt.Printf("Exact-head infrastructure evidence reconciled: %d bindings; owned sinks, schemas and cluster stopped.\n", len(receipt.Proofs))

	return nil
}
