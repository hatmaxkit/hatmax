// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Coverage must reject omitted inventory items and changed content, not just count rows.
func TestCoverageControls(t *testing.T) {
	revision := strings.Repeat("a", 40)
	original := []row{
		{id: "package:app", source: "app", digest: digest("api"), owner: "app", slice: 2, page: "missing", revision: revision, status: "inventoried", receipt: "pending"},
		{id: "page:docs/reference/app.md", source: "docs/reference/app.md", digest: digest("page"), owner: "app", slice: 2, page: "missing", revision: revision, status: "inventoried", receipt: "pending"},
	}

	cases := []struct {
		name       string
		mutate     func([]row) []row
		discovered []row
		want       string
	}{
		{name: "complete", mutate: func(rows []row) []row { return rows }, discovered: original},
		{name: "missing surface", mutate: func(rows []row) []row { return rows[1:] }, discovered: original, want: "unaccounted surface/page/example: package:app"},
		{name: "unaccounted page", mutate: func(rows []row) []row { return rows[:1] }, discovered: original, want: "unaccounted surface/page/example: page:"},
		{name: "changed snippet", mutate: func(rows []row) []row {
			rows[0].digest = digest("stale")

			return rows
		}, discovered: original, want: "source drift"},
		{name: "duplicate identity", mutate: func(rows []row) []row { return append(rows, rows[0]) }, discovered: original, want: "duplicate identity"},
		{name: "false execution claim", mutate: func(rows []row) []row {
			rows[0].status = "verified"

			return rows
		}, discovered: original, want: "pending receipts only"},
		{name: "removed page", mutate: func(rows []row) []row { return rows }, discovered: original[:1], want: "stale coverage"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := reconcile(test.discovered, test.mutate(slices.Clone(original)))
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

// Future slice and integrated modes must fail before tools or fixtures can supply a pass.
func TestUnavailableModes(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "integrated"},
		{name: "future slice", args: []string{"slice", "2"}},
		{name: "missing number", args: []string{"slice"}},
		{name: "unknown mode", args: []string{"all"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"../check-documentation-conformance.sh"}, test.args...)
			output, err := exec.Command("bash", args...).CombinedOutput()

			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 || !strings.Contains(string(output), "not yet available") {
				t.Fatalf("got %q / %v, want unavailable mode with exit 2", output, err)
			}
		})
	}
}

// Navigation needs real anchors and an index path, including rejection of circular orphans.
func TestNavigationControls(t *testing.T) {
	cases := []struct {
		name, suffix, anchor, want string
	}{
		{name: "valid anchor", anchor: "#current-contract"},
		{name: "broken anchor", anchor: "#absent", want: "broken heading anchor"},
		{name: "duplicate heading", anchor: "#current-contract-1"},
		{name: "orphan cycle", anchor: "#current-contract", suffix: "orphan", want: "unreachable from quadrant index"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			// All filesystem changes are confined to this test's owned temporary directory.
			t.Chdir(t.TempDir())

			contents := make(map[string]string)

			for _, quadrant := range []string{"tutorials", "how-to", "reference", "explanation"} {
				directory := "docs/" + quadrant

				err := os.MkdirAll(directory, 0o700)
				if err != nil {
					t.Fatal(err)
				}

				contents[directory+"/README.md"] = "# Index\n\n[Page](page.md" + test.anchor + ")\n"
				contents[directory+"/page.md"] = "# Current Contract\n\n## Current Contract\n"
			}

			if test.suffix == "orphan" {
				contents["docs/reference/a.md"] = "# Orphan\n[B](b.md)\n"
				contents["docs/reference/b.md"] = "# Orphan\n[A](a.md)\n"
			}

			for file, text := range contents {
				err := os.WriteFile(filepath.FromSlash(file), []byte(text), 0o600)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := checkNavigation(contents)
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

// Discovery must catch newly published files without using the maintained coverage list.
func TestDiscoveryControls(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := command("git", "init", "--quiet")
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir("newapi", 0o700)
	if err != nil {
		t.Fatal(err)
	}

	for file, content := range map[string]string{"newapi/api.go": "package newapi\n", "newapi/README.md": "# API\n\n```go\nNew()\n```\n", "go.mod": "module example.org/test\n", "go.sum": ""} {
		err = os.WriteFile(file, []byte(content), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	}

	discovered, err := discover()
	if err != nil {
		t.Fatal(err)
	}

	var identities []string
	for _, r := range discovered.rows {
		identities = append(identities, r.id)
	}

	for _, expected := range []string{"package:newapi", "page:newapi/README.md", "example:newapi/README.md#block-1"} {
		if !slices.Contains(identities, expected) {
			t.Fatalf("discovery omitted %q", expected)
		}
	}
}
