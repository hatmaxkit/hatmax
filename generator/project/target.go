// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
	"hatmax.adrianpk.com/generator/book"
)

// TargetAdmission classifies whether application creation may use a target.
type TargetAdmission string

const (
	// TargetAbsent means the target does not exist and can be created.
	TargetAbsent TargetAdmission = "absent"
	// TargetEmpty means the target exists as an empty directory.
	TargetEmpty TargetAdmission = "empty"
	// TargetPreservable means existing entries do not collide with the scaffold.
	TargetPreservable TargetAdmission = "preservable"
	// TargetCompatibleProject means the target is an existing compatible Hatmax project.
	TargetCompatibleProject TargetAdmission = "compatible_project"
	// TargetIncompatible means application creation must not mutate the target.
	TargetIncompatible TargetAdmission = "incompatible"
)

// TargetEntryKind classifies one existing target entry.
type TargetEntryKind string

const (
	// TargetEntryDirectory identifies a directory.
	TargetEntryDirectory TargetEntryKind = "directory"
	// TargetEntryFile identifies a regular file.
	TargetEntryFile TargetEntryKind = "file"
	// TargetEntrySymlink identifies a symbolic link, which blocks admission.
	TargetEntrySymlink TargetEntryKind = "symlink"
)

// TargetEntry records bounded metadata for content that application creation
// must preserve.
type TargetEntry struct {
	Path   string
	Kind   TargetEntryKind
	Size   int64
	Digest string
}

// TargetCollision records an existing condition that blocks a scaffold path
// or target admission.
type TargetCollision struct {
	Path   string
	Reason string
}

// RemoteEvidence is a credential-free repository remote projection.
type RemoteEvidence struct {
	Name       string
	Host       string
	Path       string
	ModulePath string
}

// TargetRequest selects one pre-project target and the exact file paths the
// application archetype intends to create.
type TargetRequest struct {
	Parent       string
	Target       string
	PlannedPaths []string
	Book         *book.Book
	Options      Options
}

// TargetInventory is a bounded, read-only snapshot of one proposed
// application target.
type TargetInventory struct {
	Parent           string
	Target           string
	Exists           bool
	Empty            bool
	Admission        TargetAdmission
	Entries          []TargetEntry
	Preserved        []TargetEntry
	Collisions       []TargetCollision
	Repository       Repository
	Remotes          []RemoteEvidence
	RemoteModulePath string
	Rules            RepositoryRules
	ExistingProject  *Inventory
	PlannedPaths     []string
}

// InspectTarget creates a read-only pre-project inventory.
func InspectTarget(ctx context.Context, request TargetRequest) (TargetInventory, error) {
	options, err := normalizeOptions(request.Options)
	if err != nil {
		return TargetInventory{}, err
	}

	parent, target, err := normalizeTarget(request.Parent, request.Target)
	if err != nil {
		return TargetInventory{}, err
	}

	plannedPaths, err := normalizeTargetPaths(request.PlannedPaths)
	if err != nil {
		return TargetInventory{}, err
	}

	result := TargetInventory{
		Parent:       parent,
		Target:       target,
		Admission:    TargetAbsent,
		PlannedPaths: plannedPaths,
	}

	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		err = inspectTargetRepository(ctx, &result, parent)
		if err != nil {
			return TargetInventory{}, err
		}

		collectTargetRules(&result)

		return result, nil
	}

	if err != nil {
		return TargetInventory{}, projectError("target_stat_failed", target, "%v", err)
	}

	result.Exists = true
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		result.Admission = TargetIncompatible
		result.Collisions = []TargetCollision{{Path: ".", Reason: "target is not a real directory"}}

		return result, nil
	}

	err = inspectTargetEntries(&result, options)
	if err != nil {
		return TargetInventory{}, err
	}

	err = inspectTargetRepository(ctx, &result, target)
	if err != nil {
		return TargetInventory{}, err
	}

	collectTargetRules(&result)
	classifyTarget(&result, request.Book, ctx)

	return result, nil
}

func normalizeTarget(parent, target string) (string, string, error) {
	normalizedParent, err := normalizeRoot(parent)
	if err != nil {
		return "", "", projectError("target_parent_invalid", parent, "%v", err)
	}

	if strings.TrimSpace(target) == "" {
		return "", "", projectError("target_path_invalid", "", "target is required")
	}

	normalizedTarget := target
	if !filepath.IsAbs(normalizedTarget) {
		normalizedTarget = filepath.Join(normalizedParent, normalizedTarget)
	}

	normalizedTarget, err = filepath.Abs(normalizedTarget)
	if err != nil {
		return "", "", projectError("target_path_invalid", target, "%v", err)
	}

	normalizedTarget = filepath.Clean(normalizedTarget)

	relative, err := filepath.Rel(normalizedParent, normalizedTarget)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", projectError("target_path_invalid", target, "target must be below the authorized parent")
	}

	info, statErr := os.Lstat(normalizedTarget)
	if statErr == nil && info.Mode()&os.ModeSymlink == 0 {
		resolved, resolveErr := filepath.EvalSymlinks(normalizedTarget)
		if resolveErr != nil {
			return "", "", projectError("target_path_invalid", target, "%v", resolveErr)
		}

		resolvedRelative, relErr := filepath.Rel(normalizedParent, resolved)
		if relErr != nil || resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(filepath.Separator)) {
			return "", "", projectError("target_path_invalid", target, "resolved target escapes the authorized parent")
		}

		normalizedTarget = resolved
	}

	return normalizedParent, normalizedTarget, nil
}

