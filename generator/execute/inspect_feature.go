// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
	"hatmax.adrianpk.com/htmx"
)

type canonicalFeature struct {
	entity   string
	table    string
	route    string
	label    string
	fields   []renderField
	contents map[string][]byte
}

type discoveredInputField struct {
	name   string
	goType string
}

type discoveredFormField struct {
	name     string
	label    string
	htmlType string
	required bool
}

var (
	formInputPattern  = regexp.MustCompile(`(?m)<label>([^\n]+)\n\s+<input type="([^"]+)" name="([^"]+)"[^>]*>`)
	queryTablePattern = regexp.MustCompile(`(?m)^FROM ([a-z][a-z0-9_]*)$`)
	routePattern      = regexp.MustCompile(`router\.Get\("([^"]+)"`)
	labelPattern      = regexp.MustCompile(`PageView\{Title: "([^"]+)"`)
)

func inspectCanonicalFeature(value plan.Plan, manifest Manifest, inventory project.Inventory) (canonicalFeature, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return canonicalFeature{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return canonicalFeature{}, err
	}

	if manifest.PlanDigest != value.Digest || manifest.ProjectFingerprint != value.ProjectFingerprint {
		return canonicalFeature{}, executionError("execution_identity_mismatch", "manifest", "manifest and plan identities do not match")
	}

	if value.Archetype != "server_rendered_crud" || (value.Intent != intent.OperationAddField && value.Intent != intent.OperationAddValidation) {
		return canonicalFeature{}, executionError("execution_recipe_unsupported", "intent", "feature inspection requires an incremental server_rendered_crud plan")
	}

	result := canonicalFeature{contents: make(map[string][]byte)}

	for _, edit := range manifest.Edits {
		if edit.Kind == EditCreateFile {
			continue
		}

		content, readErr := readFeatureTarget(inventory.Root, edit)
		if readErr != nil {
			return canonicalFeature{}, readErr
		}

		result.contents[edit.ID] = content

		validateErr := validateFeatureTarget(edit, content)
		if validateErr != nil {
			return canonicalFeature{}, validateErr
		}
	}

	canonicalFiles := []struct {
		role   string
		base   string
		recipe string
	}{
		{role: "model", base: "model.go", recipe: "server_rendered_crud.model"},
		{role: "store", base: "store.go", recipe: "server_rendered_crud.store_contract"},
		{role: "postgres_store", base: "postgres_store.go", recipe: "server_rendered_crud.postgres_store"},
		{role: "service", base: "service.go", recipe: "server_rendered_crud.service"},
		{role: "handler", base: "handler.go", recipe: "server_rendered_crud.handler"},
		{role: "model_tests", base: "model_test.go", recipe: "server_rendered_crud.model_tests"},
		{role: "service_tests", base: "service_test.go", recipe: "server_rendered_crud.service_tests"},
		{role: "handler_tests", base: "handler_test.go", recipe: "server_rendered_crud.handler_tests"},
		{role: "postgres_store_tests", base: "postgres_store_test.go", recipe: "server_rendered_crud.postgres_store_tests"},
		{role: "queries", base: value.Feature + ".sql", recipe: "server_rendered_crud.queries"},
		{role: "page_template", base: "page.html", recipe: "server_rendered_crud.page_template"},
		{role: "form_template", base: "form.html", recipe: "server_rendered_crud.form_template"},
		{role: "row_template", base: "row.html", recipe: "server_rendered_crud.row_template"},
	}

	for _, canonical := range canonicalFiles {
		path, found := canonicalFeaturePath(inventory, value.Feature, canonical.role, canonical.base)
		if !found {
			return canonicalFeature{}, featureStructureError(canonical.base, "canonical feature target is missing")
		}

		content, readErr := readFeatureTarget(inventory.Root, Edit{Target: path})
		if readErr != nil {
			return canonicalFeature{}, readErr
		}

		validateErr := validateFeatureTarget(Edit{Target: path, Recipe: canonical.recipe}, content)
		if validateErr != nil {
			return canonicalFeature{}, validateErr
		}

		result.contents[canonical.role] = content
	}

	model := result.contents["model"]

	inputs, entity, err := discoverModel(model)
	if err != nil {
		return canonicalFeature{}, featureStructureError("model.go", "%v", err)
	}

	form := result.contents["form_template"]

	formFields := discoverFormFields(form)
	if len(inputs) == 0 || len(inputs) != len(formFields) {
		return canonicalFeature{}, featureStructureError("form.html", "model input and form field counts differ")
	}

	queries := result.contents["queries"]

	tableMatch := queryTablePattern.FindSubmatch(queries)
	if len(tableMatch) != 2 {
		return canonicalFeature{}, featureStructureError("queries", "canonical table query is missing")
	}

	handler := result.contents["handler"]
	routeMatch := routePattern.FindSubmatch(handler)

	labelMatch := labelPattern.FindSubmatch(handler)
	if len(routeMatch) != 2 || len(labelMatch) != 2 {
		return canonicalFeature{}, featureStructureError("handler.go", "canonical route or page label is missing")
	}

	table := string(tableMatch[1])

	schemaTypes, err := discoverSchemaTypes(inventory, table)
	if err != nil {
		return canonicalFeature{}, err
	}

	fields := make([]renderField, 0, len(inputs))
	for index, input := range inputs {
		formField := formFields[index]
		if input.name != exportedName(formField.name) {
			return canonicalFeature{}, featureStructureError("form.html", "field %q does not match model field %q", formField.name, input.name)
		}

		kind := discoveredFieldKind(input, formField, schemaTypes[formField.name], model)

		field, fieldErr := newRenderField(intent.Field{
			Name: formField.name, Label: formField.label, Type: kind, Required: formField.required,
		})
		if fieldErr != nil || field.GoType != input.goType {
			return canonicalFeature{}, featureStructureError("model.go", "field %q has inconsistent representations", formField.name)
		}

		fields = append(fields, field)
	}

	result.entity = entity
	result.table = table
	result.route = string(routeMatch[1])
	result.label = string(labelMatch[1])
	result.fields = fields

	err = validateFeatureWiring(value, inventory)
	if err != nil {
		return canonicalFeature{}, err
	}

	return result, nil
}

