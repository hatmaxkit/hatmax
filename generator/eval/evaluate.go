package eval

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"hatmax.adrianpk.com/generator/intent"
	"hatmax.adrianpk.com/generator/plan"
)

var interpretationDiagnosticCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+$`)

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
	if value.SchemaVersion != CurrentInterpretationSchemaVersion {
		return evaluationError("evaluation_output_schema_unsupported", "interpretation.schema_version", "interpretation schema version %d is not supported", value.SchemaVersion)
	}

	switch value.Kind {
	case InterpretationIntent:
		if value.Intent == nil || len(value.Clarifications) != 0 || len(value.Diagnostics) != 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "intent result must contain only one typed intent")
		}

		err := intent.ValidateSchema(*value.Intent)
		if err != nil {
			return evaluationError("evaluation_result_invalid", "interpretation.intent", "%v", err)
		}
	case InterpretationClarification:
		if value.Intent != nil || len(value.Clarifications) == 0 || len(value.Diagnostics) != 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "clarification result must contain only focused questions")
		}

		if len(value.Clarifications) > MaximumClarificationExchanges {
			return evaluationError("evaluation_result_invalid", "interpretation.clarifications", "clarification result exceeds %d questions", MaximumClarificationExchanges)
		}

		seen := make(map[string]struct{}, len(value.Clarifications))
		for index, clarification := range value.Clarifications {
			field := strings.TrimSpace(clarification.Field)
			question := strings.TrimSpace(clarification.Question)

			if field == "" || question == "" {
				return evaluationError("evaluation_result_invalid", indexedField("interpretation.clarifications", index), "field and question are required")
			}

			if len(clarification.Field) > MaximumClarificationTextBytes || len(clarification.Question) > MaximumClarificationTextBytes {
				return evaluationError("evaluation_result_invalid", indexedField("interpretation.clarifications", index), "clarification exceeds %d bytes", MaximumClarificationTextBytes)
			}

			if _, exists := seen[field]; exists {
				return evaluationError("evaluation_result_invalid", indexedField("interpretation.clarifications", index)+".field", "clarification field %q is duplicated", field)
			}

			seen[field] = struct{}{}
		}
	case InterpretationUnsupported:
		if value.Intent != nil || len(value.Clarifications) != 0 || len(value.Diagnostics) == 0 {
			return evaluationError("evaluation_result_invalid", "interpretation", "unsupported result must contain only diagnostics")
		}

		if len(value.Diagnostics) > MaximumInterpretationDiagnostics {
			return evaluationError("evaluation_result_invalid", "interpretation.diagnostics", "unsupported result exceeds %d diagnostics", MaximumInterpretationDiagnostics)
		}

		for index, diagnostic := range value.Diagnostics {
			if !interpretationDiagnosticCodePattern.MatchString(diagnostic.Code) || strings.TrimSpace(diagnostic.Field) == "" || strings.TrimSpace(diagnostic.Message) == "" {
				return evaluationError("evaluation_result_invalid", indexedField("interpretation.diagnostics", index), "stable code, field, and message are required")
			}

			if len(diagnostic.Field) > MaximumClarificationTextBytes || len(diagnostic.Message) > MaximumClarificationTextBytes {
				return evaluationError("evaluation_result_invalid", indexedField("interpretation.diagnostics", index), "diagnostic exceeds %d bytes", MaximumClarificationTextBytes)
			}
		}
	default:
		return evaluationError("evaluation_kind_invalid", "interpretation.kind", "unknown interpretation kind %q", value.Kind)
	}

	return nil
}

func cloneInterpretation(value Interpretation) Interpretation {
	result := value
	result.Intent = nil

	if value.Intent != nil {
		result.Intent = cloneIntent(*value.Intent)
	}

	result.Clarifications = cloneClarifications(value.Clarifications)
	result.Diagnostics = cloneDiagnostics(value.Diagnostics)

	return result
}

func cloneIntent(value intent.Intent) *intent.Intent {
	result := value
	result.Capabilities = append([]string{}, value.Capabilities...)
	result.DocumentationTargets = append([]intent.DocumentationTarget{}, value.DocumentationTargets...)
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
