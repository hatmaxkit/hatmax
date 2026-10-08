// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"hatmax.adrianpk.com/htmx"
)

type evidenceInputField struct {
	name   string
	goType string
}

type evidenceFormField struct {
	name       string
	label      string
	htmlType   string
	required   bool
	attributes string
}

var (
	evidenceFormInputPattern  = regexp.MustCompile(`(?m)<label>([^\n]+)\n\s+<input type="([^"]+)" name="([^"]+)"([^>]*)>`)
	evidenceQueryTablePattern = regexp.MustCompile(`(?m)^FROM ([a-z][a-z0-9_]*)$`)
	evidenceRoutePattern      = regexp.MustCompile(`router\.Get\("([^"]+)"`)
	evidenceLabelPattern      = regexp.MustCompile(`PageView\{Title: "([^"]+)"`)
)

// InspectFeatureEvidence extracts the bounded canonical behavior needed for
// documentation planning. It never returns source bodies or arbitrary prose.
func (i Inventory) InspectFeatureEvidence(feature string) (FeatureEvidence, error) {
	featureRoot, err := i.canonicalFeatureRoot(feature)
	if err != nil {
		return FeatureEvidence{}, err
	}

	paths, err := i.canonicalEvidencePaths(feature, featureRoot)
	if err != nil {
		return FeatureEvidence{}, err
	}

	contents := make(map[string][]byte, len(paths))
	sources := make([]FeatureEvidenceSource, 0, len(paths))

	for _, role := range sortedEvidenceRoles(paths) {
		filePath := paths[role]

		content, readErr := i.readEvidenceFile(filePath)
		if readErr != nil {
			return FeatureEvidence{}, readErr
		}

		contents[role] = content

		digest, digestErr := i.digestProjectFile(filePath)
		if digestErr != nil {
			return FeatureEvidence{}, digestErr
		}

		sources = append(sources, FeatureEvidenceSource{Role: role, Path: filePath, Digest: digest})
	}

	inputs, entity, err := inspectEvidenceModel(contents["model"])
	if err != nil {
		return FeatureEvidence{}, featureEvidenceError(paths["model"], "%v", err)
	}

	formFields, err := inspectEvidenceForm(paths["form_template"], contents["form_template"])
	if err != nil {
		return FeatureEvidence{}, err
	}

	if len(inputs) == 0 || len(inputs) != len(formFields) {
		return FeatureEvidence{}, featureEvidenceError(paths["form_template"], "model input and form field counts differ")
	}

	tableMatch := evidenceQueryTablePattern.FindSubmatch(contents["queries"])
	if len(tableMatch) != 2 {
		return FeatureEvidence{}, featureEvidenceError(paths["queries"], "canonical table query is missing")
	}

	table := string(tableMatch[1])

	schemaTypes, migrationPath, err := i.inspectEvidenceSchema(table)
	if err != nil {
		return FeatureEvidence{}, err
	}

	migrationDigest, err := i.digestProjectFile(migrationPath)
	if err != nil {
		return FeatureEvidence{}, err
	}

	sources = append(sources, FeatureEvidenceSource{Role: "migration", Path: migrationPath, Digest: migrationDigest})
	sort.Slice(sources, func(left, right int) bool {
		return sources[left].Role < sources[right].Role
	})

	routeMatch := evidenceRoutePattern.FindSubmatch(contents["handler"])
	labelMatch := evidenceLabelPattern.FindSubmatch(contents["handler"])

	if len(routeMatch) != 2 || len(labelMatch) != 2 {
		return FeatureEvidence{}, featureEvidenceError(paths["handler"], "canonical route or page label is missing")
	}

	fields, validations, err := inspectEvidenceFields(inputs, formFields, schemaTypes, contents["model"])
	if err != nil {
		return FeatureEvidence{}, err
	}

	err = validateEvidenceAnchors(feature, i.Module.Path, contents)
	if err != nil {
		return FeatureEvidence{}, err
	}

	return FeatureEvidence{
		Basis:             "existing",
		Feature:           feature,
		Entity:            entity,
		Label:             string(labelMatch[1]),
		Route:             string(routeMatch[1]),
		Table:             table,
		Fields:            fields,
		Validations:       validations,
		Postgres:          true,
		HTMX:              true,
		RuntimeValidation: strings.Contains(string(contents["model"]), "Validate() error"),
		Wired:             true,
		Tested:            true,
		Sources:           sources,
	}, nil
}

func sortedEvidenceRoles(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for role := range values {
		result = append(result, role)
	}

	sort.Strings(result)

	return result
}

func (i Inventory) canonicalFeatureRoot(feature string) (string, error) {
	result := ""

	for _, candidate := range i.Layout.Features {
		if path.Base(candidate) != feature {
			continue
		}

		if result != "" {
			return "", featureEvidenceError(feature, "multiple canonical feature roots exist")
		}

		result = candidate
	}

	if result == "" {
		return "", featureEvidenceError(feature, "canonical feature root is missing")
	}

	return result, nil
}

