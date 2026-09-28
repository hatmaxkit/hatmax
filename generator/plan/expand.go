package plan

import (
	"fmt"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

// ExpansionContext supplies the validated Book and exact project fingerprint
// used to produce a plan.
type ExpansionContext struct {
	Book                  *book.Book
	Fingerprint           project.Fingerprint
	Inventory             project.Inventory
	DocumentationEvidence *project.FeatureEvidence
}

type operationCandidate struct {
	operation Operation
}

// Expand deterministically converts an admitted intent into an inspectable
// plan without reading or modifying the project.
func Expand(admission intent.Result, context ExpansionContext) (Plan, error) {
	if !admission.Admitted() {
		return Plan{}, planError("plan_intent_not_admitted", "intent", "intent status %q cannot be planned", admission.Status)
	}

	if context.Book == nil {
		return Plan{}, planError("plan_book_missing", "book_version", "a validated Hatmax Book is required")
	}

	value := admission.Intent

	err := intent.ValidateSchema(value)
	if err != nil {
		return Plan{}, planError("plan_intent_invalid", "intent", "%v", err)
	}

	manifest := context.Book.Manifest()
	if value.BookVersion != manifest.BookVersion || context.Fingerprint.BookVersion != value.BookVersion {
		return Plan{}, planError("plan_book_version_mismatch", "book_version", "intent, fingerprint, and Book versions must match")
	}

	if value.ProjectFingerprint != context.Fingerprint.Value {
		return Plan{}, planError("plan_fingerprint_mismatch", "project_fingerprint", "intent fingerprint does not match expansion context")
	}

	supported, err := context.Book.SupportsHatmax(value.HatmaxVersion)
	if err != nil || !supported {
		return Plan{}, planError("plan_hatmax_incompatible", "hatmax_version", "selected Book does not support Hatmax %q", value.HatmaxVersion)
	}

	selection, err := context.Book.Select(value.Archetype, value.Capabilities)
	if err != nil {
		return Plan{}, planError("plan_selection_invalid", "selection", "%v", err)
	}

	selectedOperation, exists := selectedBookOperation(selection.Archetype, value.Operation)
	if !exists {
		return Plan{}, planError("plan_operation_missing", "intent", "operation %q is not defined by archetype %q", value.Operation, value.Archetype)
	}

	selectedCapabilities := stringSet(selectedCapabilityIDs(selection.Capabilities))

	err = validateReferences("capabilities", selectedOperation.RequiredCapabilities, selectedCapabilities)
	if err != nil {
		return Plan{}, planError("plan_capability_missing", "capabilities", "operation %q is missing a required capability", value.Operation)
	}

	operations, err := expandOperations(selection, selectedOperation, value.Documentation != intent.DocumentationNotRequested)
	if err != nil {
		return Plan{}, err
	}

	surfaces := affectedSurfaces(selection.Archetype, operations)
	rules := expandRules(selection.Rules)

	documentationEvidence, err := expandDocumentationEvidence(value, context)
	if err != nil {
		return Plan{}, err
	}

	result := Plan{
		SchemaVersion:         CurrentSchemaVersion,
		Intent:                value.Operation,
		Archetype:             value.Archetype,
		Feature:               value.Feature,
		Domain:                cloneDomain(value.Domain),
		Capabilities:          selectedCapabilityIDs(selection.Capabilities),
		AffectedSurfaces:      surfaces,
		Documentation:         value.Documentation,
		DocumentationTargets:  cloneDocumentationTargets(value.DocumentationTargets),
		DocumentationEvidence: documentationEvidence,
		HatmaxVersion:         value.HatmaxVersion,
		BookVersion:           value.BookVersion,
		ProjectFingerprint:    value.ProjectFingerprint,
		FingerprintInputs: FingerprintInputs{
			SelectedPaths:        cloneStrings(context.Fingerprint.SelectedPaths),
			SelectedDependencies: cloneStrings(context.Fingerprint.SelectedDependencies),
			PlannedSurfaces:      cloneStrings(context.Fingerprint.PlannedSurfaces),
		},
		Rules:                rules,
		Operations:           operations,
		Preconditions:        expandPreconditions(value),
		ExpectedObservations: cloneObservations(context.Fingerprint.Observations),
		AllowedEffects: AllowedEffects{
			Surfaces:     cloneStrings(surfaces),
			Dependencies: expandDependencies(selection.Capabilities, value.Operation != intent.OperationDocumentFeature),
		},
		Validation: expandValidation(selection.Rules, operations, surfaces),
		Exceptions: cloneExceptions(value.Exceptions),
	}

	result, err = Seal(result)
	if err != nil {
		return Plan{}, fmt.Errorf("seal expanded plan: %w", err)
	}

	return result, nil
}

func expandDocumentationEvidence(value intent.Intent, context ExpansionContext) (*project.FeatureEvidence, error) {
	if value.Documentation == intent.DocumentationNotRequested {
		return nil, nil
	}

	if value.Operation == intent.OperationCreateFeature {
		result := plannedFeatureEvidence(value)

		return &result, nil
	}

	if context.DocumentationEvidence != nil {
		result := cloneFeatureEvidence(*context.DocumentationEvidence)
		if value.Operation != intent.OperationDocumentFeature {
			result.Basis = "planned"
		}

		result = applyPlannedEvidence(value, result)

		return &result, nil
	}

	evidence, err := context.Inventory.InspectFeatureEvidence(value.Feature)
	if err != nil {
		return nil, planError("plan_documentation_evidence_invalid", "documentation_evidence", "%v", err)
	}

	returnEvidence := applyPlannedEvidence(value, evidence)

	return &returnEvidence, nil
}

func applyPlannedEvidence(value intent.Intent, evidence project.FeatureEvidence) project.FeatureEvidence {
	if value.Operation == intent.OperationAddField && value.Domain.Field != nil {
		evidence.Basis = "planned"
		evidence.Fields = append(evidence.Fields, project.FeatureFieldEvidence{
			Name:     value.Domain.Field.Name,
			Type:     value.Domain.Field.Type,
			Label:    value.Domain.Field.Label,
			Required: value.Domain.Field.Required,
		})
	}

	if value.Operation == intent.OperationAddValidation && value.Domain.Validation != nil {
		evidence.Basis = "planned"
		evidence.Validations = append(evidence.Validations, project.FeatureValidationEvidence{
			Field: value.Domain.Validation.Field,
			Kind:  value.Domain.Validation.Kind,
			Value: value.Domain.Validation.Value,
			Scope: string(value.Domain.Validation.Scope),
		})

		if value.Domain.Validation.Kind == "required" {
			for index := range evidence.Fields {
				if evidence.Fields[index].Name == value.Domain.Validation.Field {
					evidence.Fields[index].Required = true
				}
			}
		}
	}

	return evidence
}

func plannedFeatureEvidence(value intent.Intent) project.FeatureEvidence {
	fields := make([]project.FeatureFieldEvidence, 0, len(value.Domain.Fields))
	validations := make([]project.FeatureValidationEvidence, 0)

	for _, field := range value.Domain.Fields {
		fields = append(fields, project.FeatureFieldEvidence{
			Name: field.Name, Type: field.Type, Label: field.Label, Required: field.Required,
		})

		if field.Required {
			validations = append(validations, project.FeatureValidationEvidence{
				Field: field.Name, Kind: "required", Scope: string(intent.ValidationDurable),
			})
		}
	}

	return project.FeatureEvidence{
		Basis:             "planned",
		Feature:           value.Feature,
		Entity:            value.Domain.Entity,
		Label:             value.Domain.Label,
		Route:             value.Domain.Route,
		Table:             pluralizeEvidenceName(value.Feature),
		Fields:            fields,
		Validations:       validations,
		Postgres:          containsString(value.Capabilities, "postgres_persistence"),
		HTMX:              containsString(value.Capabilities, "htmx_form"),
		RuntimeValidation: containsString(value.Capabilities, "runtime_validation"),
		Wired:             true,
		Tested:            true,
		Sources:           []project.FeatureEvidenceSource{},
	}
}

func pluralizeEvidenceName(value string) string {
	if strings.HasSuffix(value, "s") {
		return value + "es"
	}

	return value + "s"
}

func selectedBookOperation(archetype book.Archetype, operation intent.Operation) (book.Operation, bool) {
	for _, candidate := range archetype.Operations {
		if candidate.ID == string(operation) {
			return candidate, true
		}
	}

	return book.Operation{}, false
}

func expandOperations(selection book.Selection, selectedOperation book.Operation, includeDocumentation bool) ([]Operation, error) {
	candidates := make([]operationCandidate, 0)

	selectedArchetypeObligations := stringSet(selectedOperation.Obligations)
	if includeDocumentation && selectedOperation.ID != string(intent.OperationDocumentFeature) {
		documentationOperation, exists := selectedBookOperation(selection.Archetype, intent.OperationDocumentFeature)
		if !exists {
			return nil, planError("plan_documentation_operation_missing", "operations", "archetype %q does not define document_feature", selection.Archetype.ID)
		}

		for _, obligation := range documentationOperation.Obligations {
			selectedArchetypeObligations[obligation] = struct{}{}
		}
	}

	for _, obligation := range selection.Archetype.Obligations {
		if _, selected := selectedArchetypeObligations[obligation.ID]; !selected {
			continue
		}

		candidates = append(candidates, operationCandidate{
			operation: operationFromBook(OwnerArchetype, selection.Archetype.ID, obligation, selectedArchetypeObligations),
		})
	}

	if len(candidates) != len(selectedArchetypeObligations) {
		return nil, planError("plan_obligation_missing", "operations", "archetype operation references an unavailable obligation")
	}

	if selectedOperation.ID == string(intent.OperationDocumentFeature) {
		return orderOperations(candidates)
	}

	for _, capability := range selection.Capabilities {
		selected := make(map[string]struct{}, len(capability.Obligations))
		for _, obligation := range capability.Obligations {
			selected[obligation.ID] = struct{}{}
		}

		for _, obligation := range capability.Obligations {
			candidates = append(candidates, operationCandidate{
				operation: operationFromBook(OwnerCapability, capability.ID, obligation, selected),
			})
		}
	}

	return orderOperations(candidates)
}

func operationFromBook(kind OwnerKind, ownerID string, obligation book.Obligation, selected map[string]struct{}) Operation {
	dependencies := make([]string, 0, len(obligation.DependsOn))
	for _, dependency := range obligation.DependsOn {
		if _, exists := selected[dependency]; exists {
			dependencies = append(dependencies, operationID(kind, ownerID, dependency))
		}
	}

	return Operation{
		ID:         operationID(kind, ownerID, obligation.ID),
		Owner:      Owner{Kind: kind, ID: ownerID},
		Obligation: obligation.ID,
		Surfaces:   cloneStrings(obligation.Surfaces),
		Rules:      cloneStrings(obligation.Rules),
		DependsOn:  dependencies,
	}
}

func orderOperations(candidates []operationCandidate) ([]Operation, error) {
	result := make([]Operation, 0, len(candidates))
	emitted := make(map[string]struct{}, len(candidates))
	remaining := append([]operationCandidate(nil), candidates...)

	for len(remaining) > 0 {
		selected := -1

		for index, candidate := range remaining {
			if referencesAvailable(candidate.operation.DependsOn, emitted) {
				selected = index

				break
			}
		}

		if selected < 0 {
			return nil, planError("plan_obligation_cycle", "operations", "selected Book obligations cannot be ordered")
		}

		candidate := remaining[selected]
		result = append(result, candidate.operation)
		emitted[candidate.operation.ID] = struct{}{}

		remaining = append(remaining[:selected], remaining[selected+1:]...)
	}

	return result, nil
}

func referencesAvailable(references []string, available map[string]struct{}) bool {
	for _, reference := range references {
		if _, exists := available[reference]; !exists {
			return false
		}
	}

	return true
}

func operationID(kind OwnerKind, ownerID, obligationID string) string {
	return string(kind) + "." + ownerID + "." + obligationID
}

func affectedSurfaces(archetype book.Archetype, operations []Operation) []string {
	selected := make(map[string]struct{})

	for _, operation := range operations {
		for _, surface := range operation.Surfaces {
			selected[surface] = struct{}{}
		}
	}

	result := make([]string, 0, len(selected))
	for _, surface := range archetype.Surfaces {
		if _, exists := selected[surface]; exists {
			result = append(result, surface)
			delete(selected, surface)
		}
	}

	for _, operation := range operations {
		for _, surface := range operation.Surfaces {
			if _, exists := selected[surface]; !exists {
				continue
			}

			result = append(result, surface)
			delete(selected, surface)
		}
	}

	return result
}

func expandRules(rules []book.Rule) []RuleRef {
	result := make([]RuleRef, 0, len(rules))
	for _, rule := range rules {
		result = append(result, RuleRef{ID: rule.ID, Level: rule.Level})
	}

	return result
}

func selectedCapabilityIDs(capabilities []book.Capability) []string {
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		result = append(result, capability.ID)
	}

	return result
}

