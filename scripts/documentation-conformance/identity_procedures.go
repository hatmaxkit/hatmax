// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"strings"
)

var identityProcedures = map[string]string{
	"example:docs/reference/authentication/README.md#block-3": `export CHROMIUM_BIN="$(command -v chromium)"
export DB_HOST=/path/to/owned/postgresql/socket DB_PORT=5432
export DB_USER=postgres DB_NAME=postgres DB_PASSWORD=''
go test -race -tags=browser ./examples/ticked/internal/web \
  -run '^TestAuthenticatorBrowser$' -count=1 -timeout=240s -v`,
	"example:docs/reference/authentication/README.md#block-5": `go test -tags=browser -race -run '^TestAccountRecoveryBrowser$' \
  -count=1 -timeout=360s ./examples/ticked/internal/web`,
	"example:docs/reference/authentication/README.md#block-6": `go test -tags=browser -race -v \
  -run '^(TestAuthenticatorBrowser|TestAccountRecoveryBrowser)$' \
  -count=1 -timeout=600s ./examples/ticked/internal/web`,
	"example:examples/ticked/README.md#block-2": "createuser --pwprompt dev\ncreatedb --owner=dev tickedhm",
	"example:examples/ticked/README.md#block-4": "make build\n./ticked",
	"example:examples/ticked/README.md#block-5": "make build      # Build the binary\nmake run-fg     # Build and run in the foreground using the Makefile's DB_* values\nmake clean      # Remove binary and logs",
	"example:examples/ticked/README.md#block-6": "make db-init    # Create the selected owned database\nmake sqlc       # Regenerate sqlc queries",
	"example:examples/ticked/README.md#block-7": "make test       # Run tests",
	"example:examples/ticked/README.md#block-9": "TICKED_SERVER_PORT=:9000 ./ticked\nTICKED_DATABASE_HOST=db.example.com ./ticked",
}

func checkIdentityProcedures(discovered inventory) error {
	found := 0

	for _, r := range discovered.rows {
		if r.slice != 4 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if blockMethod(block) != "published command procedure" {
			continue
		}

		expected, ok := identityProcedures[r.id]
		if !ok || strings.TrimSpace(blockBody(block)) != expected {
			return fmt.Errorf("unaccounted identity command procedure: %s", r.id)
		}

		found++
	}

	if found != len(identityProcedures) {
		return fmt.Errorf("missing published identity command procedure")
	}

	return nil
}
