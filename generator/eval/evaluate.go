package eval

import (
	"context"
	"fmt"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
)

// Evaluate runs one provider-neutral interpretation and sends any typed intent
// through deterministic validation and planning.
func Evaluate(ctx context.Context, interpreter Interpreter, prompt string, evaluationContext Context) (Result, error) {
	if interpreter == nil {
		return Result{}, evaluationError("evaluation_interpreter_required", "interpreter", "interpreter is required")
	}

	request, err := compileRequest(prompt, evaluationContext)
	if err != nil {
		return Result{}, err
	}

	interpretation, err := interpreter.Interpret(ctx, request)
	if err != nil {
		return Result{}, evaluationError("evaluation_interpreter_failed", "interpreter", "%v", err)
	}

	err = validateInterpretation(interpretation)
	if err != nil {
		return Result{}, err
	}

	switch interpretation.Kind {
	case InterpretationClarification:
		return Result{
			Status:         intent.StatusClarificationRequired,
			Diagnostics:    []intent.Diagnostic{},
			Clarifications: cloneClarifications(interpretation.Clarifications),
		}, nil
	case InterpretationUnsupported:
		return Result{
			Status:         intent.StatusCapabilityUnsupported,
			Diagnostics:    cloneDiagnostics(interpretation.Diagnostics),
			Clarifications: []intent.Clarification{},
		}, nil
	case InterpretationIntent:
		return evaluateIntent(*interpretation.Intent, evaluationContext)
	default:
		return Result{}, evaluationError("evaluation_kind_invalid", "interpretation.kind", "unknown interpretation kind %q", interpretation.Kind)
	}
}

func evaluateIntent(value intent.Intent, context Context) (Result, error) {
	validation := intent.Validate(value, intent.ValidationContext{
		Inventory:   context.Inventory,
		Fingerprint: context.Fingerprint,
		Book:        context.Book,
	})
	result := Result{
		Status:         validation.Status,
		Intent:         cloneIntent(validation.Intent),
		Diagnostics:    cloneDiagnostics(validation.Diagnostics),
		Clarifications: cloneClarifications(validation.Clarifications),
	}

	if !validation.Admitted() {
		return result, nil
	}

	expanded, err := plan.Expand(validation, plan.ExpansionContext{
		Book:        context.Book,
		Fingerprint: context.Fingerprint,
	})
	if err != nil {
		return Result{}, fmt.Errorf("expand admitted interpretation: %w", err)
	}

	result.Plan = &expanded

	return result, nil
}

func validateInterpretation(value Interpretation) error {
	switch value.Kind {
	case InterpretationIntent:
		if value.Intent == nil || len(value.Clarifications) != 0 || len(value.Diagnostics) != 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "intent result must contain only one typed intent")
		}
	case InterpretationClarification:
		if value.Intent != nil || len(value.Clarifications) == 0 || len(value.Diagnostics) != 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "clarification result must contain only focused questions")
		}
	case InterpretationUnsupported:
		if value.Intent != nil || len(value.Clarifications) != 0 || len(value.Diagnostics) == 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "unsupported result must contain only diagnostics")
		}
	default:
		return evaluationError("evaluation_kind_invalid", "interpretation.kind", "unknown interpretation kind %q", value.Kind)
	}

	return nil
}

func cloneIntent(value intent.Intent) *intent.Intent {
	result := value
	result.Capabilities = append([]string{}, value.Capabilities...)
	result.Exceptions = append([]intent.Exception{}, value.Exceptions...)
	result.Domain.Fields = append([]intent.Field{}, value.Domain.Fields...)
	result.Domain.Rules = append([]intent.BusinessRule{}, value.Domain.Rules...)

	if value.Domain.Field != nil {
		field := *value.Domain.Field
		result.Domain.Field = &field
	}

	if value.Domain.Validation != nil {
		validation := *value.Domain.Validation
		result.Domain.Validation = &validation
	}

	return &result
}

func cloneDiagnostics(values []intent.Diagnostic) []intent.Diagnostic {
	return append([]intent.Diagnostic{}, values...)
}

func cloneClarifications(values []intent.Clarification) []intent.Clarification {
	return append([]intent.Clarification{}, values...)
}