func canonicalFeaturePath(inventory project.Inventory, feature, role, base string) (string, bool) {
	featureSegments := []string{"/feat/" + feature + "/", "/features/" + feature + "/"}
	templateSegment := "/templates/" + feature + "/"

	for _, file := range inventory.Files() {
		candidate := "/" + file.Path
		if filepath.Base(file.Path) != base {
			continue
		}

		switch role {
		case "queries":
			if containsString(file.Surfaces, "store") && filepath.Ext(file.Path) == ".sql" {
				return file.Path, true
			}
		case "page_template", "form_template", "row_template":
			if strings.Contains(candidate, templateSegment) {
				return file.Path, true
			}
		default:
			for _, featureSegment := range featureSegments {
				if strings.Contains(candidate, featureSegment) {
					return file.Path, true
				}
			}
		}
	}

	return "", false
}

func readFeatureTarget(root string, edit Edit) ([]byte, error) {
	target, err := secureTargetPath(root, edit.Target)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return nil, featureStructureError(edit.Target, "read target: %v", err)
	}

	return content, nil
}

func validateFeatureTarget(edit Edit, content []byte) error {
	for _, marker := range requiredFeatureMarkers(edit.Recipe) {
		if !strings.Contains(string(content), marker) {
			return featureStructureError(edit.Target, "canonical marker %q is missing", marker)
		}
	}

	switch filepath.Ext(edit.Target) {
	case ".go":
		parsed, err := parser.ParseFile(token.NewFileSet(), edit.Target, content, 0)
		if err != nil {
			return featureStructureError(edit.Target, "invalid Go source: %v", err)
		}

		if strings.HasSuffix(edit.Target, "_test.go") && !hasTestFunction(parsed) {
			return featureStructureError(edit.Target, "behavioral test declaration is missing")
		}
	case ".html":
		_, err := template.New(edit.Target).Funcs(htmx.FuncMap()).Parse(string(content))
		if err != nil {
			return featureStructureError(edit.Target, "invalid template: %v", err)
		}
	case ".sql":
		if !strings.Contains(string(content), "-- name:") {
			return featureStructureError(edit.Target, "named SQLC queries are missing")
		}
	}

	return nil
}

func requiredFeatureMarkers(recipe string) []string {
	switch {
	case strings.HasSuffix(recipe, "model"):
		return []string{"Input struct", "Validate() error"}
	case strings.HasSuffix(recipe, "store_contract"):
		return []string{"type Store interface", "Create(context.Context", "Update(context.Context"}
	case strings.HasSuffix(recipe, "postgres_store"):
		return []string{"type PostgresStore struct", "func NewPostgresStore", "func (store *PostgresStore) Start"}
	case strings.HasSuffix(recipe, "queries"):
		return []string{"-- name: List", "-- name: Get", "-- name: Create", "-- name: Update", "-- name: Delete"}
	case strings.HasSuffix(recipe, "service"):
		return []string{"type Service struct", "func NewService", "func (service *Service) Create"}
	case strings.HasSuffix(recipe, "handler"):
		return []string{"type Handler struct", "RegisterRoutes", "func parseInput", "web.ParseForm"}
	case strings.HasSuffix(recipe, "page_template"):
		return []string{"{{template", "-form", "-list"}
	case strings.HasSuffix(recipe, "form_template"):
		return []string{"hxPost", "hxTargetID"}
	case strings.HasSuffix(recipe, "row_template"):
		return []string{"hxPut", "hxDelete"}
	default:
		return nil
	}
}

