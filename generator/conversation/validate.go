// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package conversation

import (
	"regexp"
	"strconv"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
)

var (
	scopeKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	digestPattern   = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

// Validate checks one complete snapshot before persistence or use.
func Validate(value Conversation) error {
	if value.SchemaVersion != CurrentSchemaVersion {
		return modelError("conversation_schema_unsupported", "schema_version", "schema version %d is not supported", value.SchemaVersion)
	}

	if strings.TrimSpace(value.ID) == "" {
		return modelError("conversation_identity_invalid", "id", "conversation identifier is required")
	}

	err := validateScope(value.Scope)
	if err != nil {
		return err
	}

	if value.BookContract.BookVersion < 1 || value.BookContract.InterpreterVersion < 1 {
		return modelError("conversation_contract_invalid", "book_contract", "Book and interpreter versions must be positive")
	}

	if strings.TrimSpace(value.BackendIdentity.Adapter) == "" {
		return modelError("conversation_backend_invalid", "backend_identity.adapter", "backend adapter is required")
	}

	if !validStatus(value.Status) {
		return modelError("conversation_status_invalid", "status", "unknown conversation status %q", value.Status)
	}

	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return modelError("conversation_time_invalid", "updated_at", "valid ordered timestamps are required")
	}

	if len(value.Turns) > MaximumTurns {
		return modelError("state_limit_exceeded", "turns", "conversation exceeds %d turns", MaximumTurns)
	}

	if len(value.Operations) > MaximumOperations {
		return modelError("state_limit_exceeded", "operations", "conversation exceeds %d operations", MaximumOperations)
	}

	err = validateTurns(value)
	if err != nil {
		return err
	}

	return validateOperations(value)
}

func validateScope(scope Scope) error {
	if scope.Kind != ScopeProject && scope.Kind != ScopePreProject {
		return modelError("conversation_scope_invalid", "scope.kind", "unknown scope kind %q", scope.Kind)
	}

	if !scopeKeyPattern.MatchString(scope.Key) {
		return modelError("conversation_scope_invalid", "scope.key", "scope key must be a lowercase SHA-256 digest")
	}

	return nil
}

func validateTurns(value Conversation) error {
	var (
		previousID      uint64
		totalBytes      int
		diagnosticBytes int
	)

	operationIDs := make(map[string]struct{}, len(value.Operations))
	for _, operation := range value.Operations {
		operationIDs[operation.ID] = struct{}{}
	}

	for index, turn := range value.Turns {
		if turn.ID == 0 || turn.ID <= previousID {
			return modelError("conversation_turn_invalid", indexedField("turns", index)+".id", "turn identifiers must be positive and strictly increasing")
		}

		if turn.Role != RoleUser && turn.Role != RoleHatmax {
			return modelError("conversation_turn_invalid", indexedField("turns", index)+".role", "unknown turn role %q", turn.Role)
		}

		if !validTurnKind(turn.Kind) || strings.TrimSpace(turn.Content) == "" || turn.CreatedAt.IsZero() {
			return modelError("conversation_turn_invalid", indexedField("turns", index), "kind, content, and creation time are required")
		}

		if len(turn.Content) > MaximumTurnBytes {
			return modelError("state_limit_exceeded", indexedField("turns", index)+".content", "turn exceeds %d bytes", MaximumTurnBytes)
		}

		if turn.OperationID != "" {
			if _, exists := operationIDs[turn.OperationID]; !exists {
				return modelError("conversation_turn_invalid", indexedField("turns", index)+".operation_id", "referenced operation is not retained")
			}
		}

		previousID = turn.ID
		totalBytes += len(turn.Content)

		if turn.Kind == TurnDiagnostic {
			diagnosticBytes += len(turn.Content)
		}
	}

	if totalBytes > MaximumTurnContentBytes {
		return modelError("state_limit_exceeded", "turns", "turn content exceeds %d bytes", MaximumTurnContentBytes)
	}

	if diagnosticBytes > MaximumDiagnosticBytes {
		return modelError("state_limit_exceeded", "turns", "diagnostic content exceeds %d bytes", MaximumDiagnosticBytes)
	}

	if value.NextTurnID <= previousID {
		return modelError("conversation_turn_invalid", "next_turn_id", "next turn identifier must exceed retained turns")
	}

	return nil
}