func (i Inventory) canonicalEvidencePaths(feature, featureRoot string) (map[string]string, error) {
	paths := map[string]string{
		"model":                path.Join(featureRoot, "model.go"),
		"store":                path.Join(featureRoot, "store.go"),
		"postgres_store":       path.Join(featureRoot, "postgres_store.go"),
		"service":              path.Join(featureRoot, "service.go"),
		"handler":              path.Join(featureRoot, "handler.go"),
		"model_tests":          path.Join(featureRoot, "model_test.go"),
		"service_tests":        path.Join(featureRoot, "service_test.go"),
		"handler_tests":        path.Join(featureRoot, "handler_test.go"),
		"postgres_store_tests": path.Join(featureRoot, "postgres_store_test.go"),
	}

	for _, role := range []string{"page_template", "form_template", "row_template"} {
		base := strings.TrimSuffix(role, "_template") + ".html"

		candidate, found := i.singleEvidencePath(base, "/templates/"+feature+"/")
		if !found {
			return nil, featureEvidenceError(base, "canonical feature template is missing")
		}

		paths[role] = candidate
	}

	query, found := i.singleEvidencePath(feature+".sql", "")
	if !found {
		return nil, featureEvidenceError(feature+".sql", "canonical feature query is missing")
	}

	paths["queries"] = query

	entrypoint := ""

	for _, candidate := range i.CompositionRoots() {
		if entrypoint != "" {
			return nil, featureEvidenceError("wiring", "multiple Hatmax composition roots exist")
		}

		entrypoint = candidate
	}

	if entrypoint == "" {
		return nil, featureEvidenceError("wiring", "Hatmax composition root is missing")
	}

	paths["wiring"] = entrypoint

	for role, filePath := range paths {
		if _, exists := i.File(filePath); !exists {
			return nil, featureEvidenceError(filePath, "canonical %s source is missing", role)
		}
	}

	return paths, nil
}

func (i Inventory) singleEvidencePath(base, segment string) (string, bool) {
	result := ""

	for _, file := range i.Files() {
		candidate := "/" + file.Path
		if filepath.Base(file.Path) != base || (segment != "" && !strings.Contains(candidate, segment)) {
			continue
		}

		if result != "" {
			return "", false
		}

		result = file.Path
	}

	return result, result != ""
}

func (i Inventory) readEvidenceFile(filePath string) ([]byte, error) {
	file, exists := i.File(filePath)
	if !exists {
		return nil, featureEvidenceError(filePath, "canonical evidence source is missing")
	}

	if file.Size > i.maximumFileSize {
		return nil, featureEvidenceError(filePath, "canonical evidence source exceeds the inspection limit")
	}

	absolute, err := projectPath(i.Root, filePath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absolute)
	if err != nil {
		return nil, unexpectedReadError(filePath, err)
	}

	return content, nil
}

func inspectEvidenceModel(content []byte) ([]evidenceInputField, string, error) {
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
			if !typeOK || !strings.HasSuffix(typeSpec.Name.Name, "Input") {
				continue
			}

			structure, structOK := typeSpec.Type.(*ast.StructType)
			if !structOK {
				continue
			}

			fields := make([]evidenceInputField, 0, len(structure.Fields.List))
			for _, field := range structure.Fields.List {
				if len(field.Names) != 1 {
					return nil, "", fmt.Errorf("input fields must be named individually")
				}

				fields = append(fields, evidenceInputField{name: field.Names[0].Name, goType: evidenceExpressionName(field.Type)})
			}

			return fields, strings.TrimSuffix(typeSpec.Name.Name, "Input"), nil
		}
	}

	return nil, "", fmt.Errorf("canonical input structure is missing")
}

func evidenceExpressionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return evidenceExpressionName(value.X) + "." + value.Sel.Name
	default:
		return ""
	}
}

func inspectEvidenceForm(filePath string, content []byte) ([]evidenceFormField, error) {
	_, err := template.New(filePath).Funcs(htmx.FuncMap()).Parse(string(content))
	if err != nil {
		return nil, featureEvidenceError(filePath, "invalid canonical template: %v", err)
	}

	matches := evidenceFormInputPattern.FindAllSubmatch(content, -1)
	result := make([]evidenceFormField, 0, len(matches))

	for _, match := range matches {
		attributes := string(match[4])
		result = append(result, evidenceFormField{
			name:       string(match[3]),
			label:      strings.TrimSpace(string(match[1])),
			htmlType:   string(match[2]),
			required:   strings.Contains(attributes, " required"),
			attributes: attributes,
		})
	}

	return result, nil
}

func (i Inventory) inspectEvidenceSchema(table string) (map[string]string, string, error) {
	pattern := regexp.MustCompile(`(?s)CREATE TABLE ` + regexp.QuoteMeta(table) + ` \((.*?)\);`)

	for _, file := range i.Files() {
		if filepath.Ext(file.Path) != ".sql" || !containsValue(file.Surfaces, "migration") {
			continue
		}

		content, err := i.readEvidenceFile(file.Path)
		if err != nil {
			return nil, "", err
		}

		match := pattern.FindSubmatch(content)
		if len(match) != 2 {
			continue
		}

		result := make(map[string]string)

		for _, line := range strings.Split(string(match[1]), ",") {
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) >= 2 {
				result[parts[0]] = parts[1]
			}
		}

		return result, file.Path, nil
	}

	return nil, "", featureEvidenceError("migration", "CREATE TABLE %s is missing", table)
}

