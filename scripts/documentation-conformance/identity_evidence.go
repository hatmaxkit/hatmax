// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

func identityMethod(r row, discovered inventory) (string, error) {
	if strings.HasPrefix(r.id, "package:") || strings.HasPrefix(r.id, "example-package:") {
		return "package-tests", nil
	}

	if strings.HasPrefix(r.id, "command:") || strings.HasPrefix(r.id, "example-composition:") || strings.HasPrefix(r.id, "walkthrough:") {
		return "executed", nil
	}

	if !strings.HasPrefix(r.id, "example:") {
		return "source-inspected", nil
	}

	block, err := rowBlock(discovered, r)
	if err != nil {
		return "", err
	}

	if _, ok := identityOutcomes[r.id]; ok {
		return "executed", nil
	}

	switch {
	case r.id == "example:examples/ticked/README.md#block-3", blockMethod(block) == "published command procedure", strings.HasPrefix(block, "```javascript\n"), strings.HasPrefix(block, "```yaml\n"):
		return "executed", nil
	case strings.HasPrefix(block, "```go\n"):
		return "compiled", nil
	default:
		return "source-inspected", nil
	}
}

func identityTools() (map[string]string, error) {
	tools, err := dataTools()
	if err != nil {
		return nil, err
	}

	for _, name := range []string{"createuser", "createdb", "chromium"} {
		executable := name
		if name == "chromium" {
			executable = os.Getenv("CHROMIUM_BIN")
			if executable == "" {
				return nil, fmt.Errorf("identity conformance requires CHROMIUM_BIN")
			}
		}

		version, err := command(executable, "--version")
		if err != nil {
			return nil, err
		}

		file, err := exec.LookPath(executable)
		if err != nil {
			return nil, err
		}

		binary, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		if name == "chromium" && (len(binary) < 4 || !bytes.Equal(binary[:4], []byte{0x7f, 'E', 'L', 'F'})) {
			return nil, fmt.Errorf("CHROMIUM_BIN must name the native browser image for executable identity evidence")
		}

		tools[name] = strings.TrimSpace(version) + "; sha256=" + digest(string(binary))
	}

	return tools, nil
}

var browserSelectors = [][]string{
	{"test", "-race", "-tags=browser", "./examples/ticked/internal/web", "-run", "^TestAuthenticatorBrowser$", "-count=1", "-timeout=240s", "-v"},
	{"test", "-tags=browser", "-race", "-run", "^TestAccountRecoveryBrowser$", "-count=1", "-timeout=360s", "./examples/ticked/internal/web"},
	{"test", "-tags=browser", "-race", "-v", "-run", "^(TestAuthenticatorBrowser|TestAccountRecoveryBrowser)$", "-count=1", "-timeout=600s", "./examples/ticked/internal/web"},
}

func checkBrowserLog(index int, output []byte) error {
	if bytes.Contains(output, []byte("[no tests to run]")) || bytes.Contains(output, []byte("(cached)")) || bytes.Contains(output, []byte("--- SKIP:")) {
		return fmt.Errorf("missing actual browser execution")
	}

	// The individual recovery command intentionally omits -v. Go's package
	// result still proves a nonempty exact selector after the combined verbose
	// command separately verifies both named tests and browser stages.
	if index == 1 {
		if !bytes.Contains(output, []byte("ok  \thatmax.adrianpk.com/examples/ticked/internal/web")) && !bytes.Contains(output, []byte("ok\thatmax.adrianpk.com/examples/ticked/internal/web")) {
			return fmt.Errorf("missing recovery package completion")
		}

		return nil
	}

	if !bytes.Contains(output, []byte("--- PASS: TestAuthenticatorBrowser")) || !bytes.Contains(output, []byte("PASS actual registration/sign-in, safe cookie and strong route")) || !bytes.Contains(output, []byte("BROWSER Chrome/")) {
		return fmt.Errorf("missing actual authenticator browser completion")
	}

	if index == 2 && !bytes.Contains(output, []byte("--- PASS: TestAccountRecoveryBrowser")) {
		return fmt.Errorf("missing actual recovery browser completion")
	}

	return nil
}

