// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/mod/modfile"
)

func inspectModule(root string) (Module, error) {
	goModPath := filepath.Join(root, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Module{}, projectError("project_go_module_missing", "go.mod", "Go module file does not exist")
		}

		return Module{}, unexpectedReadError("go.mod", err)
	}

	parsed, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return Module{}, projectError("project_go_module_invalid", "go.mod", "%v", err)
	}

	if parsed.Module == nil || parsed.Module.Mod.Path == "" {
		return Module{}, projectError("project_go_module_invalid", "go.mod", "module path is required")
	}

	result := Module{
		Path:      parsed.Module.Mod.Path,
		GoModPath: "go.mod",
		Hatmax: HatmaxModule{
			Source: HatmaxSourceMissing,
		},
	}
	if parsed.Go != nil {
		result.GoVersion = parsed.Go.Version
	}

	replacements := make(map[string]Replacement, len(parsed.Replace))
	for _, replacement := range parsed.Replace {
		replacements[replacement.Old.Path] = Replacement{
			Path:    replacement.New.Path,
			Version: replacement.New.Version,
		}
	}

	for _, requirement := range parsed.Require {
		dependency := Dependency{
			Path:     requirement.Mod.Path,
			Version:  requirement.Mod.Version,
			Indirect: requirement.Indirect,
		}

		if replacement, exists := replacements[dependency.Path]; exists {
			copyOfReplacement := replacement
			dependency.Replacement = &copyOfReplacement
		}

		result.Dependencies = append(result.Dependencies, dependency)
		if dependency.Path == HatmaxModulePath {
			result.Hatmax.Version = dependency.Version
			result.Hatmax.Source = HatmaxSourceModule
		}
	}

	if result.Path == HatmaxModulePath {
		result.Hatmax.Source = HatmaxSourceMainModule
	}

	if replacement, exists := replacements[HatmaxModulePath]; exists && replacement.Version == "" {
		result.Hatmax.Source = HatmaxSourceLocalReplace
		result.Hatmax.LocalPath = resolveLocalPath(root, replacement.Path)
	}

	workPath, workHatmaxPath, workErr := inspectWorkspace(root)
	if workErr != nil {
		return Module{}, workErr
	}

	if workPath != "" {
		result.GoWorkPath = workPath
	}

	if workHatmaxPath != "" && result.Path != HatmaxModulePath {
		result.Hatmax.Source = HatmaxSourceWorkspace
		result.Hatmax.LocalPath = workHatmaxPath
	}

	sort.Slice(result.Dependencies, func(left, right int) bool {
		return result.Dependencies[left].Path < result.Dependencies[right].Path
	})

	return result, nil
}

func inspectWorkspace(root string) (string, string, error) {
	workPath := findUpward(root, "go.work")
	if workPath == "" {
		return "", "", nil
	}

	data, err := os.ReadFile(workPath)
	if err != nil {
		return "", "", unexpectedReadError(workPath, err)
	}

	parsed, err := modfile.ParseWork(workPath, data, nil)
	if err != nil {
		return "", "", projectError("project_go_workspace_invalid", workPath, "%v", err)
	}

	workRoot := filepath.Dir(workPath)
	for _, use := range parsed.Use {
		moduleRoot := resolveLocalPath(workRoot, use.Path)

		modulePath, readErr := readModulePath(moduleRoot)
		if readErr != nil {
			return "", "", readErr
		}

		if modulePath == HatmaxModulePath {
			return workPath, moduleRoot, nil
		}
	}

	for _, replacement := range parsed.Replace {
		if replacement.Old.Path == HatmaxModulePath && replacement.New.Version == "" {
			return workPath, resolveLocalPath(workRoot, replacement.New.Path), nil
		}
	}

	return workPath, "", nil
}

func readModulePath(root string) (string, error) {
	path := filepath.Join(root, "go.mod")

	data, err := os.ReadFile(path)
	if err != nil {
		return "", unexpectedReadError(path, err)
	}

	parsed, err := modfile.Parse(path, data, nil)
	if err != nil {
		return "", projectError("project_go_module_invalid", path, "%v", err)
	}

	if parsed.Module == nil {
		return "", projectError("project_go_module_invalid", path, "module path is required")
	}

	return parsed.Module.Mod.Path, nil
}

func resolveLocalPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
}

func findUpward(root, name string) string {
	current := root
	for {
		candidate := filepath.Join(current, name)

		_, err := os.Stat(candidate)
		if err == nil {
			return candidate
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}

		current = parent
	}
}
