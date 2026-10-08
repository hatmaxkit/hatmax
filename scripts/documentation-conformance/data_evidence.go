// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

type dataReceipt struct {
	runtimeReceipt
	Tools   map[string]string `json:"data_tools"`
	Cluster string            `json:"owned_cluster"`
}

var workbenchBindings = map[string]string{
	"configuration-api:config/config.go":                                          "effective YAML/env/flag/default precedence, expansion, missing/invalid input, flag-process exit and real validators",
	"configuration-api:settings/setting.go":                                       "real schema validation, typed defaults, registered-write rejection, empty values, cancellation, closed-pool errors and persisted raw-value parsing",
	"configuration:examples/guide/config.yaml":                                    "published YAML loaded and validated through Config and explicit capability validators",
	"configuration:examples/ticked/config.yaml":                                   "published YAML loaded and validated, including explicit authenticator enrollment settings",
	"example:docs/how-to/connect-postgres/README.md#block-1":                      "exact published YAML loaded and validated; fixture connection coordinates applied through supported flags",
	"example:docs/how-to/apply-migrations/README.md#block-1":                      "exact published SQL applied, tracked, rerun without replay and inspected in PostgreSQL",
	"example:db/readme.md#block-3":                                                "exact published SQL applied; UUID default and unique email constraint observed",
	"example:docs/tutorials/user-guide/persistence-and-migrations.md#block-6":     "exact published invoice migration applied in owned PostgreSQL schema",
	"example:docs/tutorials/user-guide/persistence-and-migrations.md#block-7":     "exact SQLC query input generated; compiled queries executed against the published schema",
	"example:docs/tutorials/user-guide/models-and-data-flow.md#block-2":           "exact NewInvoice function executed with application-owned validation; UUID, timestamps and persisted values inspected",
	"example:docs/tutorials/user-guide/models-and-data-flow.md#block-3":           "exact Update function executed; invalid candidate preserves original and valid candidate preserves identity/creation time",
	"example:docs/tutorials/user-guide/models-and-data-flow.md#block-5":           "exact DAL-to-domain mapping executed against a row returned by the generated query",
	"example:seed/readme.md#block-1":                                              "exact UserSeeder executed against its stated application schema; tracking-loss retry preserves durable user identity",
	"walkthrough:docs/how-to/apply-migrations/README.md":                          "owned schema, exact published Up SQL, repeated startup, tracking inspection and per-file failure rollback; Down remains unexecuted",
	"walkthrough:docs/how-to/connect-postgres/README.md":                          "effective published connection config, actual startup/ping, schema ownership, failed-ping/retained-pool failures and explicit caller cleanup",
	"walkthrough:docs/tutorials/user-guide/persistence-and-migrations.md":         "published invoice schema and SQLC query generation/execution, actual Ticked store suites, migrations and seeder ordering/tracking; illustrative composition fragments separately compiled",
	"walkthrough:docs/tutorials/user-guide/models-and-data-flow.md":               "exact illustrative constructor/update/mapping plus real generated SQLC insert/select in owned PostgreSQL; application validator is named fixture context",
	"walkthrough:docs/tutorials/user-guide/configuration-and-runtime-settings.md": "effective configuration and real SQL setting adapter operations; Guide memory setting changes and resets after restart; illustrative composition fragments separately compiled",
	"walkthrough:docs/tutorials/user-guide/forms-and-validation.md":               "Guide real HTTP form parsing, accepted/invalid name feedback, HTML escaping and rejected empty durable note; illustrative invoice handler separately compiled without a browser DOM claim",
}

