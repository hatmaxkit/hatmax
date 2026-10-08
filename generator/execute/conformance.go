// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// ConformanceResult reports independent Hatmax Book evaluation of an
// executed plan.
type ConformanceResult struct {
	Passed      bool         `json:"passed" yaml:"passed"`
	Diagnostics []Diagnostic `json:"diagnostics" yaml:"diagnostics"`
}

type conformanceSnapshot struct {
	value     plan.Plan
	manifest  Manifest
	inventory project.Inventory
	files     map[string][]byte
}

// CheckConformance evaluates selected Book rules against the current project
// without modifying it or treating ordinary test success as conformance.
func CheckConformance(value plan.Plan, manifest Manifest, inventory project.Inventory) (ConformanceResult, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return ConformanceResult{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return ConformanceResult{}, err
	}

	if manifest.PlanDigest != value.Digest || manifest.ProjectFingerprint != value.ProjectFingerprint {
		return ConformanceResult{}, executionError("execution_identity_mismatch", "manifest", "manifest and plan identities do not match")
	}

	snapshot, err := loadConformanceSnapshot(value, manifest, inventory)
	if err != nil {
		return ConformanceResult{}, err
	}

	diagnostics := make([]Diagnostic, 0)
	checkFeaturePackage(snapshot, &diagnostics)
	checkHandlerBoundary(snapshot, &diagnostics)
	checkLifecycleWiring(snapshot, &diagnostics)
	checkPersistence(snapshot, &diagnostics)
	checkServerRenderedHTMX(snapshot, &diagnostics)
	checkLayeredValidation(snapshot, &diagnostics)
	checkRequiredTests(snapshot, &diagnostics)
	checkDocumentationConformance(snapshot, &diagnostics)

	return ConformanceResult{Passed: len(diagnostics) == 0, Diagnostics: diagnostics}, nil
}

func loadConformanceSnapshot(value plan.Plan, manifest Manifest, inventory project.Inventory) (conformanceSnapshot, error) {
	result := conformanceSnapshot{
		value: value, manifest: cloneManifest(manifest), inventory: inventory, files: make(map[string][]byte),
	}

	for _, file := range inventory.Files() {
		if file.Size > MaximumStagedFileSize {
			continue
		}

		content, err := os.ReadFile(filepath.Join(inventory.Root, filepath.FromSlash(file.Path)))
		if err != nil {
			return conformanceSnapshot{}, executionError("execution_conformance_read_failed", file.Path, "%v", err)
		}

		result.files[file.Path] = content
	}

	return result, nil
}

func checkFeaturePackage(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.feature.cohesive_package"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	for _, base := range []string{"model.go", "store.go", "postgres_store.go", "service.go", "handler.go"} {
		path, found := conformanceFeatureFile(snapshot, base)
		if !found {
			addConformanceDiagnostic(diagnostics, "HMGEN-FEATURE-OWNERSHIP", rule, "model", base, "missing", "feature-owned "+base)

			continue
		}

		_, parseErr := parser.ParseFile(token.NewFileSet(), path, snapshot.files[path], parser.AllErrors)
		if parseErr != nil {
			addConformanceDiagnostic(diagnostics, "HMGEN-FEATURE-OWNERSHIP", rule, "model", path, "invalid Go source", "parseable feature-owned Go source")
		}
	}

	model := conformanceContent(snapshot, "model.go")
	if !containsAll(model, "hatmax.adrianpk.com/model", "hatmax.adrianpk.com/validation", "Validate() error") {
		addConformanceDiagnostic(diagnostics, "HMGEN-DEPENDENCY-SUBSTITUTION", "hatmax.dependencies.use_owned_primitive", "model", "model.go", "Hatmax model or validation primitive missing", "Hatmax-owned model and validation primitives")
	}
}

