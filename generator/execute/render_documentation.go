// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// RenderDocumentation renders every documentation edit from sealed Hatmax
// evidence. It does not render implementation edits from a combined manifest.
func RenderDocumentation(value plan.Plan, manifest Manifest, inventory project.Inventory) ([]Mutation, error) {
	err := validateDocumentationRenderContracts(value, manifest)
	if err != nil {
		return nil, err
	}

	targets := make(map[string]plan.DocumentationTargetEffect, len(value.DocumentationPlan.Targets))
	for _, target := range value.DocumentationPlan.Targets {
		targets[target.Path] = target
	}

	indexes := make(map[string]plan.DocumentationIndexEffect, len(value.DocumentationPlan.Indexes))
	for _, index := range value.DocumentationPlan.Indexes {
		indexes[index.Path] = index
	}

	result := make([]Mutation, 0, len(targets)+len(indexes))

	for _, edit := range manifest.Edits {
		if edit.Surface != "documentation" {
			continue
		}

		body, renderErr := renderDocumentationEdit(value, edit, targets, indexes)
		if renderErr != nil {
			return nil, renderErr
		}

		existing, readErr := existingDocumentationContent(edit, inventory)
		if readErr != nil {
			return nil, readErr
		}

		content, managedErr := renderManagedMarkdown(existing, body)
		if managedErr != nil {
			return nil, managedErr
		}

		kind := MutationCreate
		if edit.Kind == EditUpdateMarkdown {
			kind = MutationReplace
		}

		result = append(result, Mutation{EditID: edit.ID, Kind: kind, Content: content})
	}

	if len(result) != len(targets)+len(indexes) {
		return nil, executionError("execution_renderer_missing", "documentation", "manifest documentation edits do not match the sealed documentation plan")
	}

	return result, nil
}

