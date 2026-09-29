package plan

import (
	"path"
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
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

	if value.Intent != intent.OperationCreateFeature && value.Intent != intent.OperationAddField && value.Intent != intent.OperationAddValidation && value.Intent != intent.OperationDocumentFeature {
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

	err = validateDocumentationEffects(value)
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
		SchemaVersion:        intent.CurrentSchemaVersion,
		Operation:            value.Intent,
		ProjectFingerprint:   value.ProjectFingerprint,
		HatmaxVersion:        value.HatmaxVersion,
		BookVersion:          value.BookVersion,
		Archetype:            value.Archetype,
		Feature:              value.Feature,
		Domain:               value.Domain,
		Capabilities:         value.Capabilities,
		Documentation:        value.Documentation,
		DocumentationTargets: value.DocumentationTargets,
		Exceptions:           value.Exceptions,
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

func validateDocumentationEffects(value Plan) error {
	hasDocumentation := containsString(value.AffectedSurfaces, "documentation")
	if value.Documentation == intent.DocumentationNotRequested {
		if value.DocumentationEvidence != nil {
			return planError("plan_documentation_evidence_invalid", "documentation_evidence", "documentation evidence requires explicit documentation intent")
		}

		if value.DocumentationPlan != nil {
			return planError("plan_documentation_effects_invalid", "documentation_plan", "documentation effects require explicit documentation intent")
		}

		if hasDocumentation {
			return planError("plan_documentation_scope_invalid", "affected_surfaces", "documentation surface requires explicit documentation intent")
		}

		return nil
	}

	if !hasDocumentation {
		return planError("plan_documentation_scope_invalid", "affected_surfaces", "active documentation must include the documentation surface")
	}

	err := validateDocumentationEvidence(value)
	if err != nil {
		return err
	}

	err = validateDocumentationPlan(value)
	if err != nil {
		return err
	}

	if value.Intent != intent.OperationDocumentFeature {
		return nil
	}

	if len(value.AffectedSurfaces) != 1 || value.AffectedSurfaces[0] != "documentation" {
		return planError("plan_documentation_scope_invalid", "affected_surfaces", "document_feature may affect only documentation")
	}

	if len(value.AllowedEffects.Dependencies) != 0 {
		return planError("plan_documentation_scope_invalid", "allowed_effects.dependencies", "document_feature cannot add runtime dependencies")
	}

	return nil
}

func validateDocumentationPlan(value Plan) error {
	documentationPlan := value.DocumentationPlan
	if documentationPlan == nil || len(documentationPlan.Targets) != len(value.DocumentationTargets) || len(documentationPlan.Indexes) < 2 {
		return planError("plan_documentation_effects_invalid", "documentation_plan", "active documentation requires exact target and index effects")
	}

	intentTargets := make(map[string]intent.DocumentationTarget, len(value.DocumentationTargets))
	for _, target := range value.DocumentationTargets {
		intentTargets[string(target.Quadrant)+"\x00"+target.Subject] = target
	}

	targetPaths := make(map[string]struct{}, len(documentationPlan.Targets)+len(documentationPlan.Indexes))
	expectedQuadrantLinks := make(map[intent.DocumentationQuadrant]map[string]DocumentationLinkEffect)

	for index, target := range documentationPlan.Targets {
		contract, exists := intentTargets[string(target.Quadrant)+"\x00"+target.Subject]
		layout, validQuadrant := documentationQuadrants[target.Quadrant]
		expectedSlug := strings.ReplaceAll(target.Subject, "_", "-")
		expectedPath := path.Join("docs", layout.directory, expectedSlug, "README.md")

		if !exists || !validQuadrant || target.ReaderGoal != contract.ReaderGoal || target.Slug != expectedSlug || target.Path != expectedPath || target.Title != documentationTargetTitle(target.Quadrant, target.Subject) {
			return planError("plan_documentation_effects_invalid", indexedPath("documentation_plan.targets", index), "target does not match typed documentation intent")
		}

		if _, duplicate := targetPaths[target.Path]; duplicate {
			return planError("plan_documentation_target_overlap", indexedPath("documentation_plan.targets", index)+".path", "duplicate target %q", target.Path)
		}

		err := validateDocumentationSnapshot(target.Snapshot, target.Path, value.ExpectedObservations)
		if err != nil {
			return err
		}

		if expectedQuadrantLinks[target.Quadrant] == nil {
			expectedQuadrantLinks[target.Quadrant] = make(map[string]DocumentationLinkEffect)
		}

		expectedQuadrantLinks[target.Quadrant][target.Path] = DocumentationLinkEffect{
			Title: target.Title, Target: target.Path, Relative: path.Join(target.Slug, "README.md"),
		}
		targetPaths[target.Path] = struct{}{}
	}

	seenIndexes := make(map[string]struct{}, len(documentationPlan.Indexes))
	for index, effect := range documentationPlan.Indexes {
		if _, duplicate := targetPaths[effect.Path]; duplicate {
			return planError("plan_documentation_target_overlap", indexedPath("documentation_plan.indexes", index)+".path", "overlapping target %q", effect.Path)
		}

		if _, duplicate := seenIndexes[effect.Path]; duplicate {
			return planError("plan_documentation_target_overlap", indexedPath("documentation_plan.indexes", index)+".path", "duplicate index %q", effect.Path)
		}

		err := validateDocumentationIndex(effect, expectedQuadrantLinks)
		if err != nil {
			return err
		}

		err = validateDocumentationSnapshot(effect.Snapshot, effect.Path, value.ExpectedObservations)
		if err != nil {
			return err
		}

		seenIndexes[effect.Path] = struct{}{}
		targetPaths[effect.Path] = struct{}{}
	}

	if _, root := seenIndexes["docs/README.md"]; !root || len(seenIndexes) != len(expectedQuadrantLinks)+1 {
		return planError("plan_documentation_effects_invalid", "documentation_plan.indexes", "index effects must contain one root and one index per selected quadrant")
	}

	return validateDocumentationCommands(documentationPlan.ValidationCommands)
}

func validateDocumentationIndex(
	effect DocumentationIndexEffect,
	expectedQuadrantLinks map[intent.DocumentationQuadrant]map[string]DocumentationLinkEffect,
) error {
	expectedLinks := make(map[string]DocumentationLinkEffect)

	if effect.Kind == documentationIndexRoot {
		if effect.Path != "docs/README.md" || effect.Quadrant != "" || effect.Title != "Documentation" {
			return planError("plan_documentation_effects_invalid", "documentation_plan.indexes", "root index identity is invalid")
		}

		for quadrant := range expectedQuadrantLinks {
			layout := documentationQuadrants[quadrant]
			target := path.Join("docs", layout.directory, "README.md")
			expectedLinks[target] = DocumentationLinkEffect{Title: layout.title, Target: target, Relative: path.Join(layout.directory, "README.md")}
		}
	} else if effect.Kind == documentationIndexQuadrant {
		layout, exists := documentationQuadrants[effect.Quadrant]
		if !exists || effect.Path != path.Join("docs", layout.directory, "README.md") || effect.Title != layout.title {
			return planError("plan_documentation_effects_invalid", "documentation_plan.indexes", "quadrant index identity is invalid")
		}

		expectedLinks = expectedQuadrantLinks[effect.Quadrant]
	} else {
		return planError("plan_documentation_effects_invalid", "documentation_plan.indexes", "unknown index kind %q", effect.Kind)
	}

	if len(effect.RequiredLinks) != len(expectedLinks) {
		return planError("plan_documentation_effects_invalid", "documentation_plan.indexes.required_links", "index links are incomplete")
	}

	seen := make(map[string]struct{}, len(effect.RequiredLinks))
	for _, link := range effect.RequiredLinks {
		expected, exists := expectedLinks[link.Target]
		if !exists || link != expected {
			return planError("plan_documentation_effects_invalid", "documentation_plan.indexes.required_links", "index link is not derived from a selected target")
		}

		if _, duplicate := seen[link.Target]; duplicate {
			return planError("plan_duplicate_value", "documentation_plan.indexes.required_links", "duplicate link target %q", link.Target)
		}

		seen[link.Target] = struct{}{}
	}

	return nil
}

func validateDocumentationSnapshot(snapshot DocumentationSnapshot, expectedPath string, observations []project.Observation) error {
	if snapshot.Path != expectedPath {
		return planError("plan_documentation_snapshot_invalid", expectedPath, "snapshot path does not match its effect")
	}

	if !snapshot.Exists {
		if snapshot.Digest != "" || snapshot.ManagedState != "" {
			return planError("plan_documentation_snapshot_invalid", expectedPath, "absent snapshot cannot contain file state")
		}

		return nil
	}

	if !fingerprintPattern.MatchString(snapshot.Digest) || snapshot.ManagedState != project.DocumentationManaged {
		return planError("plan_documentation_snapshot_invalid", expectedPath, "existing snapshot must be managed and digest-bound")
	}

	for _, observation := range observations {
		if observation.Path == expectedPath && observation.Digest == snapshot.Digest {
			return nil
		}
	}

	return planError("plan_documentation_snapshot_invalid", expectedPath, "snapshot is not bound to the project fingerprint")
}

func validateDocumentationCommands(values []DocumentationCommand) error {
	seen := make(map[string]struct{}, len(values))
	for index, command := range values {
		commandPath := indexedPath("documentation_plan.validation_commands", index)
		if command.Kind != project.CommandValidation || strings.TrimSpace(command.Name) == "" || strings.TrimSpace(command.Source) == "" || len(command.Args) == 0 {
			return planError("plan_documentation_command_invalid", commandPath, "validation command name, source, and arguments are required")
		}

		if _, duplicate := seen[command.Name]; duplicate {
			return planError("plan_duplicate_value", commandPath+".name", "duplicate documentation command %q", command.Name)
		}

		seen[command.Name] = struct{}{}
	}

	return nil
}

func validateDocumentationEvidence(value Plan) error {
	evidence := value.DocumentationEvidence
	if evidence == nil {
		return planError("plan_documentation_evidence_invalid", "documentation_evidence", "active documentation requires feature evidence")
	}

	if evidence.Feature != value.Feature || strings.TrimSpace(evidence.Entity) == "" || strings.TrimSpace(evidence.Route) == "" || strings.TrimSpace(evidence.Table) == "" || len(evidence.Fields) == 0 {
		return planError("plan_documentation_evidence_invalid", "documentation_evidence", "feature, entity, route, table, and fields are required")
	}

	if evidence.Basis != "existing" && evidence.Basis != "planned" {
		return planError("plan_documentation_evidence_invalid", "documentation_evidence.basis", "unknown evidence basis %q", evidence.Basis)
	}

	if value.Documentation == intent.DocumentationExisting && (evidence.Basis != "existing" || len(evidence.Sources) == 0) {
		return planError("plan_documentation_evidence_invalid", "documentation_evidence", "existing behavior requires inspected source evidence")
	}

	fields := make(map[string]struct{}, len(evidence.Fields))
	for index, field := range evidence.Fields {
		path := indexedPath("documentation_evidence.fields", index)
		if strings.TrimSpace(field.Name) == "" || strings.TrimSpace(field.Type) == "" {
			return planError("plan_documentation_evidence_invalid", path, "field name and type are required")
		}

		if _, exists := fields[field.Name]; exists {
			return planError("plan_duplicate_value", path+".name", "duplicate evidence field %q", field.Name)
		}

		fields[field.Name] = struct{}{}
	}

	observedDigests := make(map[string]string, len(value.ExpectedObservations))
	for _, observation := range value.ExpectedObservations {
		if observation.Path != "" {
			observedDigests[observation.Path] = observation.Digest
		}
	}

	for index, source := range evidence.Sources {
		path := indexedPath("documentation_evidence.sources", index)
		if strings.TrimSpace(source.Role) == "" || strings.TrimSpace(source.Path) == "" || !fingerprintPattern.MatchString(source.Digest) {
			return planError("plan_documentation_evidence_invalid", path, "source role, path, and digest are required")
		}

		if observedDigests[source.Path] != source.Digest {
			return planError("plan_documentation_evidence_invalid", path, "source %q is not bound to the project fingerprint", source.Path)
		}
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
