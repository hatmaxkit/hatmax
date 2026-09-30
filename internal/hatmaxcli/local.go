// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package hatmaxcli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"hatmax.adrianpk.com/generator/backend/codex"
	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/internal/hatmaxstate"
)

// NewCodexCoordinator assembles the production coordinator around the local
// resident Codex App Server adapter.
func NewCodexCoordinator(
	root string,
	approver interaction.Approver,
	clarifier interaction.Clarifier,
) (Runner, error) {
	identity, err := opaqueProjectIdentity(root)
	if err != nil {
		return nil, err
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user cache directory: %w", err)
	}

	interpreter, err := codex.NewLocalInterpreter(codex.InterpreterConfig{
		ContextRoot:     filepath.Join(cacheRoot, "hatmax", "codex"),
		ProjectIdentity: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Codex interpreter: %w", err)
	}

	coordinator, err := interaction.New(interaction.Config{
		Interpreter: interpreter,
		Approver:    approver,
		Clarifier:   clarifier,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize interaction coordinator: %w", err)
	}

	return coordinator, nil
}

// ConversationRuntime owns the persistent conversation coordinator assembled
// around the same deterministic interaction kernel as headless generation.
type ConversationRuntime struct {
	coordinator *conversation.Coordinator
	store       conversation.Store
}

// NewCodexConversationRuntime assembles a persistent conversational runtime
// around the resident Codex App Server and user-local Hatmax state.
func NewCodexConversationRuntime(root string) (*ConversationRuntime, error) {
	identity, err := opaqueProjectIdentity(root)
	if err != nil {
		return nil, err
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user cache directory: %w", err)
	}

	interpreter, err := codex.NewLocalInterpreter(codex.InterpreterConfig{
		ContextRoot:     filepath.Join(cacheRoot, "hatmax", "codex"),
		ProjectIdentity: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize Codex interpreter: %w", err)
	}

	selectedBook, err := book.LoadDefault()
	if err != nil {
		return nil, fmt.Errorf("load default Hatmax Book: %w", err)
	}

	applicationBook, err := book.LoadRelease(2)
	if err != nil {
		return nil, fmt.Errorf("load application Hatmax Book: %w", err)
	}

	engine, err := interaction.New(interaction.Config{
		Interpreter: interpreter,
		Approver:    rejectingApprover{},
		Book:        selectedBook,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize interaction coordinator: %w", err)
	}

	store, err := hatmaxstate.New(hatmaxstate.Config{})
	if err != nil {
		return nil, fmt.Errorf("initialize conversation state: %w", err)
	}

	coordinator, err := conversation.NewCoordinator(conversation.CoordinatorConfig{
		Store:  store,
		Engine: engine,
		BookContract: conversation.BookContract{
			BookVersion:        selectedBook.Manifest().BookVersion,
			InterpreterVersion: eval.CurrentContractVersion,
		},
		ApplicationBookContract: conversation.BookContract{
			BookVersion:        applicationBook.Manifest().BookVersion,
			InterpreterVersion: eval.CurrentContractVersion,
		},
		BackendIdentity: conversation.BackendIdentity{
			Adapter: "codex-app-server",
			Model:   string(eval.ModelBackendDefault),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize conversation coordinator: %w", err)
	}

	return &ConversationRuntime{coordinator: coordinator, store: store}, nil
}

// Open resumes or deliberately creates one conversation for root.
func (runtime *ConversationRuntime) Open(
	ctx context.Context,
	root string,
	options conversation.SessionOptions,
) (*conversation.ActiveSession, error) {
	return runtime.coordinator.Open(ctx, root, options)
}

// OpenConversation selects one conversation for a headless control command.
func (runtime *ConversationRuntime) OpenConversation(
	ctx context.Context,
	root string,
	options conversation.SessionOptions,
) (ConversationSession, error) {
	return runtime.Open(ctx, root, options)
}

// List returns bounded metadata for conversations in root's exact scope.
func (runtime *ConversationRuntime) List(
	ctx context.Context,
	root string,
) ([]conversation.Summary, error) {
	absoluteRoot, kind, err := conversation.ResolveSessionRoot(root)
	if err != nil {
		return nil, err
	}

	scope, err := conversation.ResolveScope(kind, absoluteRoot)
	if err != nil {
		return nil, err
	}

	return runtime.store.List(ctx, scope)
}

type rejectingApprover struct{}

func (rejectingApprover) Approve(
	context.Context,
	interaction.ApprovalRequest,
) (interaction.ApprovalDecision, error) {
	return interaction.ApprovalRejected, nil
}

func opaqueProjectIdentity(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve canonical project root: %w", err)
	}

	digest := sha256.Sum256([]byte(filepath.Clean(canonical)))

	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
