package execute

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"hatmax.adrianpk.com/generator/project"
)

// ExecutionStatus describes the filesystem transaction outcome.
type ExecutionStatus string

const (
	// ExecutionApplied means every required target change was committed.
	ExecutionApplied ExecutionStatus = "applied"
	// ExecutionAlreadySatisfied means every target already had staged content.
	ExecutionAlreadySatisfied ExecutionStatus = "already_satisfied"
	// ExecutionFailed means no complete transaction was committed.
	ExecutionFailed ExecutionStatus = "failed"
)

// Result reports one atomic application attempt.
type Result struct {
	ManifestDigest string          `json:"manifest_digest" yaml:"manifest_digest"`
	Status         ExecutionStatus `json:"status" yaml:"status"`
	Changes        []Change        `json:"changes" yaml:"changes"`
	Diagnostics    []Diagnostic    `json:"diagnostics" yaml:"diagnostics"`
}

type commitHooks struct {
	beforeApply    func(index int, edit Edit) error
	beforeRollback func(index int, edit Edit) error
}

type preparedTarget struct {
	staged   stagedMutation
	target   string
	tempPath string
}

// Commit revalidates project state and applies every staged edit as one
// recoverable filesystem transaction. A failed application rolls back every
// target already changed by this workspace.
func (w *Workspace) Commit(ctx context.Context) (Result, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.staged) != len(w.manifest.Edits) {
		return failedResult(w.manifest.Digest, nil), executionError(
			"execution_stage_incomplete",
			"edits",
			"%d of %d manifest edits are staged",
			len(w.staged),
			len(w.manifest.Edits),
		)
	}

	if w.committed {
		return w.verifyCommittedState()
	}

	if result, err := w.revalidateProject(ctx); err != nil {
		return result, err
	}

	if result, err := w.revalidateSnapshots(); err != nil {
		return result, err
	}

	prepared, createdDirectories, err := w.prepareTargets()
	if err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-PREPARE", Edit{}, "temporary target preparation failed", "all targets prepared", false)
		cleanupPrepared(prepared)
		removeCreatedDirectories(createdDirectories)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_commit_failed", "edits", "%v", err)
	}

	applied := make([]preparedTarget, 0, len(prepared))
	for index := range prepared {
		item := prepared[index]
		if item.staged.status == ChangeAlreadySatisfied {
			continue
		}

		if result, changed, applyErr := w.applyPrepared(index, item); applyErr != nil {
			cleanupPrepared(prepared[index:])
			if changed {
				applied = append(applied, item)
			}

			return w.rollback(applied, createdDirectories, item.staged.edit, applyErr, result.Diagnostics)
		}

		prepared[index].tempPath = ""
		applied = append(applied, item)
	}

	cleanupPrepared(prepared)
	w.committed = true

	changes := w.resultChanges(ChangeApplied)
	status := ExecutionApplied
	if len(applied) == 0 {
		status = ExecutionAlreadySatisfied
	}

	return Result{
		ManifestDigest: w.manifest.Digest,
		Status:         status,
		Changes:        changes,
		Diagnostics:    []Diagnostic{},
	}, nil
}

func (w *Workspace) revalidateProject(ctx context.Context) (Result, error) {
	inventory, err := projectInventory(ctx, w.root)
	if err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-DRIFT", Edit{}, err.Error(), "project remains inspectable", false)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_project_inspection_failed", "project", "%v", err)
	}

	fingerprint, err := inventory.Fingerprint(w.fingerprintRequest)
	if err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-DRIFT", Edit{}, err.Error(), "project fingerprint can be recomputed", false)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_fingerprint_failed", "project_fingerprint", "%v", err)
	}

	if fingerprint.Value != w.manifest.ProjectFingerprint {
		diagnostic := executionDiagnostic(
			"HMGEN-EXEC-DRIFT",
			Edit{},
			fingerprint.Value,
			w.manifest.ProjectFingerprint,
			false,
		)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_plan_stale", "project_fingerprint", "relevant project state changed before commit")
	}

	return Result{}, nil
}

