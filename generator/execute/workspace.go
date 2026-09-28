package execute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

const (
	// MaximumWorkspaceEdits bounds one filesystem transaction.
	MaximumWorkspaceEdits = 1024
	// MaximumStagedFileSize bounds one original or staged target.
	MaximumStagedFileSize = int64(4 << 20)
	// MaximumStagedBytes bounds all staged target content in one workspace.
	MaximumStagedBytes = int64(64 << 20)
)

// MutationKind identifies one bounded in-memory transformation.
type MutationKind string

const (
	// MutationCreate supplies complete content for an absent target.
	MutationCreate MutationKind = "create"
	// MutationReplace supplies complete replacement content for a target.
	MutationReplace MutationKind = "replace"
	// MutationInsert adds content next to one exact semantic anchor.
	MutationInsert MutationKind = "insert"
)

// InsertionPosition identifies which side of an anchor receives content.
type InsertionPosition string

const (
	// InsertBefore places content immediately before its anchor.
	InsertBefore InsertionPosition = "before"
	// InsertAfter places content immediately after its anchor.
	InsertAfter InsertionPosition = "after"
)

// Mutation contains renderer output for one manifest-owned edit. The caller
// selects an edit ID, never a filesystem path.
type Mutation struct {
	EditID   string
	Kind     MutationKind
	Content  []byte
	Anchor   []byte
	Position InsertionPosition
}

// ChangeStatus describes the disposition of one staged or applied edit.
type ChangeStatus string

const (
	// ChangeStaged means a target differs from its desired staged content.
	ChangeStaged ChangeStatus = "staged"
	// ChangeApplied means staged content was committed to its target.
	ChangeApplied ChangeStatus = "applied"
	// ChangeAlreadySatisfied means the target already has the desired state.
	ChangeAlreadySatisfied ChangeStatus = "already_satisfied"
)

// Change reports one bounded target transformation without exposing content.
type Change struct {
	EditID       string       `json:"edit_id" yaml:"edit_id"`
	Target       string       `json:"target" yaml:"target"`
	Kind         MutationKind `json:"kind" yaml:"kind"`
	Status       ChangeStatus `json:"status" yaml:"status"`
	BeforeDigest string       `json:"before_digest" yaml:"before_digest"`
	AfterDigest  string       `json:"after_digest" yaml:"after_digest"`
}

type targetSnapshot struct {
	exists  bool
	content []byte
	mode    fs.FileMode
	digest  string
}

type stagedMutation struct {
	edit    Edit
	kind    MutationKind
	content []byte
	status  ChangeStatus
}

// Workspace owns the snapshots and staged content for one manifest. It does
// not modify the project until Commit is called.
type Workspace struct {
	mu sync.Mutex

	root               string
	manifest           Manifest
	fingerprintRequest project.FingerprintRequest
	snapshots          map[string]targetSnapshot
	staged             map[string]stagedMutation
	stagedBytes        int64
}

// OpenWorkspace verifies the sealed execution identities against current
// project state and snapshots every admitted target without modifying it.
func OpenWorkspace(
	ctx context.Context,
	manifest Manifest,
	value plan.Plan,
	inventory project.Inventory,
) (*Workspace, error) {
	if err := VerifyDigest(manifest); err != nil {
		return nil, err
	}

	if err := plan.VerifyDigest(value); err != nil {
		return nil, executionError("execution_plan_invalid", "plan", "%v", err)
	}

	if manifest.PlanDigest != value.Digest || manifest.ProjectFingerprint != value.ProjectFingerprint {
		return nil, executionError("execution_identity_mismatch", "manifest", "manifest and plan identities do not match")
	}

	if len(manifest.Edits) > MaximumWorkspaceEdits {
		return nil, executionError("execution_limit_exceeded", "edits", "manifest contains more than %d edits", MaximumWorkspaceEdits)
	}

	current, err := project.InspectWithOptions(ctx, inventory.Root, project.Options{
		MaximumFileSize: MaximumStagedFileSize,
	})
	if err != nil {
		return nil, executionError("execution_project_inspection_failed", "project", "%v", err)
	}

	if err = validateFreshness(value, current); err != nil {
		return nil, err
	}

	result := &Workspace{
		root:     current.Root,
		manifest: cloneManifest(manifest),
		fingerprintRequest: project.FingerprintRequest{
			BookVersion:          value.BookVersion,
			SelectedPaths:        append([]string{}, value.FingerprintInputs.SelectedPaths...),
			SelectedDependencies: append([]string{}, value.FingerprintInputs.SelectedDependencies...),
			PlannedSurfaces:      append([]string{}, value.FingerprintInputs.PlannedSurfaces...),
		},
		snapshots: make(map[string]targetSnapshot, len(manifest.Edits)),
		staged:    make(map[string]stagedMutation, len(manifest.Edits)),
	}

	for _, edit := range manifest.Edits {
		snapshot, snapshotErr := captureTarget(result.root, edit)
		if snapshotErr != nil {
			return nil, snapshotErr
		}

		result.snapshots[edit.ID] = snapshot
	}

	return result, nil
}

