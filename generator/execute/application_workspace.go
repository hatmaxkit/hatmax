// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package execute

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"hatmax.adrianpk.com/generator/book"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// ValidationCommandStatus classifies one scaffold validation command.
type ValidationCommandStatus string

const (
	// ValidationCommandPassed means the command completed successfully.
	ValidationCommandPassed ValidationCommandStatus = "passed"
	// ValidationCommandFailed means generated behavior failed validation.
	ValidationCommandFailed ValidationCommandStatus = "failed"
	// ValidationCommandIncomplete means external test infrastructure was unavailable.
	ValidationCommandIncomplete ValidationCommandStatus = "incomplete"
)

// ValidationCommandResult records one attempted scaffold gate.
type ValidationCommandResult struct {
	Name   string                  `json:"name" yaml:"name"`
	Status ValidationCommandStatus `json:"status" yaml:"status"`
	Output string                  `json:"output,omitempty" yaml:"output,omitempty"`
}

type applicationValidator func(context.Context, string, []Command) ([]ValidationCommandResult, error)

// ApplicationWorkspace stages one new application in memory, validates it in
// an isolated sibling directory, and publishes the whole target atomically.
type ApplicationWorkspace struct {
	mu sync.Mutex

	manifest  Manifest
	plan      plan.Plan
	target    project.TargetInventory
	book      *book.Book
	staged    map[string]stagedMutation
	bytes     int64
	committed bool
	validator applicationValidator
}

// OpenApplicationWorkspace verifies plan, manifest, and current target state
// without creating the target or a staging directory.
func OpenApplicationWorkspace(
	ctx context.Context,
	manifest Manifest,
	value plan.Plan,
	target project.TargetInventory,
	selectedBook *book.Book,
) (*ApplicationWorkspace, error) {
	err := VerifyDigest(manifest)
	if err != nil {
		return nil, err
	}

	err = validateApplicationPreparation(value, target, selectedBook)
	if err != nil {
		return nil, err
	}

	if manifest.PlanDigest != value.Digest || manifest.SourceFingerprint != value.SourceFingerprint || manifest.TargetPath != target.Target {
		return nil, executionError("execution_identity_mismatch", "manifest", "manifest, plan, and target identities do not match")
	}

	if !samePreservedPaths(target.Preserved, manifest.PreservedPaths) {
		return nil, executionError("execution_target_mismatch", "preserved_paths", "manifest does not preserve the inspected target entries")
	}

	current, err := inspectApplicationTarget(ctx, target, selectedBook)
	if err != nil {
		return nil, err
	}

	err = validateApplicationTargetFreshness(value, current)
	if err != nil {
		return nil, err
	}

	return &ApplicationWorkspace{
		manifest: cloneManifest(manifest), plan: value, target: current, book: selectedBook,
		staged: make(map[string]stagedMutation, len(manifest.Edits)), validator: validateApplicationStaging,
	}, nil
}

// Stage validates one renderer mutation and retains it without touching the
// application target.
func (w *ApplicationWorkspace) Stage(mutation Mutation) (Change, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	edit, exists := manifestEdit(w.manifest.Edits, mutation.EditID)
	if !exists {
		return Change{}, executionError("execution_edit_undeclared", mutation.EditID, "mutation does not identify a manifest edit")
	}

	if _, exists := w.staged[mutation.EditID]; exists {
		return Change{}, executionError("execution_edit_already_staged", mutation.EditID, "edit is already staged")
	}

	if mutation.Kind != MutationCreate || edit.Kind != EditCreateFile || len(mutation.Anchor) != 0 || mutation.Position != "" {
		return Change{}, executionError("execution_mutation_invalid", edit.Target, "application scaffold edits require complete create mutations")
	}

	if int64(len(mutation.Content)) > MaximumStagedFileSize || w.bytes+int64(len(mutation.Content)) > MaximumStagedBytes {
		return Change{}, executionError("execution_limit_exceeded", edit.Target, "application staging limit exceeded")
	}

	err := validatePostconditions(edit, mutation.Content)
	if err != nil {
		return Change{}, err
	}

	w.staged[edit.ID] = stagedMutation{edit: edit, kind: MutationCreate, content: append([]byte{}, mutation.Content...), status: ChangeStaged}
	w.bytes += int64(len(mutation.Content))

	return stagedChange(edit, MutationCreate, targetSnapshot{digest: "missing"}, mutation.Content, ChangeStaged), nil
}

