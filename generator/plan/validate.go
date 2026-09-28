package plan

import (
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
)

var (
	stableIDPattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	fingerprintPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

// Validate checks the structural plan contract without inspecting a project or
// recomputing its digest.
func Validate(value Plan) error {
	if value.SchemaVersion != CurrentSchemaVersion {
		return planError("plan_schema_unsupported", "schema_version", "schema version %d is not supported", value.SchemaVersion)
	}

	if value.Intent == "" || value.Archetype == "" || value.Feature == "" {
		return planError("plan_required_field", "identity", "intent, archetype, and feature are required")
	}

	if value.Intent != intent.OperationCreateFeature && value.Intent != intent.OperationAddField && value.Intent != intent.OperationAddValidation {
		return planError("plan_intent_invalid", "intent", "unknown intent %q", value.Intent)
	}

	if value.Documentation != intent.DocumentationNotRequested && value.Documentation != intent.DocumentationExisting && value.Documentation != intent.DocumentationPlanned {
		return planError("plan_documentation_invalid", "documentation", "unknown documentation intent %q", value.Documentation)
	}

	if strings.TrimSpace(value.HatmaxVersion) == "" || value.BookVersion < 1 {
		return planError("plan_required_field", "versions", "Hatmax and positive Book versions are required")
	}

	if !fingerprintPattern.MatchString(value.ProjectFingerprint) {
		return planError("plan_fingerprint_invalid", "project_fingerprint", "fingerprint must use sha256:<hex>")
	}

	if value.Digest != "" && !fingerprintPattern.MatchString(value.Digest) {
		return planError("plan_digest_invalid", "digest", "digest must use sha256:<hex>")
	}

	if len(value.Capabilities) == 0 || len(value.AffectedSurfaces) == 0 {
		return planError("plan_required_field", "selection", "capabilities and affected surfaces are required")
	}

	err := validateUniqueStrings("capabilities", value.Capabilities, true)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("affected_surfaces", value.AffectedSurfaces, true)
	if err != nil {
		return err
	}

	err = validateDomain(value)
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

	err = validatePreconditions(value.Preconditions, value)
	if err != nil {
		return err
	}

	err = validateAllowedEffects(value.AllowedEffects, value.AffectedSurfaces)
	if err != nil {
		return err
	}

	err = validateFingerprintInputs(value.FingerprintInputs, value.AffectedSurfaces, value.AllowedEffects.Dependencies)
	if err != nil {
		return err
	}

	return validateValidationObligations(value.Validation, value.AffectedSurfaces, value.Rules, rules)
}

func validateFingerprintInputs(value FingerprintInputs, affectedSurfaces []string, dependencies []DependencyEffect) error {
	err := validateUniqueStrings("fingerprint_inputs.selected_paths", value.SelectedPaths, false)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("fingerprint_inputs.selected_dependencies", value.SelectedDependencies, false)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("fingerprint_inputs.planned_surfaces", value.PlannedSurfaces, true)
	if err != nil {
		return err
	}

	selectedSurfaces := stringSet(value.PlannedSurfaces)
	for _, surface := range affectedSurfaces {
		if _, exists := selectedSurfaces[surface]; !exists {
			return planError("plan_fingerprint_incomplete", "fingerprint_inputs.planned_surfaces", "affected surface %q is not fingerprinted", surface)
		}
	}

	selectedDependencies := stringSet(value.SelectedDependencies)
	for _, dependency := range dependencies {
		if _, exists := selectedDependencies[dependency.Module]; !exists {
			return planError("plan_fingerprint_incomplete", "fingerprint_inputs.selected_dependencies", "allowed dependency %q is not fingerprinted", dependency.Module)
		}
	}

	return nil
}

func validateDomain(value Plan) error {
	intentValue := intent.Intent{
		SchemaVersion:      intent.CurrentSchemaVersion,
		Operation:          value.Intent,
		ProjectFingerprint: value.ProjectFingerprint,
		HatmaxVersion:      value.HatmaxVersion,
		BookVersion:        value.BookVersion,
		Archetype:          value.Archetype,
		Feature:            value.Feature,
		Domain:             value.Domain,
		Capabilities:       value.Capabilities,
		Documentation:      value.Documentation,
		Exceptions:         value.Exceptions,
	}

	err := intent.ValidateSchema(intentValue)
	if err != nil {
		return planError("plan_domain_invalid", "domain", "%v", err)
	}

	diagnostics, clarifications := intent.ValidateDomainDecisions(intentValue)
	if len(diagnostics) > 0 {
		return planError("plan_domain_invalid", diagnostics[0].Field, "%s", diagnostics[0].Message)
	}

	if len(clarifications) > 0 {
		return planError("plan_domain_incomplete", clarifications[0].Field, "%s", clarifications[0].Question)
	}

	return nil
}

func validateRules(values []RuleRef) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, planError("plan_required_field", "rules", "at least one rule is required")
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		path := indexedPath("rules", index)
		if !stableIDPattern.MatchString(value.ID) || !validRuleLevel(value.Level) {
			return nil, planError("plan_rule_invalid", path, "rule ID and level are required")
		}

		if _, exists := seen[value.ID]; exists {
			return nil, planError("plan_duplicate_value", path+".id", "duplicate rule %q", value.ID)
		}

		seen[value.ID] = struct{}{}
	}

	return seen, nil
}