func expandPreconditions(value intent.Intent) []Precondition {
	return []Precondition{
		{ID: "project_fingerprint", Kind: PreconditionProjectFingerprint, Expected: value.ProjectFingerprint},
		{ID: "hatmax_version", Kind: PreconditionHatmaxVersion, Expected: value.HatmaxVersion},
		{ID: "book_version", Kind: PreconditionBookVersion, Expected: strconv.Itoa(value.BookVersion)},
	}
}

func expandDependencies(capabilities []book.Capability, include bool) []DependencyEffect {
	result := make([]DependencyEffect, 0)
	if !include {
		return result
	}

	for _, capability := range capabilities {
		for _, dependency := range capability.Dependencies {
			result = append(result, DependencyEffect{
				Capability: capability.ID,
				Kind:       dependency.Kind,
				Module:     dependency.Module,
				Purpose:    dependency.Purpose,
			})
		}
	}

	return result
}

func expandValidation(rules []book.Rule, operations []Operation, affected []string) []ValidationObligation {
	result := make([]ValidationObligation, 0, len(rules))
	for _, rule := range rules {
		selectedSurfaces := make(map[string]struct{})

		for _, operation := range operations {
			if !containsString(operation.Rules, rule.ID) {
				continue
			}

			for _, surface := range operation.Surfaces {
				selectedSurfaces[surface] = struct{}{}
			}
		}

		surfaces := make([]string, 0, len(selectedSurfaces))
		for _, surface := range affected {
			if _, exists := selectedSurfaces[surface]; exists {
				surfaces = append(surfaces, surface)
			}
		}

		result = append(result, ValidationObligation{
			Rule:        rule.ID,
			Level:       rule.Level,
			Diagnostics: cloneStrings(rule.Diagnostics),
			Surfaces:    surfaces,
		})
	}

	return result
}

func cloneObservations(values []project.Observation) []project.Observation {
	return append([]project.Observation{}, values...)
}

func cloneExceptions(values []intent.Exception) []intent.Exception {
	return append([]intent.Exception{}, values...)
}

func cloneDocumentationTargets(values []intent.DocumentationTarget) []intent.DocumentationTarget {
	return append([]intent.DocumentationTarget{}, values...)
}

func cloneFeatureEvidence(value project.FeatureEvidence) project.FeatureEvidence {
	value.Fields = append([]project.FeatureFieldEvidence{}, value.Fields...)
	value.Validations = append([]project.FeatureValidationEvidence{}, value.Validations...)
	value.Sources = append([]project.FeatureEvidenceSource{}, value.Sources...)

	return value
}

func cloneDomain(value intent.Domain) intent.Domain {
	result := value
	result.Fields = append([]intent.Field{}, value.Fields...)

	result.Rules = append([]intent.BusinessRule{}, value.Rules...)
	if value.Field != nil {
		field := *value.Field
		result.Field = &field
	}

	if value.Validation != nil {
		validation := *value.Validation
		result.Validation = &validation
	}

	return result
}

func cloneStrings(values []string) []string {
	return append([]string{}, values...)
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
