// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"strings"
)

// Commands use an immutable local clone and owned ports in the gate. Reject new
// instructions until the execution harness accounts for their actual behavior.
var procedureExpectations = map[string]string{
	"example:docs/how-to/bootstrap-application/README.md#block-1": `git clone --branch dev https://forge.adrianpk.com/hatmax/hatmax.git hatmax
mkdir myapp
cd myapp
go mod init example.com/myapp
go mod edit -replace=hatmax.adrianpk.com=../hatmax
go get hatmax.adrianpk.com
git -C ../hatmax rev-parse HEAD`,
	"example:docs/how-to/bootstrap-application/README.md#block-4": "go mod tidy\ngo run .",
	"example:docs/how-to/bootstrap-application/README.md#block-5": "curl -fsS http://localhost:8080/ping",
	"example:examples/guide/README.md#block-1":                    "go run .",
	"example:examples/guide/README.md#block-3":                    "curl -fsS http://localhost:8080/ping\ncurl -fsS -H 'Origin: http://localhost:8080' -d 'name=Alice' http://localhost:8080/name",
}

func checkProcedures(discovered inventory) error {
	found := 0

	for _, r := range discovered.rows {
		if r.slice != 2 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if blockMethod(block) != "published command procedure" {
			continue
		}

		expected, ok := procedureExpectations[r.id]
		if !ok || strings.TrimSpace(blockBody(block)) != expected {
			return fmt.Errorf("unaccounted command procedure: %s", r.id)
		}

		found++
	}

	if found != len(procedureExpectations) {
		return fmt.Errorf("missing published command procedure")
	}

	return nil
}

var dataProcedureExpectations = map[string]string{
	"example:docs/how-to/apply-migrations/README.md#block-3": "psql -h localhost -U dev -d myapp -c '\\d notes'",
	"example:examples/guide/README.md#block-2":               "GUIDE_DATABASE_ENABLED=true go run .",
	"example:examples/guide/README.md#block-4":               "curl -fsS http://localhost:8080/greeting\ncurl -fsS -H 'Origin: http://localhost:8080' -d 'greeting=Welcome' http://localhost:8080/greeting\ncurl -fsS http://localhost:8080/greeting",
}

func checkDataProcedures(discovered inventory) error {
	found := 0

	for _, r := range discovered.rows {
		if r.slice != 3 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if blockMethod(block) != "published command procedure" {
			continue
		}

		expected, ok := dataProcedureExpectations[r.id]
		if !ok || strings.TrimSpace(blockBody(block)) != expected {
			return fmt.Errorf("unaccounted data command procedure: %s", r.id)
		}

		found++
	}

	if found != len(dataProcedureExpectations) {
		return fmt.Errorf("missing published data command procedure")
	}

	return nil
}
