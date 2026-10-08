// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const (
	featurePackageOperation   = "archetype.server_rendered_crud.feature_package"
	serviceBoundaryOperation  = "archetype.server_rendered_crud.service_boundary"
	featureHandlerOperation   = "archetype.server_rendered_crud.feature_handler"
	featureTemplatesOperation = "archetype.server_rendered_crud.feature_templates"
	explicitWiringOperation   = "archetype.server_rendered_crud.explicit_wiring"
	featureTestsOperation     = "archetype.server_rendered_crud.feature_tests"
	postgresMigration         = "capability.postgres_persistence.migration"
	postgresQueryAdapter      = "capability.postgres_persistence.query_adapter"
	postgresTests             = "capability.postgres_persistence.postgres_integration_tests"
	domainValidation          = "capability.runtime_validation.domain_validation"
	formValidation            = "capability.runtime_validation.form_validation"
	validationTests           = "capability.runtime_validation.validation_tests"
	fullPageResponse          = "capability.htmx_form.full_page_response"
	htmxPartialResponse       = "capability.htmx_form.htmx_partial_response"
	formHandlerTests          = "capability.htmx_form.form_handler_tests"
	documentationContent      = "archetype.server_rendered_crud.documentation_content"
	documentationNavigation   = "archetype.server_rendered_crud.documentation_navigation"
)

var migrationNamePattern = regexp.MustCompile(`^(\d+)[-_]`)

type targetSpec struct {
	id              string
	kind            EditKind
	surface         string
	target          string
	recipe          string
	operations      []string
	create          bool
	requiresCommand string
	postconditions  []Condition
	dependsOn       []string
}

func canonicalTargetSpecs(value plan.Plan, inventory project.Inventory) ([]targetSpec, error) {
	if value.Archetype != "server_rendered_crud" {
		return nil, executionError("execution_archetype_unsupported", "archetype", "archetype %q has no execution recipes", value.Archetype)
	}

	if value.Intent == intent.OperationDocumentFeature {
		return documentationTargetSpecs(value)
	}

	layout, err := resolveLayout(value, inventory)
	if err != nil {
		return nil, err
	}

	var result []targetSpec

	switch value.Intent {
	case intent.OperationCreateFeature:
		result = createFeatureTargets(value, layout)
	case intent.OperationAddField:
		result = addFieldTargets(value, layout)
	case intent.OperationAddValidation:
		result = addValidationTargets(value, layout)
	default:
		return nil, executionError("execution_intent_invalid", "intent", "intent %q has no execution recipes", value.Intent)
	}

	if value.Documentation != intent.DocumentationNotRequested {
		documentation, documentationErr := documentationTargetSpecs(value)
		if documentationErr != nil {
			return nil, documentationErr
		}

		result = append(result, documentation...)
	}

	return result, nil
}

func documentationTargetSpecs(value plan.Plan) ([]targetSpec, error) {
	if value.DocumentationPlan == nil {
		return nil, executionError("execution_documentation_plan_missing", "documentation_plan", "active documentation requires planned effects")
	}

	result := make([]targetSpec, 0, len(value.DocumentationPlan.Targets)+len(value.DocumentationPlan.Indexes))
	targetIDs := make(map[string]string, len(value.DocumentationPlan.Targets))
	allTargetIDs := make([]string, 0, len(value.DocumentationPlan.Targets))

	for _, target := range value.DocumentationPlan.Targets {
		id := "documentation.target." + string(target.Quadrant) + "." + target.Subject
		spec := documentationTargetSpec(
			id,
			target.Path,
			"server_rendered_crud.documentation."+string(target.Quadrant),
			target.Snapshot.Exists,
			[]string{documentationContent},
		)
		spec.postconditions = []Condition{
			{Kind: ConditionContentContains, Value: "<!-- hatmax:generated:start -->"},
			{Kind: ConditionContentContains, Value: "<!-- hatmax:generated:end -->"},
			{Kind: ConditionContentContains, Value: target.Title},
		}
		result = append(result, spec)
		targetIDs[target.Path] = id
		allTargetIDs = append(allTargetIDs, id)
	}

	rootIndexID := "documentation.index.root"

	for _, index := range value.DocumentationPlan.Indexes {
		id := "documentation.index." + index.Kind
		if index.Quadrant != "" {
			id += "." + string(index.Quadrant)
		}

		spec := documentationTargetSpec(
			id,
			index.Path,
			"server_rendered_crud.documentation.index",
			index.Snapshot.Exists,
			[]string{documentationNavigation},
		)

		spec.postconditions = []Condition{
			{Kind: ConditionContentContains, Value: "<!-- hatmax:generated:start -->"},
			{Kind: ConditionContentContains, Value: "<!-- hatmax:generated:end -->"},
			{Kind: ConditionContentContains, Value: index.Title},
		}
		for _, link := range index.RequiredLinks {
			spec.postconditions = append(spec.postconditions, Condition{Kind: ConditionContentContains, Value: link.Relative})
		}

		if index.Kind == "root" {
			spec.dependsOn = append(spec.dependsOn, allTargetIDs...)
		} else {
			spec.dependsOn = append(spec.dependsOn, rootIndexID)

			for _, link := range index.RequiredLinks {
				if targetID := targetIDs[link.Target]; targetID != "" {
					spec.dependsOn = append(spec.dependsOn, targetID)
				}
			}
		}

		result = append(result, spec)
	}

	return result, nil
}

