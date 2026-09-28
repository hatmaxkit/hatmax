package execute

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// Prepare converts one sealed plan and matching project inventory into a
// deterministic execution manifest without modifying the project.
func Prepare(value plan.Plan, inventory project.Inventory, selectedBook *book.Book) (Manifest, error) {
	err := plan.VerifyDigest(value)
	if err != nil {
		return Manifest{}, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	err = validateBookPlan(value, selectedBook)
	if err != nil {
		return Manifest{}, err
	}

	err = validateFreshness(value, inventory)
	if err != nil {
		return Manifest{}, err
	}

	specs, err := canonicalTargetSpecs(value, inventory)
	if err != nil {
		return Manifest{}, err
	}

	edits, err := prepareEdits(value, inventory, specs)
	if err != nil {
		return Manifest{}, err
	}

	commands, err := prepareCommands(value, inventory.Commands)
	if err != nil {
		return Manifest{}, err
	}

	result := Manifest{
		SchemaVersion:      CurrentSchemaVersion,
		PlanDigest:         value.Digest,
		ProjectFingerprint: value.ProjectFingerprint,
		Intent:             value.Intent,
		Feature:            value.Feature,
		AllowedSurfaces:    append([]string{}, value.AllowedEffects.Surfaces...),
		Edits:              edits,
		Commands:           commands,
	}

	result, err = Seal(result)
	if err != nil {
		return Manifest{}, fmt.Errorf("seal execution manifest: %w", err)
	}

	return result, nil
}

func validateBookPlan(value plan.Plan, selectedBook *book.Book) error {
	if selectedBook == nil {
		return executionError("execution_book_missing", "book_version", "a validated Hatmax Book is required")
	}

	if selectedBook.Manifest().BookVersion != value.BookVersion {
		return executionError("execution_book_version_mismatch", "book_version", "plan and Book versions must match")
	}

	selection, err := selectedBook.Select(value.Archetype, value.Capabilities)
	if err != nil {
		return executionError("execution_book_selection_invalid", "selection", "%v", err)
	}

	expected, err := selectedOperations(value, selection)
	if err != nil {
		return err
	}

	if len(expected) != len(value.Operations) {
		return executionError("execution_obligation_mismatch", "operations", "plan operations do not match selected Book obligations")
	}

	for index := range expected {
		if !sameOperation(expected[index], value.Operations[index]) {
			return executionError("execution_obligation_mismatch", indexedPath("operations", index), "plan operation does not match its Book obligation")
		}
	}

	return nil
}

func selectedOperations(value plan.Plan, selection book.Selection) ([]plan.Operation, error) {
	selected, exists := selectedArchetypeOperation(selection.Archetype, string(value.Intent))
	if !exists {
		return nil, executionError("execution_obligation_mismatch", "intent", "Book archetype does not define intent %q", value.Intent)
	}

	result := make([]plan.Operation, 0)

	for _, obligationID := range selected.Obligations {
		obligation, found := bookObligation(selection.Archetype.Obligations, obligationID)
		if !found {
			return nil, executionError("execution_obligation_mismatch", "operations", "Book archetype obligation %q is missing", obligationID)
		}

		result = append(result, plan.Operation{
			ID:         operationID(plan.OwnerArchetype, selection.Archetype.ID, obligation.ID),
			Owner:      plan.Owner{Kind: plan.OwnerArchetype, ID: selection.Archetype.ID},
			Obligation: obligation.ID,
			Surfaces:   append([]string{}, obligation.Surfaces...),
			Rules:      append([]string{}, obligation.Rules...),
			DependsOn:  ownerDependencies(plan.OwnerArchetype, selection.Archetype.ID, obligation, selected.Obligations),
		})
	}

	for _, capability := range selection.Capabilities {
		available := make([]string, 0, len(capability.Obligations))
		for _, obligation := range capability.Obligations {
			available = append(available, obligation.ID)
		}

		for _, obligation := range capability.Obligations {
			result = append(result, plan.Operation{
				ID:         operationID(plan.OwnerCapability, capability.ID, obligation.ID),
				Owner:      plan.Owner{Kind: plan.OwnerCapability, ID: capability.ID},
				Obligation: obligation.ID,
				Surfaces:   append([]string{}, obligation.Surfaces...),
				Rules:      append([]string{}, obligation.Rules...),
				DependsOn:  ownerDependencies(plan.OwnerCapability, capability.ID, obligation, available),
			})
		}
	}

	return result, nil
}

func selectedArchetypeOperation(value book.Archetype, id string) (book.Operation, bool) {
	for _, operation := range value.Operations {
		if operation.ID == id {
			return operation, true
		}
	}

	return book.Operation{}, false
}

func bookObligation(values []book.Obligation, id string) (book.Obligation, bool) {
	for _, obligation := range values {
		if obligation.ID == id {
			return obligation, true
		}
	}

	return book.Obligation{}, false
}

func ownerDependencies(kind plan.OwnerKind, owner string, obligation book.Obligation, available []string) []string {
	selected := stringSet(available)

	result := make([]string, 0, len(obligation.DependsOn))
	for _, dependency := range obligation.DependsOn {
		if _, exists := selected[dependency]; exists {
			result = append(result, operationID(kind, owner, dependency))
		}
	}

	return result
}

func operationID(kind plan.OwnerKind, owner, obligation string) string {
	return string(kind) + "." + owner + "." + obligation
}

func sameOperation(left, right plan.Operation) bool {
	return left.ID == right.ID && left.Owner == right.Owner && left.Obligation == right.Obligation &&
		sameStrings(left.Surfaces, right.Surfaces) && sameStrings(left.Rules, right.Rules) && sameStrings(left.DependsOn, right.DependsOn)
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

func validateFreshness(value plan.Plan, inventory project.Inventory) error {
	current, err := inventory.Fingerprint(project.FingerprintRequest{
		BookVersion:          value.BookVersion,
		SelectedPaths:        value.FingerprintInputs.SelectedPaths,
		SelectedDependencies: value.FingerprintInputs.SelectedDependencies,
		PlannedSurfaces:      value.FingerprintInputs.PlannedSurfaces,
	})
	if err != nil {
		return executionError("execution_fingerprint_failed", "project_fingerprint", "%v", err)
	}

	freshness, err := plan.CheckFingerprint(value, current)
	if err != nil {
		return executionError("execution_plan_invalid", "plan", "%v", err)
	}

	if freshness.Stale {
		return executionError("execution_plan_stale", "project_fingerprint", "relevant project state changed after planning")
	}

	return nil
}

func prepareEdits(value plan.Plan, inventory project.Inventory, specs []targetSpec) ([]Edit, error) {
	targets := make([]string, 0, len(specs))
	for _, spec := range specs {
		targets = append(targets, spec.target)
	}

	overlap, err := inventory.DirtyOverlap(targets)
	if err != nil {
		return nil, executionError("execution_target_invalid", "edits", "%v", err)
	}

	if len(overlap) > 0 {
		return nil, executionError("execution_dirty_overlap", overlap[0].Path, "target overlaps an uncommitted project change")
	}

	operations := make(map[string]plan.Operation, len(value.Operations))
	for _, operation := range value.Operations {
		operations[operation.ID] = operation
	}

	edits := make([]Edit, 0, len(specs))
	for _, spec := range specs {
		edit, prepareErr := prepareEdit(value, inventory, operations, spec)
		if prepareErr != nil {
			return nil, prepareErr
		}

		edits = append(edits, edit)
	}

	applyEditDependencies(edits, operations)

	return edits, nil
}

func prepareEdit(value plan.Plan, inventory project.Inventory, operations map[string]plan.Operation, spec targetSpec) (Edit, error) {
	file, exists := inventory.File(spec.target)
	if file.Generated {
		return Edit{}, executionError("execution_target_generated", spec.target, "generated files cannot be edited directly")
	}

	if protected, reason := protectedTarget(inventory.Rules.ProtectedPaths, spec.target); protected {
		return Edit{}, executionError("execution_target_protected", spec.target, "%s", reason)
	}

	if spec.create && exists {
		return Edit{}, executionError("execution_target_exists", spec.target, "create target already exists")
	}

	if !spec.create && !exists {
		return Edit{}, executionError("execution_target_missing", spec.target, "update target does not exist")
	}

	obligations := make([]Obligation, 0, len(spec.operations))
	for _, operationID := range spec.operations {
		operation, selected := operations[operationID]
		if !selected {
			continue
		}

		obligations = append(obligations, Obligation{
			Operation: operation.ID,
			Owner:     operation.Owner,
			Rules:     append([]string{}, operation.Rules...),
		})
	}

	if len(obligations) == 0 {
		return Edit{}, executionError("execution_obligation_missing", spec.id, "target has no selected Book obligation")
	}

	preconditions := []Condition{{Kind: ConditionPathPresent}}
	if spec.create {
		preconditions[0] = Condition{Kind: ConditionPathAbsent}
	} else if digest := observationDigest(value.ExpectedObservations, spec.target); digest != "" {
		preconditions = append(preconditions, Condition{Kind: ConditionPathDigest, Value: digest})
	}

	if spec.requiresCommand != "" {
		preconditions = append(preconditions, Condition{Kind: ConditionCommandAvailable, Value: spec.requiresCommand})
	}

	postconditions := []Condition{{Kind: ConditionPathPresent}}
	if path.Ext(spec.target) == ".go" {
		postconditions = append(postconditions, Condition{Kind: ConditionGoParses})
	}

	return Edit{
		ID:             spec.id,
		Kind:           spec.kind,
		Surface:        spec.surface,
		Target:         spec.target,
		Recipe:         spec.recipe,
		Obligations:    obligations,
		DependsOn:      []string{},
		Preconditions:  preconditions,
		Postconditions: postconditions,
		Slots:          implementationSlots(value, spec.surface),
	}, nil
}

func applyEditDependencies(edits []Edit, operations map[string]plan.Operation) {
	lastByOperation := make(map[string]string)
	for index := range edits {
		dependencies := make(map[string]struct{})

		for _, obligation := range edits[index].Obligations {
			for _, dependency := range operations[obligation.Operation].DependsOn {
				if editID := lastByOperation[dependency]; editID != "" {
					dependencies[editID] = struct{}{}
				}
			}
		}

		for earlier := 0; earlier < index; earlier++ {
			if _, exists := dependencies[edits[earlier].ID]; exists {
				edits[index].DependsOn = append(edits[index].DependsOn, edits[earlier].ID)
			}
		}

		for _, obligation := range edits[index].Obligations {
			lastByOperation[obligation.Operation] = edits[index].ID
		}
	}
}

func observationDigest(values []project.Observation, target string) string {
	for _, observation := range values {
		if observation.Path == target && observation.Digest != "missing" && observation.Digest != "directory" {
			return observation.Digest
		}
	}

	return ""
}

func protectedTarget(values []project.ProtectedPath, target string) (bool, string) {
	for _, value := range values {
		if pathsOverlap(value.Path, target) {
			return true, value.Reason
		}
	}

	return false, ""
}

func pathsOverlap(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func implementationSlots(value plan.Plan, surface string) []ImplementationSlot {
	result := []ImplementationSlot{{Name: "feature", Source: "plan.feature"}}

	if value.Domain.Entity != "" && (surface == "model" || surface == "store" || surface == "service" || surface == "handler" || surface == "tests") {
		result = append(result, ImplementationSlot{Name: "domain_entity", Source: "plan.domain.entity"})
	}

	if len(value.Domain.Fields) > 0 || value.Domain.Field != nil {
		result = append(result, ImplementationSlot{Name: "domain_fields", Source: "plan.domain"})
	}

	if value.Domain.Route != "" && (surface == "handler" || surface == "templates" || surface == "wiring" || surface == "tests") {
		result = append(result, ImplementationSlot{Name: "domain_route", Source: "plan.domain.route"})
	}

	if value.Domain.Validation != nil {
		result = append(result, ImplementationSlot{Name: "domain_validation", Source: "plan.domain.validation"})
	}

	if len(value.Domain.Rules) > 0 && (surface == "model" || surface == "service" || surface == "tests") {
		result = append(result, ImplementationSlot{Name: "business_rules", Source: "plan.domain.rules"})
	}

	return result
}

func prepareCommands(value plan.Plan, commands []project.Command) ([]Command, error) {
	result := make([]Command, 0, len(commands))
	foundSQLC := false

	for _, command := range commands {
		if command.Kind == project.CommandGeneration && len(command.Args) >= 2 && command.Args[0] == "sqlc" && command.Args[1] == "generate" {
			foundSQLC = true
		}

		result = append(result, Command{
			Kind:             command.Kind,
			Name:             string(command.Kind) + "." + command.Name,
			Args:             append([]string{}, command.Args...),
			WorkingDirectory: ".",
			Source:           command.Source,
		})
	}

	if containsString(value.Capabilities, "postgres_persistence") && !foundSQLC {
		return nil, executionError("execution_command_missing", "commands", "postgres_persistence requires a repository-owned sqlc generate command")
	}

	sort.Slice(result, func(left, right int) bool {
		if result[left].Kind != result[right].Kind {
			return result[left].Kind < result[right].Kind
		}

		return result[left].Name < result[right].Name
	})

	return result, nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