func (v *verification) identityBrowsers(discovered inventory) error {
	r := row{id: "example:docs/reference/authentication/README.md#block-2", source: "docs/reference/authentication/README.md#block-2"}

	block, err := rowBlock(discovered, r)
	if err != nil {
		return err
	}

	published := filepath.Join(v.fixture, "registration.js")

	err = writeFile(published, blockBody(block))
	if err != nil {
		return err
	}

	err = v.exec(v.root, "go", "test", "-tags=browser", "-race", "-run", "^TestBrowserPipe$", "-count=1", "-timeout=30s", "-v", "./examples/ticked/internal/web")
	if err != nil {
		return err
	}

	err = v.exec(v.root, "golangci-lint", "run", "--default=none", "--enable=nlreturn", "--enable=noinlineerr", "--enable=wsl_v5", "--build-tags=browser", "examples/ticked/internal/web/browser_driver_test.go", "examples/ticked/internal/web/browser_pipe_test.go")
	if err != nil {
		return err
	}

	for index, selector := range browserSelectors {
		arguments := append([]string{"HATMAX_DOC_REGISTRATION=" + published, "go"}, selector...)

		err = v.execBound(v.root, 11*time.Minute, "", "env", arguments...)
		if err != nil {
			return err
		}

		log := v.receipt.Commands[len(v.receipt.Commands)-1].Log

		output, err := os.ReadFile(filepath.Join(v.fixture, log))
		if err != nil {
			return err
		}

		err = checkBrowserLog(index, output)
		if err != nil {
			return err
		}
	}

	v.receipt.Observations = append(v.receipt.Observations,
		"all three exact published uncached browser selectors completed with race detection; each owns its schema, served production handlers, Chromium profile/private pipe and shutdown",
		"exact published parseCreationOptionsFromJSON/navigator.credentials.create/toJSON/fetch fragment executed for real enrollment and replay denied; actual signatures/proof precede protected access",
		"real cookie rotation/retirement, WebAuthn/TOTP/one-use backup MFA, proof freshness/stronger-policy denials, retained factors, neutral captured-mail recovery, foreign-origin rejection and lost committed response/replay denial",
	)

	return nil
}