func validateOperations(value Conversation) error {
	seen := make(map[string]struct{}, len(value.Operations))
	for index, operation := range value.Operations {
		field := indexedField("operations", index)
		if strings.TrimSpace(operation.ID) == "" || operation.ConversationID != value.ID {
			return modelError("conversation_operation_invalid", field, "operation and owning conversation identifiers are required")
		}

		if _, exists := seen[operation.ID]; exists {
			return modelError("conversation_operation_invalid", field+".id", "operation identifier %q is duplicated", operation.ID)
		}

		if !validOperationStatus(operation.Status) || strings.TrimSpace(operation.RequestSummary) == "" {
			return modelError("conversation_operation_invalid", field, "status and request summary are required")
		}

		if len(operation.RequestSummary) > MaximumTurnBytes || len(operation.ResultSummary) > MaximumResultSummaryBytes {
			return modelError("state_limit_exceeded", field, "operation summary exceeds its retention limit")
		}

		if len(operation.Decisions) > MaximumOperationDecisions {
			return modelError("state_limit_exceeded", field+".decisions", "operation exceeds %d decisions", MaximumOperationDecisions)
		}

		if len(operation.Pending)+len(operation.Decisions) > MaximumOperationDecisions {
			return modelError("state_limit_exceeded", field+".pending_clarifications", "operation exceeds %d clarification exchanges", MaximumOperationDecisions)
		}

		pendingFields := make(map[string]struct{}, len(operation.Pending))
		for pendingIndex, clarification := range operation.Pending {
			pendingField := indexedField(field+".pending_clarifications", pendingIndex)
			if strings.TrimSpace(clarification.Field) == "" || strings.TrimSpace(clarification.Question) == "" {
				return modelError("conversation_operation_invalid", pendingField, "field and question are required")
			}

			if len(clarification.Field) > MaximumDecisionBytes || len(clarification.Question) > MaximumDecisionBytes {
				return modelError("state_limit_exceeded", pendingField, "clarification exceeds its retention limit")
			}

			if _, exists := pendingFields[clarification.Field]; exists {
				return modelError("conversation_operation_invalid", pendingField+".field", "clarification field %q is duplicated", clarification.Field)
			}

			pendingFields[clarification.Field] = struct{}{}
		}

		decisionFields := make(map[string]struct{}, len(operation.Decisions))
		for decisionIndex, decision := range operation.Decisions {
			decisionField := indexedField(field+".decisions", decisionIndex)
			if strings.TrimSpace(decision.Field) == "" || strings.TrimSpace(decision.Question) == "" || strings.TrimSpace(decision.Answer) == "" {
				return modelError("conversation_operation_invalid", decisionField, "field, question, and answer are required")
			}

			if len(decision.Field) > MaximumDecisionBytes || len(decision.Question) > MaximumDecisionBytes || len(decision.Answer) > MaximumDecisionBytes {
				return modelError("state_limit_exceeded", decisionField, "decision exceeds its retention limit")
			}

			if _, exists := decisionFields[decision.Field]; exists {
				return modelError("conversation_operation_invalid", decisionField+".field", "decision field %q is duplicated", decision.Field)
			}

			if _, exists := pendingFields[decision.Field]; exists {
				return modelError("conversation_operation_invalid", decisionField+".field", "decision field %q is still pending", decision.Field)
			}

			decisionFields[decision.Field] = struct{}{}
		}

		if operation.CreatedAt.IsZero() || operation.UpdatedAt.IsZero() || operation.UpdatedAt.Before(operation.CreatedAt) {
			return modelError("conversation_operation_invalid", field+".updated_at", "valid ordered timestamps are required")
		}

		if operation.Intent != nil {
			if operation.IntentContract < 1 || operation.IntentContract != operation.Intent.SchemaVersion {
				return modelError("conversation_operation_invalid", field+".intent_contract", "intent contract must match the retained intent")
			}

			err := intent.ValidateSchema(*operation.Intent)
			if err != nil {
				return modelError("conversation_operation_invalid", field+".intent", "%v", err)
			}
		}

		if operation.PlanDigest != "" && !digestPattern.MatchString(operation.PlanDigest) {
			return modelError("conversation_operation_invalid", field+".plan_digest", "plan digest must use sha256:<hex>")
		}

		if operation.SourceFingerprint != "" && !digestPattern.MatchString(operation.SourceFingerprint) {
			return modelError("conversation_operation_invalid", field+".source_fingerprint", "source fingerprint must use sha256:<hex>")
		}

		seen[operation.ID] = struct{}{}
	}

	return nil
}

func validStatus(value Status) bool {
	return value == StatusActive || value == StatusReset || value == StatusIncompatible || value == StatusArchived
}

func validTurnKind(value TurnKind) bool {
	return value == TurnDialogue || value == TurnGoal || value == TurnClarification || value == TurnDecision || value == TurnResult || value == TurnDiagnostic
}

func validOperationStatus(value OperationStatus) bool {
	return value == OperationCandidate || value == OperationClarifying || value == OperationPlanned || value == OperationCancelled || value == OperationStale || value == OperationFailed || value == OperationCompleted
}

func indexedField(field string, index int) string {
	return field + "[" + strconv.Itoa(index) + "]"
}
