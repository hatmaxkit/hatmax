// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"slices"
	"strings"
	"testing"
)

// A published mail workflow requires actual execution, current source and
// complete command evidence; a compile-only substitute cannot establish it.
func TestInfrastructureControls(t *testing.T) {
	block := "```go\nmail := mailer.New(cfg, logger)\n```\n"
	r := row{id: "example:docs/how-to/configure-mailer/README.md#block-2", source: "docs/how-to/configure-mailer/README.md#block-2", slice: 5, digest: digest(block)}
	inv := inventory{rows: []row{r}, contents: map[string]string{"docs/how-to/configure-mailer/README.md": "```yaml\nmailer: {}\n```\n" + block}}
	original := runtimeReceipt{
		Commands:     []commandReceipt{{Arguments: []string{"go", "test", "-count=1"}, Expected: "exit 0", Log: "workflow.log", LogDigest: digest("local mail captured")}},
		Proofs:       []proof{{ID: r.id, Digest: r.digest, Method: "executed", Outcome: "owned SMTP capture"}},
		Observations: []string{"owned", "captured", "closed"},
	}

	for _, tc := range []struct {
		name   string
		change func(*runtimeReceipt)
		valid  bool
	}{
		{"complete", func(*runtimeReceipt) {}, true},
		{"missing", func(r *runtimeReceipt) { r.Proofs = nil }, false},
		{"compiled", func(r *runtimeReceipt) { r.Proofs[0].Method = "compiled" }, false},
		{"stale", func(r *runtimeReceipt) { r.Proofs[0].Digest = digest("previous") }, false},
		{"duplicate", func(r *runtimeReceipt) { r.Proofs = append(r.Proofs, r.Proofs[0]) }, false},
		{"failed", func(r *runtimeReceipt) { r.Commands[0].Exit = 1 }, false},
		{"unobserved", func(r *runtimeReceipt) { r.Proofs[0].Outcome = "" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			receipt := original
			receipt.Proofs = slices.Clone(original.Proofs)
			receipt.Commands = slices.Clone(original.Commands)
			tc.change(&receipt)

			err := verifyGroupProofs(inv, receipt, 5, infrastructureMethod)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

// Successful package output, skipped selectors and cached JSON cannot substitute
// for a named uncached workflow; a new runnable command cannot default to review.
func TestInfrastructureExecution(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		valid        bool
	}{
		{"passed", `{"Action":"pass","Test":"TestMail"}`, true},
		{"package only", `{"Action":"pass","Package":"fixture"}`, false},
		{"skipped", `{"Action":"skip","Test":"TestMail"}`, false},
		{"failed", `{"Action":"fail","Test":"TestMail"}`, false},
		{"cached", "{\"Action\":\"pass\",\"Test\":\"TestMail\"}\n(cached)", false},
		{"empty selector", "ok package [no tests to run]", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if namedTestPassed([]byte(tc.output), "TestMail") != tc.valid {
				t.Fatal("incorrect workflow completion classification")
			}
		})
	}

	r := row{id: "example:docs/future.md#block-1", source: "docs/future.md#block-1", slice: 5}
	inv := inventory{contents: map[string]string{"docs/future.md": "```sh\ngo run ./new-command\n```\n"}}

	_, err := infrastructureMethod(r, inv)
	if err == nil {
		t.Fatal("unbound executable command supplied a review pass")
	}
}

// Compilation context must preserve exact snippets and complete program bytes.
func TestInfrastructureContext(t *testing.T) {
	program := "package main\nfunc main(){}\n"

	source, err := infrastructureSource("example:pubsub/readme.md#block-1", program)
	if err != nil || source != program {
		t.Fatal("complete program was rewritten")
	}

	body := "mail := mailer.New(cfg, logger)\n"

	source, err = infrastructureSource("example:docs/how-to/configure-mailer/README.md#block-2", body)
	if err != nil || !strings.Contains(source, body) {
		t.Fatal("published constructor bytes were not preserved")
	}

	_, err = infrastructureSource("example:docs/fragment.md#block-1", "value := (\n")
	if err == nil {
		t.Fatal("malformed Go fragment accepted")
	}
}
