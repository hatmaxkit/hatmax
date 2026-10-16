// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"hatmax.adrianpk.com/image"
	imagelocal "hatmax.adrianpk.com/image/local"
	images3 "hatmax.adrianpk.com/image/s3"
	"hatmax.adrianpk.com/storage"
)

var objectGuideSources = []struct{ page, file, function string }{
	{"docs/tutorials/private-objects/README.md", "examples/object-storage/local/main.go", ""},
	{"docs/how-to/store-private-objects/README.md", "examples/object-storage/remote/config.go", "newStore"},
	{"docs/how-to/sign-object-uploads/README.md", "examples/object-storage/remote/main.go", "run"},
	{"docs/how-to/sign-object-uploads/README.md", "examples/object-storage/remote/upload.go", "submit"},
	{"docs/how-to/sign-object-uploads/README.md", "examples/object-storage/remote/completion.go", "complete"},
}

// Published Go bytes must stay identical to the compiled, executed composition.
func TestObjectGuideDrift(t *testing.T) {
	root := filepath.Join("..", "..")

	for _, source := range objectGuideSources {
		t.Run(filepath.Base(source.file)+" "+source.function, func(t *testing.T) {
			page, err := os.ReadFile(filepath.Join(root, source.page))
			if err != nil {
				t.Fatal(err)
			}

			code, err := os.ReadFile(filepath.Join(root, source.file))
			if err != nil {
				t.Fatal(err)
			}

			identity := source.file
			if source.function != "" {
				identity += "#" + source.function
			}

			published, err := publishedObjectSource(string(page), identity)
			if err != nil {
				t.Fatal(err)
			}

			expected, err := objectFunction(code, source.function)
			if err != nil || published != expected {
				t.Fatalf("published source drift: %s (%v)", identity, err)
			}

			err = objectOutputPolicy(code)
			if err != nil {
				t.Fatal(err)
			}
		})
	}

	pages := map[string][]string{
		"docs/tutorials/README.md":   {"private-objects/README.md"},
		"docs/how-to/README.md":      {"store-private-objects/README.md", "sign-object-uploads/README.md", "reconcile-object-storage/README.md"},
		"docs/reference/README.md":   {"storage/README.md"},
		"docs/explanation/README.md": {"private-object-ownership/README.md"},
	}
	for page, targets := range pages {
		contents, err := os.ReadFile(filepath.Join(root, page))
		if err != nil {
			t.Fatal(err)
		}

		for _, target := range targets {
			if !strings.Contains(string(contents), "]("+target+")") {
				t.Fatalf("missing quadrant navigation: %s -> %s", page, target)
			}
		}
	}
}

func publishedObjectSource(page, identity string) (string, error) {
	marker := "<!-- object-storage-source: " + identity + " -->\n```go\n"
	if strings.Count(page, marker) != 1 {
		return "", fmt.Errorf("missing or duplicate published source: %s", identity)
	}

	_, body, _ := strings.Cut(page, marker)

	code, _, found := strings.Cut(body, "```\n")
	if !found {
		return "", fmt.Errorf("unclosed published source: %s", identity)
	}

	return code, nil
}

func objectFunction(code []byte, name string) (string, error) {
	if name == "" {
		return string(code), nil
	}

	positions := token.NewFileSet()

	file, err := parser.ParseFile(positions, "guide.go", code, 0)
	if err != nil {
		return "", err
	}

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return string(code[positions.Position(function.Pos()).Offset:positions.Position(function.End()).Offset]) + "\n", nil
		}
	}

	return "", fmt.Errorf("documented function missing: %s", name)
}