func inspectEvidenceFields(
	inputs []evidenceInputField,
	formFields []evidenceFormField,
	schemaTypes map[string]string,
	model []byte,
) ([]FeatureFieldEvidence, []FeatureValidationEvidence, error) {
	fields := make([]FeatureFieldEvidence, 0, len(inputs))
	validations := make([]FeatureValidationEvidence, 0)

	for index, input := range inputs {
		form := formFields[index]
		if input.name != exportedEvidenceName(form.name) {
			return nil, nil, featureEvidenceError("form", "field %q does not match model field %q", form.name, input.name)
		}

		kind := evidenceFieldKind(input, form, schemaTypes[form.name], model)
		if kind == "" {
			return nil, nil, featureEvidenceError("model", "field %q has inconsistent representations", form.name)
		}

		fields = append(fields, FeatureFieldEvidence{
			Name: form.name, Type: kind, Label: form.label, Required: form.required,
		})

		validations = append(validations, evidenceAttributeValidations(form)...)
	}

	return fields, validations, nil
}

func evidenceFieldKind(input evidenceInputField, form evidenceFormField, sqlType string, model []byte) string {
	switch form.htmlType {
	case "checkbox":
		if input.goType == "bool" {
			return "boolean"
		}
	case "date":
		if input.goType == "time.Time" {
			return "date"
		}
	case "datetime-local":
		if input.goType == "time.Time" {
			return "timestamp"
		}
	case "number":
		if input.goType == "int64" {
			return "integer"
		}
	default:
		if input.goType != "string" {
			return ""
		}

		if strings.HasPrefix(sqlType, "VARCHAR") {
			return "string"
		}

		if strings.Contains(string(model), "model.ParseID(value."+input.name+")") {
			return "uuid"
		}

		return "text"
	}

	return ""
}

func evidenceAttributeValidations(field evidenceFormField) []FeatureValidationEvidence {
	result := make([]FeatureValidationEvidence, 0)
	if field.required {
		result = append(result, FeatureValidationEvidence{Field: field.name, Kind: "required", Scope: "durable"})
	}

	attributes := []struct {
		name string
		kind string
	}{
		{name: "min", kind: "minimum"},
		{name: "max", kind: "maximum"},
		{name: "minlength", kind: "min_length"},
		{name: "maxlength", kind: "max_length"},
		{name: "pattern", kind: "pattern"},
	}

	for _, attribute := range attributes {
		pattern := regexp.MustCompile(`\s` + attribute.name + `="([^"]+)"`)

		match := pattern.FindStringSubmatch(field.attributes)
		if len(match) == 2 {
			result = append(result, FeatureValidationEvidence{Field: field.name, Kind: attribute.kind, Value: match[1], Scope: "durable"})
		}
	}

	return result
}

func validateEvidenceAnchors(feature, module string, contents map[string][]byte) error {
	required := map[string][]string{
		"store":                {"type Store interface", "Create(context.Context", "Update(context.Context"},
		"postgres_store":       {"type PostgresStore struct", "func NewPostgresStore", "func (store *PostgresStore) Start"},
		"service":              {"type Service struct", "func NewService", "func (service *Service) Create"},
		"handler":              {"type Handler struct", "RegisterRoutes", "func parseInput", "web.ParseForm"},
		"page_template":        {"{{template", "-form", "-list"},
		"form_template":        {"hxPost", "hxTargetID"},
		"row_template":         {"hxPut", "hxDelete"},
		"queries":              {"-- name: List", "-- name: Get", "-- name: Create", "-- name: Update", "-- name: Delete"},
		"model_tests":          {"func Test"},
		"service_tests":        {"func Test"},
		"handler_tests":        {"func Test"},
		"postgres_store_tests": {"func Test"},
	}

	for role, markers := range required {
		for _, marker := range markers {
			if !strings.Contains(string(contents[role]), marker) {
				return featureEvidenceError(role, "canonical marker %q is missing", marker)
			}
		}
	}

	wiring := string(contents["wiring"])
	for _, marker := range []string{path.Join(module, "internal/feat", feature), feature + "Store", feature + "Service", feature + "Handler"} {
		if !strings.Contains(wiring, marker) {
			return featureEvidenceError("wiring", "canonical marker %q is missing", marker)
		}
	}

	return nil
}

func exportedEvidenceName(value string) string {
	parts := strings.Split(value, "_")
	for index := range parts {
		if parts[index] != "" {
			parts[index] = strings.ToUpper(parts[index][:1]) + parts[index][1:]
		}
	}

	return strings.Join(parts, "")
}

func featureEvidenceError(location, format string, arguments ...any) error {
	return projectError("project_feature_evidence_invalid", location, format, arguments...)
}
