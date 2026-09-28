package execute

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

var (
	stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	digestPattern   = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

// Validate checks an execution manifest without reading or changing its
// project.
func Validate(value Manifest) error {
	if value.SchemaVersion != CurrentSchemaVersion {
		return executionError("execution_schema_unsupported", "schema_version", "schema version %d is not supported", value.SchemaVersion)
	}

	if !digestPattern.MatchString(value.PlanDigest) || !digestPattern.MatchString(value.ProjectFingerprint) {
		return executionError("execution_identity_invalid", "identity", "plan digest and project fingerprint must use sha256:<hex>")
	}

	if value.Intent != intent.OperationCreateFeature && value.Intent != intent.OperationAddField && value.Intent != intent.OperationAddValidation {
		return executionError("execution_intent_invalid", "intent", "unknown intent %q", value.Intent)
	}

	if !stableIDPattern.MatchString(value.Feature) {
		return executionError("execution_feature_invalid", "feature", "feature must use a stable lower-case ID")
	}

	err := validateUniqueStrings("allowed_surfaces", value.AllowedSurfaces, true)
	if err != nil {
		return err
	}

	err = validateEdits(value.Edits, value.AllowedSurfaces)
	if err != nil {
		return err
	}

	err = validateCommands(value.Commands)
	if err != nil {
		return err
	}

	if value.Digest != "" && !digestPattern.MatchString(value.Digest) {
		return executionError("execution_digest_invalid", "digest", "digest must use sha256:<hex>")
	}

	return nil
}

func validateEdits(values []Edit, allowedSurfaces []string) error {
	if len(values) == 0 {
		return executionError("execution_required_field", "edits", "at least one edit is required")
	}

	surfaces := stringSet(allowedSurfaces)
	seenIDs := make(map[string]struct{}, len(values))
	seenTargets := make(map[string]struct{}, len(values))

	for index, value := range values {
		prefix := indexedPath("edits", index)
		if !stableIDPattern.MatchString(value.ID) || !stableIDPattern.MatchString(value.Operation) {
			return executionError("execution_edit_invalid", prefix, "edit and operation IDs must be stable")
		}

		if value.Owner.Kind != plan.OwnerArchetype && value.Owner.Kind != plan.OwnerCapability {
			return executionError("execution_owner_invalid", prefix+".owner.kind", "unknown owner kind %q", value.Owner.Kind)
		}

		if !stableIDPattern.MatchString(value.Owner.ID) {
			return executionError("execution_owner_invalid", prefix+".owner.id", "owner ID must be stable")
		}

		if !validEditKind(value.Kind) {
			return executionError("execution_edit_kind_invalid", prefix+".kind", "unknown edit kind %q", value.Kind)
		}

		if _, exists := surfaces[value.Surface]; !exists {
			return executionError("execution_surface_undeclared", prefix+".surface", "surface %q is not allowed by the plan", value.Surface)
		}

		target, err := normalizeRelativePath(value.Target)
		if err != nil || target != value.Target {
			return executionError("execution_target_invalid", prefix+".target", "target %q must be a clean project-relative path", value.Target)
		}

		if !stableIDPattern.MatchString(value.Recipe) {
			return executionError("execution_recipe_invalid", prefix+".recipe", "recipe must use a stable ID")
		}

		if _, exists := seenIDs[value.ID]; exists {
			return executionError("execution_duplicate_value", prefix+".id", "duplicate edit %q", value.ID)
		}

		if _, exists := seenTargets[value.Target]; exists {
			return executionError("execution_target_overlap", prefix+".target", "multiple edits target %q", value.Target)
		}

		err = validateUniqueStrings(prefix+".rules", value.Rules, true)
		if err != nil {
			return err
		}

		err = validateDependencies(prefix+".depends_on", value.DependsOn, seenIDs)
		if err != nil {
			return err
		}

		err = validateConditions(prefix+".preconditions", value.Preconditions)
		if err != nil {
			return err
		}

		err = validateConditions(prefix+".postconditions", value.Postconditions)
		if err != nil {
			return err
		}

		err = validateSlots(prefix+".implementation_slots", value.Slots)
		if err != nil {
			return err
		}

		seenIDs[value.ID] = struct{}{}
		seenTargets[value.Target] = struct{}{}
	}

	return nil
}

func validateCommands(values []Command) error {
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		prefix := indexedPath("commands", index)
		if value.Kind != project.CommandValidation && value.Kind != project.CommandGeneration && value.Kind != project.CommandFormatting {
			return executionError("execution_command_invalid", prefix+".kind", "unknown command kind %q", value.Kind)
		}

		if !stableIDPattern.MatchString(value.Name) || len(value.Args) == 0 || strings.TrimSpace(value.Source) == "" {
			return executionError("execution_command_invalid", prefix, "name, arguments, and source are required")
		}

		for argumentIndex, argument := range value.Args {
			if strings.TrimSpace(argument) == "" {
				return executionError("execution_command_invalid", indexedPath(prefix+".args", argumentIndex), "argument is required")
			}
		}

		if value.WorkingDirectory == "" {
			return executionError("execution_command_invalid", prefix+".working_directory", "working directory is required")
		}

		if value.WorkingDirectory != "." {
			normalized, err := normalizeRelativePath(value.WorkingDirectory)
			if err != nil || normalized != value.WorkingDirectory {
				return executionError("execution_command_invalid", prefix+".working_directory", "working directory must stay inside the project")
			}
		}

		if _, exists := seen[value.Name]; exists {
			return executionError("execution_duplicate_value", prefix+".name", "duplicate command %q", value.Name)
		}

		seen[value.Name] = struct{}{}
	}

	return nil
}

