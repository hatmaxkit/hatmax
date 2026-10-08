// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package project

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultMaximumFiles    = 100000
	defaultMaximumFileSize = int64(4 << 20)
)

// Options bounds project inspection.
type Options struct {
	MaximumFiles    int
	MaximumFileSize int64
}

// Inspect creates a read-only semantic inventory of root.
func Inspect(ctx context.Context, root string) (Inventory, error) {
	return InspectWithOptions(ctx, root, Options{})
}

// InspectWithOptions creates a bounded read-only semantic inventory of root.
func InspectWithOptions(ctx context.Context, root string, options Options) (Inventory, error) {
	normalizedOptions, err := normalizeOptions(options)
	if err != nil {
		return Inventory{}, err
	}

	normalizedRoot, err := normalizeRoot(root)
	if err != nil {
		return Inventory{}, err
	}

	result := Inventory{
		Root:            normalizedRoot,
		maximumFileSize: normalizedOptions.MaximumFileSize,
	}

	result.Module, err = inspectModule(normalizedRoot)
	if err != nil {
		return Inventory{}, err
	}

	result.Repository, err = inspectRepository(ctx, normalizedRoot)
	if err != nil {
		return Inventory{}, err
	}

	err = inspectFiles(&result, normalizedOptions)
	if err != nil {
		return Inventory{}, err
	}

	result.Commands, err = inspectCommands(normalizedRoot, result.files)
	if err != nil {
		return Inventory{}, err
	}

	err = inspectDocumentation(&result)
	if err != nil {
		return Inventory{}, err
	}

	sortInventory(&result)

	return result, nil
}

func normalizeOptions(options Options) (Options, error) {
	if options.MaximumFiles < 0 || options.MaximumFileSize < 0 {
		return Options{}, projectError("project_invalid_options", "", "inspection limits cannot be negative")
	}

	if options.MaximumFiles == 0 {
		options.MaximumFiles = defaultMaximumFiles
	}

	if options.MaximumFileSize == 0 {
		options.MaximumFileSize = defaultMaximumFileSize
	}

	return options, nil
}

func normalizeRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", projectError("project_root_invalid", "", "root is required")
	}

	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", projectError("project_root_invalid", root, "%v", err)
	}

	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", projectError("project_root_invalid", absolute, "%v", err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", projectError("project_root_invalid", resolved, "%v", err)
	}

	if !info.IsDir() {
		return "", projectError("project_root_invalid", resolved, "root is not a directory")
	}

	return resolved, nil
}

func inspectFiles(inventory *Inventory, options Options) error {
	seenDirectories := make(map[string]map[string]struct{})
	generatedPaths := make([]string, 0)

	inventory.Rules.ProtectedPaths = append(inventory.Rules.ProtectedPaths, ProtectedPath{
		Path:   ".git",
		Reason: "repository metadata",
	})

	err := filepath.WalkDir(inventory.Root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, relErr := filepath.Rel(inventory.Root, path)
		if relErr != nil {
			return relErr
		}

		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			if relative == "vendor" {
				inventory.Rules.ProtectedPaths = append(inventory.Rules.ProtectedPaths, ProtectedPath{
					Path:   "vendor",
					Reason: "vendored dependency tree",
				})
			}

			if relative != "." && shouldSkipDirectory(relative) {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		if len(inventory.files) >= options.MaximumFiles {
			return projectError("project_limit_exceeded", relative, "project contains more than %d files", options.MaximumFiles)
		}

		record := fileRecord{path: relative, size: info.Size()}
		if record.size <= options.MaximumFileSize {
			record.generated, record.surfaces, infoErr = inspectFileSemantics(path, relative, inventory)
			if infoErr != nil {
				return infoErr
			}
		}

		inventory.files = append(inventory.files, record)
		collectLayouts(relative, record, seenDirectories)
		collectRepositoryRules(relative, record, inventory, &generatedPaths)

		return nil
	})
	if err != nil {
		var projectErr Error
		if !asProjectError(err, &projectErr) {
			return projectError("project_walk_failed", inventory.Root, "%v", err)
		}

		return projectErr
	}

	applyLayouts(&inventory.Layout, seenDirectories)

	for _, path := range generatedPaths {
		inventory.Rules.ProtectedPaths = append(inventory.Rules.ProtectedPaths, ProtectedPath{
			Path:   path,
			Reason: "generated file",
		})
	}

	return nil
}

func shouldSkipDirectory(path string) bool {
	base := filepath.Base(path)

	return base == ".git" || base == ".tmp" || base == "tmp" || base == "node_modules" || base == "vendor"
}

func sortInventory(inventory *Inventory) {
	sort.Strings(inventory.compositionRoots)
	sort.Slice(inventory.Entrypoints, func(left, right int) bool {
		return inventory.Entrypoints[left].Path < inventory.Entrypoints[right].Path
	})
	sort.Slice(inventory.Commands, func(left, right int) bool {
		if inventory.Commands[left].Kind != inventory.Commands[right].Kind {
			return inventory.Commands[left].Kind < inventory.Commands[right].Kind
		}

		return inventory.Commands[left].Name < inventory.Commands[right].Name
	})
	sort.Strings(inventory.Rules.Instructions)
	sort.Slice(inventory.Rules.ProtectedPaths, func(left, right int) bool {
		return inventory.Rules.ProtectedPaths[left].Path < inventory.Rules.ProtectedPaths[right].Path
	})
	sort.Slice(inventory.files, func(left, right int) bool {
		return inventory.files[left].path < inventory.files[right].path
	})
	sort.Slice(inventory.Documentation.Files, func(left, right int) bool {
		return inventory.Documentation.Files[left].Path < inventory.Documentation.Files[right].Path
	})
	sort.Strings(inventory.Documentation.OtherPaths)
	sort.Slice(inventory.Documentation.ProtectedPaths, func(left, right int) bool {
		return inventory.Documentation.ProtectedPaths[left].Path < inventory.Documentation.ProtectedPaths[right].Path
	})
	sort.Slice(inventory.Documentation.ValidationCommands, func(left, right int) bool {
		return inventory.Documentation.ValidationCommands[left].Name < inventory.Documentation.ValidationCommands[right].Name
	})
}

func asProjectError(err error, target *Error) bool {
	value, ok := err.(Error)
	if !ok {
		return false
	}

	*target = value

	return true
}

func projectPath(root, path string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(path))
	if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", projectError("project_path_invalid", path, "path must identify content inside the project")
	}

	joined := filepath.Join(root, cleaned)

	relative, err := filepath.Rel(root, joined)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", projectError("project_path_invalid", path, "path escapes the project root")
	}

	return joined, nil
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func formatCommand(command Command) string {
	return strings.Join(command.Args, " ")
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func unexpectedReadError(path string, err error) error {
	return projectError("project_read_failed", path, "%v", fmt.Errorf("read project file: %w", err))
}