func projectInventory(ctx context.Context, root string) (project.Inventory, error) {
	return project.InspectWithOptions(ctx, root, project.Options{MaximumFileSize: MaximumStagedFileSize})
}

func (w *Workspace) revalidateSnapshots() (Result, error) {
	for _, edit := range w.manifest.Edits {
		current, err := readTargetState(w.root, edit.Target)
		if err != nil {
			diagnostic := executionDiagnostic("HMGEN-EXEC-CONFLICT", edit, err.Error(), "target remains readable", false)

			return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_conflict", edit.Target, "%v", err)
		}

		expected := w.snapshots[edit.ID]
		if !sameSnapshot(current, expected) {
			diagnostic := executionDiagnostic("HMGEN-EXEC-CONFLICT", edit, current.digest, expected.digest, false)

			return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_conflict", edit.Target, "target changed after workspace snapshot")
		}
	}

	return Result{}, nil
}

func (w *Workspace) prepareTargets() ([]preparedTarget, []string, error) {
	prepared := make([]preparedTarget, 0, len(w.manifest.Edits))
	createdDirectories := make([]string, 0)

	for _, edit := range w.manifest.Edits {
		staged := w.staged[edit.ID]
		item := preparedTarget{staged: staged}
		prepared = append(prepared, item)

		if staged.status == ChangeAlreadySatisfied {
			continue
		}

		target, err := secureTargetPath(w.root, edit.Target)
		if err != nil {
			return prepared, createdDirectories, err
		}

		created, err := ensureTargetDirectory(w.root, filepath.Dir(target))
		if err != nil {
			return prepared, append(createdDirectories, created...), err
		}
		createdDirectories = append(createdDirectories, created...)

		target, err = secureTargetPath(w.root, edit.Target)
		if err != nil {
			return prepared, createdDirectories, err
		}

		mode := fs.FileMode(0o644)
		if snapshot := w.snapshots[edit.ID]; snapshot.exists {
			mode = snapshot.mode
		}

		temporary, err := os.CreateTemp(filepath.Dir(target), ".hatmax-stage-*")
		if err != nil {
			return prepared, createdDirectories, fmt.Errorf("create temporary target for %s: %w", edit.Target, err)
		}

		tempPath := temporary.Name()
		prepared[len(prepared)-1].target = target
		prepared[len(prepared)-1].tempPath = tempPath

		if err = writeTemporary(temporary, staged.content, mode); err != nil {
			return prepared, createdDirectories, fmt.Errorf("prepare %s: %w", edit.Target, err)
		}
	}

	return prepared, uniqueDeepestFirst(createdDirectories), nil
}

func (w *Workspace) applyPrepared(index int, item preparedTarget) (Result, bool, error) {
	current, err := readTargetState(w.root, item.staged.edit.Target)
	if err != nil || !sameSnapshot(current, w.snapshots[item.staged.edit.ID]) {
		observed := "unreadable"
		if err == nil {
			observed = current.digest
		}

		diagnostic := executionDiagnostic(
			"HMGEN-EXEC-CONFLICT",
			item.staged.edit,
			observed,
			w.snapshots[item.staged.edit.ID].digest,
			false,
		)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), false, executionError("execution_conflict", item.staged.edit.Target, "target changed during commit")
	}

	if w.hooks.beforeApply != nil {
		if err = w.hooks.beforeApply(index, item.staged.edit); err != nil {
			diagnostic := executionDiagnostic("HMGEN-EXEC-APPLY", item.staged.edit, err.Error(), "target applied", true)

			return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), false, err
		}
	}

	if err = os.Rename(item.tempPath, item.target); err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-APPLY", item.staged.edit, err.Error(), "target applied", true)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), false, err
	}

	if err = syncDirectory(filepath.Dir(item.target)); err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-APPLY", item.staged.edit, err.Error(), "target directory synchronized", true)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), true, err
	}

	current, err = readTargetState(w.root, item.staged.edit.Target)
	if err != nil || current.digest != contentDigest(item.staged.content) {
		diagnostic := executionDiagnostic("HMGEN-EXEC-APPLY", item.staged.edit, "committed content differs", contentDigest(item.staged.content), true)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), true, executionError("execution_postcondition_failed", item.staged.edit.Target, "committed target does not match staged content")
	}

	if err = validatePostconditions(item.staged.edit, current.content); err != nil {
		diagnostic := executionDiagnostic("HMGEN-EXEC-APPLY", item.staged.edit, err.Error(), "postconditions satisfied", true)

		return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), true, err
	}

	return Result{}, true, nil
}