func documentationTargetSpec(id, target, recipe string, exists bool, operations []string) targetSpec {
	if exists {
		return updateTarget(id, EditUpdateMarkdown, "documentation", target, recipe, operations)
	}

	return createTarget(id, "documentation", target, recipe, operations, "")
}

type resolvedLayout struct {
	featureRoot   string
	templateRoot  string
	migrationRoot string
	queryRoot     string
	entrypoint    string
	migration     string
}

func resolveLayout(value plan.Plan, inventory project.Inventory) (resolvedLayout, error) {
	featureRoot, err := resolveFeatureRoot(value, inventory.Layout.Features)
	if err != nil {
		return resolvedLayout{}, err
	}

	entrypoint := ""

	for _, candidate := range inventory.CompositionRoots() {
		if entrypoint != "" {
			return resolvedLayout{}, executionError("execution_layout_ambiguous", "wiring", "project has multiple Hatmax composition roots")
		}

		entrypoint = candidate
	}

	if entrypoint == "" {
		return resolvedLayout{}, executionError("execution_layout_missing", "wiring", "project has no Hatmax composition root")
	}

	migrationRoot, err := singleLayoutRoot("migration", inventory.Layout.Migrations, "assets/migration/postgres")
	if err != nil {
		return resolvedLayout{}, err
	}

	queryRoot, err := singleLayoutRoot("store", inventory.Layout.Queries, "db/queries")
	if err != nil {
		return resolvedLayout{}, err
	}

	templateRoot, err := singleLayoutRoot("templates", inventory.Layout.Templates, "assets/templates")
	if err != nil {
		return resolvedLayout{}, err
	}

	return resolvedLayout{
		featureRoot:   featureRoot,
		templateRoot:  templateRoot,
		migrationRoot: migrationRoot,
		queryRoot:     queryRoot,
		entrypoint:    entrypoint,
		migration:     nextMigrationPath(inventory.Files(), migrationRoot, value.Feature),
	}, nil
}

func resolveFeatureRoot(value plan.Plan, features []string) (string, error) {
	if value.Intent != intent.OperationCreateFeature {
		for _, feature := range features {
			if path.Base(feature) == value.Feature {
				return feature, nil
			}
		}

		return "", executionError("execution_layout_missing", "feature", "canonical feature %q does not exist", value.Feature)
	}

	parents := make(map[string]struct{})
	for _, feature := range features {
		parents[path.Dir(feature)] = struct{}{}
	}

	if len(parents) > 1 {
		return "", executionError("execution_layout_ambiguous", "feature", "project has multiple feature roots")
	}

	root := "internal/feat"
	for parent := range parents {
		root = parent
	}

	return path.Join(root, value.Feature), nil
}

func singleLayoutRoot(surface string, values []string, fallback string) (string, error) {
	if len(values) == 0 {
		return fallback, nil
	}

	if len(values) > 1 {
		return "", executionError("execution_layout_ambiguous", surface, "project has multiple candidate roots")
	}

	return values[0], nil
}

