// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/plan"
)

// CheckApplicationConformance evaluates the staged application independently
// from compilation and generated tests.
func CheckApplicationConformance(value plan.Plan, manifest Manifest, root string) (ConformanceResult, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return ConformanceResult{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return ConformanceResult{}, err
	}

	if manifest.PlanDigest != value.Digest || manifest.SourceFingerprint != value.SourceFingerprint {
		return ConformanceResult{}, executionError("execution_identity_mismatch", "manifest", "manifest and application plan identities do not match")
	}

	diagnostics := make([]Diagnostic, 0)
	checkApplicationEffects(value, manifest, &diagnostics)
	checkApplicationModule(value, root, &diagnostics)
	checkApplicationMain(root, &diagnostics)
	checkApplicationComposition(value, root, &diagnostics)
	checkApplicationWeb(root, &diagnostics)

	return ConformanceResult{Passed: len(diagnostics) == 0, Diagnostics: diagnostics}, nil
}

func checkApplicationEffects(value plan.Plan, manifest Manifest, diagnostics *[]Diagnostic) {
	planned := make([]string, 0, len(value.AllowedEffects.Files))
	for _, effect := range value.AllowedEffects.Files {
		planned = append(planned, effect.Path)
	}

	prepared := make([]string, 0, len(manifest.Edits))
	for _, edit := range manifest.Edits {
		prepared = append(prepared, edit.Target)
		if len(value.Units) == 1 && strings.Contains(edit.Target, "migration") {
			addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-BOOTSTRAP-MIGRATION", "hatmax.application.no_bootstrap_migration", edit.Target, "migration scaffold", "no migration before a domain owner")
		}
	}

	sort.Strings(planned)
	sort.Strings(prepared)

	if !sameStrings(planned, prepared) {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-EFFECTS", "", "manifest", strings.Join(prepared, ","), strings.Join(planned, ","))
	}
}

func checkApplicationModule(value plan.Plan, root string, diagnostics *[]Diagnostic) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-MODULE", "hatmax.application.module_identity", "go.mod", err.Error(), "readable module declaration")

		return
	}

	version := value.HatmaxVersion
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	text := string(content)
	if !strings.Contains(text, "module "+value.Application.ModulePath) || !strings.Contains(text, "hatmax.adrianpk.com "+version) {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-MODULE", "hatmax.application.module_identity", "go.mod", "module identity differs", "admitted module path and Hatmax version")
	}
}

func checkApplicationMain(root string, diagnostics *[]Diagnostic) {
	target := filepath.Join(root, "main.go")

	file, err := parser.ParseFile(token.NewFileSet(), target, nil, parser.SkipObjectResolution)
	if err != nil {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-THIN-MAIN", "hatmax.application.thin_main", "main.go", err.Error(), "parseable thin main")

		return
	}

	functions := make([]*ast.FuncDecl, 0)

	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok {
			functions = append(functions, function)

			continue
		}

		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.IMPORT {
			addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-THIN-MAIN", "hatmax.application.thin_main", "main.go", "non-import declaration", "only imports and main")
		}
	}

	if len(functions) != 1 || functions[0].Name.Name != "main" || functions[0].Recv != nil {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-THIN-MAIN", "hatmax.application.thin_main", "main.go", "unexpected functions", "exactly one main function")
	}

	content, readErr := os.ReadFile(target)
	if readErr != nil || !strings.Contains(string(content), "application.Run(") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-THIN-MAIN", "hatmax.application.thin_main", "main.go", "missing application delegation", "application.Run")
	}
}

func checkApplicationComposition(value plan.Plan, root string, diagnostics *[]Diagnostic) {
	application := applicationContent(root, "internal/application/application.go", diagnostics)
	database := applicationContent(root, "internal/application/database.go", diagnostics)

	if !containsAll(application, "app.Setup", "app.Start", "app.Shutdown", "newDatabase", "newWeb") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-COMPOSITION", "hatmax.application.explicit_composition", "internal/application/application.go", "incomplete lifecycle", "explicit setup, start, serve, and shutdown")
	}

	if !strings.Contains(database, "db.New(") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-BOOTSTRAP-MIGRATION", "hatmax.application.no_bootstrap_migration", "internal/application/database.go", "invalid database lifecycle", "database wiring without migrator")
	}

	if len(value.Units) == 1 && strings.Contains(application+database, "NewMigrator") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-BOOTSTRAP-MIGRATION", "hatmax.application.no_bootstrap_migration", "internal/application/database.go", "invalid database lifecycle", "database wiring without migrator")
	}

	if len(value.Units) > 1 && !strings.Contains(application, "db.NewMigrator") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-MIGRATION", "hatmax.persistence.postgres_sqlc", "internal/application/application.go", "missing migrator", "migrator owned by initial persistent features")
	}
}

func checkApplicationWeb(root string, diagnostics *[]Diagnostic) {
	webSource := applicationContent(root, "internal/application/web.go", diagnostics)
	home := applicationContent(root, "internal/application/assets/templates/pages/home.html", diagnostics)
	layout := applicationContent(root, "internal/application/assets/templates/layouts/base.html", diagnostics)
	styles := applicationContent(root, "internal/application/assets/static/app.css", diagnostics)

	if !containsAll(webSource, "app.NewRouter", "middleware.DefaultStack()", "web.NewTemplateManager", "RegisterRoutes") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-NEUTRAL-WEB", "hatmax.application.neutral_web", "internal/application/web.go", "incomplete web composition", "Hatmax router, middleware, templates, and routes")
	}

	if home == "" || layout == "" || styles == "" || !strings.Contains(home, "ApplicationName") || !strings.Contains(layout, "/static/app.css") {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-NEUTRAL-WEB", "hatmax.application.neutral_web", "internal/application/assets", "incomplete landing surface", "embedded layout, landing page, and stylesheet")
	}
}

func applicationContent(root, target string, diagnostics *[]Diagnostic) string {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(target)))
	if err != nil {
		addApplicationDiagnostic(diagnostics, "HMGEN-APPLICATION-FILE", "", target, err.Error(), "readable scaffold file")

		return ""
	}

	return string(content)
}

func addApplicationDiagnostic(diagnostics *[]Diagnostic, code, rule, location, observed, expected string) {
	*diagnostics = append(*diagnostics, Diagnostic{
		Code: code, Rule: rule, Severity: SeverityError, Location: location,
		Observed: observed, Expected: expected, RepairableWithinPlan: true,
	})
}
