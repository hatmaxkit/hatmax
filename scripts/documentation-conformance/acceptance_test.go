// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Expected-failure assertions and inventories cannot become successful closure.
func TestClosureControls(t *testing.T) {
	for _, tc := range []struct {
		name, status, receipt, page string
		reject                      bool
	}{
		{name: "executed", status: "executed", receipt: "slice-7/acceptance-receipts.json", page: "README.md"},
		{name: "blocked journey", status: "blocked", receipt: "slice-6/generator-receipts.json", page: "README.md", reject: true},
		{name: "unexecuted inventory", status: "inventoried", receipt: "pending", page: "README.md", reject: true},
		{name: "pending receipt", status: "executed", receipt: "pending", page: "README.md", reject: true},
		{name: "missing page", status: "source-inspected", receipt: "slice-7/acceptance-receipts.json", page: "missing", reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := closureCoverage([]row{{id: "example:request", status: tc.status, receipt: tc.receipt, page: tc.page}})
			if (err != nil) != tc.reject {
				t.Fatalf("closure rejection %v: %v", tc.reject, err)
			}
		})
	}
}

// Native execution must retain the exact finite executable set and its bytes.
func TestNativeControls(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testing.T, string, string)
		reject bool
	}{
		{name: "complete", mutate: func(*testing.T, string, string) {}},
		{name: "inherited path", reject: true, mutate: func(t *testing.T, bin, _ string) { t.Setenv("PATH", bin+":/usr/bin") }},
		{name: "unapproved executable", reject: true, mutate: func(t *testing.T, bin, _ string) {
			err := os.WriteFile(filepath.Join(bin, "auxiliary-runtime"), []byte("runtime"), 0600)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{name: "changed binary", reject: true, mutate: func(t *testing.T, bin, _ string) {
			err := os.WriteFile(filepath.Join(bin, "go"), []byte("changed"), 0600)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing tool", reject: true, mutate: func(t *testing.T, bin, _ string) {
			err := os.Remove(filepath.Join(bin, "make"))
			if err != nil {
				t.Fatal(err)
			}
		}},
		{name: "automatic toolchain", reject: true, mutate: func(t *testing.T, _, _ string) { t.Setenv("GOTOOLCHAIN", "auto") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			image := string([]byte{0x7f, 'E', 'L', 'F'}) + "native-fixture"

			var manifest strings.Builder

			for _, name := range strings.Fields(nativeToolNames) {
				file := filepath.Join(bin, name)

				err := os.WriteFile(file, []byte(image), 0700)
				if err != nil {
					t.Fatal(err)
				}

				manifest.WriteString(name + "\t" + file + "\t" + digest(image) + "\n")
			}

			file := filepath.Join(t.TempDir(), "tools.tsv")

			err := os.WriteFile(file, []byte(manifest.String()), 0600)
			if err != nil {
				t.Fatal(err)
			}

			t.Setenv("PATH", bin)
			t.Setenv("HATMAX_DOC_NATIVE_BIN", bin)
			t.Setenv("GOTOOLCHAIN", "local")
			tc.mutate(t, bin, file)

			err = checkNativeTools(file)
			if (err != nil) != tc.reject {
				t.Fatalf("native rejection %v: %v", tc.reject, err)
			}
		})
	}
}

// New published commands require a deliberate execution binding before acceptance.
func TestAcceptanceBindings(t *testing.T) {
	for _, id := range []string{"example:README.md#block-6", "page:docs/new-entrypoint.md", "command:unbound"} {
		t.Run(id, func(t *testing.T) {
			_, err := acceptanceMethod(row{id: id}, inventory{})
			if err == nil {
				t.Fatal("unaccounted execution accepted")
			}
		})
	}
}
