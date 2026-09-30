// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"errors"
	"fmt"
	"go/format"
	"path"
	"sort"
	"strings"
	"unicode"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

type renderContext struct {
	plan       plan.Plan
	manifest   Manifest
	inventory  project.Inventory
	modulePath string
	entity     string
	feature    string
	table      string
	route      string
	label      string
	plural     string
	fields     []renderField
	validation *intent.ValidationRule
}

type renderField struct {
	Name      string
	GoName    string
	GoType    string
	SQLType   string
	Label     string
	InputType string
	ParseKind string
	HTMLType  string
	Required  bool
}

type recipeRenderer func(renderContext, Edit) ([]byte, error)

func newRenderContext(value plan.Plan, manifest Manifest, inventory project.Inventory) (renderContext, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return renderContext{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return renderContext{}, err
	}

	if manifest.PlanDigest != value.Digest || manifest.ProjectFingerprint != value.ProjectFingerprint {
		return renderContext{}, executionError("execution_identity_mismatch", "manifest", "manifest and plan identities do not match")
	}

	if value.Intent != intent.OperationCreateFeature || value.Archetype != "server_rendered_crud" {
		return renderContext{}, executionError("execution_recipe_unsupported", "intent", "CRUD creation recipes require create_feature and server_rendered_crud")
	}

	if len(value.Domain.Rules) > 0 {
		return renderContext{}, executionError("execution_slot_unresolved", "domain.rules", "business-rule implementation requires resolved Book implementation slots")
	}

	fields := make([]renderField, 0, len(value.Domain.Fields))
	for _, field := range value.Domain.Fields {
		rendered, err := newRenderField(field)
		if err != nil {
			return renderContext{}, err
		}

		fields = append(fields, rendered)
	}

	label := strings.TrimSpace(value.Domain.Label)
	if label == "" {
		label = value.Domain.Entity + "s"
	}

	table := strings.TrimPrefix(value.Domain.Route, "/")
	table = strings.ReplaceAll(table, "-", "_")
	table = strings.ReplaceAll(table, "/", "_")

	return renderContext{
		plan:       value,
		manifest:   cloneManifest(manifest),
		inventory:  inventory,
		modulePath: inventory.Module.Path,
		entity:     value.Domain.Entity,
		feature:    value.Feature,
		table:      table,
		route:      value.Domain.Route,
		label:      label,
		plural:     exportedName(table),
		fields:     fields,
	}, nil
}

func newRenderField(field intent.Field) (renderField, error) {
	result := renderField{
		Name:      field.Name,
		GoName:    exportedName(field.Name),
		Label:     field.Label,
		Required:  field.Required,
		HTMLType:  "text",
		ParseKind: field.Type,
	}
	if result.Label == "" {
		result.Label = titleWords(field.Name)
	}

	switch field.Type {
	case "boolean":
		result.GoType, result.InputType, result.SQLType, result.HTMLType = "bool", "bool", "BOOLEAN", "checkbox"
	case "date":
		result.GoType, result.InputType, result.SQLType, result.HTMLType = "time.Time", "time.Time", "DATE", "date"
	case "decimal":
		result.GoType, result.InputType, result.SQLType, result.HTMLType = "string", "string", "TEXT", "text"
	case "integer":
		result.GoType, result.InputType, result.SQLType, result.HTMLType = "int64", "int64", "BIGINT", "number"
	case "string":
		result.GoType, result.InputType, result.SQLType = "string", "string", "VARCHAR(255)"
	case "text":
		result.GoType, result.InputType, result.SQLType = "string", "string", "TEXT"
	case "timestamp":
		result.GoType, result.InputType, result.SQLType, result.HTMLType = "time.Time", "time.Time", "TIMESTAMPTZ", "datetime-local"
	case "uuid":
		result.GoType, result.InputType, result.SQLType = "string", "string", "TEXT"
	default:
		return renderField{}, executionError("execution_field_type_unsupported", field.Name, "field type %q has no canonical renderer", field.Type)
	}

	return result, nil
}

func renderSelectedRecipes(
	value plan.Plan,
	manifest Manifest,
	inventory project.Inventory,
	renderers map[string]recipeRenderer,
) ([]Mutation, error) {
	context, err := newRenderContext(value, manifest, inventory)
	if err != nil {
		return nil, err
	}

	result := make([]Mutation, 0, len(manifest.Edits))
	for _, edit := range manifest.Edits {
		renderer, selected := renderers[edit.Recipe]
		if !selected {
			continue
		}

		content, renderErr := renderer(context, edit)
		if renderErr != nil {
			var typedError Error
			if errors.As(renderErr, &typedError) {
				return nil, renderErr
			}

			return nil, executionError("execution_render_failed", edit.ID, "%v", renderErr)
		}

		kind := MutationCreate
		if edit.Kind != EditCreateFile {
			kind = MutationReplace
		}

		result = append(result, Mutation{EditID: edit.ID, Kind: kind, Content: content})
	}

	return result, nil
}

// RenderCreateFeature renders every edit in a canonical create-feature
// manifest. It does not modify the inspected project or execute commands.
func RenderCreateFeature(value plan.Plan, manifest Manifest, inventory project.Inventory) ([]Mutation, error) {
	renderers := make(map[string]recipeRenderer, len(domainRenderers)+len(transportRenderers)+len(wiringRenderers))

	for recipe, renderer := range domainRenderers {
		renderers[recipe] = renderer
	}

	for recipe, renderer := range transportRenderers {
		renderers[recipe] = renderer
	}

	for recipe, renderer := range wiringRenderers {
		renderers[recipe] = renderer
	}

	for _, edit := range manifest.Edits {
		if edit.Surface == "documentation" {
			continue
		}

		if _, exists := renderers[edit.Recipe]; !exists {
			return nil, executionError("execution_renderer_missing", edit.ID, "recipe %q has no canonical renderer", edit.Recipe)
		}
	}

	return renderSelectedRecipes(value, manifest, inventory, renderers)
}

func formatGo(target string, source string) ([]byte, error) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format %s: %w", target, err)
	}

	return formatted, nil
}

func exportedName(value string) string {
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return character == '_' || character == '-'
	})

	var result strings.Builder

	for _, part := range parts {
		if strings.EqualFold(part, "id") {
			result.WriteString("ID")

			continue
		}

		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}

		result.WriteRune(unicode.ToUpper(runes[0]))
		result.WriteString(string(runes[1:]))
	}

	return result.String()
}

func titleWords(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "_", " ")), " ")
}

func sortedRendererKeys(values map[string]recipeRenderer) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}

	sort.Strings(result)

	return result
}

func featureImport(context renderContext) string {
	return path.Join(context.modulePath, "internal", "feat", context.feature)
}