func nextMigrationPath(files []project.File, root, feature string) string {
	maximum := 0
	width := 3

	for _, file := range files {
		if path.Dir(file.Path) != root {
			continue
		}

		matches := migrationNamePattern.FindStringSubmatch(path.Base(file.Path))
		if len(matches) != 2 {
			continue
		}

		sequence, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		if sequence > maximum {
			maximum = sequence
		}

		if len(matches[1]) > width {
			width = len(matches[1])
		}
	}

	name := fmt.Sprintf("%0*d-%s.sql", width, maximum+1, strings.ReplaceAll(feature, "_", "-"))

	return path.Join(root, name)
}

func createFeatureTargets(value plan.Plan, layout resolvedLayout) []targetSpec {
	feature := layout.featureRoot
	templates := path.Join(layout.templateRoot, value.Feature)

	return []targetSpec{
		createTarget("create.migration", "migration", layout.migration, "server_rendered_crud.migration", []string{postgresMigration}, "sqlc"),
		createTarget("create.model", "model", path.Join(feature, "model.go"), "server_rendered_crud.model", []string{featurePackageOperation, domainValidation}, ""),
		createTarget("create.store_contract", "store", path.Join(feature, "store.go"), "server_rendered_crud.store_contract", []string{featurePackageOperation}, ""),
		createTarget("create.postgres_store", "store", path.Join(feature, "postgres_store.go"), "server_rendered_crud.postgres_store", []string{postgresQueryAdapter}, "sqlc"),
		createTarget("create.queries", "store", path.Join(layout.queryRoot, value.Feature+".sql"), "server_rendered_crud.queries", []string{postgresQueryAdapter}, "sqlc"),
		createTarget("create.service", "service", path.Join(feature, "service.go"), "server_rendered_crud.service", []string{serviceBoundaryOperation}, ""),
		createTarget("create.handler", "handler", path.Join(feature, "handler.go"), "server_rendered_crud.handler", []string{featureHandlerOperation, formValidation, fullPageResponse, htmxPartialResponse}, ""),
		createTarget("create.page_template", "templates", path.Join(templates, "page.html"), "server_rendered_crud.page_template", []string{featureTemplatesOperation, fullPageResponse}, ""),
		createTarget("create.form_template", "templates", path.Join(templates, "form.html"), "server_rendered_crud.form_template", []string{featureTemplatesOperation, fullPageResponse, htmxPartialResponse}, ""),
		createTarget("create.row_template", "templates", path.Join(templates, "row.html"), "server_rendered_crud.row_template", []string{featureTemplatesOperation, htmxPartialResponse}, ""),
		updateTarget("create.wiring", EditUpdateGo, "wiring", layout.entrypoint, "server_rendered_crud.wiring", []string{explicitWiringOperation}),
		createTarget("create.model_tests", "tests", path.Join(feature, "model_test.go"), "server_rendered_crud.model_tests", []string{featureTestsOperation, validationTests}, ""),
		createTarget("create.service_tests", "tests", path.Join(feature, "service_test.go"), "server_rendered_crud.service_tests", []string{featureTestsOperation}, ""),
		createTarget("create.handler_tests", "tests", path.Join(feature, "handler_test.go"), "server_rendered_crud.handler_tests", []string{featureTestsOperation, validationTests, formHandlerTests}, ""),
		createTarget("create.postgres_store_tests", "tests", path.Join(feature, "postgres_store_test.go"), "server_rendered_crud.postgres_store_tests", []string{postgresTests}, "sqlc"),
	}
}

