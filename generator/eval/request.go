package eval

import (
	"path"
	"strings"
)

func compileRequest(prompt string, context Context) (Request, error) {
	if strings.TrimSpace(prompt) == "" {
		return Request{}, evaluationError("evaluation_prompt_required", "prompt", "prompt is required")
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

	return Request{
		Prompt: prompt,
		Project: ProjectContext{
			Fingerprint:      context.Fingerprint.Value,
			HatmaxVersion:    context.Inventory.Module.Hatmax.Version,
			ExistingFeatures: existingFeatures(context),
		},
		Book: BookContext{
			Version:      manifest.BookVersion,
			Archetypes:   archetypeContexts(context),
			Capabilities: capabilityContexts(context),
		},
	}, nil
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