func validateConditions(path string, values []Condition) error {
	if len(values) == 0 {
		return executionError("execution_required_field", path, "at least one condition is required")
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		conditionPath := indexedPath(path, index)
		if !validConditionKind(value.Kind) {
			return executionError("execution_condition_invalid", conditionPath+".kind", "unknown condition kind %q", value.Kind)
		}

		if conditionNeedsValue(value.Kind) && strings.TrimSpace(value.Value) == "" {
			return executionError("execution_condition_invalid", conditionPath+".value", "condition %q requires a value", value.Kind)
		}

		identity := string(value.Kind) + "\x00" + value.Value
		if _, exists := seen[identity]; exists {
			return executionError("execution_duplicate_value", conditionPath, "duplicate condition %q", value.Kind)
		}

		seen[identity] = struct{}{}
	}

	return nil
}

func validateSlots(path string, values []ImplementationSlot) error {
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		slotPath := indexedPath(path, index)
		if !stableIDPattern.MatchString(value.Name) || strings.TrimSpace(value.Source) == "" {
			return executionError("execution_slot_invalid", slotPath, "slot name and source are required")
		}

		if _, exists := seen[value.Name]; exists {
			return executionError("execution_duplicate_value", slotPath+".name", "duplicate slot %q", value.Name)
		}

		seen[value.Name] = struct{}{}
	}

	return nil
}

func validateDependencies(path string, values []string, available map[string]struct{}) error {
	err := validateUniqueStrings(path, values, false)
	if err != nil {
		return err
	}

	for index, value := range values {
		if _, exists := available[value]; !exists {
			return executionError("execution_dependency_order_invalid", indexedPath(path, index), "dependency %q must reference an earlier edit", value)
		}
	}

	return nil
}

func validateUniqueStrings(path string, values []string, required bool) error {
	if required && len(values) == 0 {
		return executionError("execution_required_field", path, "at least one value is required")
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return executionError("execution_required_field", indexedPath(path, index), "value is required")
		}

		if _, exists := seen[value]; exists {
			return executionError("execution_duplicate_value", indexedPath(path, index), "duplicate value %q", value)
		}

		seen[value] = struct{}{}
	}

	return nil
}

func validEditKind(value EditKind) bool {
	return value == EditCreateFile || value == EditUpdateGo || value == EditUpdateSQL || value == EditUpdateTemplate || value == EditUpdateConfiguration || value == EditUpdateTest
}

func validConditionKind(value ConditionKind) bool {
	return value == ConditionPathAbsent || value == ConditionPathPresent || value == ConditionPathDigest || value == ConditionGoParses || value == ConditionContentContains || value == ConditionCommandAvailable
}

func conditionNeedsValue(value ConditionKind) bool {
	return value == ConditionPathDigest || value == ConditionContentContains || value == ConditionCommandAvailable
}

func normalizeRelativePath(value string) (string, error) {
	if value == "" || strings.Contains(value, `\`) {
		return "", executionError("execution_target_invalid", value, "path is empty or uses a non-portable separator")
	}

	cleaned := path.Clean(value)
	if cleaned == "." || strings.HasPrefix(cleaned, "/") || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", executionError("execution_target_invalid", value, "path must stay inside the project")
	}

	return cleaned, nil
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}

	return result
}

func indexedPath(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}