// Commit builds the complete application in an isolated directory and only
// then swaps it into the admitted target path.
func (w *ApplicationWorkspace) Commit(ctx context.Context) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.staged) != len(w.manifest.Edits) {
		return failedResult(w.manifest.Digest, nil), executionError("execution_stage_incomplete", "edits", "%d of %d manifest edits are staged", len(w.staged), len(w.manifest.Edits))
	}

	if w.committed {
		return w.verifyPublished()
	}

	current, err := inspectApplicationTarget(ctx, w.target, w.book)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	err = validateApplicationTargetFreshness(w.plan, current)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	staging, err := os.MkdirTemp(w.target.Parent, ".hatmax-scaffold-*")
	if err != nil {
		return failedResult(w.manifest.Digest, nil), executionError("execution_stage_failed", w.target.Parent, "%v", err)
	}
	defer os.RemoveAll(staging)

	if current.Exists {
		err = copyApplicationTree(current.Target, staging)
		if err != nil {
			return failedResult(w.manifest.Digest, nil), executionError("execution_stage_failed", current.Target, "%v", err)
		}
	}

	err = w.writeStaging(staging)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	conformance, err := CheckApplicationConformance(w.plan, w.manifest, staging)
	if err != nil || !conformance.Passed {
		if err == nil {
			err = executionError("execution_conformance_failed", "target", "staged scaffold violates its Book rules")
		}

		return failedResult(w.manifest.Digest, conformance.Diagnostics), err
	}

	validation, err := w.validator(ctx, staging, w.manifest.Commands)
	if err != nil {
		result := failedResult(w.manifest.Digest, []Diagnostic{executionDiagnostic("HMGEN-APPLICATION-VALIDATION", Edit{}, err.Error(), "generated application builds and tests", true)})
		result.Validation = validation

		return result, err
	}

	err = ctx.Err()
	if err != nil {
		return failedResult(w.manifest.Digest, nil), executionError("execution_cancelled", "target", "%v", err)
	}

	current, err = inspectApplicationTarget(ctx, w.target, w.book)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	err = validateApplicationTargetFreshness(w.plan, current)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	err = publishApplicationTarget(staging, current)
	if err != nil {
		return failedResult(w.manifest.Digest, nil), err
	}

	w.committed = true
	changes := w.applicationChanges(ChangeApplied)
	result := Result{ManifestDigest: w.manifest.Digest, Status: ExecutionApplied, Changes: changes, Diagnostics: []Diagnostic{}, Validation: validation}

	for _, command := range validation {
		if command.Status == ValidationCommandIncomplete {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Code: "HMGEN-VALIDATION-INCOMPLETE", Severity: SeverityWarning,
				Observed: command.Output, Expected: "declared test infrastructure is available", RepairableWithinPlan: false,
			})
		}
	}

	return result, nil
}

func (w *ApplicationWorkspace) writeStaging(root string) error {
	for _, edit := range w.manifest.Edits {
		staged := w.staged[edit.ID]

		target, err := secureTargetPath(root, edit.Target)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Dir(target), 0o755)
		if err != nil {
			return executionError("execution_stage_failed", edit.Target, "%v", err)
		}

		err = os.WriteFile(target, staged.content, 0o644)
		if err != nil {
			return executionError("execution_stage_failed", edit.Target, "%v", err)
		}
	}

	return nil
}

func (w *ApplicationWorkspace) verifyPublished() (Result, error) {
	for _, edit := range w.manifest.Edits {
		content, err := os.ReadFile(filepath.Join(w.target.Target, filepath.FromSlash(edit.Target)))
		if err != nil {
			return failedResult(w.manifest.Digest, nil), executionError("execution_committed_state_invalid", edit.Target, "%v", err)
		}

		if edit.Target != "go.sum" && contentDigest(content) != contentDigest(w.staged[edit.ID].content) {
			return failedResult(w.manifest.Digest, nil), executionError("execution_committed_state_invalid", edit.Target, "published content changed")
		}
	}

	return Result{ManifestDigest: w.manifest.Digest, Status: ExecutionAlreadySatisfied, Changes: w.applicationChanges(ChangeAlreadySatisfied), Diagnostics: []Diagnostic{}}, nil
}

