// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"bytes"
	"go/parser"
	"go/token"
)

// Stage validates and records one structured mutation without changing the
// project. Every manifest edit must be staged before commit.
func (w *Workspace) Stage(mutation Mutation) (Change, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	edit, exists := manifestEdit(w.manifest.Edits, mutation.EditID)
	if !exists {
		return Change{}, executionError("execution_edit_undeclared", mutation.EditID, "mutation does not identify a manifest edit")
	}

	if _, exists = w.staged[mutation.EditID]; exists {
		return Change{}, executionError("execution_edit_already_staged", mutation.EditID, "edit is already staged")
	}

	if int64(len(mutation.Content)) > MaximumStagedFileSize {
		return Change{}, executionError("execution_limit_exceeded", edit.Target, "staged content exceeds the %d-byte file limit", MaximumStagedFileSize)
	}

	snapshot := w.snapshots[edit.ID]

	content, status, err := applyMutation(edit, snapshot, mutation)
	if err != nil {
		return Change{}, err
	}

	if int64(len(content)) > MaximumStagedFileSize {
		return Change{}, executionError("execution_limit_exceeded", edit.Target, "result exceeds the %d-byte file limit", MaximumStagedFileSize)
	}

	if w.stagedBytes+int64(len(content)) > MaximumStagedBytes {
		return Change{}, executionError("execution_limit_exceeded", "edits", "staged content exceeds the %d-byte workspace limit", MaximumStagedBytes)
	}

	err = validatePostconditions(edit, content)
	if err != nil {
		return Change{}, err
	}

	w.staged[edit.ID] = stagedMutation{
		edit:    edit,
		kind:    mutation.Kind,
		content: append([]byte{}, content...),
		status:  status,
	}
	w.stagedBytes += int64(len(content))

	return stagedChange(edit, mutation.Kind, snapshot, content, status), nil
}

// Changes returns target-ordered metadata for all staged mutations.
func (w *Workspace) Changes() []Change {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]Change, 0, len(w.staged))
	for _, edit := range w.manifest.Edits {
		staged, exists := w.staged[edit.ID]
		if !exists {
			continue
		}

		result = append(result, stagedChange(edit, staged.kind, w.snapshots[edit.ID], staged.content, staged.status))
	}

	return result
}

func applyMutation(edit Edit, snapshot targetSnapshot, mutation Mutation) ([]byte, ChangeStatus, error) {
	switch mutation.Kind {
	case MutationCreate:
		if edit.Kind != EditCreateFile || snapshot.exists {
			return nil, "", executionError("execution_mutation_invalid", edit.Target, "create requires an absent create-file target")
		}

		if len(mutation.Anchor) != 0 || mutation.Position != "" {
			return nil, "", executionError("execution_mutation_invalid", edit.Target, "create does not accept an insertion anchor")
		}

		return append([]byte{}, mutation.Content...), ChangeStaged, nil

	case MutationReplace:
		if edit.Kind == EditCreateFile || !snapshot.exists {
			return nil, "", executionError("execution_mutation_invalid", edit.Target, "replace requires an existing update target")
		}

		if len(mutation.Anchor) != 0 || mutation.Position != "" {
			return nil, "", executionError("execution_mutation_invalid", edit.Target, "replace does not accept an insertion anchor")
		}

		status := ChangeStaged
		if bytes.Equal(snapshot.content, mutation.Content) {
			status = ChangeAlreadySatisfied
		}

		return append([]byte{}, mutation.Content...), status, nil

	case MutationInsert:
		return applyInsertion(edit, snapshot, mutation)

	default:
		return nil, "", executionError("execution_mutation_invalid", edit.Target, "unknown mutation kind %q", mutation.Kind)
	}
}

func applyInsertion(edit Edit, snapshot targetSnapshot, mutation Mutation) ([]byte, ChangeStatus, error) {
	if edit.Kind == EditCreateFile || !snapshot.exists {
		return nil, "", executionError("execution_mutation_invalid", edit.Target, "insert requires an existing update target")
	}

	if len(mutation.Anchor) == 0 || len(mutation.Content) == 0 {
		return nil, "", executionError("execution_mutation_invalid", edit.Target, "insert requires non-empty anchor and content")
	}

	if mutation.Position != InsertBefore && mutation.Position != InsertAfter {
		return nil, "", executionError("execution_mutation_invalid", edit.Target, "insert position must be before or after")
	}

	if bytes.Count(snapshot.content, mutation.Anchor) != 1 {
		return nil, "", executionError("execution_semantic_conflict", edit.Target, "insertion anchor must occur exactly once")
	}

	anchorIndex := bytes.Index(snapshot.content, mutation.Anchor)

	insertAt := anchorIndex
	if mutation.Position == InsertAfter {
		insertAt += len(mutation.Anchor)
	}

	if insertionSatisfied(snapshot.content, mutation.Content, insertAt, mutation.Position) {
		return append([]byte{}, snapshot.content...), ChangeAlreadySatisfied, nil
	}

	if bytes.Contains(snapshot.content, mutation.Content) {
		return nil, "", executionError("execution_semantic_conflict", edit.Target, "insertion content already exists outside the selected anchor")
	}

	result := make([]byte, 0, len(snapshot.content)+len(mutation.Content))
	result = append(result, snapshot.content[:insertAt]...)
	result = append(result, mutation.Content...)
	result = append(result, snapshot.content[insertAt:]...)

	return result, ChangeStaged, nil
}

func insertionSatisfied(existing, content []byte, insertAt int, position InsertionPosition) bool {
	if position == InsertBefore {
		return insertAt >= len(content) && bytes.Equal(existing[insertAt-len(content):insertAt], content)
	}

	return len(existing)-insertAt >= len(content) && bytes.Equal(existing[insertAt:insertAt+len(content)], content)
}

func validatePostconditions(edit Edit, content []byte) error {
	for _, condition := range edit.Postconditions {
		switch condition.Kind {
		case ConditionPathPresent:
			continue
		case ConditionGoParses:
			_, err := parser.ParseFile(token.NewFileSet(), edit.Target, content, parser.AllErrors)
			if err != nil {
				return executionError("execution_postcondition_failed", edit.Target, "staged Go source does not parse: %v", err)
			}
		case ConditionContentContains:
			if !bytes.Contains(content, []byte(condition.Value)) {
				return executionError("execution_postcondition_failed", edit.Target, "staged content is missing required marker %q", condition.Value)
			}
		case ConditionManagedOutsideDigest:
			digest, err := managedOutsideDigest(content)
			if err != nil {
				return executionError("execution_postcondition_failed", edit.Target, "%v", err)
			}

			if digest != condition.Value {
				return executionError("execution_postcondition_failed", edit.Target, "content outside the managed section changed")
			}
		case ConditionCommandAvailable:
			continue
		default:
			return executionError("execution_postcondition_unsupported", edit.Target, "unsupported postcondition %q", condition.Kind)
		}
	}

	return nil
}

func manifestEdit(values []Edit, id string) (Edit, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}

	return Edit{}, false
}

func stagedChange(edit Edit, kind MutationKind, snapshot targetSnapshot, content []byte, status ChangeStatus) Change {
	return Change{
		EditID:       edit.ID,
		Target:       edit.Target,
		Kind:         kind,
		Status:       status,
		BeforeDigest: snapshot.digest,
		AfterDigest:  contentDigest(content),
	}
}
