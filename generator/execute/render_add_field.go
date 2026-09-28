package execute

import (
	"errors"
	"fmt"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

var addFieldRenderers = map[string]recipeRenderer{
	"server_rendered_crud.add_field_migration":            renderAddFieldMigration,
	"server_rendered_crud.add_field_model":                renderModel,
	"server_rendered_crud.add_field_store_contract":       renderStoreContract,
	"server_rendered_crud.add_field_postgres_store":       renderPostgresStore,
	"server_rendered_crud.add_field_queries":              renderQueries,
	"server_rendered_crud.add_field_service":              renderService,
	"server_rendered_crud.add_field_handler":              renderHandler,
	"server_rendered_crud.add_field_page_template":        renderPageTemplate,
	"server_rendered_crud.add_field_form_template":        renderFormTemplate,
	"server_rendered_crud.add_field_row_template":         renderRowTemplate,
	"server_rendered_crud.add_field_model_tests":          renderModelTests,
	"server_rendered_crud.add_field_service_tests":        renderServiceTests,
	"server_rendered_crud.add_field_handler_tests":        renderHandlerTests,
	"server_rendered_crud.add_field_postgres_store_tests": renderPostgresStoreTests,
}

// RenderAddField renders one admitted field across every canonical CRUD
// representation after inspecting the existing feature structure.
func RenderAddField(value plan.Plan, manifest Manifest, inventory project.Inventory) ([]Mutation, error) {
	if value.Intent != intent.OperationAddField || value.Domain.Field == nil {
		return nil, executionError("execution_recipe_unsupported", "intent", "add-field rendering requires an add_field plan")
	}

	feature, err := inspectCanonicalFeature(value, manifest, inventory)
	if err != nil {
		return nil, err
	}

	for _, existing := range feature.fields {
		if existing.Name == value.Domain.Field.Name || existing.GoName == exportedName(value.Domain.Field.Name) {
			return nil, executionError("execution_semantic_conflict", "domain.field.name", "field %q already exists", value.Domain.Field.Name)
		}
	}

	field, err := newRenderField(*value.Domain.Field)
	if err != nil {
		return nil, err
	}

	context := incrementalRenderContext(value, manifest, inventory, feature)
	context.fields = append(context.fields, field)

	return renderIncrementalManifest(context, manifest, addFieldRenderers)
}

func incrementalRenderContext(
	value plan.Plan,
	manifest Manifest,
	inventory project.Inventory,
	feature canonicalFeature,
) renderContext {
	return renderContext{
		plan:       value,
		manifest:   cloneManifest(manifest),
		inventory:  inventory,
		modulePath: inventory.Module.Path,
		entity:     feature.entity,
		feature:    value.Feature,
		table:      feature.table,
		route:      feature.route,
		label:      feature.label,
		plural:     exportedName(feature.table),
		fields:     append([]renderField{}, feature.fields...),
	}
}

func renderIncrementalManifest(
	context renderContext,
	manifest Manifest,
	renderers map[string]recipeRenderer,
) ([]Mutation, error) {
	result := make([]Mutation, 0, len(manifest.Edits))

	for _, edit := range manifest.Edits {
		renderer, exists := renderers[edit.Recipe]
		if !exists {
			return nil, executionError("execution_renderer_missing", edit.ID, "recipe %q has no canonical renderer", edit.Recipe)
		}

		content, err := renderer(context, edit)
		if err != nil {
			var typedError Error
			if errors.As(err, &typedError) {
				return nil, err
			}

			return nil, executionError("execution_render_failed", edit.ID, "%v", err)
		}

		kind := MutationReplace
		if edit.Kind == EditCreateFile {
			kind = MutationCreate
		}

		result = append(result, Mutation{EditID: edit.ID, Kind: kind, Content: content})
	}

	return result, nil
}

func renderAddFieldMigration(context renderContext, _ Edit) ([]byte, error) {
	field := context.fields[len(context.fields)-1]

	nullability := ""
	if field.Required {
		nullability = " NOT NULL"
	}

	return []byte(fmt.Sprintf(`-- +migrate Up
ALTER TABLE %s ADD COLUMN %s %s%s;

-- +migrate Down
ALTER TABLE %s DROP COLUMN %s;
`, context.table, field.Name, field.SQLType, nullability, context.table, field.Name)), nil
}