func normalizeTargetPaths(paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	for _, path := range uniqueSorted(paths) {
		cleaned := filepath.Clean(filepath.FromSlash(path))
		if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return nil, projectError("target_planned_path_invalid", path, "planned path must identify content inside the target")
		}

		result = append(result, filepath.ToSlash(cleaned))
	}

	return result, nil
}

func inspectTargetEntries(inventory *TargetInventory, options Options) error {
	err := filepath.WalkDir(inventory.Target, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(inventory.Target, path)
		if err != nil {
			return err
		}

		relative = filepath.ToSlash(relative)
		if relative == "." {
			return nil
		}

		if relative == ".git" && entry.IsDir() {
			inventory.Entries = append(inventory.Entries, TargetEntry{Path: relative, Kind: TargetEntryDirectory})

			return filepath.SkipDir
		}

		if len(inventory.Entries) >= options.MaximumFiles {
			return projectError("target_limit_exceeded", relative, "target contains more than %d entries", options.MaximumFiles)
		}

		if entry.Type()&os.ModeSymlink != 0 {
			inventory.Entries = append(inventory.Entries, TargetEntry{Path: relative, Kind: TargetEntrySymlink})
			inventory.Collisions = append(inventory.Collisions, TargetCollision{Path: relative, Reason: "symbolic links are not admitted"})

			return nil
		}

		if entry.IsDir() {
			inventory.Entries = append(inventory.Entries, TargetEntry{Path: relative, Kind: TargetEntryDirectory})

			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			inventory.Collisions = append(inventory.Collisions, TargetCollision{Path: relative, Reason: "non-regular files are not admitted"})

			return nil
		}

		if info.Size() > options.MaximumFileSize {
			inventory.Collisions = append(inventory.Collisions, TargetCollision{Path: relative, Reason: "file exceeds the inspection limit"})
			inventory.Entries = append(inventory.Entries, TargetEntry{Path: relative, Kind: TargetEntryFile, Size: info.Size()})

			return nil
		}

		digest, err := digestTargetFile(path)
		if err != nil {
			return err
		}

		inventory.Entries = append(inventory.Entries, TargetEntry{
			Path: relative, Kind: TargetEntryFile, Size: info.Size(), Digest: digest,
		})

		return nil
	})
	if err != nil {
		var targetErr Error
		if asProjectError(err, &targetErr) {
			return targetErr
		}

		return projectError("target_walk_failed", inventory.Target, "%v", err)
	}

	sort.Slice(inventory.Entries, func(left, right int) bool {
		return inventory.Entries[left].Path < inventory.Entries[right].Path
	})

	inventory.Empty = len(inventory.Entries) == 0

	return nil
}

func digestTargetFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", projectError("target_read_failed", path, "%v", err)
	}
	defer file.Close()

	digest := sha256.New()

	_, err = io.Copy(digest, file)
	if err != nil {
		return "", projectError("target_read_failed", path, "%v", err)
	}

	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}

func classifyTarget(inventory *TargetInventory, selectedBook *book.Book, ctx context.Context) {
	if inventory.Empty {
		inventory.Admission = TargetEmpty

		return
	}

	if !inventory.Repository.Available || filepath.Clean(inventory.Repository.Root) != inventory.Target {
		inventory.Collisions = append(inventory.Collisions, TargetCollision{
			Path: ".", Reason: "non-empty target is not the root of an existing Git repository",
		})
	}

	if _, hasModule := targetEntry(inventory.Entries, "go.mod"); hasModule {
		inspectExistingTarget(inventory, selectedBook, ctx)

		if inventory.Admission == TargetCompatibleProject {
			return
		}

		inventory.Collisions = append(inventory.Collisions, TargetCollision{Path: "go.mod", Reason: "target contains another or incomplete Go module"})
	}

	for _, planned := range inventory.PlannedPaths {
		for _, entry := range inventory.Entries {
			if entry.Path == planned || entry.Kind != TargetEntryDirectory && strings.HasPrefix(planned, entry.Path+"/") {
				inventory.Collisions = append(inventory.Collisions, TargetCollision{Path: entry.Path, Reason: "entry collides with a planned scaffold path"})
			}
		}
	}

	inventory.Collisions = uniqueTargetCollisions(inventory.Collisions)
	if len(inventory.Collisions) > 0 {
		inventory.Admission = TargetIncompatible

		return
	}

	inventory.Admission = TargetPreservable
	inventory.Preserved = cloneTargetEntries(inventory.Entries)
}

