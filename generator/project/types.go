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
	Root        string
	Repository  Repository
	Module      Module
	Entrypoints []Entrypoint
	Layout      Layout
	Commands    []Command
	Rules       RepositoryRules

	files           []fileRecord
	maximumFileSize int64
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