func checkHandlerBoundary(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.http.feature_owned_handler"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	handler := conformanceContent(snapshot, "handler.go")
	if !containsAll(handler, "type FeatureService interface", "RegisterRoutes", "chi.Router", "web.ParseForm") {
		addConformanceDiagnostic(diagnostics, "HMGEN-HANDLER-BOUNDARY", rule, "handler", "handler.go", "canonical handler boundary incomplete", "consumer-owned service interface and Chi route registration")
	}

	if strings.Contains(handler, "database/sql") || strings.Contains(handler, "/internal/dal") {
		addConformanceDiagnostic(diagnostics, "HMGEN-HANDLER-BOUNDARY", rule, "handler", "handler.go", "handler accesses persistence directly", "handler depends only on its service boundary")
	}
}

func checkLifecycleWiring(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.lifecycle.explicit_wiring"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	entrypoint, content := conformanceEntrypoint(snapshot)
	markers := []string{
		featureImportPath(snapshot.value, snapshot.inventory),
		snapshot.value.Feature + "Store :=",
		snapshot.value.Feature + "Service :=",
		snapshot.value.Feature + "Handler :=",
	}

	if entrypoint == "" || !containsAll(content, markers...) {
		addConformanceDiagnostic(diagnostics, "HMGEN-WIRING-ORDER", rule, "wiring", entrypoint, "explicit feature wiring incomplete", "visible store, service, handler, and dependency-list composition")

		return
	}

	store := strings.Index(content, snapshot.value.Feature+"Store :=")
	service := strings.Index(content, snapshot.value.Feature+"Service :=")
	handler := strings.Index(content, snapshot.value.Feature+"Handler :=")

	dependencies := strings.Index(content, "deps := []any")
	if dependencies < 0 {
		dependencies = strings.Index(content, "components := append([]any{")
	}

	if !(store < service && service < handler && handler < dependencies) {
		addConformanceDiagnostic(diagnostics, "HMGEN-WIRING-ORDER", rule, "wiring", entrypoint, "feature construction order is invalid", "store, service, handler, then dependency list")
	}
}

func checkPersistence(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.persistence.postgres_sqlc"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	queries := conformanceQuery(snapshot)
	if !containsAll(queries, "-- name: List", "-- name: Get", "-- name: Create", "-- name: Update", "-- name: Delete") {
		addConformanceDiagnostic(diagnostics, "HMGEN-SQLC-QUERY", rule, "store", snapshot.value.Feature+".sql", "CRUD query set incomplete", "named SQLC list, get, create, update, and delete queries")
	}

	store := conformanceContent(snapshot, "postgres_store.go")
	if !containsAll(store, "dal.New(database)", "ErrNotFound", "Start(context.Context)", "Stop(context.Context)") {
		addConformanceDiagnostic(diagnostics, "HMGEN-STORE-MAPPING", rule, "store", "postgres_store.go", "Postgres adapter lifecycle or mapping incomplete", "SQLC adapter with lifecycle and stable not-found translation")
	}

	if !hasReversibleMigration(snapshot) {
		addConformanceDiagnostic(diagnostics, "HMGEN-POSTGRES-MIGRATION", rule, "migration", "db migrations", "reversible migration missing", "migration containing Up and Down sections")
	}
}

func checkServerRenderedHTMX(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.web.server_rendered_htmx"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	form := conformanceTemplate(snapshot, "form.html")
	row := conformanceTemplate(snapshot, "row.html")
	handler := conformanceContent(snapshot, "handler.go")

	if !containsAll(form, "hxAttrs", "hxPost") || !containsAll(row, "hxPut", "hxDelete") || !containsAll(handler, "htmx.IsHTMXRequest", "RenderPartial") {
		addConformanceDiagnostic(diagnostics, "HMGEN-HTMX-HELPER", rule, "templates", "assets/templates", "Hatmax HTMX helper path incomplete", "Hatmax helpers and full or partial response selection")
	}

	if strings.Contains(form, " hx-") || strings.Contains(row, " hx-") {
		addConformanceDiagnostic(diagnostics, "HMGEN-HTMX-HELPER", rule, "templates", "assets/templates", "raw HTMX attribute found", "Hatmax HTMX helper")
	}
}