func validateOperations(values []Operation, affectedSurfaces []string, rules map[string]struct{}) error {
	if len(values) == 0 {
		return planError("plan_required_field", "operations", "at least one logical operation is required")
	}

	knownSurfaces := stringSet(affectedSurfaces)
	seen := make(map[string]struct{}, len(values))

	for index, value := range values {
		path := indexedPath("operations", index)
		if !stableIDPattern.MatchString(value.ID) || !stableIDPattern.MatchString(value.Obligation) {
			return planError("plan_operation_invalid", path, "operation and obligation IDs must be stable")
		}

		if value.Owner.Kind != OwnerArchetype && value.Owner.Kind != OwnerCapability {
			return planError("plan_owner_invalid", path+".owner.kind", "unknown owner kind %q", value.Owner.Kind)
		}

		if !stableIDPattern.MatchString(value.Owner.ID) {
			return planError("plan_owner_invalid", path+".owner.id", "owner ID must be stable")
		}

		if _, exists := seen[value.ID]; exists {
			return planError("plan_duplicate_value", path+".id", "duplicate operation %q", value.ID)
		}

		if len(value.Surfaces) == 0 || len(value.Rules) == 0 {
			return planError("plan_operation_invalid", path, "surfaces and rules are required")
		}

		err := validateUniqueStrings(path+".surfaces", value.Surfaces, true)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".rules", value.Rules, true)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".depends_on", value.DependsOn, false)
		if err != nil {
			return err
		}

		err = validateReferences(path+".surfaces", value.Surfaces, knownSurfaces)
		if err != nil {
			return err
		}

		err = validateReferences(path+".rules", value.Rules, rules)
		if err != nil {
			return err
		}

		err = validateReferences(path+".depends_on", value.DependsOn, seen)
		if err != nil {
			return planError("plan_operation_order_invalid", path+".depends_on", "%v", err)
		}

		seen[value.ID] = struct{}{}
	}

	return nil
}

func validatePreconditions(values []Precondition, plan Plan) error {
	if len(values) == 0 {
		return planError("plan_required_field", "preconditions", "at least one precondition is required")
	}

	seen := make(map[string]struct{}, len(values))
	kinds := make(map[PreconditionKind]string, len(values))

	for index, value := range values {
		path := indexedPath("preconditions", index)
		if !stableIDPattern.MatchString(value.ID) || strings.TrimSpace(value.Expected) == "" {
			return planError("plan_precondition_invalid", path, "precondition ID and expected value are required")
		}

		if value.Kind != PreconditionProjectFingerprint && value.Kind != PreconditionHatmaxVersion && value.Kind != PreconditionBookVersion {
			return planError("plan_precondition_invalid", path+".kind", "unknown precondition kind %q", value.Kind)
		}

		if _, exists := seen[value.ID]; exists {
			return planError("plan_duplicate_value", path+".id", "duplicate precondition %q", value.ID)
		}

		seen[value.ID] = struct{}{}

		if _, exists := kinds[value.Kind]; exists {
			return planError("plan_duplicate_value", path+".kind", "duplicate precondition kind %q", value.Kind)
		}

		kinds[value.Kind] = value.Expected
	}

	expected := map[PreconditionKind]string{
		PreconditionProjectFingerprint: plan.ProjectFingerprint,
		PreconditionHatmaxVersion:      plan.HatmaxVersion,
		PreconditionBookVersion:        strconv.Itoa(plan.BookVersion),
	}
	for kind, want := range expected {
		if kinds[kind] != want {
			return planError("plan_precondition_mismatch", "preconditions", "%s must expect %q", kind, want)
		}
	}

	return nil
}