func addFieldTargets(value plan.Plan, layout resolvedLayout) []targetSpec {
	feature := layout.featureRoot
	templates := path.Join(layout.templateRoot, value.Feature)

	return []targetSpec{
		createTarget("add_field.migration", "migration", layout.migration, "server_rendered_crud.add_field_migration", []string{postgresMigration}, "sqlc"),
		updateTarget("add_field.model", EditUpdateGo, "model", path.Join(feature, "model.go"), "server_rendered_crud.add_field_model", []string{featurePackageOperation}),
		updateTarget("add_field.store_contract", EditUpdateGo, "store", path.Join(feature, "store.go"), "server_rendered_crud.add_field_store_contract", []string{featurePackageOperation}),
		updateTarget("add_field.postgres_store", EditUpdateGo, "store", path.Join(feature, "postgres_store.go"), "server_rendered_crud.add_field_postgres_store", []string{postgresQueryAdapter}),
		updateTarget("add_field.queries", EditUpdateSQL, "store", path.Join(layout.queryRoot, value.Feature+".sql"), "server_rendered_crud.add_field_queries", []string{postgresQueryAdapter}),
		updateTarget("add_field.service", EditUpdateGo, "service", path.Join(feature, "service.go"), "server_rendered_crud.add_field_service", []string{serviceBoundaryOperation}),
		updateTarget("add_field.handler", EditUpdateGo, "handler", path.Join(feature, "handler.go"), "server_rendered_crud.add_field_handler", []string{featureHandlerOperation}),
		updateTarget("add_field.page_template", EditUpdateTemplate, "templates", path.Join(templates, "page.html"), "server_rendered_crud.add_field_page_template", []string{featureTemplatesOperation}),
		updateTarget("add_field.form_template", EditUpdateTemplate, "templates", path.Join(templates, "form.html"), "server_rendered_crud.add_field_form_template", []string{featureTemplatesOperation}),
		updateTarget("add_field.row_template", EditUpdateTemplate, "templates", path.Join(templates, "row.html"), "server_rendered_crud.add_field_row_template", []string{featureTemplatesOperation}),
		updateTarget("add_field.model_tests", EditUpdateTest, "tests", path.Join(feature, "model_test.go"), "server_rendered_crud.add_field_model_tests", []string{featureTestsOperation}),
		updateTarget("add_field.service_tests", EditUpdateTest, "tests", path.Join(feature, "service_test.go"), "server_rendered_crud.add_field_service_tests", []string{featureTestsOperation}),
		updateTarget("add_field.handler_tests", EditUpdateTest, "tests", path.Join(feature, "handler_test.go"), "server_rendered_crud.add_field_handler_tests", []string{featureTestsOperation}),
		updateTarget("add_field.postgres_store_tests", EditUpdateTest, "tests", path.Join(feature, "postgres_store_test.go"), "server_rendered_crud.add_field_postgres_store_tests", []string{postgresTests}),
	}
}

func addValidationTargets(value plan.Plan, layout resolvedLayout) []targetSpec {
	feature := layout.featureRoot
	templates := path.Join(layout.templateRoot, value.Feature)
	result := make([]targetSpec, 0, 6)

	if value.Domain.Validation != nil && value.Domain.Validation.Scope == intent.ValidationDurable {
		result = append(result,
			createTarget("add_validation.migration", "migration", layout.migration, "server_rendered_crud.add_validation_migration", []string{postgresMigration}, "sqlc"),
			updateTarget("add_validation.model", EditUpdateGo, "model", path.Join(feature, "model.go"), "server_rendered_crud.add_validation_model", []string{featurePackageOperation, domainValidation}),
		)
	}

	result = append(result,
		updateTarget("add_validation.handler", EditUpdateGo, "handler", path.Join(feature, "handler.go"), "server_rendered_crud.add_validation_handler", []string{featureHandlerOperation, formValidation}),
		updateTarget("add_validation.form_template", EditUpdateTemplate, "templates", path.Join(templates, "form.html"), "server_rendered_crud.add_validation_form_template", []string{featureTemplatesOperation}),
		updateTarget("add_validation.model_tests", EditUpdateTest, "tests", path.Join(feature, "model_test.go"), "server_rendered_crud.add_validation_model_tests", []string{featureTestsOperation, validationTests}),
		updateTarget("add_validation.handler_tests", EditUpdateTest, "tests", path.Join(feature, "handler_test.go"), "server_rendered_crud.add_validation_handler_tests", []string{featureTestsOperation, validationTests}),
	)

	return result
}

func createTarget(id, surface, target, recipe string, operations []string, command string) targetSpec {
	return targetSpec{
		id:              id,
		kind:            EditCreateFile,
		surface:         surface,
		target:          target,
		recipe:          recipe,
		operations:      operations,
		create:          true,
		requiresCommand: command,
	}
}

func updateTarget(id string, kind EditKind, surface, target, recipe string, operations []string) targetSpec {
	return targetSpec{
		id:         id,
		kind:       kind,
		surface:    surface,
		target:     target,
		recipe:     recipe,
		operations: operations,
	}
}
