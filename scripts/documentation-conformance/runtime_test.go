// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Evidence must reject missing or stale bindings and refuse compilation as execution proof.
func TestRuntimeControls(t *testing.T) {
	block := "```go\npackage main\nfunc main() {}\n```\n"
	r := row{id: "example:docs/app.md#block-1", source: "docs/app.md#block-1", slice: 2, digest: digest(block)}
	discovered := inventory{rows: []row{r}, contents: map[string]string{"docs/app.md": block}}
	original := runtimeReceipt{
		Commands:     []commandReceipt{{Arguments: []string{"go", "build", "."}, Expected: "exit 0", Log: "build.log", LogDigest: digest("build log")}},
		Proofs:       []proof{{ID: r.id, Digest: r.digest, Method: "executed", Outcome: "observed shutdown"}},
		Observations: []string{"health", "page", "redirect"},
	}

	cases := []struct {
		name, want string
		change     func(*runtimeReceipt)
	}{
		{name: "complete", change: func(*runtimeReceipt) {}},
		{name: "missing example", want: "missing, stale", change: func(receipt *runtimeReceipt) { receipt.Proofs = nil }},
		{name: "changed snippet", want: "missing, stale", change: func(receipt *runtimeReceipt) { receipt.Proofs[0].Digest = digest("different") }},
		{name: "compile only", want: "misclassified", change: func(receipt *runtimeReceipt) { receipt.Proofs[0].Method = "compiled" }},
		{name: "duplicate receipt", want: "duplicate", change: func(receipt *runtimeReceipt) { receipt.Proofs = append(receipt.Proofs, receipt.Proofs[0]) }},
		{name: "empty outcome", want: "misclassified", change: func(receipt *runtimeReceipt) { receipt.Proofs[0].Outcome = "" }},
		{name: "empty command", want: "empty runtime", change: func(receipt *runtimeReceipt) { receipt.Commands = nil }},
		{name: "failed command", want: "unsuccessful", change: func(receipt *runtimeReceipt) { receipt.Commands[0].Exit = 1 }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			receipt := original
			receipt.Proofs = slices.Clone(original.Proofs)
			receipt.Commands = slices.Clone(original.Commands)
			test.change(&receipt)

			err := verifyProofs(discovered, receipt)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want rejection containing %q", err, test.want)
			}
		})
	}
}

// A changed fragment must reach the compiler unchanged, and malformed source must fail.
func TestContextBindings(t *testing.T) {
	cases := []struct {
		name, body string
		invalid    bool
	}{
		{name: "actual API", body: "attrs := htmx.HX().Post(\"/items\")\n"},
		{name: "unknown API", body: "attrs := htmx.Nonexistent(\"/items\")\n"},
		{name: "malformed source", body: "attrs := (\n", invalid: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			source, err := contextualSource("example:htmx/readme.md#block-1", test.body)
			if test.invalid {
				if err == nil {
					t.Fatal("malformed source accepted")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(source, test.body) || !strings.Contains(source, "_ = attrs") {
				t.Fatal("fragment bytes or unused-result context lost")
			}
		})
	}
}

// Added shell instructions must fail until the owned execution harness accounts for them.
func TestProcedureControls(t *testing.T) {
	cases := []struct {
		name  string
		extra bool
		want  string
	}{
		{name: "published commands"},
		{name: "unaccounted instruction", extra: true, want: "unaccounted command procedure"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			discovered := inventory{contents: make(map[string]string)}
			pageBlocks := make(map[string][]string)

			for id, expected := range procedureExpectations {
				source := strings.TrimPrefix(id, "example:")
				parts := strings.Split(source, "#block-")

				number, err := strconv.Atoi(parts[1])
				if err != nil {
					t.Fatal(err)
				}

				for len(pageBlocks[parts[0]]) < number {
					pageBlocks[parts[0]] = append(pageBlocks[parts[0]], "```text\ncontext\n```\n")
				}

				if test.extra && strings.Contains(id, "#block-1") {
					expected += "\nunknown-command"
				}

				pageBlocks[parts[0]][number-1] = "```sh\n" + expected + "\n```\n"

				discovered.rows = append(discovered.rows, row{id: id, source: source, slice: 2})
			}

			for page, blocks := range pageBlocks {
				discovered.contents[page] = strings.Join(blocks, "\n")
			}

			err := checkProcedures(discovered)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %s", err, test.want)
			}
		})
	}
}