func captureTarget(root string, edit Edit) (targetSnapshot, error) {
	target, err := secureTargetPath(root, edit.Target)
	if err != nil {
		return targetSnapshot{}, err
	}

	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		snapshot := targetSnapshot{digest: "missing"}
		if conditionPresent(edit.Preconditions, ConditionPathPresent) || conditionPresent(edit.Preconditions, ConditionPathDigest) {
			return targetSnapshot{}, executionError("execution_precondition_failed", edit.Target, "target must exist")
		}

		return snapshot, nil
	}

	if err != nil {
		return targetSnapshot{}, executionError("execution_snapshot_failed", edit.Target, "%v", err)
	}

	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return targetSnapshot{}, executionError("execution_target_unsupported", edit.Target, "target must be a regular file")
	}

	if info.Size() > MaximumStagedFileSize {
		return targetSnapshot{}, executionError("execution_limit_exceeded", edit.Target, "target exceeds the %d-byte workspace limit", MaximumStagedFileSize)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		return targetSnapshot{}, executionError("execution_snapshot_failed", edit.Target, "%v", err)
	}

	snapshot := targetSnapshot{
		exists:  true,
		content: append([]byte{}, content...),
		mode:    info.Mode().Perm(),
		digest:  contentDigest(content),
	}

	if conditionPresent(edit.Preconditions, ConditionPathAbsent) {
		return targetSnapshot{}, executionError("execution_precondition_failed", edit.Target, "target must be absent")
	}

	for _, condition := range edit.Preconditions {
		if condition.Kind == ConditionPathDigest && condition.Value != snapshot.digest {
			return targetSnapshot{}, executionError("execution_precondition_failed", edit.Target, "target content does not match its prepared digest")
		}
	}

	return snapshot, nil
}

func secureTargetPath(root, target string) (string, error) {
	normalized, err := normalizeRelativePath(target)
	if err != nil || normalized != target {
		return "", executionError("execution_target_invalid", target, "target must be a clean project-relative path")
	}

	absolute := filepath.Join(root, filepath.FromSlash(target))
	parent := filepath.Dir(absolute)
	for parent != root {
		info, statErr := os.Lstat(parent)
		if errors.Is(statErr, os.ErrNotExist) {
			parent = filepath.Dir(parent)
			continue
		}

		if statErr != nil {
			return "", executionError("execution_target_invalid", target, "%v", statErr)
		}

		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", executionError("execution_target_unsupported", target, "target parent must be a project directory")
		}

		parent = filepath.Dir(parent)
	}

	return absolute, nil
}

func contentDigest(content []byte) string {
	digest := sha256.Sum256(content)

	return "sha256:" + hex.EncodeToString(digest[:])
}

func conditionPresent(values []Condition, expected ConditionKind) bool {
	for _, value := range values {
		if value.Kind == expected {
			return true
		}
	}

	return false
}

func cloneManifest(value Manifest) Manifest {
	result := value
	result.AllowedSurfaces = append([]string{}, value.AllowedSurfaces...)
	result.Edits = make([]Edit, len(value.Edits))
	for index, edit := range value.Edits {
		result.Edits[index] = edit
		result.Edits[index].Obligations = make([]Obligation, len(edit.Obligations))
		for obligationIndex, obligation := range edit.Obligations {
			result.Edits[index].Obligations[obligationIndex] = obligation
			result.Edits[index].Obligations[obligationIndex].Rules = append([]string{}, obligation.Rules...)
		}
		result.Edits[index].DependsOn = append([]string{}, edit.DependsOn...)
		result.Edits[index].Preconditions = append([]Condition{}, edit.Preconditions...)
		result.Edits[index].Postconditions = append([]Condition{}, edit.Postconditions...)
		result.Edits[index].Slots = append([]ImplementationSlot{}, edit.Slots...)
	}
	result.Commands = make([]Command, len(value.Commands))
	for index, command := range value.Commands {
		result.Commands[index] = command
		result.Commands[index].Args = append([]string{}, command.Args...)
	}

	return result
}