func dataMethod(r row, discovered inventory) (string, error) {
	if _, ok := workbenchBindings[r.id]; ok {
		return "executed", nil
	}

	if strings.HasPrefix(r.id, "package:") || r.id == "example-package:examples/ticked/internal/feat/list" {
		return "package-tests", nil
	}

	if r.id == "example-package:examples/ticked/internal/dal" {
		return "compiled", nil
	}

	if strings.HasPrefix(r.id, "configuration:") || strings.HasPrefix(r.id, "configuration-api:") || strings.HasPrefix(r.id, "walkthrough:") {
		return "executed", nil
	}

	if !strings.HasPrefix(r.id, "example:") {
		return "source-inspected", nil
	}

	block, err := rowBlock(discovered, r)
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(block, "```go\n"):
		return "compiled", nil
	case strings.HasPrefix(block, "```sql\n"), strings.HasPrefix(block, "```yaml\n"), blockMethod(block) == "published command procedure":
		return "executed", nil
	default:
		return "source-inspected", nil
	}
}

func dataTools() (map[string]string, error) {
	tools := make(map[string]string)

	for _, name := range []string{"initdb", "pg_ctl", "postgres", "psql", "sqlc"} {
		argument := "--version"
		if name == "sqlc" {
			argument = "version"
		}

		version, err := command(name, argument)
		if err != nil {
			return nil, err
		}

		executable, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}

		binary, err := os.ReadFile(executable)
		if err != nil {
			return nil, err
		}

		tools[name] = strings.TrimSpace(version) + "; sha256=" + digest(string(binary))
	}

	if !strings.HasPrefix(tools["sqlc"], "v1.30.0;") {
		return nil, fmt.Errorf("data conformance requires the published DAL's SQLC v1.30.0")
	}

	return tools, nil
}

func (v *verification) bindData(r row, method, outcome string) {
	for index, p := range v.receipt.Proofs {
		if p.ID == r.id {
			v.receipt.Proofs[index] = proof{ID: r.id, Digest: r.digest, Method: method, Outcome: outcome}

			return
		}
	}

	v.record(r, method, outcome)
}

