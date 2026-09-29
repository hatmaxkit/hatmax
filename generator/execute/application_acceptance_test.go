//go:build acceptance

package execute

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestApplicationScaffoldAcceptance(t *testing.T) {
	value, target, selectedBook := applicationExecutionPlan(t, nil)
	manifest, err := PrepareApplication(value, target, selectedBook)
	if err != nil {
		t.Fatalf("PrepareApplication() error = %v", err)
	}

	mutations, err := RenderApplication(value, manifest)
	if err != nil {
		t.Fatalf("RenderApplication() error = %v", err)
	}

	workspace, err := OpenApplicationWorkspace(context.Background(), manifest, value, target, selectedBook)
	if err != nil {
		t.Fatalf("OpenApplicationWorkspace() error = %v", err)
	}

	for _, mutation := range mutations {
		if _, err := workspace.Stage(mutation); err != nil {
			t.Fatalf("Stage(%s) error = %v", mutation.EditID, err)
		}
	}

	result, err := workspace.Commit(context.Background())
	if err != nil {
		t.Fatalf("Commit() error = %v, result = %#v", err, result)
	}

	if result.Status != ExecutionApplied || len(result.Validation) != 3 {
		t.Fatalf("Commit() = %#v, want applied with module, build, and test validation", result)
	}

	for _, validation := range result.Validation {
		if validation.Status != ValidationCommandPassed {
			t.Errorf("validation %s = %q, want passed", validation.Name, validation.Status)
		}
	}

	command := exec.Command("go", "test", "-mod=readonly", "./...")
	command.Dir = target.Target
	command.Env = append(os.Environ(), "GOWORK=off")

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("published scaffold is not reproducible: %v\n%s", err, output)
	}
}
