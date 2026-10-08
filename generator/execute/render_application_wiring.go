// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

func applicationWiringPrerequisites(file *ast.File, function *ast.FuncDecl, dependencies *ast.CompositeLit, declared map[string]bool) ([]ast.Stmt, error) {
	for _, parameter := range function.Type.Params.List {
		for _, name := range parameter.Names {
			declared[name.Name] = true
		}
	}

	if !declared["database"] || !declared["logger"] {
		return nil, fmt.Errorf("database and logger in application composition")
	}

	var source string

	for _, requirement := range []struct {
		name, statement string
		imports         []string
	}{
		{"migrator", "migrator := db.NewMigrator(database, assetsFS, db.Postgres, logger)", []string{"db"}},
		{"templates", "templates := web.NewTemplateManager(assetsFS, logger, web.WithFuncMap(ui.FuncMap()))", []string{"web", "ui"}},
	} {
		if declared[requirement.name] {
			continue
		}

		if identifierExists(file, requirement.name) {
			return nil, fmt.Errorf("unambiguous %s declaration", requirement.name)
		}

		for _, name := range requirement.imports {
			path := "hatmax.adrianpk.com/" + name
			if !importPathExists(file, path) {
				if identifierExists(file, name) {
					return nil, fmt.Errorf("available %s import name", name)
				}

				addNamedImport(file, "", path)
			}
		}

		source += requirement.statement + "\n"
		dependencies.Elts = append(dependencies.Elts, ast.NewIdent(requirement.name))
	}

	if source == "" {
		return nil, nil
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), "prerequisites.go", "package application\nfunc prerequisites() {\n"+source+"}\n", 0)
	if err != nil {
		return nil, fmt.Errorf("parse application prerequisites: %w", err)
	}

	return parsed.Decls[0].(*ast.FuncDecl).Body.List, nil
}