func validateAllowedEffects(value AllowedEffects, affectedSurfaces []string) error {
	if len(value.Surfaces) == 0 {
		return planError("plan_required_field", "allowed_effects.surfaces", "at least one surface is required")
	}

	err := validateUniqueStrings("allowed_effects.surfaces", value.Surfaces, true)
	if err != nil {
		return err
	}

	err = validateReferences("allowed_effects.surfaces", value.Surfaces, stringSet(affectedSurfaces))
	if err != nil {
		return err
	}

	if len(value.Surfaces) != len(affectedSurfaces) {
		return planError("plan_effects_incomplete", "allowed_effects.surfaces", "allowed effects must cover every affected surface")
	}

	seen := make(map[string]struct{}, len(value.Dependencies))
	for index, dependency := range value.Dependencies {
		path := indexedPath("allowed_effects.dependencies", index)
		if !stableIDPattern.MatchString(dependency.Capability) || dependency.Kind == "" || dependency.Module == "" || dependency.Purpose == "" {
			return planError("plan_dependency_invalid", path, "capability, kind, module, and purpose are required")
		}

		identity := dependency.Kind + "\x00" + dependency.Module
		if _, exists := seen[identity]; exists {
			return planError("plan_duplicate_value", path, "duplicate dependency %q", dependency.Module)
		}

		seen[identity] = struct{}{}
	}

	return nil
}

func validateValidationObligations(
	values []ValidationObligation,
	affectedSurfaces []string,
	ruleRefs []RuleRef,
	rules map[string]struct{},
) error {
	if len(values) == 0 {
		return planError("plan_required_field", "validation", "at least one validation obligation is required")
	}

	knownSurfaces := stringSet(affectedSurfaces)

	ruleLevels := make(map[string]book.Level, len(ruleRefs))
	for _, rule := range ruleRefs {
		ruleLevels[rule.ID] = rule.Level
	}

	seen := make(map[string]struct{}, len(values))

	for index, value := range values {
		path := indexedPath("validation", index)
		if _, exists := rules[value.Rule]; !exists {
			return planError("plan_reference_missing", path+".rule", "unknown rule %q", value.Rule)
		}

		if !validRuleLevel(value.Level) || len(value.Diagnostics) == 0 {
			return planError("plan_validation_invalid", path, "level and diagnostics are required")
		}

		if value.Level != ruleLevels[value.Rule] {
			return planError("plan_validation_invalid", path+".level", "validation level must match selected rule")
		}

		if _, exists := seen[value.Rule]; exists {
			return planError("plan_duplicate_value", path+".rule", "duplicate validation rule %q", value.Rule)
		}

		err := validateUniqueStrings(path+".diagnostics", value.Diagnostics, true)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".surfaces", value.Surfaces, false)
		if err != nil {
			return err
		}

		err = validateReferences(path+".surfaces", value.Surfaces, knownSurfaces)
		if err != nil {
			return err
		}

		seen[value.Rule] = struct{}{}
	}

	if len(seen) != len(rules) {
		return planError("plan_validation_incomplete", "validation", "every selected rule requires one validation obligation")
	}

	return nil
}

func validRuleLevel(level book.Level) bool {
	return level == book.LevelRequired || level == book.LevelRecommended || level == book.LevelOptional || level == book.LevelProhibited
}

func validateUniqueStrings(path string, values []string, required bool) error {
	if required && len(values) == 0 {
		return planError("plan_required_field", path, "at least one value is required")
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if strings.TrimSpace(value) == "" {
			return planError("plan_required_field", indexedPath(path, index), "value is required")
		}

		if _, exists := seen[value]; exists {
			return planError("plan_duplicate_value", indexedPath(path, index), "duplicate value %q", value)
		}

		seen[value] = struct{}{}
	}

	return nil
}

func validateReferences(path string, values []string, known map[string]struct{}) error {
	for index, value := range values {
		if _, exists := known[value]; !exists {
			return planError("plan_reference_missing", indexedPath(path, index), "unknown reference %q", value)
		}
	}

	return nil
}

func indexedPath(path string, index int) string {
	return path + "[" + strconv.Itoa(index) + "]"
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}

	return result
}
