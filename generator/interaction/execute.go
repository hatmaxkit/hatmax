package interaction

import (
	"context"
	"errors"
	"fmt"

	"hatmax.adrianpk.com/generator/execute"
	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
	"hatmax.adrianpk.com/generator/project"
)

// Run coordinates one complete interaction from project inspection through
// post-commit conformance and repository validation.
func (c *Coordinator) Run(ctx context.Context, root, prompt string) Result {
	prepared, terminal := c.prepareApproved(ctx, root, prompt)
	if terminal != nil {
		return *terminal
	}

	result := prepared.result
	result.State = StateExecuting

	manifest, err := execute.Prepare(prepared.plan, prepared.inventory, prepared.book)
	if err != nil {
		return *terminalFailure(result, OutcomeExecutionFailed, PhasePreparation, "HMGEN-EXECUTION-PREPARATION-FAILED", err)
	}

	result.Manifest = &manifest

	mutations, err := renderPlan(prepared.plan, manifest, prepared.inventory)
	if err != nil {
		return *terminalFailure(result, OutcomeExecutionFailed, PhaseRendering, "HMGEN-EXECUTION-RENDERING-FAILED", err)
	}

	workspace, err := execute.OpenWorkspace(ctx, manifest, prepared.plan, prepared.inventory)
	if err != nil {
		return *terminalFailure(result, OutcomeExecutionFailed, PhaseApplication, "HMGEN-WORKSPACE-OPEN-FAILED", err)
	}

	for _, mutation := range mutations {
		_, err = workspace.Stage(mutation)
		if err != nil {
			return *terminalFailure(result, OutcomeExecutionFailed, PhaseApplication, "HMGEN-EXECUTION-STAGE-FAILED", err)
		}
	}

	executionResult, err := workspace.Commit(ctx)
	result.Execution = &executionResult
	result.Diagnostics = append(result.Diagnostics, executionDiagnostics(executionResult.Diagnostics, PhaseApplication)...)

	if err != nil {
		return *terminalFailure(result, cancelledExecutionOutcome(ctx), PhaseApplication, "HMGEN-EXECUTION-COMMIT-FAILED", err)
	}

	result.RetainedChanges = retainedChanges(executionResult.Changes)
	result.State = StateValidating

	currentInventory, err := project.InspectWithOptions(ctx, prepared.inventory.Root, c.inspectionOptions)
	if err != nil {
		return *terminalFailure(result, OutcomeExecutionFailed, PhaseValidation, "HMGEN-VALIDATION-INSPECTION-FAILED", err)
	}

	report, err := execute.ValidateExecution(ctx, prepared.plan, manifest, currentInventory)
	result.Report = &report
	result.Diagnostics = append(result.Diagnostics, executionDiagnostics(report.Conformance.Diagnostics, PhaseValidation)...)

	if err != nil {
		return *terminalFailure(result, cancelledExecutionOutcome(ctx), PhaseValidation, "HMGEN-EXECUTION-VALIDATION-FAILED", err)
	}

	result.State = StateFinished
	result.Outcome = OutcomeCompleted

	return result
}

func renderPlan(value plan.Plan, manifest execute.Manifest, inventory project.Inventory) ([]execute.Mutation, error) {
	mutations := make([]execute.Mutation, 0, len(manifest.Edits))

	switch value.Intent {
	case intent.OperationCreateFeature:
		implementation, err := execute.RenderCreateFeature(value, manifest, inventory)
		if err != nil {
			return nil, err
		}

		mutations = append(mutations, implementation...)
	case intent.OperationAddField:
		implementation, err := execute.RenderAddField(value, manifest, inventory)
		if err != nil {
			return nil, err
		}

		mutations = append(mutations, implementation...)
	case intent.OperationAddValidation:
		implementation, err := execute.RenderAddValidation(value, manifest, inventory)
		if err != nil {
			return nil, err
		}

		mutations = append(mutations, implementation...)
	case intent.OperationDocumentFeature:
	default:
		return nil, fmt.Errorf("no canonical renderer for operation %q", value.Intent)
	}

	if value.Documentation != intent.DocumentationNotRequested {
		documentation, err := execute.RenderDocumentation(value, manifest, inventory)
		if err != nil {
			return nil, err
		}

		mutations = append(mutations, documentation...)
	}

	return mutations, nil
}

func executionDiagnostics(values []execute.Diagnostic, phase Phase) []Diagnostic {
	result := make([]Diagnostic, 0, len(values))
	for _, value := range values {
		message := fmt.Sprintf("observed %q; expected %q", value.Observed, value.Expected)
		result = append(result, Diagnostic{
			Code:    value.Code,
			Phase:   phase,
			Field:   value.Location,
			Message: message,
		})
	}

	return result
}

func retainedChanges(values []execute.Change) []execute.Change {
	result := make([]execute.Change, 0, len(values))
	for _, value := range values {
		if value.Status == execute.ChangeApplied {
			result = append(result, value)
		}
	}

	return result
}

func cancelledExecutionOutcome(ctx context.Context) Outcome {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return OutcomeCancelled
	}

	return OutcomeExecutionFailed
}
