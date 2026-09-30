// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

var addValidationRenderers = map[string]recipeRenderer{
	"server_rendered_crud.add_validation_migration":     renderAddValidationMigration,
	"server_rendered_crud.add_validation_model":         renderModel,
	"server_rendered_crud.add_validation_handler":       renderHandler,
	"server_rendered_crud.add_validation_form_template": renderFormTemplate,
	"server_rendered_crud.add_validation_model_tests":   renderModelTests,
	"server_rendered_crud.add_validation_handler_tests": renderHandlerTests,
}

// RenderAddValidation renders one admitted validation at its canonical
// durable and interaction boundaries.
func RenderAddValidation(value plan.Plan, manifest Manifest, inventory project.Inventory) ([]Mutation, error) {
	if value.Intent != intent.OperationAddValidation || value.Domain.Validation == nil {
		return nil, executionError("execution_recipe_unsupported", "intent", "validation rendering requires an add_validation plan")
	}

	feature, err := inspectCanonicalFeature(value, manifest, inventory)
	if err != nil {
		return nil, err
	}

	rule := *value.Domain.Validation
	fieldIndex := -1

	for index, field := range feature.fields {
		if field.Name == rule.Field {
			fieldIndex = index

			break
		}
	}

	if fieldIndex < 0 {
		return nil, executionError("execution_semantic_conflict", "domain.validation.field", "field %q does not exist", rule.Field)
	}

	err = validateValidationRule(feature.fields[fieldIndex], rule)
	if err != nil {
		return nil, err
	}

	context := incrementalRenderContext(value, manifest, inventory, feature)

	context.validation = &rule
	if rule.Kind == "required" && rule.Scope == intent.ValidationDurable {
		context.fields[fieldIndex].Required = true
	}

	return renderIncrementalManifest(context, manifest, addValidationRenderers)
}

func validateValidationRule(field renderField, rule intent.ValidationRule) error {
	switch rule.Kind {
	case "required":
		if field.ParseKind == "boolean" {
			return executionError("execution_validation_unsupported", "domain.validation", "required boolean validation has no canonical meaning")
		}
	case "minimum", "maximum":
		if field.ParseKind != "integer" {
			return executionError("execution_validation_unsupported", "domain.validation", "%s validation requires an integer field", rule.Kind)
		}

		_, err := strconv.ParseInt(rule.Value, 10, 64)
		if err != nil {
			return executionError("execution_validation_invalid", "domain.validation.value", "%s validation requires an integer value", rule.Kind)
		}
	case "min_length", "max_length":
		if !isTextualField(field) {
			return executionError("execution_validation_unsupported", "domain.validation", "%s validation requires a textual field", rule.Kind)
		}

		length, err := strconv.Atoi(rule.Value)
		if err != nil || length < 1 {
			return executionError("execution_validation_invalid", "domain.validation.value", "%s validation requires a positive integer", rule.Kind)
		}
	case "pattern":
		if !isTextualField(field) {
			return executionError("execution_validation_unsupported", "domain.validation", "pattern validation requires a textual field")
		}

		_, err := regexp.Compile(rule.Value)
		if err != nil {
			return executionError("execution_validation_invalid", "domain.validation.value", "pattern is not a valid regular expression")
		}
	case "unique":
		if rule.Scope != intent.ValidationDurable {
			return executionError("execution_validation_unsupported", "domain.validation", "unique validation must be durable")
		}
	default:
		return executionError("execution_validation_unsupported", "domain.validation.kind", "validation kind %q has no canonical renderer", rule.Kind)
	}

	return nil
}

func isTextualField(field renderField) bool {
	return field.ParseKind == "string" || field.ParseKind == "text" || field.ParseKind == "decimal" || field.ParseKind == "uuid"
}

func renderAddValidationMigration(context renderContext, _ Edit) ([]byte, error) {
	rule := *context.validation
	constraint := context.table + "_" + rule.Field + "_" + rule.Kind
	up := ""
	down := ""

	switch rule.Kind {
	case "required":
		up = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", context.table, rule.Field)
		down = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", context.table, rule.Field)
	case "minimum":
		up = addCheckConstraint(context.table, constraint, rule.Field+" >= "+rule.Value)
		down = dropConstraint(context.table, constraint)
	case "maximum":
		up = addCheckConstraint(context.table, constraint, rule.Field+" <= "+rule.Value)
		down = dropConstraint(context.table, constraint)
	case "min_length":
		up = addCheckConstraint(context.table, constraint, "char_length("+rule.Field+") >= "+rule.Value)
		down = dropConstraint(context.table, constraint)
	case "max_length":
		up = addCheckConstraint(context.table, constraint, "char_length("+rule.Field+") <= "+rule.Value)
		down = dropConstraint(context.table, constraint)
	case "pattern":
		pattern := strings.ReplaceAll(rule.Value, "'", "''")
		up = addCheckConstraint(context.table, constraint, rule.Field+" ~ '"+pattern+"'")
		down = dropConstraint(context.table, constraint)
	case "unique":
		up = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);", context.table, constraint, rule.Field)
		down = dropConstraint(context.table, constraint)
	}

	return []byte("-- +migrate Up\n" + up + "\n\n-- +migrate Down\n" + down + "\n"), nil
}

func addCheckConstraint(table, name, expression string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s);", table, name, expression)
}

func dropConstraint(table, name string) string {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", table, name)
}