func (w *Workspace) rollback(
	applied []preparedTarget,
	createdDirectories []string,
	failedEdit Edit,
	cause error,
	diagnostics []Diagnostic,
) (Result, error) {
	recoveryPaths := make([]string, 0)

	for index := len(applied) - 1; index >= 0; index-- {
		item := applied[index]
		current, err := readTargetState(w.root, item.staged.edit.Target)
		if err != nil || current.digest != contentDigest(item.staged.content) {
			recoveryPaths = append(recoveryPaths, item.staged.edit.Target)
			continue
		}

		if w.hooks.beforeRollback != nil {
			if err = w.hooks.beforeRollback(index, item.staged.edit); err != nil {
				recoveryPaths = append(recoveryPaths, item.staged.edit.Target)
				continue
			}
		}

		if err = restoreSnapshot(item.target, w.snapshots[item.staged.edit.ID]); err != nil {
			recoveryPaths = append(recoveryPaths, item.staged.edit.Target)
		}
	}

	removeCreatedDirectories(createdDirectories)

	if len(recoveryPaths) > 0 {
		sort.Strings(recoveryPaths)
		diagnostics = append(diagnostics, executionDiagnostic(
			"HMGEN-EXEC-ROLLBACK-CONFLICT",
			failedEdit,
			fmt.Sprintf("manual recovery required for %v", recoveryPaths),
			"all applied targets restored",
			false,
		))

		return failedResult(w.manifest.Digest, diagnostics), executionError("execution_conflict", recoveryPaths[0], "rollback would overwrite a concurrent change")
	}

	diagnostics = append(diagnostics, executionDiagnostic(
		"HMGEN-EXEC-ROLLED-BACK",
		failedEdit,
		cause.Error(),
		"transaction committed",
		true,
	))

	return failedResult(w.manifest.Digest, diagnostics), executionError("execution_commit_failed", failedEdit.Target, "transaction rolled back: %v", cause)
}

func (w *Workspace) verifyCommittedState() (Result, error) {
	for _, edit := range w.manifest.Edits {
		current, err := readTargetState(w.root, edit.Target)
		if err != nil || current.digest != contentDigest(w.staged[edit.ID].content) {
			observed := "unreadable"
			if err == nil {
				observed = current.digest
			}

			diagnostic := executionDiagnostic("HMGEN-EXEC-CONFLICT", edit, observed, contentDigest(w.staged[edit.ID].content), false)

			return failedResult(w.manifest.Digest, []Diagnostic{diagnostic}), executionError("execution_conflict", edit.Target, "committed target changed after execution")
		}
	}

	return Result{
		ManifestDigest: w.manifest.Digest,
		Status:         ExecutionAlreadySatisfied,
		Changes:        w.resultChanges(ChangeAlreadySatisfied),
		Diagnostics:    []Diagnostic{},
	}, nil
}

func (w *Workspace) resultChanges(appliedStatus ChangeStatus) []Change {
	result := make([]Change, 0, len(w.manifest.Edits))
	for _, edit := range w.manifest.Edits {
		staged := w.staged[edit.ID]
		status := staged.status
		if status == ChangeStaged {
			status = appliedStatus
		}

		result = append(result, stagedChange(edit, staged.kind, w.snapshots[edit.ID], staged.content, status))
	}

	return result
}