func checkLayeredValidation(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.validation.layered"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	validation := snapshot.value.Domain.Validation
	if validation == nil || validation.Scope == intent.ValidationClientOnly {
		return
	}

	model := conformanceContent(snapshot, "model.go")
	form := conformanceTemplate(snapshot, "form.html")

	if validation.Kind != "unique" && !strings.Contains(model, validationMessage(*validation)) {
		addConformanceDiagnostic(diagnostics, "HMGEN-VALIDATION-OWNER", rule, "model", "model.go", "durable rule missing from domain", "domain validation for "+validation.Field)
	}

	if !strings.Contains(form, `name="`+validation.Field+`"`) {
		addConformanceDiagnostic(diagnostics, "HMGEN-VALIDATION-OWNER", rule, "templates", "form.html", "validated field missing from form", "form field "+validation.Field)
	}
}

func checkRequiredTests(snapshot conformanceSnapshot, diagnostics *[]Diagnostic) {
	rule := "hatmax.testing.required_boundaries"
	if !selectedRule(snapshot.value, rule) {
		return
	}

	for _, base := range []string{"model_test.go", "service_test.go", "handler_test.go", "postgres_store_test.go"} {
		content := conformanceContent(snapshot, base)
		if !strings.Contains(content, "func Test") {
			addConformanceDiagnostic(diagnostics, "HMGEN-TEST-BOUNDARY", rule, "tests", base, "behavioral test missing", "at least one behavioral Test function")
		}
	}
}

func selectedRule(value plan.Plan, expected string) bool {
	for _, rule := range value.Rules {
		if rule.ID == expected {
			return true
		}
	}

	return false
}

func conformanceFeatureFile(snapshot conformanceSnapshot, base string) (string, bool) {
	return canonicalFeaturePath(snapshot.inventory, snapshot.value.Feature, "feature", base)
}

func conformanceContent(snapshot conformanceSnapshot, base string) string {
	path, found := conformanceFeatureFile(snapshot, base)
	if !found {
		return ""
	}

	return string(snapshot.files[path])
}

func conformanceQuery(snapshot conformanceSnapshot) string {
	path, found := canonicalFeaturePath(snapshot.inventory, snapshot.value.Feature, "queries", snapshot.value.Feature+".sql")
	if !found {
		return ""
	}

	return string(snapshot.files[path])
}

func conformanceTemplate(snapshot conformanceSnapshot, base string) string {
	role := strings.TrimSuffix(base, ".html") + "_template"

	path, found := canonicalFeaturePath(snapshot.inventory, snapshot.value.Feature, role, base)
	if !found {
		return ""
	}

	return string(snapshot.files[path])
}

func conformanceEntrypoint(snapshot conformanceSnapshot) (string, string) {
	for _, composition := range snapshot.inventory.CompositionRoots() {
		return composition, string(snapshot.files[composition])
	}

	return "", ""
}

func hasReversibleMigration(snapshot conformanceSnapshot) bool {
	for path, content := range snapshot.files {
		file, exists := snapshot.inventory.File(path)
		if exists && containsString(file.Surfaces, "migration") && containsAll(string(content), "-- +migrate Up", "-- +migrate Down") {
			return true
		}
	}

	return false
}

func containsAll(content string, markers ...string) bool {
	for _, marker := range markers {
		if !strings.Contains(content, marker) {
			return false
		}
	}

	return true
}

func addConformanceDiagnostic(
	diagnostics *[]Diagnostic,
	code string,
	rule string,
	surface string,
	location string,
	observed string,
	expected string,
) {
	*diagnostics = append(*diagnostics, Diagnostic{
		Code: code, Rule: rule, Severity: SeverityError, Surface: surface, Location: location,
		Observed: observed, Expected: expected, RepairableWithinPlan: false,
	})
}
