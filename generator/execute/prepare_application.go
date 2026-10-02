// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package execute

import (
	"path"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// PrepareApplication converts one sealed application plan and matching target
// inventory into the exact manifest consumed by scaffold rendering and
// publication.
func PrepareApplication(value plan.Plan, target project.TargetInventory, selectedBook *book.Book) (Manifest, error) {
	err := validateApplicationPreparation(value, target, selectedBook)
	if err != nil {
		return Manifest{}, err
	}

	edits, err := prepareApplicationEdits(value)
	if err != nil {
		return Manifest{}, err
	}

	preserved := make([]string, 0, len(target.Preserved))
	for _, entry := range target.Preserved {
		preserved = append(preserved, entry.Path)
	}

	result := Manifest{
		SchemaVersion: CurrentSchemaVersion,
		PlanDigest:    value.Digest, SourceFingerprint: value.SourceFingerprint,
		Intent: value.Intent, TargetPath: target.Target, PreservedPaths: preserved,
		AllowedSurfaces: append([]string{}, value.AllowedEffects.Surfaces...), Edits: edits,
		Commands: []Command{},
	}
	if len(value.Units) > 1 {
		result.Commands = append(result.Commands,
			Command{Kind: project.CommandGeneration, Name: "generation.sqlc", Args: []string{"sqlc", "generate"}, WorkingDirectory: ".", Source: "hatmax.application.initial_features"},
		)
	}

	result.Commands = append(result.Commands,
		Command{Kind: project.CommandGeneration, Name: "generation.modules", Args: []string{"go", "mod", "tidy"}, WorkingDirectory: ".", Source: "hatmax.application.scaffold"},
		Command{Kind: project.CommandValidation, Name: "validation.build", Args: []string{"go", "build", "./..."}, WorkingDirectory: ".", Source: "hatmax.application.scaffold"},
		Command{Kind: project.CommandValidation, Name: "validation.test", Args: []string{"go", "test", "./..."}, WorkingDirectory: ".", Source: "hatmax.application.scaffold"},
	)

	result, err = Seal(result)
	if err != nil {
		return Manifest{}, executionError("execution_manifest_invalid", "manifest", "%v", err)
	}

	return result, nil
}

func validateApplicationPreparation(value plan.Plan, target project.TargetInventory, selectedBook *book.Book) error {
	err := plan.VerifyDigest(value)
	if err != nil {
		return executionError("execution_plan_invalid", "plan", "%v", err)
	}

	if selectedBook == nil || selectedBook.Manifest().BookVersion != value.BookVersion {
		return executionError("execution_book_version_mismatch", "book_version", "plan and Book versions must match")
	}

	if value.SchemaVersion != plan.ApplicationSchemaVersion || value.Intent != intent.OperationCreateApplication || value.Target == nil {
		return executionError("execution_intent_invalid", "intent", "application preparation requires a create_application plan")
	}

	if len(value.Units) == 0 || value.Units[0].Intent != intent.OperationCreateApplication {
		return executionError("execution_units_invalid", "units", "application unit must lead scaffold execution")
	}

	if target.Target != value.Target.Path || target.Parent != value.Target.Parent || target.Admission != value.Target.Admission {
		return executionError("execution_target_mismatch", "target", "target inventory does not match the sealed application plan")
	}

	fingerprint, err := target.Fingerprint(value.BookVersion)
	if err != nil {
		return executionError("execution_fingerprint_failed", "source_fingerprint", "%v", err)
	}

	freshness, err := plan.CheckFingerprint(value, fingerprint)
	if err != nil {
		return executionError("execution_plan_invalid", "plan", "%v", err)
	}

	if freshness.Stale {
		return executionError("execution_plan_stale", "source_fingerprint", "target state changed after planning")
	}

	err = validateApplicationRecipeCoverage(value)
	if err != nil {
		return executionError("execution_recipe_unsupported", "allowed_effects.files", "%v", err)
	}

	return nil
}

func prepareApplicationEdits(value plan.Plan) ([]Edit, error) {
	operations := make(map[string]plan.Operation, len(value.Operations))
	for _, operation := range value.Operations {
		operations[operation.ID] = operation
	}

	edits := make([]Edit, 0, len(value.AllowedEffects.Files))
	for _, effect := range value.AllowedEffects.Files {
		if effect.Effect != plan.FileEffectCreate {
			return nil, executionError("execution_effect_invalid", effect.Path, "application publication requires final create effects")
		}

		recipeID, recipe, exists := applicationRecipeForPath(effect.Path)
		if !exists {
			recipeID, exists = applicationFeatureRecipe(value, effect.Path)
			recipe = applicationRecipe{target: effect.Path, goSource: path.Ext(effect.Path) == ".go"}
		}

		if !exists || recipe.target != effect.Path {
			return nil, executionError("execution_recipe_unsupported", effect.Path, "planned path has no canonical application recipe")
		}

		operations := applicationPlanOperations(value.Operations, effect.Path)
		if len(operations) == 0 {
			return nil, executionError("execution_obligation_missing", effect.Path, "planned path has no owning operation")
		}

		obligations := make([]Obligation, 0, len(operations))
		for _, operation := range operations {
			obligations = append(obligations, Obligation{
				Operation: operation.ID, Owner: operation.Owner, Rules: append([]string{}, operation.Rules...),
			})
		}

		postconditions := []Condition{{Kind: ConditionPathPresent}}
		if recipe.goSource {
			postconditions = append(postconditions, Condition{Kind: ConditionGoParses})
		}

		edits = append(edits, Edit{
			ID:   "application.scaffold." + applicationEditID(effect.Path),
			Kind: EditCreateFile, Surface: operations[0].Surfaces[0], Target: effect.Path, Recipe: recipeID,
			Obligations:   obligations,
			Preconditions: []Condition{{Kind: ConditionPathAbsent}}, Postconditions: postconditions,
			Slots: implementationSlots(value, operations[0].Surfaces[0]),
		})
	}

	applyEditDependencies(edits, operations)

	return edits, nil
}

func applicationPlanOperations(values []plan.Operation, target string) []plan.Operation {
	result := make([]plan.Operation, 0)

	for _, operation := range values {
		for _, effect := range operation.Files {
			if effect.Path == target {
				result = append(result, operation)

				break
			}
		}
	}

	return result
}

func applicationEditID(target string) string {
	value := strings.ToLower(strings.TrimPrefix(target, "."))
	value = strings.NewReplacer("/", ".", "-", "_").Replace(value)
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return character == '.'
	})

	result := strings.Join(parts, ".")
	if result == "" {
		return path.Base(target)
	}

	return result
}

func samePreservedPaths(entries []project.TargetEntry, paths []string) bool {
	left := make([]string, 0, len(entries))
	for _, entry := range entries {
		left = append(left, entry.Path)
	}

	right := append([]string{}, paths...)

	sort.Strings(left)
	sort.Strings(right)

	return sameStrings(left, right)
}
