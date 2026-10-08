// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

var wiringRenderers = map[string]recipeRenderer{
	"server_rendered_crud.wiring": renderWiring,
}

func renderWiring(context renderContext, edit Edit) ([]byte, error) {
	target, err := secureTargetPath(context.inventory.Root, edit.Target)
	if err != nil {
		return nil, err
	}

	source, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read composition root: %w", err)
	}

	files := token.NewFileSet()

	parsed, err := parser.ParseFile(files, edit.Target, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse composition root: %w", err)
	}

	mainFunction := findCompositionFunction(parsed)
	if mainFunction == nil {
		return nil, wiringPrerequisite(edit.Target, "composition function")
	}

	depsIndex, dependencies := findDependencyList(mainFunction)
	if dependencies == nil {
		return nil, wiringPrerequisite(edit.Target, "deps := []any{...}")
	}

	required := []string{"database", "migrator", "tmplMgr", "logger"}

	declared := declaredBefore(mainFunction.Body.List[:depsIndex])
	templateName := "tmplMgr"

	var prerequisites []ast.Stmt

	if parsed.Name.Name == "application" && mainFunction.Name.Name == "run" {
		prerequisites, err = applicationWiringPrerequisites(parsed, mainFunction, dependencies, declared)
		if err != nil {
			return nil, wiringPrerequisite(edit.Target, err.Error())
		}

		templateName = "templates"
		required = []string{"database", "logger"}
	}

	for _, name := range required {
		if !declared[name] {
			return nil, wiringPrerequisite(edit.Target, name)
		}
	}

	featureAlias := context.feature + "feat"
	if identifierExists(parsed, featureAlias) {
		return nil, executionError("execution_semantic_conflict", edit.Target, "identifier %q already exists", featureAlias)
	}

	featurePath := featureImport(context)
	if importPathExists(parsed, featurePath) {
		return nil, executionError("execution_semantic_conflict", edit.Target, "feature import %q already exists", featurePath)
	}

	storeName := context.feature + "Store"
	serviceName := context.feature + "Service"
	handlerName := context.feature + "Handler"

	for _, name := range []string{storeName, serviceName, handlerName} {
		if identifierExists(parsed, name) {
			return nil, executionError("execution_semantic_conflict", edit.Target, "identifier %q already exists", name)
		}
	}

	statements := append(prerequisites, wiringStatements(featureAlias, storeName, serviceName, handlerName, templateName)...)
	mainFunction.Body.List = insertStatements(mainFunction.Body.List, depsIndex, statements)

	insertLifecycleDependencies(dependencies, storeName, handlerName)
	addNamedImport(parsed, featureAlias, featurePath)
	ast.SortImports(files, parsed)

	var rendered bytes.Buffer

	err = format.Node(&rendered, files, parsed)
	if err != nil {
		return nil, fmt.Errorf("format composition root: %w", err)
	}

	return rendered.Bytes(), nil
}

func findCompositionFunction(file *ast.File) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && (file.Name.Name == "main" && function.Name.Name == "main" || file.Name.Name == "application" && function.Name.Name == "run") {
			return function
		}
	}

	return nil
}

func findDependencyList(function *ast.FuncDecl) (int, *ast.CompositeLit) {
	for index, statement := range function.Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			continue
		}

		identifier, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || identifier.Name != "deps" && identifier.Name != "components" {
			continue
		}

		expression := assignment.Rhs[0]
		if call, ok := expression.(*ast.CallExpr); ok && len(call.Args) == 2 {
			name, isName := call.Fun.(*ast.Ident)
			if isName && name.Name == "append" && call.Ellipsis.IsValid() {
				expression = call.Args[0]
			}
		}

		literal, ok := expression.(*ast.CompositeLit)
		if !ok || !isAnySlice(literal.Type) {
			return index, nil
		}

		return index, literal
	}

	return -1, nil
}

func isAnySlice(expression ast.Expr) bool {
	slice, ok := expression.(*ast.ArrayType)
	if !ok || slice.Len != nil {
		return false
	}

	element, ok := slice.Elt.(*ast.Ident)

	return ok && element.Name == "any"
}

