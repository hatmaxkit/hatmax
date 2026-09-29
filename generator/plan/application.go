package plan

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

const applicationFeatureArchetype = "server_rendered_crud"

type applicationUnitDraft struct {
	unit       Unit
	operation  intent.Operation
	selection  book.Selection
	operations []Operation
}

// ApplicationTargetPaths returns the exact Book-owned paths that must be
// included in target inventory before a create_application intent is sealed.
func ApplicationTargetPaths(value intent.Intent, selectedBook *book.Book) ([]string, error) {
	drafts, err := applicationUnitDrafts(value, selectedBook)
	if err != nil {
		return nil, err
	}

	paths := make(map[string]struct{})

	for _, draft := range drafts {
		for _, operation := range draft.operations {
			for _, effect := range operation.Files {
				paths[effect.Path] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(paths))
	for effectPath := range paths {
		result = append(result, effectPath)
	}

	sort.Strings(result)

	return result, nil
}

func expandApplication(value intent.Intent, context ExpansionContext) (Plan, error) {
	if context.Target == nil {
		return Plan{}, planError("plan_target_missing", "target", "application planning requires inspected target state")
	}

	fingerprint, err := context.Target.Fingerprint(value.BookVersion)
	if err != nil {
		return Plan{}, planError("plan_target_invalid", "target", "%v", err)
	}

	if fingerprint.Value != context.Fingerprint.Value {
		return Plan{}, planError("plan_fingerprint_mismatch", "source_fingerprint", "target state does not match expansion context")
	}

	drafts, err := applicationUnitDrafts(value, context.Book)
	if err != nil {
		return Plan{}, err
	}

	operations := make([]Operation, 0)
	units := make([]Unit, 0, len(drafts))
	selectedRules := make(map[string]struct{})
	selectedCapabilities := make(map[string]struct{})
	dependencies := make([]DependencyEffect, 0)
	seenDependencies := make(map[string]struct{})

	for _, draft := range drafts {
		unit := draft.unit

		unit.Operations = make([]string, 0, len(draft.operations))
		for _, operation := range draft.operations {
			unit.Operations = append(unit.Operations, operation.ID)
		}

		operations = append(operations, draft.operations...)
		units = append(units, unit)

		for _, rule := range draft.selection.Rules {
			selectedRules[rule.ID] = struct{}{}
		}

		for _, capability := range draft.selection.Capabilities {
			selectedCapabilities[capability.ID] = struct{}{}
		}

		for _, dependency := range expandDependencies(draft.selection.Capabilities, draft.operation, true) {
			identity := dependency.Capability + "\x00" + dependency.Kind + "\x00" + dependency.Module
			if _, exists := seenDependencies[identity]; exists {
				continue
			}

			seenDependencies[identity] = struct{}{}

			dependencies = append(dependencies, dependency)
		}
	}

	operations, units, files, err := resolveApplicationEffects(operations, units, context.Target)
	if err != nil {
		return Plan{}, err
	}

	plannedPaths := make([]string, 0, len(files))
	for _, effect := range files {
		plannedPaths = append(plannedPaths, effect.Path)
	}

	sort.Strings(plannedPaths)

	if !sameStrings(plannedPaths, context.Target.PlannedPaths) {
		return Plan{}, planError("plan_target_inventory_incomplete", "target.planned_paths", "target inventory does not cover the exact application plan paths")
	}

	capabilities := selectedBookCapabilityIDs(context.Book, selectedCapabilities)
	rules := selectedBookRules(context.Book, selectedRules)
	surfaces := operationSurfaces(operations)
	application := *value.Application

	result := Plan{
		SchemaVersion:        ApplicationSchemaVersion,
		Intent:               value.Operation,
		Archetype:            value.Archetype,
		Capabilities:         capabilities,
		AffectedSurfaces:     surfaces,
		Documentation:        value.Documentation,
		DocumentationTargets: cloneDocumentationTargets(value.DocumentationTargets),
		HatmaxVersion:        value.HatmaxVersion,
		BookVersion:          value.BookVersion,
		SourceFingerprint:    value.SourceFingerprint,
		Application:          &application,
		Target: &ApplicationTarget{
			Base: value.Target.Base, Directory: value.Target.Directory,
			Parent: context.Target.Parent, Path: context.Target.Target,
			Admission: context.Target.Admission, Preserved: cloneTargetEntries(context.Target.Preserved),
		},
		Units: units,
		FingerprintInputs: FingerprintInputs{
			SelectedPaths:        cloneStrings(context.Fingerprint.SelectedPaths),
			SelectedDependencies: cloneStrings(context.Fingerprint.SelectedDependencies),
			PlannedSurfaces:      []string{},
		},
		Rules:                rules,
		Operations:           operations,
		Preconditions:        expandApplicationPreconditions(value),
		ExpectedObservations: cloneObservations(context.Fingerprint.Observations),
		AllowedEffects: AllowedEffects{
			Surfaces: surfaces, Dependencies: dependencies, Files: files,
		},
		Validation: expandValidation(selectedRuleDefinitions(context.Book, selectedRules), operations, surfaces),
		Exceptions: cloneExceptions(value.Exceptions),
	}

	result, err = Seal(result)
	if err != nil {
		return Plan{}, fmt.Errorf("seal expanded application plan: %w", err)
	}

	return result, nil
}

func applicationUnitDrafts(value intent.Intent, selectedBook *book.Book) ([]applicationUnitDraft, error) {
	if selectedBook == nil {
		return nil, planError("plan_book_missing", "book_version", "a validated Hatmax Book is required")
	}

	if value.Operation != intent.OperationCreateApplication || value.Application == nil || value.Target == nil {
		return nil, planError("plan_application_invalid", "intent", "create_application identity and target are required")
	}

	application, err := buildApplicationUnitDraft(
		selectedBook,
		"application",
		value.Archetype,
		value.Operation,
		"",
		intent.Domain{},
		value.Capabilities,
		0,
	)
	if err != nil {
		return nil, err
	}

	result := []applicationUnitDraft{application}

	for index, feature := range value.InitialFeatures {
		archetype, exists := selectedBook.Archetype(applicationFeatureArchetype)
		if !exists {
			return nil, planError("plan_selection_invalid", "initial_features", "Book does not define %q", applicationFeatureArchetype)
		}

		operation, exists := selectedBookOperation(archetype, intent.OperationCreateFeature)
		if !exists {
			return nil, planError("plan_operation_missing", "initial_features", "archetype %q does not define create_feature", applicationFeatureArchetype)
		}

		capabilities := append([]string{}, operation.RequiredCapabilities...)

		featureDraft, draftErr := buildApplicationUnitDraft(
			selectedBook,
			"feature."+feature.Feature,
			applicationFeatureArchetype,
			intent.OperationCreateFeature,
			feature.Feature,
			feature.Domain,
			capabilities,
			index+1,
		)
		if draftErr != nil {
			return nil, draftErr
		}

		featureDraft.unit.DependsOn = []string{"application"}
		result = append(result, featureDraft)
	}

	return result, nil
}

func buildApplicationUnitDraft(
	selectedBook *book.Book,
	unitID string,
	archetypeID string,
	operationID intent.Operation,
	feature string,
	domain intent.Domain,
	capabilities []string,
	sequence int,
) (applicationUnitDraft, error) {
	selection, err := selectedBook.Select(archetypeID, capabilities)
	if err != nil {
		return applicationUnitDraft{}, planError("plan_selection_invalid", unitID, "%v", err)
	}

	operation, exists := selectedBookOperation(selection.Archetype, operationID)
	if !exists {
		return applicationUnitDraft{}, planError("plan_operation_missing", unitID, "operation %q is not defined by archetype %q", operationID, archetypeID)
	}

	selected := stringSet(selectedCapabilityIDs(selection.Capabilities))

	err = validateReferences("capabilities", operation.RequiredCapabilities, selected)
	if err != nil {
		return applicationUnitDraft{}, planError("plan_capability_missing", unitID, "operation %q is missing a required capability", operationID)
	}

	operations, err := expandOperations(selection, operation, false)
	if err != nil {
		return applicationUnitDraft{}, err
	}

	operations, err = scopeApplicationOperations(operations, unitID, feature, sequence)
	if err != nil {
		return applicationUnitDraft{}, err
	}

	return applicationUnitDraft{
		unit: Unit{
			ID: unitID, Intent: operationID, Archetype: archetypeID, Feature: feature,
			Domain: cloneDomain(domain), Capabilities: selectedCapabilityIDs(selection.Capabilities),
			AffectedSurfaces: affectedSurfaces(selection.Archetype, operations), Effects: []FileEffect{},
		},
		operation:  operationID,
		selection:  selection,
		operations: operations,
	}, nil
}

func scopeApplicationOperations(values []Operation, unitID, feature string, sequence int) ([]Operation, error) {
	ids := make(map[string]string, len(values))
	for _, value := range values {
		ids[value.ID] = unitID + "." + value.ID
	}

	result := cloneOperations(values)
	for index := range result {
		result[index].ID = ids[result[index].ID]
		for dependencyIndex, dependency := range result[index].DependsOn {
			result[index].DependsOn[dependencyIndex] = ids[dependency]
		}

		for effectIndex := range result[index].Files {
			effectPath := result[index].Files[effectIndex].Path
			effectPath = strings.ReplaceAll(effectPath, "{feature}", feature)

			effectPath = strings.ReplaceAll(effectPath, "{sequence}", fmt.Sprintf("%04d", sequence))
			if strings.ContainsAny(effectPath, "{}") || strings.Contains(effectPath, "\\") || !path.IsAbs("/"+effectPath) || path.Clean(effectPath) != effectPath || effectPath == "." {
				return nil, planError("plan_file_effect_invalid", result[index].ID, "Book path %q could not be expanded safely", result[index].Files[effectIndex].Path)
			}

			result[index].Files[effectIndex].Path = effectPath
		}
	}

	return result, nil
}

func resolveApplicationEffects(
	operations []Operation,
	units []Unit,
	target *project.TargetInventory,
) ([]Operation, []Unit, []FileEffect, error) {
	available := make(map[string]struct{})

	for _, entry := range target.Entries {
		if entry.Kind == project.TargetEntryFile {
			available[entry.Path] = struct{}{}
		}
	}

	unitIndex := make(map[string]int, len(units))
	for index, unit := range units {
		unitIndex[unit.ID] = index
	}

	allowed := make([]FileEffect, 0)
	allowedIndex := make(map[string]int)

	unitSeen := make([]map[string]struct{}, len(units))
	for index := range unitSeen {
		unitSeen[index] = make(map[string]struct{})
	}

	for operationIndex := range operations {
		unitID := operationUnitID(operations[operationIndex].ID, units)

		index, exists := unitIndex[unitID]
		if !exists {
			return nil, nil, nil, planError("plan_unit_invalid", operations[operationIndex].ID, "operation has no owning application unit")
		}

		for effectIndex := range operations[operationIndex].Files {
			effect := &operations[operationIndex].Files[effectIndex]

			_, pathExists := available[effect.Path]
			switch book.FileEffectMode(effect.Effect) {
			case book.FileEffectCreate:
				if pathExists {
					return nil, nil, nil, planError("plan_target_collision", effect.Path, "planned create path already exists")
				}

				effect.Effect = FileEffectCreate
				available[effect.Path] = struct{}{}
			case book.FileEffectUpdate:
				if !pathExists {
					return nil, nil, nil, planError("plan_file_effect_invalid", effect.Path, "planned update path is not produced by an earlier effect")
				}

				effect.Effect = FileEffectUpdate
			case book.FileEffectEnsure:
				if pathExists {
					effect.Effect = FileEffectUpdate
				} else {
					effect.Effect = FileEffectCreate
					available[effect.Path] = struct{}{}
				}
			default:
				return nil, nil, nil, planError("plan_file_effect_invalid", effect.Path, "unknown Book file effect %q", effect.Effect)
			}

			if _, seen := unitSeen[index][effect.Path]; !seen {
				units[index].Effects = append(units[index].Effects, *effect)
				unitSeen[index][effect.Path] = struct{}{}
			}

			if allowedPosition, seen := allowedIndex[effect.Path]; seen {
				if allowed[allowedPosition].Effect == FileEffectUpdate && effect.Effect == FileEffectCreate {
					allowed[allowedPosition].Effect = FileEffectCreate
				}

				continue
			}

			allowedIndex[effect.Path] = len(allowed)
			allowed = append(allowed, *effect)
		}
	}

	return operations, units, allowed, nil
}

func operationUnitID(operationID string, units []Unit) string {
	for _, unit := range units {
		if strings.HasPrefix(operationID, unit.ID+".") {
			return unit.ID
		}
	}

	return ""
}

func selectedBookCapabilityIDs(selectedBook *book.Book, selected map[string]struct{}) []string {
	result := make([]string, 0, len(selected))
	for _, capability := range selectedBook.Capabilities() {
		if _, exists := selected[capability.ID]; exists {
			result = append(result, capability.ID)
		}
	}

	return result
}

func selectedBookRules(selectedBook *book.Book, selected map[string]struct{}) []RuleRef {
	result := make([]RuleRef, 0, len(selected))
	for _, rule := range selectedBook.Rules() {
		if _, exists := selected[rule.ID]; exists {
			result = append(result, RuleRef{ID: rule.ID, Level: rule.Level})
		}
	}

	return result
}

func selectedRuleDefinitions(selectedBook *book.Book, selected map[string]struct{}) []book.Rule {
	result := make([]book.Rule, 0, len(selected))
	for _, rule := range selectedBook.Rules() {
		if _, exists := selected[rule.ID]; exists {
			result = append(result, rule)
		}
	}

	return result
}

func operationSurfaces(operations []Operation) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})

	for _, operation := range operations {
		for _, surface := range operation.Surfaces {
			if _, exists := seen[surface]; exists {
				continue
			}

			seen[surface] = struct{}{}
			result = append(result, surface)
		}
	}

	return result
}

func expandApplicationPreconditions(value intent.Intent) []Precondition {
	return []Precondition{
		{ID: "source_fingerprint", Kind: PreconditionSourceFingerprint, Expected: value.SourceFingerprint},
		{ID: "hatmax_version", Kind: PreconditionHatmaxVersion, Expected: value.HatmaxVersion},
		{ID: "book_version", Kind: PreconditionBookVersion, Expected: strconv.Itoa(value.BookVersion)},
	}
}

func cloneTargetEntries(values []project.TargetEntry) []project.TargetEntry {
	return append([]project.TargetEntry{}, values...)
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func intentFingerprint(value intent.Intent) string {
	if value.SchemaVersion == intent.ApplicationSchemaVersion {
		return value.SourceFingerprint
	}

	return value.ProjectFingerprint
}

func intentFingerprintField(value intent.Intent) string {
	if value.SchemaVersion == intent.ApplicationSchemaVersion {
		return "source_fingerprint"
	}

	return "project_fingerprint"
}
