// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var knownSurfaces = map[string]struct{}{
	"documentation": {},
	"handler":       {},
	"migration":     {},
	"model":         {},
	"service":       {},
	"store":         {},
	"templates":     {},
	"tests":         {},
	"wiring":        {},
}

func inspectFileSemantics(path, relative string, inventory *Inventory) (bool, []string, error) {
	surfaces := classifySurfaces(relative)
	if filepath.Ext(path) != ".go" {
		return false, surfaces, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil, unexpectedReadError(relative, err)
	}

	generated := isGeneratedGo(data)

	parsed, err := parser.ParseFile(token.NewFileSet(), relative, data, 0)
	if err != nil {
		return false, nil, projectError("project_go_source_invalid", relative, "%v", err)
	}

	if parsed.Name.Name == "main" && hasMainFunction(parsed) {
		inventory.Entrypoints = append(inventory.Entrypoints, Entrypoint{
			Path:            relative,
			CompositionRoot: hasHatmaxSetup(parsed),
		})
	}

	if !strings.HasSuffix(relative, "_test.go") && hasHatmaxSetup(parsed) {
		inventory.compositionRoots = append(inventory.compositionRoots, relative)
	}

	return generated, surfaces, nil
}

func hasMainFunction(file *ast.File) bool {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "main" {
			return true
		}
	}

	return false
}

func hasHatmaxSetup(file *ast.File) bool {
	aliases := make(map[string]struct{})

	for _, imported := range file.Imports {
		if strings.Trim(imported.Path.Value, `"`) != HatmaxModulePath+"/app" {
			continue
		}

		name := "app"
		if imported.Name != nil {
			name = imported.Name.Name
		}

		aliases[name] = struct{}{}
	}

	found := false

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Setup" {
			return true
		}

		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		_, found = aliases[identifier.Name]

		return !found
	})

	return found
}

func isGeneratedGo(data []byte) bool {
	firstLines := string(data)
	if len(firstLines) > 2048 {
		firstLines = firstLines[:2048]
	}

	return strings.Contains(firstLines, "Code generated") && strings.Contains(firstLines, "DO NOT EDIT")
}

func classifySurfaces(path string) []string {
	base := filepath.Base(path)
	extension := strings.ToLower(filepath.Ext(base))
	segments := strings.Split(path, "/")
	seen := make(map[string]struct{})

	if extension == ".md" && (segments[0] == "docs" || base == "README.md") {
		seen["documentation"] = struct{}{}
	}

	if extension == ".html" || extension == ".tmpl" || containsSegment(segments, "templates") {
		seen["templates"] = struct{}{}
	}

	if extension == ".sql" && (containsSegment(segments, "migration") || containsSegment(segments, "migrations") || strings.Contains(base, "migration")) {
		seen["migration"] = struct{}{}
	}

	if extension == ".sql" && containsSegment(segments, "queries") {
		seen["store"] = struct{}{}
	}

	if strings.HasSuffix(base, "_test.go") {
		seen["tests"] = struct{}{}
	}

	switch base {
	case "handler.go", "handlers.go":
		seen["handler"] = struct{}{}
	case "model.go", "models.go":
		seen["model"] = struct{}{}
	case "service.go", "services.go":
		seen["service"] = struct{}{}
	case "store.go", "repository.go", "queries.go":
		seen["store"] = struct{}{}
	case "main.go":
		seen["wiring"] = struct{}{}
	}

	if strings.HasSuffix(base, "_store.go") {
		seen["store"] = struct{}{}
	}

	if containsSegment(segments, "model") {
		seen["model"] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for surface := range seen {
		result = append(result, surface)
	}

	sort.Strings(result)

	return result
}

func collectLayouts(path string, record fileRecord, directories map[string]map[string]struct{}) {
	segments := strings.Split(path, "/")
	addLayoutDirectory(directories, "features", directoryThroughSegment(segments, "feat", 1))
	addLayoutDirectory(directories, "features", directoryThroughSegment(segments, "features", 1))
	addLayoutDirectory(directories, "assets", directoryThroughSegment(segments, "assets", 0))
	addLayoutDirectory(directories, "assets", directoryThroughSegment(segments, "static", 0))
	addLayoutDirectory(directories, "migrations", directoryThroughSegment(segments, "migrations", 0))
	addLayoutDirectory(directories, "migrations", directoryThroughSegment(segments, "migration", 1))
	addLayoutDirectory(directories, "templates", directoryThroughSegment(segments, "templates", 0))
	addLayoutDirectory(directories, "queries", directoryThroughSegment(segments, "queries", 0))

	if filepath.Ext(path) == ".sql" && !containsSegment(segments, "migrations") && !containsSegment(segments, "migration") {
		addLayoutDirectory(directories, "queries", filepath.ToSlash(filepath.Dir(path)))
	}

	if record.generated {
		addLayoutDirectory(directories, "generated", filepath.ToSlash(filepath.Dir(path)))
	}
}

func directoryThroughSegment(segments []string, name string, following int) string {
	for index, segment := range segments {
		if segment != name || index+following >= len(segments) {
			continue
		}

		end := index + following + 1
		if end >= len(segments) {
			end = len(segments) - 1
		}

		return strings.Join(segments[:end], "/")
	}

	return ""
}

func containsSegment(segments []string, name string) bool {
	for _, segment := range segments {
		if segment == name {
			return true
		}
	}

	return false
}

func addLayoutDirectory(directories map[string]map[string]struct{}, kind, path string) {
	if path == "" || path == "." {
		return
	}

	if directories[kind] == nil {
		directories[kind] = make(map[string]struct{})
	}

	directories[kind][path] = struct{}{}
}

func applyLayouts(layout *Layout, directories map[string]map[string]struct{}) {
	layout.Features = sortedKeys(directories["features"])
	layout.Assets = sortedKeys(directories["assets"])
	layout.Migrations = sortedKeys(directories["migrations"])
	layout.Templates = sortedKeys(directories["templates"])
	layout.Queries = sortedKeys(directories["queries"])
	layout.Generated = sortedKeys(directories["generated"])
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func collectRepositoryRules(path string, record fileRecord, inventory *Inventory, generatedPaths *[]string) {
	base := filepath.Base(path)
	if base == "AGENTS.md" || strings.HasPrefix(path, ".agents/") || base == "CODEOWNERS" {
		inventory.Rules.Instructions = append(inventory.Rules.Instructions, path)
	}

	if record.generated {
		*generatedPaths = append(*generatedPaths, path)
	}
}
