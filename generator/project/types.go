// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package project inspects Hatmax projects without modifying them.
package project

import "fmt"

// HatmaxModulePath is the canonical Hatmax Go module path.
const HatmaxModulePath = "hatmax.adrianpk.com"

// HatmaxSource describes how a project resolves the Hatmax module.
type HatmaxSource string

const (
	// HatmaxSourceMissing means the project does not declare Hatmax.
	HatmaxSourceMissing HatmaxSource = "missing"
	// HatmaxSourceModule means the project uses a versioned module dependency.
	HatmaxSourceModule HatmaxSource = "module"
	// HatmaxSourceLocalReplace means go.mod replaces Hatmax with a local path.
	HatmaxSourceLocalReplace HatmaxSource = "local_replace"
	// HatmaxSourceWorkspace means a go.work file supplies the Hatmax module.
	HatmaxSourceWorkspace HatmaxSource = "workspace"
	// HatmaxSourceMainModule means the inspected project is Hatmax itself.
	HatmaxSourceMainModule HatmaxSource = "main_module"
)

// CommandKind classifies a repository-owned command without executing it.
type CommandKind string

const (
	// CommandValidation identifies a validation command.
	CommandValidation CommandKind = "validation"
	// CommandGeneration identifies a source generation command.
	CommandGeneration CommandKind = "generation"
	// CommandFormatting identifies a formatting command.
	CommandFormatting CommandKind = "formatting"
)

// Repository records Git observations for the inspected project.
type Repository struct {
	Available bool
	Root      string
	Revision  string
	Dirty     []DirtyPath
}

// DirtyPath records one index or worktree change relative to the project root.
type DirtyPath struct {
	Path           string
	OriginalPath   string
	IndexStatus    byte
	WorktreeStatus byte
}

// Replacement describes one Go module replacement.
type Replacement struct {
	Path    string
	Version string
}

// Dependency describes one declared Go module dependency.
type Dependency struct {
	Path        string
	Version     string
	Indirect    bool
	Replacement *Replacement
}

// HatmaxModule describes the effective Hatmax dependency.
type HatmaxModule struct {
	Version   string
	Source    HatmaxSource
	LocalPath string
}

// Module describes the project's Go module and workspace context.
type Module struct {
	Path         string
	GoVersion    string
	GoModPath    string
	GoWorkPath   string
	Hatmax       HatmaxModule
	Dependencies []Dependency
}

// Entrypoint describes one executable entrypoint and whether it assembles
// Hatmax components.
type Entrypoint struct {
	Path            string
	CompositionRoot bool
}

// Layout groups semantic project paths without loading their content.
type Layout struct {
	Features   []string
	Assets     []string
	Migrations []string
	Templates  []string
	Queries    []string
	Generated  []string
}

// DocumentationManagedState classifies Hatmax ownership markers without
// interpreting surrounding prose.
type DocumentationManagedState string

const (
	// DocumentationUnmanaged means a canonical Markdown file has no Hatmax
	// ownership markers.
	DocumentationUnmanaged DocumentationManagedState = "unmanaged"
	// DocumentationManaged means a canonical Markdown file has one balanced
	// Hatmax-owned section.
	DocumentationManaged DocumentationManagedState = "managed"
	// DocumentationMarkersInvalid means ownership markers are duplicated,
	// incomplete, or reversed.
	DocumentationMarkersInvalid DocumentationManagedState = "markers_invalid"
	// DocumentationUnreadable means the file exceeds the bounded inspection
	// limit.
	DocumentationUnreadable DocumentationManagedState = "unreadable"
)

// DocumentationLink is one local Markdown link resolved inside the project.
type DocumentationLink struct {
	Target string
	Exists bool
}

// DocumentationFile records only canonical path, ownership, digest, and link
// structure. Its prose is not exposed as project authority.
type DocumentationFile struct {
	Path         string
	Quadrant     string
	Subject      string
	Index        bool
	Size         int64
	Digest       string
	ManagedState DocumentationManagedState
	LocalLinks   []DocumentationLink
}

// DocumentationInventory describes the bounded canonical Diataxis surface.
type DocumentationInventory struct {
	Root               string
	Files              []DocumentationFile
	OtherPaths         []string
	ProtectedPaths     []ProtectedPath
	ValidationCommands []Command
}

// FeatureFieldEvidence is one field established from matching model, form,
// and persistence structure.
type FeatureFieldEvidence struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Label    string `json:"label" yaml:"label"`
	Required bool   `json:"required" yaml:"required"`
}

// FeatureValidationEvidence is one validation established at an observed
// canonical boundary.
type FeatureValidationEvidence struct {
	Field string `json:"field" yaml:"field"`
	Kind  string `json:"kind" yaml:"kind"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	Scope string `json:"scope" yaml:"scope"`
}

// FeatureEvidenceSource identifies one structurally parsed canonical source
// without retaining its content.
type FeatureEvidenceSource struct {
	Role   string `json:"role" yaml:"role"`
	Path   string `json:"path" yaml:"path"`
	Digest string `json:"digest" yaml:"digest"`
}

// FeatureEvidence is the bounded product contract established for one
// canonical server-rendered CRUD feature.
type FeatureEvidence struct {
	Basis             string                      `json:"basis" yaml:"basis"`
	Feature           string                      `json:"feature" yaml:"feature"`
	Entity            string                      `json:"entity" yaml:"entity"`
	Label             string                      `json:"label" yaml:"label"`
	Route             string                      `json:"route" yaml:"route"`
	Table             string                      `json:"table" yaml:"table"`
	Fields            []FeatureFieldEvidence      `json:"fields" yaml:"fields"`
	Validations       []FeatureValidationEvidence `json:"validations" yaml:"validations"`
	Postgres          bool                        `json:"postgres" yaml:"postgres"`
	HTMX              bool                        `json:"htmx" yaml:"htmx"`
	RuntimeValidation bool                        `json:"runtime_validation" yaml:"runtime_validation"`
	Wired             bool                        `json:"wired" yaml:"wired"`
	Tested            bool                        `json:"tested" yaml:"tested"`
	Sources           []FeatureEvidenceSource     `json:"sources" yaml:"sources"`
}

// Command describes a bounded repository-owned command.
type Command struct {
	Kind   CommandKind
	Name   string
	Args   []string
	Source string
}

// ProtectedPath describes a path the generator must not edit directly.
type ProtectedPath struct {
	Path   string
	Reason string
}

// File describes bounded file metadata captured during project inspection.
// It does not expose file content.
type File struct {
	Path      string
	Size      int64
	Generated bool
	Surfaces  []string
}

// RepositoryRules identifies instruction files and protected paths.
type RepositoryRules struct {
	Instructions   []string
	ProtectedPaths []ProtectedPath
}

// Inventory is a bounded semantic snapshot of a Hatmax project.
type Inventory struct {
	Root          string
	Repository    Repository
	Module        Module
	Entrypoints   []Entrypoint
	Layout        Layout
	Commands      []Command
	Rules         RepositoryRules
	Documentation DocumentationInventory

	files            []fileRecord
	compositionRoots []string
	maximumFileSize  int64
}

// Error describes a project inspection failure.
type Error struct {
	Code    string
	Path    string
	Message string
}

func (e Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

func projectError(code, path, format string, arguments ...any) error {
	return Error{
		Code:    code,
		Path:    path,
		Message: fmt.Sprintf(format, arguments...),
	}
}

type fileRecord struct {
	path      string
	size      int64
	generated bool
	surfaces  []string
}