func readTargetState(root, target string) (targetSnapshot, error) {
	absolute, err := secureTargetPath(root, target)
	if err != nil {
		return targetSnapshot{}, err
	}

	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) {
		return targetSnapshot{digest: "missing"}, nil
	}

	if err != nil {
		return targetSnapshot{}, err
	}

	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return targetSnapshot{}, executionError("execution_target_unsupported", target, "target must be a regular file")
	}

	if info.Size() > MaximumStagedFileSize {
		return targetSnapshot{}, executionError("execution_limit_exceeded", target, "target exceeds the %d-byte workspace limit", MaximumStagedFileSize)
	}

	content, err := os.ReadFile(absolute)
	if err != nil {
		return targetSnapshot{}, err
	}

	return targetSnapshot{
		exists:  true,
		content: content,
		mode:    info.Mode().Perm(),
		digest:  contentDigest(content),
	}, nil
}

func sameSnapshot(left, right targetSnapshot) bool {
	return left.exists == right.exists && left.digest == right.digest && (!left.exists || left.mode == right.mode)
}

func ensureTargetDirectory(root, directory string) ([]string, error) {
	missing := make([]string, 0)
	current := directory
	for current != root {
		_, err := os.Lstat(current)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return missing, err
		}

		missing = append(missing, current)
		current = filepath.Dir(current)
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return missing, err
	}

	return missing, nil
}

func writeTemporary(file *os.File, content []byte, mode fs.FileMode) error {
	_, writeErr := file.Write(content)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if writeErr == nil {
		writeErr = file.Chmod(mode)
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}

	return closeErr
}

func restoreSnapshot(target string, snapshot targetSnapshot) error {
	if !snapshot.exists {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		return syncDirectory(filepath.Dir(target))
	}

	temporary, err := os.CreateTemp(filepath.Dir(target), ".hatmax-rollback-*")
	if err != nil {
		return err
	}

	tempPath := temporary.Name()
	if err = writeTemporary(temporary, snapshot.content, snapshot.mode); err != nil {
		_ = os.Remove(tempPath)

		return err
	}

	if err = os.Rename(tempPath, target); err != nil {
		_ = os.Remove(tempPath)

		return err
	}

	return syncDirectory(filepath.Dir(target))
}

func syncDirectory(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}

	syncErr := file.Sync()
	closeErr := file.Close()
	if syncErr != nil {
		return syncErr
	}

	return closeErr
}

func cleanupPrepared(values []preparedTarget) {
	for _, value := range values {
		if value.tempPath != "" {
			_ = os.Remove(value.tempPath)
		}
	}
}

func uniqueDeepestFirst(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Slice(result, func(left, right int) bool {
		return len(result[left]) > len(result[right])
	})

	return result
}

func removeCreatedDirectories(values []string) {
	for _, value := range uniqueDeepestFirst(values) {
		_ = os.Remove(value)
	}
}

func executionDiagnostic(code string, edit Edit, observed, expected string, repairable bool) Diagnostic {
	rule := ""
	if len(edit.Obligations) > 0 && len(edit.Obligations[0].Rules) > 0 {
		rule = edit.Obligations[0].Rules[0]
	}

	return Diagnostic{
		Code:                 code,
		Rule:                 rule,
		Severity:             SeverityError,
		Surface:              edit.Surface,
		Location:             edit.Target,
		Observed:             observed,
		Expected:             expected,
		RepairableWithinPlan: repairable,
	}
}

func failedResult(digest string, diagnostics []Diagnostic) Result {
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}

	return Result{
		ManifestDigest: digest,
		Status:         ExecutionFailed,
		Changes:        []Change{},
		Diagnostics:    diagnostics,
	}
}
