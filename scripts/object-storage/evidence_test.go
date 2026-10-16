// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Receipts must reject missing/reordered commands and changed retained log bytes.
func TestStepEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{"complete", nil},
		{"missing command", func(t *testing.T, directory string) {
			err := os.Remove(filepath.Join(directory, "step-002.json"))
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"changed log", func(t *testing.T, directory string) {
			err := os.WriteFile(filepath.Join(directory, "log-001.txt"), []byte("changed"), 0600)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"reordered command", func(t *testing.T, directory string) {
			err := os.Rename(filepath.Join(directory, "step-001.json"), filepath.Join(directory, "step-999.json"))
			if err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()

			m := manifest{Mode: "slice-5"}
			for _, group := range []string{"ordinary", "remote", "guide", "guide-child"} {
				m.Roots = append(m.Roots, root{group, "storage", "TestRequired"})
			}

			groups := map[string]string{
				"ordinary storage and image adapter roots": "ordinary", "race storage and image adapter roots": "race",
				"private S3 integration roots": "remote", "published actual signed remote workflow": "guide",
				"documented storage child results": "guide-child", "documentation structure, links and compiled examples": "docs",
				"owned PostgreSQL cleanup": "postgres",
			}
			for index, label := range expectedLabels(m.Mode) {
				group := groups[label]
				if group == "" {
					group = "check"
				}

				log := filepath.Join(directory, fmt.Sprintf("log-%03d.txt", index+1))

				err := os.WriteFile(log, []byte("{\"Action\":\"pass\",\"Package\":\"hatmax.adrianpk.com/storage\",\"Test\":\"TestRequired\"}\n"), 0600)
				if err != nil {
					t.Fatal(err)
				}

				digest, err := hashFile(log)
				if err != nil {
					t.Fatal(err)
				}

				s := step{Index: index + 1, Label: label, Group: group, Log: log, SHA256: digest, Arguments: []string{"fixture-command"}}

				err = writeJSON(filepath.Join(directory, fmt.Sprintf("step-%03d.json", index+1)), s)
				if err != nil {
					t.Fatal(err)
				}
			}

			if test.mutate != nil {
				test.mutate(t, directory)
			}

			_, _, err := verifySteps(directory, m)
			if (err != nil) != (test.mutate != nil) {
				t.Fatalf("command/log evidence: %v", err)
			}
		})
	}
}

// Package-level no-test events cannot hide a missing, skipped or failed root.
func TestResultControls(t *testing.T) {
	required := []root{{"ordinary", "storage", "TestKeys"}}

	pass := `{"Action":"pass","Package":"hatmax.adrianpk.com/storage","Test":"TestKeys"}` + "\n"
	for _, test := range []struct {
		name, events string
		repository   bool
		rejected     bool
	}{
		{"complete", pass, false, false},
		{"missing root", `{"Action":"skip","Package":"hatmax.adrianpk.com/examples/command"}`, false, true},
		{"required skip", strings.Replace(pass, "pass", "skip", 1), false, true},
		{"required skip in full", strings.Replace(pass, "pass", "skip", 1), true, true},
		{"required subcase skip", pass + `{"Action":"skip","Package":"hatmax.adrianpk.com/storage","Test":"TestKeys/case"}`, true, true},
		{"failure after pass", pass + `{"Action":"fail","Package":"hatmax.adrianpk.com/storage"}`, false, true},
		{"wrong package", strings.Replace(pass, "storage", "other", 1), false, true},
		{"declared opt in", pass + `{"Action":"skip","Package":"hatmax.adrianpk.com/generator/backend/codex","Test":"TestAuthenticatedLiveSmoke"}`, true, false},
		{"opt in cannot replace OS root", `{"Action":"skip","Package":"hatmax.adrianpk.com/generator/backend/codex","Test":"TestAuthenticatedLiveSmoke"}`, true, true},
		{"malformed result", pass + "{malformed", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := rootResults(strings.NewReader(test.events), required, test.repository)
			if (err != nil) != test.rejected {
				t.Fatalf("result evidence rejection=%v: %v", test.rejected, err)
			}
		})
	}
}

// Accepted roots still require the actual pinned fixture and complete teardown.
func TestFixtureEvidence(t *testing.T) {
	complete := strings.Join([]string{
		"source: " + fixtureSource, "module: " + fixtureModule, "module-sum: " + fixtureSum,
		"go: go1.27.1", "binary-sha256: observed", "objects-after-cleanup: 0",
		"bucket: removed", "server: stopped", "temporary-root: removed",
		"scope-probe-bucket: removed", "lifecycle: local and remote snapshots, failed replacement, cancellation and absence passed",
	}, "\n") + "\n"
	for _, test := range []struct{ name, remove string }{
		{"complete", ""}, {"source mismatch", fixtureSource},
		{"binary mismatch", "binary-sha256: observed"},
		{"remaining objects", "objects-after-cleanup: 0"},
		{"live server", "server: stopped"}, {"leftover root", "temporary-root: removed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := complete
			if test.remove != "" {
				text = strings.Replace(text, test.remove, "missing", 1)
			}

			path := filepath.Join(t.TempDir(), "fixture.txt")

			err := os.WriteFile(path, []byte(text), 0600)
			if err != nil {
				t.Fatal(err)
			}

			err = checkFixture(path, "observed", false)
			if (err != nil) != (test.remove != "") {
				t.Fatalf("fixture evidence: %v", err)
			}
		})
	}
}
