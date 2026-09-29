package eval

import (
	"fmt"
	"path"
	"strings"
)

func compileRequest(prompt string, clarifications []ClarificationExchange, context Context) (Request, error) {
	if strings.TrimSpace(prompt) == "" {
		return Request{}, evaluationError("evaluation_prompt_required", "prompt", "prompt is required")
	}

	if len(prompt) > MaximumPromptBytes {
		return Request{}, evaluationError("evaluation_prompt_too_large", "prompt", "prompt exceeds %d bytes", MaximumPromptBytes)
	}

	if context.Book == nil {
		return Request{}, evaluationError("evaluation_book_required", "book", "validated Book is required")
	}

	if context.Fingerprint.Value == "" || context.Fingerprint.BookVersion < 1 {
		return Request{}, evaluationError("evaluation_fingerprint_required", "project.fingerprint", "versioned project fingerprint is required")
	}

	manifest := context.Book.Manifest()
	if manifest.BookVersion != context.Fingerprint.BookVersion {
		return Request{}, evaluationError("evaluation_book_mismatch", "book.version", "Book and fingerprint versions must match")
	}

	result := Request{
		ContractVersion: CurrentContractVersion,
		Prompt:          prompt,
		Clarifications:  cloneClarificationExchanges(clarifications),
		Book: BookContext{
			Version:      manifest.BookVersion,
			Archetypes:   archetypeContexts(context),
			Capabilities: capabilityContexts(context),
		},
	}

	if context.Target == nil {
		result.Project = ProjectContext{
			Fingerprint:      context.Fingerprint.Value,
			HatmaxVersion:    context.Inventory.Module.Hatmax.Version,
			ExistingFeatures: existingFeatures(context),
		}
	} else {
		result.Target = &TargetContext{
			SourceFingerprint: context.Fingerprint.Value,
			HatmaxVersion:     manifest.Hatmax.Minimum,
			Admission:         context.Target.Admission,
			RemoteModulePath:  context.Target.RemoteModulePath,
		}
	}

	err := validateRequest(result)
	if err != nil {
		return Request{}, err
	}

	return result, nil
}

func validateRequest(value Request) error {
	if value.ContractVersion != CurrentContractVersion {
		return evaluationError("evaluation_contract_unsupported", "contract_version", "contract version %d is not supported", value.ContractVersion)
	}

	hasProject := strings.TrimSpace(value.Project.Fingerprint) != ""
	hasTarget := value.Target != nil
	if hasProject == hasTarget {
		return evaluationError("evaluation_source_invalid", "project", "exactly one project or target context is required")
	}

	if hasTarget {
		if strings.TrimSpace(value.Target.SourceFingerprint) == "" || strings.TrimSpace(value.Target.HatmaxVersion) == "" || strings.TrimSpace(string(value.Target.Admission)) == "" {
			return evaluationError("evaluation_target_invalid", "target", "target fingerprint, Hatmax version, and admission are required")
		}
	}

	if len(value.Clarifications) > MaximumClarificationExchanges {
		return evaluationError("evaluation_clarification_limit", "clarifications", "clarification history exceeds %d exchanges", MaximumClarificationExchanges)
	}

	seen := make(map[string]struct{}, len(value.Clarifications))
	for index, exchange := range value.Clarifications {
		field := strings.TrimSpace(exchange.Field)
		question := strings.TrimSpace(exchange.Question)
		answer := strings.TrimSpace(exchange.Answer)

		if field == "" || question == "" || answer == "" {
			return evaluationError("evaluation_clarification_invalid", indexedField("clarifications", index), "field, question, and answer are required")
		}

		if len(exchange.Field) > MaximumClarificationTextBytes || len(exchange.Question) > MaximumClarificationTextBytes || len(exchange.Answer) > MaximumClarificationTextBytes {
			return evaluationError("evaluation_clarification_too_large", indexedField("clarifications", index), "clarification values exceed %d bytes", MaximumClarificationTextBytes)
		}

		if _, exists := seen[field]; exists {
			return evaluationError("evaluation_clarification_duplicate", indexedField("clarifications", index)+".field", "clarification field %q is duplicated", field)
		}

		seen[field] = struct{}{}
	}

	return nil
}

func indexedField(field string, index int) string {
	return fmt.Sprintf("%s[%d]", field, index)
}

func existingFeatures(context Context) []string {
	result := make([]string, 0, len(context.Inventory.Layout.Features))
	for _, featurePath := range context.Inventory.Layout.Features {
		result = append(result, path.Base(featurePath))
	}

	return result
}

func archetypeContexts(context Context) []ArchetypeContext {
	archetypes := context.Book.Archetypes()
	result := make([]ArchetypeContext, 0, len(archetypes))

	for _, archetype := range archetypes {
		operations := make([]string, 0, len(archetype.Operations))
		for _, operation := range archetype.Operations {
			operations = append(operations, operation.ID)
		}

		result = append(result, ArchetypeContext{
			ID:                   archetype.ID,
			Operations:           operations,
			RequiredCapabilities: append([]string{}, archetype.RequiredCapabilities...),
			OptionalCapabilities: append([]string{}, archetype.OptionalCapabilities...),
		})
	}

	return result
}

func capabilityContexts(context Context) []CapabilityContext {
	capabilities := context.Book.Capabilities()
	result := make([]CapabilityContext, 0, len(capabilities))

	for _, capability := range capabilities {
		result = append(result, CapabilityContext{
			ID:          capability.ID,
			Intent:      capability.Intent,
			Requires:    append([]string{}, capability.Requires...),
			Unsupported: append([]string{}, capability.Unsupported...),
		})
	}

	return result
}