func runData(fixture string) error {
	v, discovered, err := newDataVerification(fixture)
	if err != nil {
		return err
	}

	cluster := filepath.Join(v.fixture, "postgres-data")
	if os.Getenv("HATMAX_DOC_PGDATA") != cluster || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("data conformance requires the invocation-owned PostgreSQL fixture")
	}

	_, err = os.Stat(filepath.Join(v.fixture, "data-receipts.json"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("fixture already contains data evidence")
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	err = checkDataProcedures(discovered)
	if err != nil {
		return err
	}

	for _, check := range []func(inventory) error{
		func(inv inventory) error { return v.packageGroup(inv, 3) },
		func(inv inventory) error { return compileDataContexts(v, inv) },
		v.dataWorkbench, v.tickedSQLC, v.dataGuide,
	} {
		err = check(discovered)
		if err != nil {
			return err
		}
	}

	err = v.exec(v.root, "env", "PGOPTIONS=-c search_path=migration_history", "PGPORT="+os.Getenv("DB_PORT"), "psql", "-h", os.Getenv("DB_HOST"), "-U", os.Getenv("DB_USER"), "-d", os.Getenv("DB_NAME"), "-c", `\d notes`)
	if err != nil {
		return err
	}

	v.receipt.Observations = append(v.receipt.Observations,
		"fresh owned PostgreSQL cluster on a unique Unix socket with TCP disabled; documented role/database names, isolated schemas and no production service",
		"exact published migrations/query/model/seeder bytes executed in a real SQLC/PostgreSQL workbench; caller-owned fixture adapters and historical rollback/retry semantics verified",
		"published psql table inspection executed with the fixture socket/port and explicit search_path; SQLC v1.30.0 configuration generates and compiles Ticked DAL",
	)

	for _, r := range discovered.rows {
		if r.slice != 3 {
			continue
		}

		if outcome, ok := workbenchBindings[r.id]; ok {
			v.bindData(r, "executed", outcome)

			continue
		}

		switch r.id {
		case "configuration:examples/ticked/sqlc.yaml":
			v.bindData(r, "executed", "exact published SQLC config generated the real queries/migrations and compiled its output through SQLC v1.30.0")
		case "example-package:examples/ticked/internal/dal":
			v.bindData(r, "compiled", "published DAL built; generated output compiled; actual Ticked store suites independently execute persistence contracts")
		case "example:docs/how-to/apply-migrations/README.md#block-3":
			v.bindData(r, "executed", "published psql inspection executed with owned socket/port/schema substitutions; table exists after repeated startup")
		case "example:examples/guide/README.md#block-2", "example:examples/guide/README.md#block-4":
			v.bindData(r, "executed", "Guide built and launched directly with published config and explicit owned database/HTTP coordinates; note persistence and greeting update/reset observed")
		default:
			method, err := dataMethod(r, discovered)
			if err != nil {
				return err
			}

			if method == "source-inspected" {
				v.bindData(r, method, "T3.1 source/API comparison; conceptual notation or reference page; execution bindings remain separate")
			}
		}
	}

	err = verifyGroupProofs(discovered, v.receipt, 3, dataMethod)
	if err != nil {
		return err
	}

	output, err := json.MarshalIndent(dataReceipt{runtimeReceipt: v.receipt, Tools: tools, Cluster: cluster}, "", "  ")
	if err != nil {
		return err
	}

	err = writeFile(filepath.Join(v.fixture, "data-receipts.json"), string(output)+"\n")
	if err != nil {
		return err
	}

	fmt.Printf("Data/configuration receipts passed: %d bindings; real PostgreSQL, SQLC, runtime settings, migration/seed history and Guide restart workflows.\n", len(v.receipt.Proofs))

	return nil
}

func checkDataEvidence(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var receipt dataReceipt

	err = json.Unmarshal(data, &receipt)
	if err != nil {
		return err
	}

	v, discovered, err := newDataVerification(filepath.Dir(file))
	if err != nil {
		return err
	}

	tools, err := dataTools()
	if err != nil {
		return err
	}

	if receipt.Head != v.receipt.Head || receipt.Inputs != v.receipt.Inputs || receipt.Go != v.receipt.Go || receipt.Compiler != v.receipt.Compiler || !reflect.DeepEqual(receipt.Tools, tools) {
		return fmt.Errorf("data evidence source/head/tool identity mismatch")
	}

	err = verifyGroupProofs(discovered, receipt.runtimeReceipt, 3, dataMethod)
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
		if r.slice != 3 {
			continue
		}

		method, err := dataMethod(r, discovered)
		if err != nil {
			return err
		}

		if r.status != method || r.receipt != "slice-3/data-receipts.json" {
			return fmt.Errorf("coverage status disagrees with actual data verification: %s", r.id)
		}
	}

	for _, cmd := range receipt.Commands {
		if filepath.Base(cmd.Log) != cmd.Log {
			return fmt.Errorf("invalid data diagnostic path")
		}

		output, err := os.ReadFile(filepath.Join(filepath.Dir(file), cmd.Log))
		if err != nil {
			return err
		}

		if digest(string(output)) != cmd.LogDigest {
			return fmt.Errorf("data command diagnostic changed")
		}
	}

	if len(receipt.Processes) != 2 {
		return fmt.Errorf("missing Guide data process evidence")
	}

	for _, process := range receipt.Processes {
		configuration, err := os.ReadFile(filepath.Join(process.Directory, "config.yaml"))
		if err != nil {
			return err
		}

		if process.Configuration != digest(string(configuration)) || process.Actual != "signal: interrupt" || process.Expected != "signal interrupt; coordinator absent" {
			return fmt.Errorf("Guide data process configuration/completion mismatch")
		}
	}

	if receipt.Cluster != filepath.Join(v.fixture, "postgres-data") {
		return fmt.Errorf("data cluster ownership mismatch")
	}

	_, err = os.Stat(filepath.Join(receipt.Cluster, "postmaster.pid"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("owned PostgreSQL cluster has not stopped")
	}

	stopped, err := os.ReadFile(filepath.Join(v.fixture, "postgres-stop.log"))
	if err != nil || !strings.Contains(string(stopped), "server stopped") {
		return fmt.Errorf("missing owned PostgreSQL shutdown evidence")
	}

	fmt.Printf("Exact-head data evidence reconciled: %d bindings; owned cluster and HTTP processes stopped.\n", len(receipt.Proofs))

	return nil
}
