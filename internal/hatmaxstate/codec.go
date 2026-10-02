// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package hatmaxstate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"hatmax.adrianpk.com/generator/conversation"
)

const maximumSnapshotBytes = 10 << 20

type scopeIndex struct {
	SchemaVersion   int                `json:"schema_version"`
	Scope           conversation.Scope `json:"scope"`
	Active          string             `json:"active_conversation_id,omitempty"`
	ArchiveOrder    []string           `json:"archive_order"`
	IncompatibleIDs []string           `json:"incompatible_ids"`
}

func readJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maximumSnapshotBytes+1))
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	if len(data) > maximumSnapshotBytes {
		return stateError("state_limit_exceeded", path, "state file exceeds %d bytes", maximumSnapshotBytes)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	if err != nil {
		return stateError("state_incompatible", path, "decode state: %v", err)
	}

	var trailing any

	err = decoder.Decode(&trailing)
	if err != io.EOF {
		return stateError("state_incompatible", path, "state file contains trailing data")
	}

	return nil
}

func writeJSONAtomic(path string, value any, beforeRename func(string) error) error {
	directory := filepath.Dir(path)

	err := os.MkdirAll(directory, 0o700)
	if err != nil {
		return stateError("state_persistence_failed", path, "create state directory: %v", err)
	}

	err = os.Chmod(directory, 0o700)
	if err != nil {
		return stateError("state_persistence_failed", path, "secure state directory: %v", err)
	}

	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*")
	if err != nil {
		return stateError("state_persistence_failed", path, "create temporary state file: %v", err)
	}

	temporaryPath := temporary.Name()
	keepTemporary := false

	defer func() {
		_ = temporary.Close()

		if !keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	err = temporary.Chmod(0o600)
	if err != nil {
		return stateError("state_persistence_failed", path, "secure temporary state file: %v", err)
	}

	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(false)

	err = encoder.Encode(value)
	if err != nil {
		return stateError("state_persistence_failed", path, "encode state: %v", err)
	}

	err = temporary.Sync()
	if err != nil {
		return stateError("state_persistence_failed", path, "synchronize temporary state: %v", err)
	}

	err = temporary.Close()
	if err != nil {
		return stateError("state_persistence_failed", path, "close temporary state: %v", err)
	}

	if beforeRename != nil {
		err = beforeRename(path)
		if err != nil {
			return stateError("state_persistence_failed", path, "before atomic replacement: %v", err)
		}
	}

	err = os.Rename(temporaryPath, path)
	if err != nil {
		return stateError("state_persistence_failed", path, "replace state atomically: %v", err)
	}

	keepTemporary = true

	err = syncDirectory(directory)
	if err != nil {
		return stateError("state_persistence_failed", path, "synchronize state directory: %v", err)
	}

	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()

	return directory.Sync()
}

func validatePrivacy(value conversation.Conversation) error {
	for index, turn := range value.Turns {
		if containsSensitiveMaterial(turn.Content) {
			return stateError("state_sensitive_content", fmt.Sprintf("turns[%d].content", index), "credential-like content cannot be persisted")
		}
	}

	for index, operation := range value.Operations {
		if containsSensitiveMaterial(operation.RequestSummary) || containsSensitiveMaterial(operation.ResultSummary) {
			return stateError("state_sensitive_content", fmt.Sprintf("operations[%d]", index), "credential-like content cannot be persisted")
		}
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return stateError("state_persistence_failed", "conversation", "encode privacy boundary: %v", err)
	}

	if containsSensitiveMaterial(string(encoded)) {
		return stateError("state_sensitive_content", "conversation", "credential-like content cannot be persisted")
	}

	return nil
}

func containsSensitiveMaterial(value string) bool {
	lower := strings.ToLower(value)
	markers := []string{
		"-----begin private key-----",
		"-----begin rsa private key-----",
		"authorization: bearer ",
		"openai_api_key=",
		"codex_api_key=",
		"aws_secret_access_key=",
		"github_pat_",
		"ghp_",
		"sk-proj-",
	}

	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	return false
}

func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
