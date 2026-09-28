package plan

import (
	"testing"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/project"
)

func TestCheckFingerprintAcceptsExactRelevantState(t *testing.T) {
	value, context := expandedTestPlan(t)

	result, err := CheckFingerprint(value, context.Fingerprint)
	if err != nil {
		t.Fatalf("CheckFingerprint() error = %v", err)
	}

	if result.Stale || len(result.Changes) != 0 || len(result.Diagnostics) != 0 {
		t.Errorf("CheckFingerprint() = %#v, want fresh result", result)
	}
}

func TestCheckFingerprintReportsRelevantDrift(t *testing.T) {
	value, context := expandedTestPlan(t)
	current := context.Fingerprint
	current.Value = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	current.Observations = cloneObservations(current.Observations)
	current.Observations[0].Digest = "changed-digest"

	result, err := CheckFingerprint(value, current)
	if err != nil {
		t.Fatalf("CheckFingerprint() error = %v", err)
	}

	if !result.Stale || !hasPlanDiagnostic(result.Diagnostics, "HMGEN-PLAN-STALE") {
		t.Fatalf("CheckFingerprint() = %#v, want stale diagnostic", result)
	}

	if len(result.Changes) != 1 || result.Changes[0].Kind != project.ChangeModified || result.Changes[0].Path != "go.mod" {
		t.Errorf("Changes = %#v, want modified go.mod observation", result.Changes)
	}
}

func TestCheckFingerprintReportsBookDriftWithoutInventingFileChanges(t *testing.T) {
	value, context := expandedTestPlan(t)
	current := context.Fingerprint
	current.BookVersion = 2

	result, err := CheckFingerprint(value, current)
	if err != nil {
		t.Fatalf("CheckFingerprint() error = %v", err)
	}

	if !result.Stale || len(result.Changes) != 0 || !hasPlanDiagnostic(result.Diagnostics, "HMGEN-PLAN-STALE") {
		t.Errorf("CheckFingerprint() = %#v, want stale Book context without observation changes", result)
	}
}

func TestCheckFingerprintRequiresSealedPlan(t *testing.T) {
	_, context := admittedExpansion(t, intent.OperationAddField, []string{"postgres_persistence"})

	requirePlanCode(t, func() error {
		_, err := CheckFingerprint(validTestPlan(), context.Fingerprint)

		return err
	}(), "plan_digest_missing")
}

func expandedTestPlan(t *testing.T) (Plan, ExpansionContext) {
	t.Helper()

	admission, context := admittedExpansion(t, intent.OperationAddField, []string{"postgres_persistence"})

	value, err := Expand(admission, context)
	if err != nil {
		t.Fatalf("Expand() error = %v", err)
	}

	return value, context
}

func hasPlanDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}

	return false
}
