package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
)

// Evaluate runs one provider-neutral interpretation and sends any typed intent
// through deterministic validation and planning.
func Evaluate(ctx context.Context, interpreter Interpreter, prompt string, evaluationContext Context) (Result, error) {
	return EvaluateConversation(ctx, interpreter, prompt, nil, evaluationContext)
}

// EvaluateConversation runs one interpretation with explicit, bounded
// clarification history before deterministic validation and planning.
func EvaluateConversation(
	ctx context.Context,
	interpreter Interpreter,
	prompt string,
	clarifications []ClarificationExchange,
	evaluationContext Context,
) (Result, error) {
	if interpreter == nil {
		return Result{}, evaluationError("evaluation_interpreter_required", "interpreter", "interpreter is required")
	}

	request, err := compileRequest(prompt, clarifications, evaluationContext)
	if err != nil {
		return Result{}, err
	}

	backendResult, err := interpreter.Interpret(ctx, request)
	if err != nil {
		var backendErr BackendError
		if errors.As(err, &backendErr) {
			return Result{}, backendErr
		}

		return Result{}, evaluationError("evaluation_interpreter_failed", "interpreter", "%v", err)
	}

	err = validateProvenance(backendResult.Provenance)
	if err != nil {
		return Result{}, err
	}

	interpretation := backendResult.Interpretation

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
			Provenance:     backendResult.Provenance,
		}, nil
	case InterpretationUnsupported:
		return Result{
			Status:         intent.StatusCapabilityUnsupported,
			Diagnostics:    cloneDiagnostics(interpretation.Diagnostics),
			Clarifications: []intent.Clarification{},
			Provenance:     backendResult.Provenance,
		}, nil
	case InterpretationIntent:
		return evaluateIntent(*interpretation.Intent, backendResult.Provenance, evaluationContext)
	default:
		return Result{}, evaluationError("evaluation_kind_invalid", "interpretation.kind", "unknown interpretation kind %q", interpretation.Kind)
	}
}

func evaluateIntent(value intent.Intent, provenance Provenance, context Context) (Result, error) {
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
		Provenance:     provenance,
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

func validateProvenance(value Provenance) error {
	if value.ContractVersion != CurrentContractVersion {
		return evaluationError("evaluation_provenance_invalid", "provenance.contract_version", "interpreter provenance must use contract version %d", CurrentContractVersion)
	}

	if strings.TrimSpace(value.Adapter) == "" {
		return evaluationError("evaluation_provenance_invalid", "provenance.adapter", "interpreter adapter is required")
	}

	if value.ModelSelection != ModelBackendDefault && value.ModelSelection != ModelExplicit && value.ModelSelection != ModelNotApplicable {
		return evaluationError("evaluation_provenance_invalid", "provenance.model_selection", "unknown model selection %q", value.ModelSelection)
	}

	if value.Timing != TimingNotMeasured && value.Timing != TimingUnderSecond && value.Timing != TimingUnderTenSeconds && value.Timing != TimingUnderThirtySeconds && value.Timing != TimingUnderTwoMinutes {
		return evaluationError("evaluation_provenance_invalid", "provenance.timing", "unknown timing classification %q", value.Timing)
	}

	return nil
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

func cloneClarificationExchanges(values []ClarificationExchange) []ClarificationExchange {
	return append([]ClarificationExchange{}, values...)
}