// Only fixed literal messages may reach the examples' output sinks.
func objectOutputPolicy(code []byte) error {
	file, err := parser.ParseFile(token.NewFileSet(), "guide.go", code, 0)
	if err != nil {
		return err
	}

	for _, imported := range file.Imports {
		path, _ := strconv.Unquote(imported.Path.Value)
		if path == "log" || strings.HasPrefix(path, "log/") {
			return fmt.Errorf("example logging import is not permitted")
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			name, ok := call.Fun.(*ast.Ident)
			if ok && (name.Name == "print" || name.Name == "println") {
				err = fmt.Errorf("unbounded example output")
			}

			return true
		}

		owner, ok := selector.X.(*ast.Ident)
		if !ok {
			// Do not permit direct stdout/stderr writes as another output path.
			if inner, ok := selector.X.(*ast.SelectorExpr); ok && (inner.Sel.Name == "Stdout" || inner.Sel.Name == "Stderr") {
				err = fmt.Errorf("direct example output")
			}

			return true
		}

		if owner.Name == "fmt" && strings.Contains(strings.ToLower(selector.Sel.Name), "print") {
			arguments := call.Args
			if selector.Sel.Name == "Fprintln" && len(arguments) == 2 {
				destination, ok := arguments[0].(*ast.SelectorExpr)
				if !ok || destination.Sel.Name != "Stderr" {
					err = fmt.Errorf("example output destination differs")
				}

				arguments = arguments[1:]
			} else if selector.Sel.Name != "Println" {
				err = fmt.Errorf("unapproved example output sink")
			}

			if len(arguments) != 1 {
				err = fmt.Errorf("example output must be one literal")
			}

			for _, argument := range arguments {
				literal, ok := argument.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					err = fmt.Errorf("sensitive or dynamic example output")
				}
			}
		}

		return true
	})

	return err
}

// Drift and secret-output controls must reject missing and changed examples.
func TestObjectGuideControls(t *testing.T) {
	for _, test := range []struct {
		name, code string
		rejected   bool
	}{
		{"fixed outcome", `func main() { fmt.Println("complete") }`, false},
		{"grant fields", `func main() { fmt.Println(grant.Fields) }`, true},
		{"provider body", `func main() { fmt.Fprintf(os.Stderr, "%s", response) }`, true},
		{"logger", `func main() { log.Print(grant) }`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			code := "package main\n" + test.code
			if test.name == "logger" {
				code = "package main\nimport \"log\"\n" + test.code
			}

			err := objectOutputPolicy([]byte(code))
			if (err != nil) != test.rejected {
				t.Fatalf("output policy: %v", err)
			}
		})
	}

	_, err := publishedObjectSource("```go\nstale\n```\n", "missing")
	if err == nil {
		t.Fatal("missing source marker accepted")
	}
}

// Run the actual published command; a success message alone cannot hide leftovers.
func TestObjectLocalGuide(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	owned := t.TempDir()
	binary := filepath.Join(t.TempDir(), "local-guide")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./examples/object-storage/local")
	build.Dir = root

	buildOutput, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build local guide: %v: %s", err, buildOutput)
	}

	command := exec.CommandContext(ctx, binary, owned)

	output, err := command.CombinedOutput()
	if err != nil || string(output) != "local object workflow complete\n" {
		t.Fatal("local guide failed or emitted unexpected output")
	}

	err = filepath.WalkDir(owned, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			return fmt.Errorf("local example left an object or spool")
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The new canonical key contract must not tighten historical image keys or URLs.
func TestObjectImageCompatibility(t *testing.T) {
	key := "photos/my photo.jpg"
	if storage.ValidateKey(key) == nil {
		t.Fatal("fixture must distinguish historical and new key contracts")
	}

	ctx := context.Background()

	var local image.Store = imagelocal.NewStore(t.TempDir(), "/legacy/")

	data := []byte("legacy bytes")

	err := local.Put(ctx, key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	reader, err := local.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	actual, readErr := io.ReadAll(reader)

	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(actual, data) || local.URL(key) != "/legacy//"+key {
		t.Fatal("historical local read or URL behavior changed")
	}

	err = local.Delete(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	remote, err := images3.NewStore(ctx, images3.Options{Bucket: "legacy", Region: "us-east-1", AccessKey: "fixture", SecretKey: "fixture", BaseURL: "https://example.invalid/legacy/"})
	if err != nil || remote.URL(key) != "https://example.invalid/legacy/"+key {
		t.Fatal("historical remote constructor or URL behavior changed")
	}
}