func runIdentity(fixture string) error {
	v, discovered, err := newGroupVerification(fixture, 4)
	if err != nil {
		return err
	}

	cluster := filepath.Join(v.fixture, "postgres-data")
	if os.Getenv("HATMAX_DOC_PGDATA") != cluster || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("identity conformance requires the invocation-owned PostgreSQL fixture")
	}

	_, err = os.Stat(filepath.Join(v.fixture, "identity-receipts.json"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("fixture already contains identity evidence")
	}

	tools, err := identityTools()
	if err != nil {
		return err
	}

	err = checkIdentityProcedures(discovered)
	if err != nil {
		return err
	}

	for _, check := range []func(inventory) error{
		func(inv inventory) error { return v.packageGroup(inv, 4) },
		func(inv inventory) error { return compileIdentityContexts(v, inv) },
		v.identityTicked, v.identityBrowsers,
	} {
		err = check(discovered)
		if err != nil {
			return err
		}
	}

	for _, r := range discovered.rows {
		if r.slice != 4 {
			continue
		}

		method, err := identityMethod(r, discovered)
		if err != nil {
			return err
		}

		switch method {
		case "executed":
			if strings.HasPrefix(r.id, "walkthrough:") {
				v.bindData(r, method, "actual Ticked production handlers/core/PostgreSQL and navigator journeys execute sign-in, management and recovery; illustrative invoice wiring is separately compiled, hardware/provider/deployment assurance excluded")
			} else if !strings.HasPrefix(r.id, "example:crypto/") && r.id != "example:examples/ticked/README.md#block-3" {
				v.bindData(r, method, "published command/configuration/entrypoint or browser fragment executed with owned database/HTTP/profile substitutions; observed proof, cookie, persistence and shutdown outcomes recorded")
			}
		case "source-inspected":
			v.bindData(r, method, "T4.1 current source/API comparison and navigation; reference page or conceptual notation, separate from workflow execution")
		}
	}

	err = verifyGroupProofs(discovered, v.receipt, 4, identityMethod)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(dataReceipt{runtimeReceipt: v.receipt, Tools: tools, Cluster: cluster}, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(v.fixture, "identity-receipts.json"), string(output)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Identity/recovery receipts passed: %d bindings; actual Ticked entrypoint, native Chromium, published fragments and proof/denial/recovery workflows.\n", len(v.receipt.Proofs))

	return nil
}

func checkIdentityEvidence(file string) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var receipt dataReceipt

	err = json.Unmarshal(contents, &receipt)
	if err != nil {
		return err
	}

	v, discovered, err := newGroupVerification(filepath.Dir(file), 4)
	if err != nil {
		return err
	}

	tools, err := identityTools()
	if err != nil {
		return err
	}

	if receipt.Head != v.receipt.Head || receipt.Inputs != v.receipt.Inputs || receipt.Go != v.receipt.Go || receipt.Compiler != v.receipt.Compiler || !reflect.DeepEqual(receipt.Tools, tools) {
		return fmt.Errorf("identity evidence source/head/tool identity mismatch")
	}

	err = verifyGroupProofs(discovered, receipt.runtimeReceipt, 4, identityMethod)
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
		if r.slice != 4 {
			continue
		}

		method, err := identityMethod(r, discovered)
		if err != nil {
			return err
		}

		if r.status != method || r.receipt != "slice-4/identity-receipts.json" {
			return fmt.Errorf("coverage status disagrees with actual identity verification: %s", r.id)
		}
	}

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid identity diagnostic path")
		}

		output, err := os.ReadFile(filepath.Join(v.fixture, cmd.Log))
		if err != nil || digest(string(output)) != cmd.LogDigest {
			return fmt.Errorf("identity command diagnostic changed")
		}
	}

	err = verifyIdentityCommands(receipt.runtimeReceipt, v.fixture)
	if err != nil {
		return err
	}

	if len(receipt.Processes) != 3 {
		return fmt.Errorf("missing Ticked startup/restart/Make process evidence")
	}

	for index, process := range receipt.Processes {
		expectedFile := filepath.Join(v.fixture, fmt.Sprintf("ticked-config-%d.yaml", index+1))

		configuration, err := os.ReadFile(expectedFile)
		if err != nil || process.ConfigurationFile != expectedFile || process.Configuration != digest(string(configuration)) {
			return fmt.Errorf("Ticked configuration identity mismatch")
		}

		if index < 2 && (process.Actual != "exit 0" || process.Expected != "exit 0 after Ctrl+C") || index == 2 && (process.Expected != "owned make group interrupted; application coordinated shutdown" || process.Actual != "exit 0" && process.Actual != "exit status 2") {
			return fmt.Errorf("Ticked process completion mismatch")
		}

		if filepath.Base(process.Log) != process.Log {
			return fmt.Errorf("invalid Ticked diagnostic path")
		}

		log, err := os.ReadFile(filepath.Join(v.fixture, process.Log))
		if err != nil || process.LogDigest != digest(string(log)) || !strings.Contains(string(log), "Shutting down gracefully") || !strings.Contains(string(log), "Closing database connection") || strings.Contains(string(log), "error stopping component") || strings.Contains(string(log), "server shutdown failed") {
			return fmt.Errorf("Ticked coordinated shutdown evidence incomplete")
		}
	}

	if receipt.Cluster != filepath.Join(v.fixture, "postgres-data") {
		return fmt.Errorf("identity cluster ownership mismatch")
	}

	_, err = os.Stat(filepath.Join(receipt.Cluster, "postmaster.pid"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("owned identity PostgreSQL cluster has not stopped")
	}

	stopped, err := os.ReadFile(filepath.Join(v.fixture, "postgres-stop.log"))
	if err != nil || !strings.Contains(string(stopped), "server stopped") {
		return fmt.Errorf("missing owned identity PostgreSQL shutdown evidence")
	}

	fmt.Printf("Exact-head identity evidence reconciled: %d bindings; owned database, Ticked and Chromium processes stopped.\n", len(receipt.Proofs))

	return nil
}

func verifyIdentityCommands(receipt runtimeReceipt, fixture string) error {
	next := 0

	for _, cmd := range receipt.Commands {
		if len(cmd.Arguments) < 3 || cmd.Arguments[0] != "env" || !strings.HasPrefix(cmd.Arguments[1], "HATMAX_DOC_REGISTRATION=") {
			continue
		}

		if next >= len(browserSelectors) {
			return fmt.Errorf("duplicate browser command evidence")
		}

		want := append([]string{"env", "HATMAX_DOC_REGISTRATION=" + filepath.Join(fixture, "registration.js"), "go"}, browserSelectors[next]...)
		if !reflect.DeepEqual(cmd.Arguments, want) {
			return fmt.Errorf("changed browser command selector")
		}

		output, err := os.ReadFile(filepath.Join(fixture, cmd.Log))
		if err != nil {
			return err
		}

		err = checkBrowserLog(next, output)
		if err != nil {
			return err
		}

		next++
	}

	if next != len(browserSelectors) {
		return fmt.Errorf("missing published browser command evidence")
	}

	return nil
}