func declaredBefore(statements []ast.Stmt) map[string]bool {
	result := make(map[string]bool)

	for _, statement := range statements {
		switch value := statement.(type) {
		case *ast.AssignStmt:
			if value.Tok != token.DEFINE {
				continue
			}

			for _, expression := range value.Lhs {
				if identifier, ok := expression.(*ast.Ident); ok {
					result[identifier.Name] = true
				}
			}
		case *ast.DeclStmt:
			declaration, ok := value.Decl.(*ast.GenDecl)
			if !ok || declaration.Tok != token.VAR {
				continue
			}

			for _, specification := range declaration.Specs {
				values, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, name := range values.Names {
					result[name.Name] = true
				}
			}
		}
	}

	return result
}

func identifierExists(file *ast.File, name string) bool {
	found := false

	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && identifier.Name == name {
			found = true

			return false
		}

		return !found
	})

	return found
}

func importPathExists(file *ast.File, expected string) bool {
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err == nil && path == expected {
			return true
		}
	}

	return false
}

func wiringStatements(featureAlias, storeName, serviceName, handlerName, templateName string) []ast.Stmt {
	return []ast.Stmt{
		defineCall(storeName, featureAlias, "NewPostgresStore", "database"),
		defineCall(serviceName, featureAlias, "NewService", storeName),
		defineCall(handlerName, featureAlias, "NewHandler", serviceName, templateName, "logger"),
	}
}

func defineCall(target, receiver, function string, arguments ...string) ast.Stmt {
	values := make([]ast.Expr, 0, len(arguments))
	for _, argument := range arguments {
		values = append(values, ast.NewIdent(argument))
	}

	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(target)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent(function)},
			Args: values,
		}},
	}
}

func insertStatements(existing []ast.Stmt, index int, added []ast.Stmt) []ast.Stmt {
	result := make([]ast.Stmt, 0, len(existing)+len(added))
	result = append(result, existing[:index]...)
	result = append(result, added...)
	result = append(result, existing[index:]...)

	return result
}

func insertLifecycleDependencies(dependencies *ast.CompositeLit, storeName, handlerName string) {
	insertAt := 0

	for index, expression := range dependencies.Elts {
		identifier, ok := expression.(*ast.Ident)
		if ok && (identifier.Name == "database" || identifier.Name == "migrator" || identifier.Name == "tmplMgr" || identifier.Name == "templates") {
			insertAt = index + 1
		}
	}

	added := []ast.Expr{ast.NewIdent(storeName), ast.NewIdent(handlerName)}

	dependencies.Elts = append(dependencies.Elts, nil, nil)
	copy(dependencies.Elts[insertAt+len(added):], dependencies.Elts[insertAt:len(dependencies.Elts)-len(added)])
	copy(dependencies.Elts[insertAt:], added)
}

func addNamedImport(file *ast.File, name, path string) {
	specification := &ast.ImportSpec{
		Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(path)},
	}
	if name != "" {
		specification.Name = ast.NewIdent(name)
	}

	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			continue
		}

		if !imports.Lparen.IsValid() {
			imports.Lparen = imports.Pos()
		}

		insertAt := len(imports.Specs)
		for index, existing := range imports.Specs {
			imported, importOK := existing.(*ast.ImportSpec)
			if !importOK {
				continue
			}

			value, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr == nil && strings.Contains(value, ".") {
				insertAt = index

				if specification.Name != nil {
					specification.Name.NamePos = imported.Pos()
				}

				specification.Path.ValuePos = imported.Pos()

				break
			}
		}

		imports.Specs = append(imports.Specs, nil)
		copy(imports.Specs[insertAt+1:], imports.Specs[insertAt:len(imports.Specs)-1])
		imports.Specs[insertAt] = specification
		file.Imports = append(file.Imports, specification)

		return
	}

	declaration := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{specification}}
	file.Decls = append([]ast.Decl{declaration}, file.Decls...)
	file.Imports = append(file.Imports, specification)
}

func wiringPrerequisite(target, prerequisite string) error {
	return executionError(
		"execution_wiring_prerequisite_missing",
		target,
		"composition root requires canonical %s before feature wiring",
		prerequisite,
	)
}