func (w *ApplicationWorkspace) applicationChanges(status ChangeStatus) []Change {
	result := make([]Change, 0, len(w.manifest.Edits))
	for _, edit := range w.manifest.Edits {
		staged := w.staged[edit.ID]
		result = append(result, stagedChange(edit, MutationCreate, targetSnapshot{digest: "missing"}, staged.content, status))
	}

	return result
}

func inspectApplicationTarget(ctx context.Context, prior project.TargetInventory, selectedBook *book.Book) (project.TargetInventory, error) {
	current, err := project.InspectTarget(ctx, project.TargetRequest{
		Parent: prior.Parent, Target: prior.Target, PlannedPaths: prior.PlannedPaths, Book: selectedBook,
		Options: project.Options{MaximumFileSize: MaximumStagedFileSize},
	})
	if err != nil {
		return project.TargetInventory{}, executionError("execution_target_inspection_failed", prior.Target, "%v", err)
	}

	return current, nil
}

func validateApplicationTargetFreshness(value plan.Plan, current project.TargetInventory) error {
	fingerprint, err := current.Fingerprint(value.BookVersion)
	if err != nil {
		return executionError("execution_fingerprint_failed", "source_fingerprint", "%v", err)
	}

	freshness, err := plan.CheckFingerprint(value, fingerprint)
	if err != nil {
		return executionError("execution_plan_invalid", "plan", "%v", err)
	}

	if freshness.Stale {
		return executionError("execution_plan_stale", "source_fingerprint", "target state changed after planning")
	}

	return nil
}

func copyApplicationTree(source, target string) error {
	return filepath.WalkDir(source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(source, sourcePath)
		if err != nil || relative == "." {
			return err
		}

		targetPath := filepath.Join(target, relative)

		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.Type()&os.ModeSymlink != 0 {
			link, readErr := os.Readlink(sourcePath)
			if readErr != nil {
				return readErr
			}

			return os.Symlink(link, targetPath)
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported preserved entry %s", relative)
		}

		return copyApplicationFile(sourcePath, targetPath, info.Mode().Perm())
	})
}

func copyApplicationFile(source, target string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()

	if copyErr != nil {
		return copyErr
	}

	return closeErr
}

func publishApplicationTarget(staging string, target project.TargetInventory) error {
	backup := ""

	if target.Exists {
		file, err := os.CreateTemp(target.Parent, ".hatmax-backup-*")
		if err != nil {
			return executionError("execution_publish_failed", target.Target, "%v", err)
		}

		backup = file.Name()
		file.Close()
		os.Remove(backup)

		err = os.Rename(target.Target, backup)
		if err != nil {
			return executionError("execution_publish_failed", target.Target, "%v", err)
		}
	}

	err := os.Rename(staging, target.Target)
	if err != nil {
		if backup != "" {
			_ = os.Rename(backup, target.Target)
		}

		return executionError("execution_publish_failed", target.Target, "%v", err)
	}

	if backup != "" {
		err = os.RemoveAll(backup)
		if err != nil {
			return executionError("execution_cleanup_failed", backup, "%v", err)
		}
	}

	return syncDirectory(target.Parent)
}

func validateApplicationStaging(ctx context.Context, root string, commands []Command) ([]ValidationCommandResult, error) {
	results := make([]ValidationCommandResult, 0, len(commands))
	for _, command := range commands {
		process := exec.CommandContext(ctx, command.Args[0], command.Args[1:]...)
		process.Dir = filepath.Join(root, filepath.FromSlash(command.WorkingDirectory))

		process.Env = append(os.Environ(), "GOWORK=off")

		output := commandOutput{checkInfrastructure: command.Name == "validation.test"}
		process.Stdout = &output
		process.Stderr = &output

		err := process.Run()

		result := ValidationCommandResult{Name: command.Name, Status: ValidationCommandPassed}
		if err != nil {
			result.Output = strings.TrimSpace(output.String())
			result.Status = ValidationCommandFailed

			if command.Name == "validation.test" && output.infrastructureUnavailable {
				result.Status = ValidationCommandIncomplete
				results = append(results, result)

				continue
			}

			results = append(results, result)

			return results, executionError("execution_validation_failed", command.Name, "%v", err)
		}

		results = append(results, result)
	}

	return results, nil
}

func testInfrastructureUnavailable(output string) bool {
	normalized := strings.ToLower(output)
	markers := []string{
		"connection refused", "cannot connect to the docker daemon", "docker daemon is not running",
		"no such host", "testcontainers", "dial unix /var/run/docker.sock",
	}

	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return false
}
