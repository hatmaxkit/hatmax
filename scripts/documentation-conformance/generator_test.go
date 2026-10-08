// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// All named workflows must run uncached; package-only, skipped or altered logs
// cannot substitute for generated-project and actual terminal execution.
func TestGeneratorEvidence(t *testing.T) {
	complete := "{\"Action\":\"pass\",\"Test\":\"TestBuilder\"}\n{\"Action\":\"pass\",\"Test\":\"TestHeadless\"}\n{\"Action\":\"pass\",\"Test\":\"TestCommands\"}\nfresh-scaffold evolution completed: resumed conversation and project checks passed\ntimestamp evolution completed: SQLC and generated project checks passed\nvalidation evolution completed: generated model tests and project lint passed\n"
	for _, tc := range []struct {
		name, output  string
		tamper, valid bool
	}{
		{"complete", complete, false, true},
		{"missing commands", strings.ReplaceAll(complete, "TestCommands", "Other"), false, false},
		{"old failure evidence", strings.ReplaceAll(complete, "fresh-scaffold evolution completed: resumed conversation and project checks passed", "fresh-scaffold evolution blocked: HMGEN-EXECUTION-LAYOUT-MISSING; no mutation"), false, false},
		{"skipped builder", strings.Replace(complete, "pass", "skip", 1), false, false},
		{"cached", complete + "(cached)", false, false},
		{"package only", `{"Action":"pass","Package":"fixture"}`, false, false},
		{"changed log", complete, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()

			err := writeFile(filepath.Join(fixture, "workflow.log"), tc.output)
			if err != nil {
				t.Fatal(err)
			}

			logDigest := digest(tc.output)
			if tc.tamper {
				logDigest = digest("previous")
			}

			receipt := runtimeReceipt{Commands: []commandReceipt{{
				Directory: "slice-6.fixture/generator-workbench",
				Arguments: []string{"go", "test", "-json", "-count=1", "-timeout=9m", "./..."},
				Expected:  "exit 0", Log: "workflow.log", LogDigest: logDigest,
			}}}

			err = verifyGeneratorCommands(receipt, fixture)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

// Adding a published command cannot silently inherit source-review evidence.
func TestGeneratorCoverage(t *testing.T) {
	r := row{id: "example:docs/reference/generator/README.md#block-4", source: "docs/reference/generator/README.md#block-4"}
	inv := inventory{contents: map[string]string{"docs/reference/generator/README.md": strings.Repeat("```sh\nhm\n```\n", 4)}}

	_, err := generatorMethod(r, inv)
	if err == nil {
		t.Fatal("new unbound command accepted")
	}
}
