// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package plan

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

func validateApplicationPlan(value Plan) error {
	if value.Intent != intent.OperationCreateApplication || value.Archetype != "server_rendered_hatmax_application" {
		return planError("plan_intent_invalid", "intent", "application plan must use the canonical create_application archetype")
	}

	if value.Feature != "" || domainHasValues(value.Domain) || value.ProjectFingerprint != "" {
		return planError("plan_application_invalid", "identity", "application plan cannot carry existing-project identity")
	}

	if !fingerprintPattern.MatchString(value.SourceFingerprint) {
		return planError("plan_fingerprint_invalid", "source_fingerprint", "fingerprint must use sha256:<hex>")
	}

	if strings.TrimSpace(value.HatmaxVersion) == "" || value.BookVersion < 1 {
		return planError("plan_required_field", "versions", "Hatmax and positive Book versions are required")
	}

	if value.Digest != "" && !fingerprintPattern.MatchString(value.Digest) {
		return planError("plan_digest_invalid", "digest", "digest must use sha256:<hex>")
	}

	if value.Application == nil || value.Target == nil {
		return planError("plan_required_field", "application", "application identity and target are required")
	}

	if value.Documentation != intent.DocumentationNotRequested || len(value.DocumentationTargets) != 0 || value.DocumentationEvidence != nil || value.DocumentationPlan != nil {
		return planError("plan_documentation_invalid", "documentation", "application bootstrap does not imply documentation")
	}

	err := validateApplicationIdentityAndTarget(value)
	if err != nil {
		return err
	}

	if len(value.Capabilities) == 0 || len(value.AffectedSurfaces) == 0 {
		return planError("plan_required_field", "selection", "capabilities and affected surfaces are required")
	}

	err = validateUniqueStrings("capabilities", value.Capabilities, true)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("affected_surfaces", value.AffectedSurfaces, true)
	if err != nil {
		return err
	}

	rules, err := validateRules(value.Rules)
	if err != nil {
		return err
	}

	err = validateOperations(value.Operations, value.AffectedSurfaces, rules)
	if err != nil {
		return err
	}

	err = validateApplicationUnits(value)
	if err != nil {
		return err
	}

	err = validatePreconditions(value.Preconditions, value)
	if err != nil {
		return err
	}

	err = validateAllowedEffects(value.AllowedEffects, value.AffectedSurfaces)
	if err != nil {
		return err
	}

	err = validateApplicationFileEffects(value)
	if err != nil {
		return err
	}

	err = validateApplicationFingerprintInputs(value)
	if err != nil {
		return err
	}

	return validateValidationObligations(value.Validation, value.AffectedSurfaces, value.Rules, rules)
}

func validateApplicationIdentityAndTarget(value Plan) error {
	application := value.Application
	target := value.Target

	if strings.TrimSpace(application.DisplayName) == "" || application.ProjectSlug == "" || application.ModulePath == "" {
		return planError("plan_required_field", "application", "display name, project slug, and module path are required")
	}

	if target.Base != "session_directory" || target.Directory != application.ProjectSlug {
		return planError("plan_target_invalid", "target", "target must use the normalized application child directory")
	}

	if !filepath.IsAbs(target.Parent) || !filepath.IsAbs(target.Path) || filepath.Clean(target.Path) != target.Path || filepath.Dir(target.Path) != target.Parent || filepath.Base(target.Path) != target.Directory {
		return planError("plan_target_invalid", "target.path", "target path must be the inspected child of its authorized parent")
	}

	if target.Admission != project.TargetAbsent && target.Admission != project.TargetEmpty && target.Admission != project.TargetPreservable {
		return planError("plan_target_invalid", "target.admission", "target admission %q cannot create an application", target.Admission)
	}

	seen := make(map[string]struct{}, len(target.Preserved))
	for index, entry := range target.Preserved {
		entryPath := indexedPath("target.preserved", index)
		if !validRelativeFileOrDirectory(entry.Path) {
			return planError("plan_target_invalid", entryPath+".path", "preserved path %q is invalid", entry.Path)
		}

		if _, exists := seen[entry.Path]; exists {
			return planError("plan_duplicate_value", entryPath+".path", "duplicate preserved path %q", entry.Path)
		}

		seen[entry.Path] = struct{}{}
	}

	return nil
}

