package hatmaxcli

import (
	"bytes"
	"strings"
	"testing"

	"hatmax.adrianpk.com/generator/eval"
	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/interaction"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

func TestWriteResultReportsBoundedExecutionEvidence(t *testing.T) {
	result := interaction.Result{
		Outcome: interaction.OutcomeExecutionFailed,
		Plan: &plan.Plan{
			Intent:             intent.OperationCreateFeature,
			Feature:            "invoice",
			Digest:             "sha256:plan",
			ProjectFingerprint: "sha256:project",
			HatmaxVersion:      "v0.4.0",
			BookVersion:        1,
			AffectedSurfaces:   []string{"handler", "templates"},
		},
		Provenance: interaction.Provenance{Interpretations: []eval.Provenance{{
			Adapter:         "codex_app_server",
			ContractVersion: 1,
			BackendVersion:  "codex-cli 1.2.3",
			ProtocolVersion: "2",
			ModelSelection:  eval.ModelBackendDefault,
			EffectiveModel:  "gpt-example",
			RuntimeReused:   true,
			ThreadReused:    false,
			Timing:          eval.TimingUnderTenSeconds,
		}}},
		Manifest: &execute.Manifest{
			AllowedSurfaces: []string{"handler", "templates"},
			Edits: []execute.Edit{
				{ID: "handler", Surface: "handler"},
				{ID: "templates", Surface: "templates"},
			},
		},
		Execution: &execute.Result{Changes: []execute.Change{
			{EditID: "handler", Target: "internal/invoice/handler.go", Kind: execute.MutationCreate, Status: execute.ChangeApplied},
			{EditID: "templates", Target: "internal/invoice/templates/index.html", Kind: execute.MutationCreate, Status: execute.ChangeAlreadySatisfied},
		}},
		Report: &execute.ExecutionReport{
			Conformance: execute.ConformanceResult{Passed: true},
			Commands: []execute.CommandEvidence{{
				Name:             "check",
				Kind:             project.CommandValidation,
				Args:             []string{"make", "check"},
				WorkingDirectory: ".",
				ExitCode:         7,
				Output:           "first line\nsecond line\n",
			}},
			Repairs:  []string{"formatted generated source"},
			Warnings: []string{"validation retained generated files"},
		},
		Diagnostics: []interaction.Diagnostic{{
			Code:    "HMGEN-EXECUTION-VALIDATION-FAILED",
			Phase:   interaction.PhaseValidation,
			Field:   "check",
			Message: "repository validation failed",
		}},
	}

	var output bytes.Buffer

	err := writeResult(&output, result)
	if err != nil {
		t.Fatalf("writeResult() error = %v", err)
	}

	for _, expected := range []string{
		"Outcome: execution_failed",
		"Intent: create_feature",
		"Affected surfaces: [\"handler\",\"templates\"]",
		"Adapter: codex_app_server",
		"Runtime reused: true",
		"applied create internal/invoice/handler.go",
		"Changed surfaces: [\"handler\"]",
		"Unchanged surfaces: [\"templates\"]",
		"Conformance passed: true",
		"Command check: kind=validation cwd=. args=[\"make\",\"check\"] exit=7",
		"Output: first line\n    second line",
		"Repair: formatted generated source",
		"Warning: validation retained generated files",
		"HMGEN-EXECUTION-VALIDATION-FAILED phase=validation field=check: repository validation failed",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output does not contain %q:\n%s", expected, output.String())
		}
	}

	for _, forbidden := range []string{"reasoning", "auth", "raw_event"} {
		if strings.Contains(output.String(), forbidden) {
			t.Errorf("output contains forbidden backend detail %q:\n%s", forbidden, output.String())
		}
	}
}

func TestWriteResultReportsClarificationAndProjectDrift(t *testing.T) {
	result := interaction.Result{
		Outcome: interaction.OutcomePlanStale,
		Clarifications: []intent.Clarification{{
			Field:    "domain.route",
			Question: "Which route should expose invoices?",
		}},
		FreshnessChanges: []project.Change{{
			Kind:    project.ChangeModified,
			Class:   project.ObservationPlannedSurface,
			Path:    "cmd/app/main.go",
			Surface: "wiring",
		}},
	}

	var output bytes.Buffer

	err := writeResult(&output, result)
	if err != nil {
		t.Fatalf("writeResult() error = %v", err)
	}

	for _, expected := range []string{
		"Pending clarifications:\n  domain.route: Which route should expose invoices?",
		"Project drift:\n  modified planned_surface path=cmd/app/main.go surface=wiring",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output does not contain %q:\n%s", expected, output.String())
		}
	}
}