func validateDocumentationRenderContracts(value plan.Plan, manifest Manifest) error {
	err := plan.VerifyDigest(value)
	if err != nil {
		return executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = VerifyDigest(manifest)
	if err != nil {
		return err
	}

	if manifest.PlanDigest != value.Digest || manifest.ProjectFingerprint != value.ProjectFingerprint {
		return executionError("execution_identity_mismatch", "manifest", "manifest and plan identities do not match")
	}

	if value.Documentation == intent.DocumentationNotRequested || value.DocumentationPlan == nil || value.DocumentationEvidence == nil {
		return executionError("execution_recipe_unsupported", "documentation", "documentation rendering requires active planned documentation and evidence")
	}

	return nil
}

func renderDocumentationEdit(
	value plan.Plan,
	edit Edit,
	targets map[string]plan.DocumentationTargetEffect,
	indexes map[string]plan.DocumentationIndexEffect,
) ([]byte, error) {
	if target, exists := targets[edit.Target]; exists {
		expectedRecipe := "server_rendered_crud.documentation." + string(target.Quadrant)
		if edit.Recipe != expectedRecipe {
			return nil, executionError("execution_renderer_missing", edit.ID, "recipe %q does not match %q", edit.Recipe, expectedRecipe)
		}

		return renderDocumentationTarget(target, *value.DocumentationEvidence, value.DocumentationPlan.ValidationCommands)
	}

	if index, exists := indexes[edit.Target]; exists {
		if edit.Recipe != "server_rendered_crud.documentation.index" {
			return nil, executionError("execution_renderer_missing", edit.ID, "recipe %q is not the canonical documentation index renderer", edit.Recipe)
		}

		return renderDocumentationIndex(index), nil
	}

	return nil, executionError("execution_renderer_missing", edit.ID, "documentation target %q is not sealed in the plan", edit.Target)
}

func existingDocumentationContent(edit Edit, inventory project.Inventory) ([]byte, error) {
	if edit.Kind == EditCreateFile {
		return nil, nil
	}

	if edit.Kind != EditUpdateMarkdown {
		return nil, executionError("execution_mutation_invalid", edit.Target, "documentation edit must create or update Markdown")
	}

	content, err := os.ReadFile(filepath.Join(inventory.Root, filepath.FromSlash(edit.Target)))
	if err != nil {
		return nil, executionError("execution_render_failed", edit.Target, "read managed Markdown: %v", err)
	}

	return content, nil
}

func renderDocumentationTarget(
	target plan.DocumentationTargetEffect,
	evidence project.FeatureEvidence,
	commands []plan.DocumentationCommand,
) ([]byte, error) {
	switch target.Quadrant {
	case intent.DocumentationTutorial:
		return renderDocumentationTutorial(target, evidence, commands), nil
	case intent.DocumentationHowTo:
		return renderDocumentationHowTo(target, evidence, commands), nil
	case intent.DocumentationReference:
		return renderDocumentationReference(target, evidence), nil
	case intent.DocumentationExplanation:
		return renderDocumentationExplanation(target, evidence), nil
	default:
		return nil, executionError("execution_render_failed", target.Path, "unknown documentation quadrant %q", target.Quadrant)
	}
}

func renderDocumentationTutorial(
	target plan.DocumentationTargetEffect,
	evidence project.FeatureEvidence,
	commands []plan.DocumentationCommand,
) []byte {
	var result strings.Builder

	fmt.Fprintf(&result, "# %s\n\n", target.Title)
	result.WriteString("## Outcome\n\n")
	fmt.Fprintf(&result, "Use the server-rendered %s workflow at `%s` and recognize its canonical fields.\n\n", markdownText(evidence.Entity), markdownCode(evidence.Route))
	result.WriteString("## Before You Begin\n\n")
	fmt.Fprintf(&result, "The application must provide the `%s` route and its configured PostgreSQL store.\n\n", markdownCode(evidence.Route))
	result.WriteString("## Guided Steps\n\n")
	fmt.Fprintf(&result, "1. Open `%s` in the running application.\n", markdownCode(evidence.Route))
	fmt.Fprintf(&result, "2. Complete the form using the documented fields: %s.\n", documentationFieldNames(evidence.Fields))
	result.WriteString("3. Submit the form. Hatmax returns a server-rendered response and uses an HTMX partial response when the request is an HTMX request.\n")
	result.WriteString("4. Confirm that the created record appears in the feature list.\n\n")
	writeDocumentationCommand(&result, commands)
	result.WriteString("## Continue\n\n")
	fmt.Fprintf(&result, "[Back to %s](../README.md)\n", documentationQuadrantLabel(target.Quadrant))

	return []byte(result.String())
}

func renderDocumentationHowTo(
	target plan.DocumentationTargetEffect,
	evidence project.FeatureEvidence,
	commands []plan.DocumentationCommand,
) []byte {
	var result strings.Builder

	fmt.Fprintf(&result, "# %s\n\n", target.Title)
	result.WriteString("## Goal\n\n")
	fmt.Fprintf(&result, "Complete the `%s` workflow for %s using the canonical server-rendered form.\n\n", markdownCode(evidence.Route), markdownText(evidence.Entity))
	result.WriteString("## Procedure\n\n")
	fmt.Fprintf(&result, "1. Navigate to `%s`.\n", markdownCode(evidence.Route))
	fmt.Fprintf(&result, "2. Supply the applicable fields: %s.\n", documentationFieldNames(evidence.Fields))
	result.WriteString("3. Submit the form and correct any validation messages returned with the form.\n")
	result.WriteString("4. Verify the updated row in the server-rendered list.\n\n")
	writeDocumentationCommand(&result, commands)
	result.WriteString("## Related Documentation\n\n")
	fmt.Fprintf(&result, "[Back to %s](../README.md)\n", documentationQuadrantLabel(target.Quadrant))

	return []byte(result.String())
}

func renderDocumentationReference(target plan.DocumentationTargetEffect, evidence project.FeatureEvidence) []byte {
	var result strings.Builder

	fmt.Fprintf(&result, "# %s\n\n", target.Title)
	result.WriteString("## Contract\n\n")
	result.WriteString("| Property | Value |\n| --- | --- |\n")
	fmt.Fprintf(&result, "| Entity | `%s` |\n", markdownTable(evidence.Entity))
	fmt.Fprintf(&result, "| Route | `%s` |\n", markdownTable(evidence.Route))
	fmt.Fprintf(&result, "| PostgreSQL table | `%s` |\n\n", markdownTable(evidence.Table))
	result.WriteString("## Fields\n\n")
	result.WriteString("| Field | Type | Label | Required |\n| --- | --- | --- | --- |\n")

	for _, field := range evidence.Fields {
		required := "no"
		if field.Required {
			required = "yes"
		}

		label := field.Label
		if strings.TrimSpace(label) == "" {
			label = documentationTitleWords(field.Name)
		}

		fmt.Fprintf(&result, "| `%s` | `%s` | %s | %s |\n", markdownTable(field.Name), markdownTable(field.Type), markdownTable(label), required)
	}

	result.WriteString("\n## Validation\n\n")

	if len(evidence.Validations) == 0 {
		result.WriteString("No explicit field validation is established by the canonical feature evidence.\n")
	} else {
		for _, validation := range evidence.Validations {
			value := ""
			if validation.Value != "" {
				value = " `" + markdownCode(validation.Value) + "`"
			}

			fmt.Fprintf(&result, "- `%s`: `%s`%s at the `%s` scope.\n", markdownCode(validation.Field), markdownCode(validation.Kind), value, markdownCode(validation.Scope))
		}
	}

	result.WriteString("\n## Integration\n\n")
	fmt.Fprintf(&result, "- PostgreSQL persistence: %s\n", documentationBoolean(evidence.Postgres))
	fmt.Fprintf(&result, "- HTMX form behavior: %s\n", documentationBoolean(evidence.HTMX))
	fmt.Fprintf(&result, "- Runtime validation: %s\n", documentationBoolean(evidence.RuntimeValidation))
	fmt.Fprintf(&result, "- Composition-root wiring: %s\n", documentationBoolean(evidence.Wired))
	fmt.Fprintf(&result, "- Canonical boundary tests: %s\n\n", documentationBoolean(evidence.Tested))
	fmt.Fprintf(&result, "[Back to %s](../README.md)\n", documentationQuadrantLabel(target.Quadrant))

	return []byte(result.String())
}

func renderDocumentationExplanation(target plan.DocumentationTargetEffect, evidence project.FeatureEvidence) []byte {
	var result strings.Builder

	fmt.Fprintf(&result, "# %s\n\n", target.Title)
	result.WriteString("## Context\n\n")
	fmt.Fprintf(&result, "The %s feature is assembled as one server-rendered CRUD slice around the `%s` route.\n\n", markdownText(evidence.Entity), markdownCode(evidence.Route))
	result.WriteString("## Ownership\n\n")
	result.WriteString("- The feature package owns its model, store contract, service boundary, HTTP handler, templates, and boundary tests.\n")
	result.WriteString("- PostgreSQL persistence remains behind the feature store contract.\n")
	result.WriteString("- The composition root constructs the store, service, and handler explicitly.\n")
	result.WriteString("- Hatmax HTMX helpers own partial-response behavior at the template and handler boundaries.\n\n")
	result.WriteString("## Tradeoffs\n\n")
	result.WriteString("This structure keeps feature behavior cohesive and wiring visible. It favors one canonical implementation path over interchangeable libraries or implicit runtime assembly.\n\n")
	result.WriteString("## Relationship to the Application\n\n")
	fmt.Fprintf(&result, "The `%s` table persists the feature state, while `%s` remains the user-facing route.\n\n", markdownCode(evidence.Table), markdownCode(evidence.Route))
	fmt.Fprintf(&result, "[Back to %s](../README.md)\n", documentationQuadrantLabel(target.Quadrant))

	return []byte(result.String())
}

func renderDocumentationIndex(index plan.DocumentationIndexEffect) []byte {
	var result strings.Builder

	fmt.Fprintf(&result, "# %s\n\n", index.Title)
	result.WriteString("## Generated Navigation\n\n")

	for _, link := range index.RequiredLinks {
		fmt.Fprintf(&result, "- [%s](%s)\n", markdownText(link.Title), link.Relative)
	}

	return []byte(result.String())
}

func writeDocumentationCommand(result *strings.Builder, commands []plan.DocumentationCommand) {
	if len(commands) == 0 {
		return
	}

	ordered := append([]plan.DocumentationCommand{}, commands...)
	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].Name < ordered[right].Name
	})

	result.WriteString("## Verify\n\n")
	result.WriteString("Run the repository-owned documentation check:\n\n```sh\n")
	result.WriteString(strings.Join(ordered[0].Args, " "))
	result.WriteString("\n```\n\n")
}

func documentationFieldNames(fields []project.FeatureFieldEvidence) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		values = append(values, "`"+markdownCode(field.Name)+"`")
	}

	return strings.Join(values, ", ")
}

func documentationBoolean(value bool) string {
	if value {
		return "enabled"
	}

	return "not established"
}

func documentationQuadrantLabel(quadrant intent.DocumentationQuadrant) string {
	switch quadrant {
	case intent.DocumentationTutorial:
		return "Tutorials"
	case intent.DocumentationHowTo:
		return "How-to Guides"
	case intent.DocumentationReference:
		return "Reference"
	case intent.DocumentationExplanation:
		return "Explanation"
	default:
		return "Documentation"
	}
}

func documentationTitleWords(value string) string {
	words := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for index, word := range words {
		if word == "" {
			continue
		}

		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}

	return strings.Join(words, " ")
}

func markdownText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")

	return value
}

func markdownTable(value string) string {
	return strings.ReplaceAll(markdownText(value), "|", "\\|")
}

func markdownCode(value string) string {
	return strings.ReplaceAll(strings.Join(strings.Fields(value), " "), "`", "\\`")
}