func inspectExistingTarget(inventory *TargetInventory, selectedBook *book.Book, ctx context.Context) {
	if selectedBook == nil {
		return
	}

	existing, err := Inspect(ctx, inventory.Target)
	if err != nil {
		return
	}

	compatibility, err := existing.CompatibilityWith(selectedBook)
	if err != nil || compatibility != BookCompatibilityCompatible || len(existing.Entrypoints) == 0 {
		return
	}

	inventory.ExistingProject = &existing
	inventory.Admission = TargetCompatibleProject
}

func inspectTargetRepository(ctx context.Context, inventory *TargetInventory, root string) error {
	repository, err := inspectRepository(ctx, root)
	if err != nil {
		return err
	}

	inventory.Repository = repository
	if !repository.Available {
		return nil
	}

	remotes, err := inspectRemoteEvidence(ctx, root, repository.Root, inventory.Target)
	if err != nil {
		return err
	}

	inventory.Remotes = remotes
	inventory.RemoteModulePath = unambiguousModulePath(remotes)

	return nil
}

func inspectRemoteEvidence(ctx context.Context, root, repositoryRoot, target string) ([]RemoteEvidence, error) {
	namesOutput, err := runGit(ctx, root, "remote")
	if err != nil {
		return nil, projectError("target_git_failed", root, "%v", err)
	}

	names := strings.Fields(namesOutput)
	sort.Strings(names)
	result := make([]RemoteEvidence, 0, len(names))

	for _, name := range names {
		remoteURL, remoteErr := runGit(ctx, root, "remote", "get-url", name)
		if remoteErr != nil {
			return nil, projectError("target_git_failed", root, "%v", remoteErr)
		}

		evidence, ok := parseRemoteEvidence(name, strings.TrimSpace(remoteURL))
		if !ok {
			continue
		}

		relative, relErr := filepath.Rel(repositoryRoot, target)
		if relErr == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			evidence.ModulePath += "/" + filepath.ToSlash(relative)
		}

		if module.CheckPath(evidence.ModulePath) != nil {
			continue
		}

		result = append(result, evidence)
	}

	return result, nil
}

func parseRemoteEvidence(name, value string) (RemoteEvidence, bool) {
	host := ""
	remotePath := ""

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Hostname() == "" {
			return RemoteEvidence{}, false
		}

		host = strings.ToLower(parsed.Hostname())
		remotePath = parsed.Path
	} else {
		separator := strings.Index(value, ":")
		if separator < 0 {
			return RemoteEvidence{}, false
		}

		hostPart := value[:separator]
		if at := strings.LastIndex(hostPart, "@"); at >= 0 {
			hostPart = hostPart[at+1:]
		}

		host = strings.ToLower(hostPart)
		remotePath = value[separator+1:]
	}

	remotePath = strings.Trim(strings.TrimSuffix(remotePath, ".git"), "/")
	if host == "" || remotePath == "" {
		return RemoteEvidence{}, false
	}

	return RemoteEvidence{
		Name: name, Host: host, Path: remotePath, ModulePath: host + "/" + remotePath,
	}, true
}

func unambiguousModulePath(remotes []RemoteEvidence) string {
	for _, remote := range remotes {
		if remote.Name == "origin" {
			return remote.ModulePath
		}
	}

	values := make(map[string]struct{}, len(remotes))
	for _, remote := range remotes {
		values[remote.ModulePath] = struct{}{}
	}

	if len(values) != 1 {
		return ""
	}

	for value := range values {
		return value
	}

	return ""
}

func collectTargetRules(inventory *TargetInventory) {
	for _, root := range []string{inventory.Parent, inventory.Target} {
		path := findUpward(root, "AGENTS.md")
		if path == "" {
			continue
		}

		inventory.Rules.Instructions = append(inventory.Rules.Instructions, path)
	}

	inventory.Rules.Instructions = uniqueSorted(inventory.Rules.Instructions)
}

func targetEntry(entries []TargetEntry, path string) (TargetEntry, bool) {
	for _, entry := range entries {
		if entry.Path == path {
			return entry, true
		}
	}

	return TargetEntry{}, false
}

func uniqueTargetCollisions(values []TargetCollision) []TargetCollision {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Path != values[right].Path {
			return values[left].Path < values[right].Path
		}

		return values[left].Reason < values[right].Reason
	})

	result := make([]TargetCollision, 0, len(values))
	for _, value := range values {
		if len(result) > 0 && result[len(result)-1] == value {
			continue
		}

		result = append(result, value)
	}

	return result
}

func cloneTargetEntries(values []TargetEntry) []TargetEntry {
	return append([]TargetEntry(nil), values...)
}
