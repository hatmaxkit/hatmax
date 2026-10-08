// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// SQL execution cannot be replaced by a compile receipt, omitted binding or
// altered snippet. The same evidence rules apply to the active data group.
func TestDataControls(t *testing.T) {
	block := "```sql\n-- +migrate Up\nCREATE TABLE notes(id text);\n```\n"
	r := row{id: "example:docs/how-to/apply-migrations/README.md#block-1", source: "docs/how-to/apply-migrations/README.md#block-1", slice: 3, digest: digest(block)}
	discovered := inventory{rows: []row{r}, contents: map[string]string{"docs/how-to/apply-migrations/README.md": block}}
	original := runtimeReceipt{
		Commands:     []commandReceipt{{Arguments: []string{"go", "test", "./..."}, Expected: "exit 0", Log: "data.log", LogDigest: digest("executed")}},
		Proofs:       []proof{{ID: r.id, Digest: r.digest, Method: "executed", Outcome: "real migration tracked"}},
		Observations: []string{"migration", "settings", "restart"},
	}

	for _, tc := range []struct {
		name   string
		change func(*runtimeReceipt)
		valid  bool
	}{
		{"complete", func(*runtimeReceipt) {}, true},
		{"missing migration", func(r *runtimeReceipt) { r.Proofs = nil }, false},
		{"compile only", func(r *runtimeReceipt) { r.Proofs[0].Method = "compiled" }, false},
		{"changed SQL", func(r *runtimeReceipt) { r.Proofs[0].Digest = digest("altered") }, false},
		{"duplicate", func(r *runtimeReceipt) { r.Proofs = append(r.Proofs, r.Proofs[0]) }, false},
		{"failed command", func(r *runtimeReceipt) { r.Commands[0].Exit = 1 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			receipt := original
			receipt.Proofs = slices.Clone(original.Proofs)
			receipt.Commands = slices.Clone(original.Commands)
			tc.change(&receipt)

			err := verifyGroupProofs(discovered, receipt, 3, dataMethod)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

// Local values named errors are not package imports. Exact source and its
// candidate mutation remain present in the application compilation context.
func TestDataContext(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"field errors", "errors := validation.Field(\"name\", \"\").Required().Errors()\n_ = errors.HasErrors()\n", true},
		{"malformed", "value := (\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := dataSource("example:validation/readme.md#block-1", tc.body, inventory{})
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}

			if tc.valid && (!strings.Contains(source, tc.body) || strings.Contains(source, "errors \"errors\"")) {
				t.Fatal("source lost or local variable treated as package")
			}
		})
	}
}

// Storage outside the dedicated invocation is rejected before any database
// process starts, regardless of whether the requested command would succeed.
func TestDataFixtureOwnership(t *testing.T) {
	// The gate sets TMPDIR inside owned storage, so choose an outside path
	// explicitly instead of inheriting that directory through t.TempDir.
	fixture, err := os.MkdirTemp("/tmp", "hatmax-doc-unowned-")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(fixture)
		if err != nil {
			t.Error(err)
		}
	})

	output, err := exec.Command("bash", "data-fixture.sh", fixture, "true").CombinedOutput()

	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || !strings.Contains(string(output), "owned Slice 3 through Slice 6 storage") {
		t.Fatalf("unowned fixture accepted: %s / %v", output, err)
	}
}