func validateApplicationUnits(value Plan) error {
	if len(value.Units) == 0 {
		return planError("plan_required_field", "units", "application plan requires at least one unit")
	}

	var err error

	knownOperations := make(map[string]Operation, len(value.Operations))
	for _, operation := range value.Operations {
		knownOperations[operation.ID] = operation
	}

	knownCapabilities := stringSet(value.Capabilities)
	knownSurfaces := stringSet(value.AffectedSurfaces)
	seenUnits := make(map[string]struct{}, len(value.Units))
	seenOperations := make(map[string]struct{}, len(value.Operations))
	usedCapabilities := make(map[string]struct{}, len(value.Capabilities))
	usedSurfaces := make(map[string]struct{}, len(value.AffectedSurfaces))
	flattenedOperations := make([]string, 0, len(value.Operations))
	previousFeature := ""
	initialFeatures := make([]intent.InitialFeature, 0, len(value.Units)-1)

	for index, unit := range value.Units {
		unitPath := indexedPath("units", index)
		if !stableIDPattern.MatchString(unit.ID) {
			return planError("plan_unit_invalid", unitPath+".id", "unit ID must be stable")
		}

		if _, exists := seenUnits[unit.ID]; exists {
			return planError("plan_duplicate_value", unitPath+".id", "duplicate unit %q", unit.ID)
		}

		if index == 0 {
			if unit.ID != "application" || unit.Intent != intent.OperationCreateApplication || unit.Archetype != value.Archetype || unit.Feature != "" || domainHasValues(unit.Domain) || len(unit.DependsOn) != 0 {
				return planError("plan_unit_invalid", unitPath, "first unit must be the canonical application scaffold")
			}
		} else {
			if unit.ID != "feature."+unit.Feature || unit.Intent != intent.OperationCreateFeature || unit.Archetype != applicationFeatureArchetype || len(unit.DependsOn) != 1 || unit.DependsOn[0] != "application" {
				return planError("plan_unit_invalid", unitPath, "initial feature unit must depend on the application unit")
			}

			if previousFeature != "" && unit.Feature <= previousFeature {
				return planError("plan_unit_order_invalid", unitPath+".feature", "initial feature units must use stable feature order")
			}

			previousFeature = unit.Feature
			initialFeatures = append(initialFeatures, intent.InitialFeature{Feature: unit.Feature, Domain: cloneDomain(unit.Domain)})
		}

		err = validateUniqueStrings(unitPath+".capabilities", unit.Capabilities, true)
		if err != nil {
			return err
		}

		err = validateReferences(unitPath+".capabilities", unit.Capabilities, knownCapabilities)
		if err != nil {
			return err
		}

		for _, capability := range unit.Capabilities {
			usedCapabilities[capability] = struct{}{}
		}

		err = validateUniqueStrings(unitPath+".affected_surfaces", unit.AffectedSurfaces, true)
		if err != nil {
			return err
		}

		err = validateReferences(unitPath+".affected_surfaces", unit.AffectedSurfaces, knownSurfaces)
		if err != nil {
			return err
		}

		for _, surface := range unit.AffectedSurfaces {
			usedSurfaces[surface] = struct{}{}
		}

		err = validateUniqueStrings(unitPath+".operations", unit.Operations, true)
		if err != nil {
			return err
		}

		for _, operationID := range unit.Operations {
			if _, exists := knownOperations[operationID]; !exists {
				return planError("plan_reference_missing", unitPath+".operations", "unknown operation %q", operationID)
			}

			if _, exists := seenOperations[operationID]; exists {
				return planError("plan_duplicate_value", unitPath+".operations", "operation %q belongs to multiple units", operationID)
			}

			if !strings.HasPrefix(operationID, unit.ID+".") {
				return planError("plan_unit_invalid", unitPath+".operations", "operation %q is outside unit scope", operationID)
			}

			seenOperations[operationID] = struct{}{}
			flattenedOperations = append(flattenedOperations, operationID)
		}

		unitOperations := make([]Operation, 0, len(unit.Operations))
		for _, operationID := range unit.Operations {
			unitOperations = append(unitOperations, knownOperations[operationID])
		}

		if !sameStringSet(unit.AffectedSurfaces, operationSurfaces(unitOperations)) {
			return planError("plan_unit_surfaces_invalid", unitPath+".affected_surfaces", "unit surfaces must match its operations")
		}

		err = validateUnitEffects(unitPath, unit, knownOperations)
		if err != nil {
			return err
		}

		seenUnits[unit.ID] = struct{}{}
	}

	if len(flattenedOperations) != len(value.Operations) {
		return planError("plan_unit_incomplete", "units.operations", "every operation must belong to exactly one unit")
	}

	if len(usedCapabilities) != len(knownCapabilities) || len(usedSurfaces) != len(knownSurfaces) {
		return planError("plan_unit_incomplete", "units", "units must cover every selected capability and affected surface")
	}

	for index, operation := range value.Operations {
		if flattenedOperations[index] != operation.ID {
			return planError("plan_unit_order_invalid", "units.operations", "unit operation order must match plan operation order")
		}
	}

	intentValue := intent.Intent{
		SchemaVersion: intent.ApplicationSchemaVersion, Operation: value.Intent,
		SourceFingerprint: value.SourceFingerprint, HatmaxVersion: value.HatmaxVersion,
		BookVersion: value.BookVersion, Archetype: value.Archetype,
		Capabilities: []string{}, Documentation: value.Documentation,
		DocumentationTargets: cloneDocumentationTargets(value.DocumentationTargets),
		Exceptions:           cloneExceptions(value.Exceptions), Application: value.Application,
		Target:          &intent.ApplicationTarget{Base: value.Target.Base, Directory: value.Target.Directory},
		InitialFeatures: initialFeatures,
	}

	err = intent.ValidateSchema(intentValue)
	if err != nil {
		return planError("plan_application_invalid", "units", "%v", err)
	}

	return nil
}

