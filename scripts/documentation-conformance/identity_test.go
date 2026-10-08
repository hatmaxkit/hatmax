// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"slices"
	"testing"
)

// A real browser fragment cannot be replaced by compilation, stale source or
// a successful command whose selector ran no tests.
func TestIdentityControls(t *testing.T) {
	block := "```javascript\nawait navigator.credentials.create({publicKey});\n```\n"
	r := row{id: "example:docs/auth.md#block-1", source: "docs/auth.md#block-1", slice: 4, digest: digest(block)}
	inv := inventory{rows: []row{r}, contents: map[string]string{"docs/auth.md": block}}
	original := runtimeReceipt{
		Commands:     []commandReceipt{{Arguments: []string{"go", "test"}, Expected: "exit 0", Log: "browser.log", LogDigest: digest("actual browser")}},
		Proofs:       []proof{{ID: r.id, Digest: r.digest, Method: "executed", Outcome: "real registration accepted"}},
		Observations: []string{"proof", "cookie", "recovery"},
	}

	for _, tc := range []struct {
		name   string
		change func(*runtimeReceipt)
		valid  bool
	}{
		{"complete", func(*runtimeReceipt) {}, true},
		{"missing", func(r *runtimeReceipt) { r.Proofs = nil }, false},
		{"compiled", func(r *runtimeReceipt) { r.Proofs[0].Method = "compiled" }, false},
		{"changed", func(r *runtimeReceipt) { r.Proofs[0].Digest = digest("other") }, false},
		{"failed", func(r *runtimeReceipt) { r.Commands[0].Exit = 1 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			receipt := original
			receipt.Proofs = slices.Clone(original.Proofs)
			receipt.Commands = slices.Clone(original.Commands)
			tc.change(&receipt)

			err := verifyGroupProofs(inv, receipt, 4, identityMethod)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

// Missing selectors, skipped tests and cached output never establish actual
// browser execution, even when Go exits successfully.
func TestBrowserEvidence(t *testing.T) {
	complete := "--- PASS: TestAuthenticatorBrowser\nPASS actual registration/sign-in, safe cookie and strong route\nBROWSER Chrome/151\n--- PASS: TestAccountRecoveryBrowser\n"
	for _, tc := range []struct {
		name, output string
		valid        bool
	}{
		{"complete", complete, true},
		{"empty", "ok package [no tests to run]", false},
		{"cached", complete + "(cached)", false},
		{"skipped", complete + "--- SKIP: TestAuthenticatorBrowser", false},
		{"no browser", "--- PASS: TestAuthenticatorBrowser\n--- PASS: TestAccountRecoveryBrowser", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkBrowserLog(2, []byte(tc.output))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}

// Foreground Make may receive SIGINT itself after the application's orderly
// shutdown. Accept that wrapper outcome while retaining strict direct-process
// success and rejecting unrelated signals, failures and missing expectations.
func TestTickedCompletion(t *testing.T) {
	for _, tc := range []struct {
		name, actual string
		index        int
		valid        bool
	}{
		{"direct success", "exit 0", 0, true},
		{"direct interrupted", "signal: interrupt", 0, false},
		{"make success", "exit 0", 2, true},
		{"make recipe interrupted", "exit status 2", 2, true},
		{"make interrupted", "signal: interrupt", 2, true},
		{"make terminated", "signal: terminated", 2, false},
		{"make failed", "exit status 1", 2, false},
		{"missing", "", 2, false},
		{"extra process", "exit 0", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := "exit 0 after Ctrl+C"
			if tc.index >= 2 {
				expected = "owned make group interrupted; application coordinated shutdown"
			}

			if validTickedCompletion(tc.index, processReceipt{Actual: tc.actual, Expected: expected}) != tc.valid {
				t.Fatalf("unexpected foreground completion classification")
			}
		})
	}
}
