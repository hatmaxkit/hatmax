package book

import "fmt"

// Select resolves the complete deterministic Book context for an archetype
// and requested capability set.
func (b *Book) Select(archetypeID string, requested []string) (Selection, error) {
	archetype, exists := b.archetypes[archetypeID]
	if !exists {
		return Selection{}, validationError("book_missing_reference", "selection.archetype", "unknown archetype %q", archetypeID)
	}

	allowed := make(map[string]struct{}, len(archetype.RequiredCapabilities)+len(archetype.OptionalCapabilities))
	for _, id := range archetype.RequiredCapabilities {
		allowed[id] = struct{}{}
	}

	for _, id := range archetype.OptionalCapabilities {
		allowed[id] = struct{}{}
	}

	selected := make(map[string]struct{}, len(allowed))
	for _, id := range archetype.RequiredCapabilities {
		selected[id] = struct{}{}
	}

	for _, id := range requested {
		if _, exists := allowed[id]; !exists {
			return Selection{}, validationError(
				"book_capability_not_allowed",
				"selection.capabilities",
				"capability %q is not admitted by archetype %q",
				id,
				archetypeID,
			)
		}

		selected[id] = struct{}{}
	}

	changed := true
	for changed {
		changed = false

		for id := range selected {
			for _, dependency := range b.capabilities[id].Requires {
				if _, exists := selected[dependency]; exists {
					continue
				}

				selected[dependency] = struct{}{}
				changed = true
			}
		}
	}

	for id := range selected {
		for _, incompatible := range b.capabilities[id].IncompatibleWith {
			if _, exists := selected[incompatible]; exists {
				return Selection{}, validationError(
					"book_capability_conflict",
					"selection.capabilities",
					"capabilities %q and %q are incompatible",
					id,
					incompatible,
				)
			}
		}
	}

	capabilities := make([]Capability, 0, len(selected))

	ruleIDs := make(map[string]struct{})
	for _, id := range archetype.Rules {
		ruleIDs[id] = struct{}{}
	}

	for _, id := range b.capabilityOrder {
		if _, exists := selected[id]; !exists {
			continue
		}

		capability := b.capabilities[id]

		capabilities = append(capabilities, cloneCapability(capability))
		for _, ruleID := range capability.Rules {
			ruleIDs[ruleID] = struct{}{}
		}

		for _, obligation := range capability.Obligations {
			for _, ruleID := range obligation.Rules {
				ruleIDs[ruleID] = struct{}{}
			}
		}
	}

	for _, obligation := range archetype.Obligations {
		for _, ruleID := range obligation.Rules {
			ruleIDs[ruleID] = struct{}{}
		}
	}

	rules := make([]Rule, 0, len(ruleIDs))
	for _, id := range b.ruleOrder {
		if _, exists := ruleIDs[id]; exists {
			rules = append(rules, cloneRule(b.rules[id]))
		}
	}

	if len(rules) != len(ruleIDs) {
		return Selection{}, fmt.Errorf("selected rule set is incomplete")
	}

	return Selection{
		Archetype:    cloneArchetype(archetype),
		Capabilities: capabilities,
		Rules:        rules,
	}, nil
}
