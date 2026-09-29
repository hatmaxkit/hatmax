package conversation

import (
	"sort"
	"time"

	"hatmax.adrianpk.com/generator/intent"
)

// New creates one empty active conversation.
func New(
	id string,
	scope Scope,
	contract BookContract,
	backend BackendIdentity,
	now time.Time,
) (Conversation, error) {
	value := Conversation{
		SchemaVersion:   CurrentSchemaVersion,
		ID:              id,
		Scope:           scope,
		BookContract:    contract,
		BackendIdentity: backend,
		Status:          StatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
		NextTurnID:      1,
		Turns:           []Turn{},
		Operations:      []ProposedOperation{},
	}

	err := Validate(value)
	if err != nil {
		return Conversation{}, err
	}

	return value, nil
}

// AppendTurn adds one visible turn with a monotonic identifier.
func (c *Conversation) AppendTurn(role Role, kind TurnKind, content, operationID string, now time.Time) error {
	turn := Turn{
		ID:          c.NextTurnID,
		Role:        role,
		Kind:        kind,
		Content:     content,
		CreatedAt:   now,
		OperationID: operationID,
	}

	next := clone(*c)
	next.NextTurnID++
	next.UpdatedAt = now
	next.Turns = append(next.Turns, turn)

	pruned, err := Prune(next)
	if err != nil {
		return err
	}

	*c = pruned

	return nil
}

