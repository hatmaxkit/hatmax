// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func inspectRepository(ctx context.Context, root string) (Repository, error) {
	repositoryRoot, err := runGit(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		var executionErr *exec.Error

		var exitErr *exec.ExitError
		if errors.As(err, &executionErr) || errors.As(err, &exitErr) {
			return Repository{}, nil
		}

		return Repository{}, projectError("project_git_failed", root, "%v", err)
	}

	repositoryRoot = strings.TrimSpace(repositoryRoot)

	resolvedRepositoryRoot, err := filepath.EvalSymlinks(repositoryRoot)
	if err != nil {
		return Repository{}, projectError("project_git_failed", repositoryRoot, "%v", err)
	}

	revision, revisionErr := runGit(ctx, root, "rev-parse", "HEAD")
	if revisionErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(revisionErr, &exitErr) {
			return Repository{}, projectError("project_git_failed", root, "%v", revisionErr)
		}
	}

	status, err := runGitBytes(ctx, root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return Repository{}, projectError("project_git_failed", root, "%v", err)
	}

	dirty, err := parseGitStatus(resolvedRepositoryRoot, root, status)
	if err != nil {
		return Repository{}, err
	}

	return Repository{
		Available: true,
		Root:      resolvedRepositoryRoot,
		Revision:  strings.TrimSpace(revision),
		Dirty:     dirty,
	}, nil
}

func runGit(ctx context.Context, root string, arguments ...string) (string, error) {
	output, err := runGitBytes(ctx, root, arguments...)

	return string(output), err
}

func runGitBytes(ctx context.Context, root string, arguments ...string) ([]byte, error) {
	commandArguments := append([]string{"-C", root}, arguments...)
	command := exec.CommandContext(ctx, "git", commandArguments...)

	return command.Output()
}

func parseGitStatus(repositoryRoot, projectRoot string, data []byte) ([]DirtyPath, error) {
	entries := bytes.Split(data, []byte{0})
	result := make([]DirtyPath, 0, len(entries))

	for index := 0; index < len(entries); index++ {
		entry := entries[index]
		if len(entry) == 0 {
			continue
		}

		if len(entry) < 4 || entry[2] != ' ' {
			return nil, projectError("project_git_status_invalid", "", "unexpected porcelain record %q", entry)
		}

		path := string(entry[3:])
		originalPath := ""

		if entry[0] == 'R' || entry[0] == 'C' || entry[1] == 'R' || entry[1] == 'C' {
			index++
			if index >= len(entries) || len(entries[index]) == 0 {
				return nil, projectError("project_git_status_invalid", "", "rename record has no original path")
			}

			originalPath = string(entries[index])
		}

		projectPath, inside := relativeProjectPath(repositoryRoot, projectRoot, path)
		if !inside {
			continue
		}

		originalProjectPath := ""
		if originalPath != "" {
			originalProjectPath, _ = relativeProjectPath(repositoryRoot, projectRoot, originalPath)
		}

		result = append(result, DirtyPath{
			Path:           projectPath,
			OriginalPath:   originalProjectPath,
			IndexStatus:    entry[0],
			WorktreeStatus: entry[1],
		})
	}

	sort.Slice(result, func(left, right int) bool {
		return result[left].Path < result[right].Path
	})

	return result, nil
}

func relativeProjectPath(repositoryRoot, projectRoot, repositoryPath string) (string, bool) {
	absolute := filepath.Join(repositoryRoot, filepath.FromSlash(repositoryPath))

	relative, err := filepath.Rel(projectRoot, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(relative), true
}

// DirtyOverlap returns dirty paths that intersect one of the supplied project
// paths. Both files and directory prefixes are supported.
func (i Inventory) DirtyOverlap(paths []string) ([]DirtyPath, error) {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		_, err := projectPath(i.Root, path)
		if err != nil {
			return nil, err
		}

		normalized = append(normalized, filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))))
	}

	result := make([]DirtyPath, 0)

	for _, dirty := range i.Repository.Dirty {
		for _, path := range normalized {
			if pathsOverlap(dirty.Path, path) || pathsOverlap(dirty.OriginalPath, path) {
				result = append(result, dirty)

				break
			}
		}
	}

	return result, nil
}

func pathsOverlap(left, right string) bool {
	if left == "" || right == "" {
		return false
	}

	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}