func validateUnitEffects(unitPath string, unit Unit, operations map[string]Operation) error {
	expected := make([]FileEffect, 0)
	seen := make(map[string]struct{})

	for _, operationID := range unit.Operations {
		for _, effect := range operations[operationID].Files {
			if _, exists := seen[effect.Path]; exists {
				continue
			}

			seen[effect.Path] = struct{}{}
			expected = append(expected, effect)
		}
	}

	if len(expected) != len(unit.Effects) {
		return planError("plan_unit_effects_invalid", unitPath+".effects", "unit effects do not match owned operations")
	}

	for index := range expected {
		if expected[index] != unit.Effects[index] {
			return planError("plan_unit_effects_invalid", indexedPath(unitPath+".effects", index), "unit effect does not match owned operations")
		}
	}

	return nil
}

func validateApplicationFileEffects(value Plan) error {
	var err error

	allowed := make(map[string]FileEffectKind, len(value.AllowedEffects.Files))

	paths := make([]string, 0, len(value.AllowedEffects.Files))
	for index, effect := range value.AllowedEffects.Files {
		effectPath := indexedPath("allowed_effects.files", index)

		err = validateFileEffect(effectPath, effect)
		if err != nil {
			return err
		}

		if _, exists := allowed[effect.Path]; exists {
			return planError("plan_duplicate_value", effectPath+".path", "duplicate file effect %q", effect.Path)
		}

		allowed[effect.Path] = effect.Effect
		paths = append(paths, effect.Path)
	}

	if len(allowed) == 0 {
		return planError("plan_required_field", "allowed_effects.files", "application plan requires exact file effects")
	}

	for index, operation := range value.Operations {
		for fileIndex, effect := range operation.Files {
			effectPath := indexedPath(indexedPath("operations", index)+".files", fileIndex)

			err = validateFileEffect(effectPath, effect)
			if err != nil {
				return err
			}

			allowedEffect, exists := allowed[effect.Path]
			if !exists || allowedEffect == FileEffectUpdate && effect.Effect == FileEffectCreate {
				return planError("plan_effects_incomplete", effectPath, "operation file effect is outside allowed effects")
			}
		}
	}

	for _, entry := range value.Target.Preserved {
		if _, collision := allowed[entry.Path]; collision {
			return planError("plan_target_collision", entry.Path, "preserved path overlaps a planned file effect")
		}
	}

	sorted := append([]string{}, paths...)
	sort.Strings(sorted)

	if !sameStrings(sorted, value.FingerprintInputs.SelectedPaths) {
		return planError("plan_fingerprint_incomplete", "fingerprint_inputs.selected_paths", "target fingerprint paths must match allowed file effects")
	}

	return nil
}

func validateFileEffect(effectPath string, effect FileEffect) error {
	if !validRelativeFileOrDirectory(effect.Path) || effect.Effect != FileEffectCreate && effect.Effect != FileEffectUpdate {
		return planError("plan_file_effect_invalid", effectPath, "file effect requires a safe relative path and create or update action")
	}

	return nil
}

func validateApplicationFingerprintInputs(value Plan) error {
	err := validateUniqueStrings("fingerprint_inputs.selected_paths", value.FingerprintInputs.SelectedPaths, true)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("fingerprint_inputs.selected_dependencies", value.FingerprintInputs.SelectedDependencies, false)
	if err != nil {
		return err
	}

	if len(value.FingerprintInputs.PlannedSurfaces) != 0 {
		return planError("plan_fingerprint_inputs_invalid", "fingerprint_inputs.planned_surfaces", "target fingerprints use exact planned paths rather than project surfaces")
	}

	return nil
}

func validRelativeFileOrDirectory(value string) bool {
	return value != "" && value != "." && !strings.Contains(value, "\\") && !path.IsAbs(value) && path.Clean(value) == value && !strings.HasPrefix(value, "../")
}

func domainHasValues(value intent.Domain) bool {
	return value.Entity != "" || value.Route != "" || value.Label != "" || value.Ownership != "" ||
		len(value.Fields) != 0 || value.Field != nil || value.Validation != nil || len(value.Rules) != 0
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	rightSet := stringSet(right)
	for _, value := range left {
		if _, exists := rightSet[value]; !exists {
			return false
		}
	}

	return true
}