// AddOperation adds one candidate operation without approval authority.
func (c *Conversation) AddOperation(id, requestSummary string, now time.Time) error {
	operation := ProposedOperation{
		ID:             id,
		ConversationID: c.ID,
		Status:         OperationCandidate,
		RequestSummary: requestSummary,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	next := clone(*c)
	next.UpdatedAt = now
	next.Operations = append(next.Operations, operation)

	pruned, err := Prune(next)
	if err != nil {
		return err
	}

	*c = pruned

	return nil
}

// TransitionOperation applies one valid status transition and bounded summary.
func (c *Conversation) TransitionOperation(
	id string,
	status OperationStatus,
	resultSummary string,
	now time.Time,
) error {
	next := clone(*c)

	index := operationIndex(next.Operations, id)
	if index < 0 {
		return modelError("conversation_operation_missing", "operation.id", "operation %q is not retained", id)
	}

	operation := next.Operations[index]
	if !allowedTransition(operation.Status, status) {
		return modelError("conversation_operation_transition_invalid", "operation.status", "cannot transition %q to %q", operation.Status, status)
	}

	operation.Status = status
	operation.ResultSummary = resultSummary
	operation.UpdatedAt = now
	next.Operations[index] = operation
	next.UpdatedAt = now

	err := Validate(next)
	if err != nil {
		return err
	}

	*c = next

	return nil
}

// SetPlannedOperation retains a typed intent and sealed plan identity without
// retaining or restoring approval.
func (c *Conversation) SetPlannedOperation(
	id string,
	intentContract int,
	value intent.Intent,
	planDigest string,
	sourceFingerprint string,
	now time.Time,
) error {
	next := clone(*c)

	index := operationIndex(next.Operations, id)
	if index < 0 {
		return modelError("conversation_operation_missing", "operation.id", "operation %q is not retained", id)
	}

	operation := next.Operations[index]
	if !allowedTransition(operation.Status, OperationPlanned) {
		return modelError("conversation_operation_transition_invalid", "operation.status", "cannot transition %q to %q", operation.Status, OperationPlanned)
	}

	operation.Status = OperationPlanned
	operation.IntentContract = intentContract
	operation.Intent = cloneIntent(value)
	operation.PlanDigest = planDigest
	operation.SourceFingerprint = sourceFingerprint
	operation.UpdatedAt = now
	next.Operations[index] = operation
	next.UpdatedAt = now

	err := Validate(next)
	if err != nil {
		return err
	}

	*c = next

	return nil
}

// ReconcileResume invalidates planned operations whose contract or source
// fingerprint is no longer current. It never recreates approval.
func (c *Conversation) ReconcileResume(contract BookContract, currentFingerprint string, now time.Time) {
	next := clone(*c)
	if next.BookContract != contract {
		next.Status = StatusIncompatible
		next.BackendThreadID = ""
		next.UpdatedAt = now
		*c = next

		return
	}

	for index, operation := range next.Operations {
		if operation.Status != OperationPlanned {
			continue
		}

		if operation.SourceFingerprint != currentFingerprint {
			operation.Status = OperationStale
			operation.ResultSummary = "Stored plan requires fresh source inspection and replanning."
			operation.UpdatedAt = now
			next.Operations[index] = operation
		}
	}

	next.UpdatedAt = now
	*c = next
}

// Reset archives the current identity as reset and returns one new active
// conversation for the same scope and contracts.
func (c Conversation) Reset(id string, now time.Time) (Conversation, Conversation, error) {
	previous := clone(c)
	previous.Status = StatusReset
	previous.BackendThreadID = ""
	previous.UpdatedAt = now

	next, err := New(id, c.Scope, c.BookContract, c.BackendIdentity, now)
	if err != nil {
		return Conversation{}, Conversation{}, err
	}

	err = Validate(previous)
	if err != nil {
		return Conversation{}, Conversation{}, err
	}

	return previous, next, nil
}

// Rebind moves a pre-project conversation to an exact proposed target or the
// created project scope and clears backend thread identity. Planned operations
// become stale.
func (c *Conversation) Rebind(scope Scope, now time.Time) error {
	if c.Scope.Kind != ScopePreProject || (scope.Kind != ScopePreProject && scope.Kind != ScopeProject) {
		return modelError("conversation_rebind_invalid", "scope.kind", "rebind requires a pre_project source and pre_project or project target")
	}

	err := validateScope(scope)
	if err != nil {
		return err
	}

	next := clone(*c)
	next.Scope = scope
	next.BackendThreadID = ""
	next.UpdatedAt = now

	for index, operation := range next.Operations {
		if operation.Status != OperationPlanned {
			continue
		}

		operation.Status = OperationStale
		operation.ResultSummary = "Stored plan requires replanning after scope rebinding."
		operation.UpdatedAt = now
		next.Operations[index] = operation
	}

	err = Validate(next)
	if err != nil {
		return err
	}

	*c = next

	return nil
}

// RecentTurns returns the newest stable-order replay window.
func (c Conversation) RecentTurns(maxTurns, maxBytes int) []Turn {
	if maxTurns <= 0 || maxBytes <= 0 {
		return []Turn{}
	}

	start := len(c.Turns)
	total := 0

	for start > 0 && len(c.Turns)-start < maxTurns {
		candidate := c.Turns[start-1]
		if total+len(candidate.Content) > maxBytes {
			break
		}

		total += len(candidate.Content)
		start--
	}

	return append([]Turn{}, c.Turns[start:]...)
}

// NonTerminalOperations returns candidate, clarifying, and planned operations.
func (c Conversation) NonTerminalOperations() []ProposedOperation {
	result := make([]ProposedOperation, 0)

	for _, operation := range c.Operations {
		if operation.Status == OperationCandidate || operation.Status == OperationClarifying || operation.Status == OperationPlanned {
			result = append(result, cloneOperation(operation))
		}
	}

	return result
}

// Prune applies per-conversation retention while preserving non-terminal
// operations, their linked turns, and the newest terminal result.
func Prune(value Conversation) (Conversation, error) {
	result := clone(value)
	protectedOperations := make(map[string]struct{})
	latestTerminal := -1

	for index, operation := range result.Operations {
		if operation.Status == OperationCandidate || operation.Status == OperationClarifying || operation.Status == OperationPlanned {
			protectedOperations[operation.ID] = struct{}{}
		} else if latestTerminal < 0 || operation.UpdatedAt.After(result.Operations[latestTerminal].UpdatedAt) {
			latestTerminal = index
		}
	}

	if latestTerminal >= 0 {
		protectedOperations[result.Operations[latestTerminal].ID] = struct{}{}
	}

	for len(result.Operations) > MaximumOperations {
		removed := false

		for index, operation := range result.Operations {
			if _, protected := protectedOperations[operation.ID]; protected {
				continue
			}

			result.Operations = append(result.Operations[:index], result.Operations[index+1:]...)
			removed = true

			break
		}

		if !removed {
			return Conversation{}, modelError("state_limit_exceeded", "operations", "protected operations exceed retention limits")
		}
	}

	retainedOperationIDs := make(map[string]struct{}, len(result.Operations))
	for _, operation := range result.Operations {
		retainedOperationIDs[operation.ID] = struct{}{}
	}

	retainedTurns := result.Turns[:0]
	for _, turn := range result.Turns {
		if turn.OperationID != "" {
			if _, retained := retainedOperationIDs[turn.OperationID]; !retained {
				continue
			}
		}

		retainedTurns = append(retainedTurns, turn)
	}

	result.Turns = retainedTurns

	turnBytes := 0
	diagnosticBytes := 0

	for _, turn := range result.Turns {
		turnBytes += len(turn.Content)
		if turn.Kind == TurnDiagnostic {
			diagnosticBytes += len(turn.Content)
		}
	}

	for len(result.Turns) > MaximumTurns || turnBytes > MaximumTurnContentBytes || diagnosticBytes > MaximumDiagnosticBytes {
		removed := false

		for index, turn := range result.Turns {
			_, protected := protectedOperations[turn.OperationID]
			if turn.OperationID != "" && protected {
				continue
			}

			turnBytes -= len(turn.Content)
			if turn.Kind == TurnDiagnostic {
				diagnosticBytes -= len(turn.Content)
			}

			result.Turns = append(result.Turns[:index], result.Turns[index+1:]...)
			removed = true

			break
		}

		if !removed {
			return Conversation{}, modelError("state_limit_exceeded", "turns", "protected turns exceed retention limits")
		}
	}

	sort.SliceStable(result.Operations, func(left, right int) bool {
		return result.Operations[left].CreatedAt.Before(result.Operations[right].CreatedAt)
	})

	err := Validate(result)
	if err != nil {
		return Conversation{}, err
	}

	return result, nil
}

func allowedTransition(from, to OperationStatus) bool {
	if from == to {
		return true
	}

	switch from {
	case OperationCandidate:
		return to == OperationClarifying || to == OperationPlanned || to == OperationCancelled || to == OperationFailed
	case OperationClarifying:
		return to == OperationCandidate || to == OperationPlanned || to == OperationCancelled || to == OperationFailed || to == OperationStale
	case OperationPlanned:
		return to == OperationCancelled || to == OperationStale || to == OperationFailed || to == OperationCompleted
	default:
		return false
	}
}

func operationIndex(values []ProposedOperation, id string) int {
	for index, operation := range values {
		if operation.ID == id {
			return index
		}
	}

	return -1
}

func clone(value Conversation) Conversation {
	result := value
	result.Turns = append([]Turn{}, value.Turns...)
	result.Operations = make([]ProposedOperation, len(value.Operations))

	for index, operation := range value.Operations {
		result.Operations[index] = cloneOperation(operation)
	}

	return result
}

// Clone returns a mutation-isolated copy of one conversation snapshot.
func Clone(value Conversation) Conversation {
	return clone(value)
}

func cloneOperation(value ProposedOperation) ProposedOperation {
	result := value
	if value.Intent != nil {
		result.Intent = cloneIntent(*value.Intent)
	}

	return result
}

func cloneIntent(value intent.Intent) *intent.Intent {
	result := value
	result.Capabilities = append([]string{}, value.Capabilities...)
	result.DocumentationTargets = append([]intent.DocumentationTarget{}, value.DocumentationTargets...)
	result.Exceptions = append([]intent.Exception{}, value.Exceptions...)
	result.Domain.Fields = append([]intent.Field{}, value.Domain.Fields...)
	result.Domain.Rules = append([]intent.BusinessRule{}, value.Domain.Rules...)
	result.InitialFeatures = make([]intent.InitialFeature, len(value.InitialFeatures))

	for index, feature := range value.InitialFeatures {
		result.InitialFeatures[index] = feature
		result.InitialFeatures[index].Domain.Fields = append([]intent.Field{}, feature.Domain.Fields...)
		result.InitialFeatures[index].Domain.Rules = append([]intent.BusinessRule{}, feature.Domain.Rules...)
	}

	if value.Application != nil {
		application := *value.Application
		result.Application = &application
	}

	if value.Target != nil {
		target := *value.Target
		result.Target = &target
	}

	if value.Domain.Field != nil {
		field := *value.Domain.Field
		result.Domain.Field = &field
	}

	if value.Domain.Validation != nil {
		validation := *value.Domain.Validation
		result.Domain.Validation = &validation
	}

	return &result
}
