package execute

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/intent"
)

func TestCommittedWorkspaceReapplicationIsIdempotent(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)

	first, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("first Commit() error = %v", err)
	}

	before := snapshotTree(t, root)

	second, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("second Commit() error = %v", err)
	}

	if second.Status != ExecutionAlreadySatisfied || len(second.Changes) != len(first.Changes) {
		t.Fatalf("second Commit() = %#v, want already satisfied", second)
	}

	for _, change := range second.Changes {
		if change.Status != ChangeAlreadySatisfied {
			t.Errorf("second Commit() change = %#v, want already satisfied", change)
		}
	}

	after := snapshotTree(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Error("idempotent Commit() changed the project")
	}
}

func TestInsertionReportsAlreadySatisfiedOrSemanticConflict(t *testing.T) {
	const insertion = "func buildInvoice() {}\n\n"

	t.Run("already adjacent", func(t *testing.T) {
		root := copyExecutionFixture(t)
		replaceExecutionText(t, root, "main.go", "func main() {", insertion+"func main() {")
		workspace := createFeatureWorkspaceAt(t, root)

		change, err := workspace.Stage(Mutation{
			EditID:   "create.wiring",
			Kind:     MutationInsert,
			Anchor:   []byte("func main() {"),
			Position: InsertBefore,
			Content:  []byte(insertion),
		})
		if err != nil {
			t.Fatalf("Stage() error = %v", err)
		}

		if change.Status != ChangeAlreadySatisfied {
			t.Errorf("Stage() status = %q, want %q", change.Status, ChangeAlreadySatisfied)
		}
	})

	t.Run("content elsewhere", func(t *testing.T) {
		root := copyExecutionFixture(t)
		appendExecutionFile(t, root, "main.go", "\n"+insertion)
		workspace := createFeatureWorkspaceAt(t, root)

		_, err := workspace.Stage(Mutation{
			EditID:   "create.wiring",
			Kind:     MutationInsert,
			Anchor:   []byte("func main() {"),
			Position: InsertBefore,
			Content:  []byte(insertion),
		})
		requireExecutionCode(t, err, "execution_semantic_conflict")
	})
}

func TestFailedCommitRestoresFixtureExactly(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)
	before := snapshotTree(t, root)

	workspace.hooks.beforeApply = func(index int, _ Edit) error {
		if index == 2 {
			return errors.New("injected application failure")
		}

		return nil
	}

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_commit_failed")

	if result.Status != ExecutionFailed || !hasDiagnostic(result.Diagnostics, "HMGEN-EXEC-ROLLED-BACK") {
		t.Errorf("Commit() = %#v, want rolled-back failure", result)
	}

	after := snapshotTree(t, root)
	if !reflect.DeepEqual(after, before) {
		t.Errorf("failed Commit() changed fixture\nbefore: %#v\nafter: %#v", before, after)
	}
}

func TestRollbackRefusesToOverwriteConcurrentChange(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)

	workspace.hooks.beforeApply = func(index int, _ Edit) error {
		if index != 2 {
			return nil
		}

		writeExecutionFile(t, root, "db/migrations/002-invoice.sql", "-- external replacement\n")

		return errors.New("injected failure after concurrent change")
	}

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_conflict")

	if !hasDiagnostic(result.Diagnostics, "HMGEN-EXEC-ROLLBACK-CONFLICT") {
		t.Errorf("Commit() diagnostics = %#v, want rollback conflict", result.Diagnostics)
	}

	content, readErr := os.ReadFile(filepath.Join(root, "db", "migrations", "002-invoice.sql"))
	if readErr != nil {
		t.Fatalf("read concurrent target: %v", readErr)
	}

	if string(content) != "-- external replacement\n" {
		t.Errorf("rollback overwrote concurrent content: %q", content)
	}
}

func TestCommitDetectsTargetChangeDuringApplication(t *testing.T) {
	workspace, root := createFeatureWorkspace(t)
	stageCreateFeature(t, workspace, root)

	workspace.hooks.beforeApply = func(index int, edit Edit) error {
		if index == 2 {
			writeExecutionFile(t, root, edit.Target, "external content\n")
		}

		return nil
	}

	result, err := workspace.Commit(context.Background())
	requireExecutionCode(t, err, "execution_commit_failed")

	if !hasDiagnostic(result.Diagnostics, "HMGEN-EXEC-CONFLICT") || !hasDiagnostic(result.Diagnostics, "HMGEN-EXEC-ROLLED-BACK") {
		t.Errorf("Commit() diagnostics = %#v, want conflict and rollback", result.Diagnostics)
	}

	_, statErr := os.Stat(filepath.Join(root, "db", "migrations", "002-invoice.sql"))
	if !os.IsNotExist(statErr) {
		t.Errorf("earlier applied target was not rolled back, stat error = %v", statErr)
	}
}

func createFeatureWorkspaceAt(t *testing.T, root string) *Workspace {
	t.Helper()

	value, inventory, selectedBook := executionPlan(t, root, intent.OperationCreateFeature, intent.Domain{
		Entity: "Invoice",
		Route:  "/invoices",
		Fields: []intent.Field{{Name: "number", Type: "string"}},
	}, []string{"postgres_persistence"})

	manifest, err := Prepare(value, inventory, selectedBook)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	workspace, err := OpenWorkspace(context.Background(), manifest, value, inventory)
	if err != nil {
		t.Fatalf("OpenWorkspace() error = %v", err)
	}

	return workspace
}

func replaceExecutionText(t *testing.T, root, path, old, replacement string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))

	content, err := os.ReadFile(absolute)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	updated := strings.Replace(string(content), old, replacement, 1)
	if updated == string(content) {
		t.Fatalf("replace %q: marker %q not found", path, old)
	}

	err = os.WriteFile(absolute, []byte(updated), 0o644)
	if err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()

	result := make(map[string]string)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if relative == "." {
			return nil
		}

		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			result[relative] = "directory"

			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		result[relative] = contentDigest(content)

		return nil
	})
	if err != nil {
		t.Fatalf("snapshot tree: %v", err)
	}

	return result
}

func hasDiagnostic(values []Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}

	return false
}