func hasTestFunction(file *ast.File) bool {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && strings.HasPrefix(function.Name.Name, "Test") {
			return true
		}
	}

	return false
}

func discoverModel(content []byte) ([]discoveredInputField, string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "model.go", content, 0)
	if err != nil {
		return nil, "", err
	}

	for _, declaration := range parsed.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || generic.Tok != token.TYPE {
			continue
		}

		for _, specification := range generic.Specs {
			typeSpec, typeOK := specification.(*ast.TypeSpec)
			if !typeOK {
				continue
			}

			structure, structOK := typeSpec.Type.(*ast.StructType)
			if !structOK || !strings.HasSuffix(typeSpec.Name.Name, "Input") {
				continue
			}

			fields := make([]discoveredInputField, 0, len(structure.Fields.List))
			for _, field := range structure.Fields.List {
				if len(field.Names) != 1 {
					return nil, "", fmt.Errorf("input fields must be named individually")
				}

				fields = append(fields, discoveredInputField{name: field.Names[0].Name, goType: expressionName(field.Type)})
			}

			return fields, strings.TrimSuffix(typeSpec.Name.Name, "Input"), nil
		}
	}

	return nil, "", fmt.Errorf("canonical input structure is missing")
}

func expressionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return expressionName(value.X) + "." + value.Sel.Name
	default:
		return ""
	}
}

func discoverFormFields(content []byte) []discoveredFormField {
	matches := formInputPattern.FindAllSubmatch(content, -1)
	result := make([]discoveredFormField, 0, len(matches))

	for _, match := range matches {
		result = append(result, discoveredFormField{
			label: strings.TrimSpace(string(match[1])), htmlType: string(match[2]), name: string(match[3]),
			required: strings.Contains(string(match[0]), " required"),
		})
	}

	return result
}

func discoverSchemaTypes(inventory project.Inventory, table string) (map[string]string, error) {
	pattern := regexp.MustCompile(`(?s)CREATE TABLE ` + regexp.QuoteMeta(table) + ` \((.*?)\);`)
	result := make(map[string]string)

	for _, file := range inventory.Files() {
		if filepath.Ext(file.Path) != ".sql" || !containsString(file.Surfaces, "migration") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(inventory.Root, filepath.FromSlash(file.Path)))
		if err != nil {
			return nil, featureStructureError(file.Path, "read migration: %v", err)
		}

		match := pattern.FindSubmatch(content)
		if len(match) != 2 {
			continue
		}

		for _, line := range strings.Split(string(match[1]), ",") {
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) >= 2 {
				result[parts[0]] = parts[1]
			}
		}
	}

	if len(result) == 0 {
		return nil, featureStructureError("migration", "CREATE TABLE %s is missing", table)
	}

	return result, nil
}

func discoveredFieldKind(input discoveredInputField, form discoveredFormField, sqlType string, model []byte) string {
	switch form.htmlType {
	case "checkbox":
		return "boolean"
	case "date":
		return "date"
	case "datetime-local":
		return "timestamp"
	case "number":
		return "integer"
	}

	if strings.HasPrefix(sqlType, "VARCHAR") {
		return "string"
	}

	if strings.Contains(string(model), "model.ParseID(value."+input.name+")") {
		return "uuid"
	}

	return "text"
}

func validateFeatureWiring(value plan.Plan, inventory project.Inventory) error {
	for _, entrypoint := range inventory.Entrypoints {
		if !entrypoint.CompositionRoot {
			continue
		}

		content, err := os.ReadFile(filepath.Join(inventory.Root, filepath.FromSlash(entrypoint.Path)))
		if err != nil {
			return featureStructureError(entrypoint.Path, "read composition root: %v", err)
		}

		source := string(content)
		for _, marker := range []string{featureImportPath(value, inventory), value.Feature + "Store", value.Feature + "Service", value.Feature + "Handler"} {
			if !strings.Contains(source, marker) {
				return featureStructureError(entrypoint.Path, "canonical wiring marker %q is missing", marker)
			}
		}

		return nil
	}

	return featureStructureError("wiring", "composition root is missing")
}

func featureImportPath(value plan.Plan, inventory project.Inventory) string {
	return filepath.ToSlash(filepath.Join(inventory.Module.Path, "internal", "feat", value.Feature))
}

func featureStructureError(location, format string, arguments ...any) error {
	return executionError("execution_feature_structure_invalid", location, format, arguments...)
}
