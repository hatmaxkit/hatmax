package book

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

var (
	stableIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)
	diagnosticPattern = regexp.MustCompile(`^HMGEN-[A-Z0-9]+(?:-[A-Z0-9]+)*$`)
)

// ValidationError describes one invalid Book contract.
type ValidationError struct {
	Code    string
	Path    string
	Message string
}

func (e ValidationError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}

	return fmt.Sprintf("%s at %s: %s", e.Code, e.Path, e.Message)
}

func validationError(code, path, format string, arguments ...any) error {
	return ValidationError{
		Code:    code,
		Path:    path,
		Message: fmt.Sprintf(format, arguments...),
	}
}

func (b *Book) validate() error {
	err := b.validateManifest()
	if err != nil {
		return err
	}

	err = b.validateCapabilities()
	if err != nil {
		return err
	}

	err = b.validateArchetypes()
	if err != nil {
		return err
	}

	err = b.validateRules()
	if err != nil {
		return err
	}

	err = b.validateReferencedFiles("capabilities", b.manifest.Capabilities)
	if err != nil {
		return err
	}

	err = b.validateReferencedFiles("archetypes", b.manifest.Archetypes)
	if err != nil {
		return err
	}

	err = b.validateReferencedFiles("rules", b.manifest.Rules)
	if err != nil {
		return err
	}

	return b.validateCapabilityCycles()
}

func (b *Book) validateManifest() error {
	if b.manifest.SchemaVersion != CurrentSchemaVersion {
		return validationError(
			"book_schema_unsupported",
			manifestPath,
			"schema version %d is not supported",
			b.manifest.SchemaVersion,
		)
	}

	if b.manifest.BookVersion < 1 {
		return validationError("book_invalid_version", manifestPath, "book version must be positive")
	}

	minimum, err := parseVersion(b.manifest.Hatmax.Minimum)
	if err != nil {
		return validationError("book_invalid_version", manifestPath, "%v", err)
	}

	maximum, err := parseVersion(b.manifest.Hatmax.MaximumExclusive)
	if err != nil {
		return validationError("book_invalid_version", manifestPath, "%v", err)
	}

	if compareVersion(minimum, maximum) >= 0 {
		return validationError(
			"book_invalid_version",
			manifestPath,
			"minimum Hatmax version must be lower than maximum_exclusive",
		)
	}

	err = validateUniqueStrings("manifest.capabilities", b.manifest.Capabilities, true)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("manifest.archetypes", b.manifest.Archetypes, true)
	if err != nil {
		return err
	}

	err = validateUniqueStrings("manifest.rules", b.manifest.Rules, true)
	if err != nil {
		return err
	}

	return nil
}

func (b *Book) validateCapabilities() error {
	for _, id := range b.capabilityOrder {
		capability := b.capabilities[id]
		path := "capabilities." + id

		err := validateStableID(path+".id", capability.ID)
		if err != nil {
			return err
		}

		if strings.TrimSpace(capability.Intent) == "" {
			return validationError("book_required_field", path+".intent", "intent is required")
		}

		err = validateUniqueStrings(path+".surfaces", capability.Surfaces, true)
		if err != nil {
			return err
		}

		err = validateObligations(path+".obligations", capability.Obligations, b.rules)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".rules", capability.Rules, true)
		if err != nil {
			return err
		}

		err = validateReferences(path+".rules", capability.Rules, b.rules)
		if err != nil {
			return err
		}

		err = validateReferences(path+".requires", capability.Requires, b.capabilities)
		if err != nil {
			return err
		}

		err = validateReferences(path+".incompatible_with", capability.IncompatibleWith, b.capabilities)
		if err != nil {
			return err
		}

		for index, dependency := range capability.Dependencies {
			dependencyPath := fmt.Sprintf("%s.dependencies[%d]", path, index)
			if dependency.Kind == "" || dependency.Module == "" || dependency.Purpose == "" {
				return validationError(
					"book_required_field",
					dependencyPath,
					"kind, module, and purpose are required",
				)
			}
		}
	}

	return nil
}

func (b *Book) validateArchetypes() error {
	for _, id := range b.archetypeOrder {
		archetype := b.archetypes[id]
		path := "archetypes." + id

		err := validateStableID(path+".id", archetype.ID)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".surfaces", archetype.Surfaces, true)
		if err != nil {
			return err
		}

		err = validateObligations(path+".obligations", archetype.Obligations, b.rules)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(path+".rules", archetype.Rules, true)
		if err != nil {
			return err
		}

		err = validateReferences(path+".rules", archetype.Rules, b.rules)
		if err != nil {
			return err
		}

		err = validateReferences(path+".required_capabilities", archetype.RequiredCapabilities, b.capabilities)
		if err != nil {
			return err
		}

		err = validateReferences(path+".optional_capabilities", archetype.OptionalCapabilities, b.capabilities)
		if err != nil {
			return err
		}

		err = b.validateOperations(path, archetype)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Book) validateOperations(path string, archetype Archetype) error {
	if len(archetype.Operations) == 0 {
		return validationError("book_required_field", path+".operations", "at least one operation is required")
	}

	obligations := make(map[string]Obligation, len(archetype.Obligations))
	for _, obligation := range archetype.Obligations {
		obligations[obligation.ID] = obligation
	}

	operationIDs := make([]string, 0, len(archetype.Operations))
	for index, operation := range archetype.Operations {
		operationPath := fmt.Sprintf("%s.operations[%d]", path, index)

		operationIDs = append(operationIDs, operation.ID)

		err := validateStableID(operationPath+".id", operation.ID)
		if err != nil {
			return err
		}

		err = validateReferences(operationPath+".required_capabilities", operation.RequiredCapabilities, b.capabilities)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(operationPath+".obligations", operation.Obligations, true)
		if err != nil {
			return err
		}

		err = validateReferences(operationPath+".obligations", operation.Obligations, obligations)
		if err != nil {
			return err
		}
	}

	return validateUniqueStrings(path+".operations", operationIDs, true)
}

func (b *Book) validateRules() error {
	for _, id := range b.ruleOrder {
		rule := b.rules[id]
		path := "rules." + id

		err := validateStableID(path+".id", rule.ID)
		if err != nil {
			return err
		}

		if rule.Level != LevelRequired && rule.Level != LevelRecommended && rule.Level != LevelOptional && rule.Level != LevelProhibited {
			return validationError("book_invalid_level", path+".level", "unknown rule level %q", rule.Level)
		}

		if strings.TrimSpace(rule.Title) == "" || strings.TrimSpace(rule.Rationale) == "" {
			return validationError("book_required_field", path, "title and rationale are required")
		}

		err = validateUniqueStrings(path+".diagnostics", rule.Diagnostics, true)
		if err != nil {
			return err
		}

		for _, diagnostic := range rule.Diagnostics {
			if !diagnosticPattern.MatchString(diagnostic) {
				return validationError("book_invalid_diagnostic", path+".diagnostics", "invalid code %q", diagnostic)
			}
		}

		err = validateReferences(path+".capabilities", rule.Capabilities, b.capabilities)
		if err != nil {
			return err
		}

		err = validateReferences(path+".archetypes", rule.Archetypes, b.archetypes)
		if err != nil {
			return err
		}

		for _, examplePath := range append(cloneStrings(rule.PositiveCases), rule.NegativeCases...) {
			if !fs.ValidPath(examplePath) || !strings.HasPrefix(examplePath, "examples/") {
				return validationError("book_invalid_path", path, "invalid example path %q", examplePath)
			}

			_, statErr := fs.Stat(b.source, examplePath)
			if statErr != nil {
				return validationError("book_missing_reference", path, "example %q does not exist", examplePath)
			}
		}
	}

	return nil
}

func (b *Book) validateReferencedFiles(directory string, referenced []string) error {
	known := make(map[string]struct{}, len(referenced))
	for _, entryPath := range referenced {
		if _, exists := known[entryPath]; exists {
			return validationError("book_duplicate_path", manifestPath, "duplicate entry path %q", entryPath)
		}

		known[entryPath] = struct{}{}
	}

	entries, err := fs.ReadDir(b.source, directory)
	if err != nil {
		return fmt.Errorf("read %s: %w", directory, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		entryPath := directory + "/" + entry.Name()
		if _, exists := known[entryPath]; !exists {
			return validationError("book_unreferenced_entry", entryPath, "entry is not listed in the manifest")
		}
	}

	return nil
}

func (b *Book) validateCapabilityCycles() error {
	state := make(map[string]int, len(b.capabilities))
	path := make([]string, 0, len(b.capabilities))

	var visit func(string) error

	visit = func(id string) error {
		if state[id] == 2 {
			return nil
		}

		if state[id] == 1 {
			cycle := append(path, id)

			return validationError("book_capability_cycle", "capabilities."+id, "cycle: %s", strings.Join(cycle, " -> "))
		}

		state[id] = 1

		path = append(path, id)
		for _, dependency := range b.capabilities[id].Requires {
			err := visit(dependency)
			if err != nil {
				return err
			}
		}

		path = path[:len(path)-1]
		state[id] = 2

		return nil
	}

	for _, id := range b.capabilityOrder {
		err := visit(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateStableID(path, id string) error {
	if !stableIDPattern.MatchString(id) {
		return validationError("book_invalid_id", path, "invalid stable ID %q", id)
	}

	return nil
}

func validateUniqueStrings(path string, values []string, required bool) error {
	if required && len(values) == 0 {
		return validationError("book_required_field", path, "at least one value is required")
	}

	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return validationError("book_required_field", path, "empty values are not allowed")
		}

		if _, exists := seen[value]; exists {
			return validationError("book_duplicate_value", path, "duplicate value %q", value)
		}

		seen[value] = struct{}{}
	}

	return nil
}

func validateObligations(path string, values []Obligation, rules map[string]Rule) error {
	if len(values) == 0 {
		return validationError("book_required_field", path, "at least one obligation is required")
	}

	ids := make([]string, 0, len(values))
	for index, obligation := range values {
		obligationPath := fmt.Sprintf("%s[%d]", path, index)

		ids = append(ids, obligation.ID)

		err := validateStableID(obligationPath+".id", obligation.ID)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(obligationPath+".surfaces", obligation.Surfaces, true)
		if err != nil {
			return err
		}

		err = validateUniqueStrings(obligationPath+".rules", obligation.Rules, true)
		if err != nil {
			return err
		}

		err = validateReferences(obligationPath+".rules", obligation.Rules, rules)
		if err != nil {
			return err
		}
	}

	err := validateUniqueStrings(path, ids, true)
	if err != nil {
		return err
	}

	known := make(map[string]Obligation, len(values))
	for _, obligation := range values {
		known[obligation.ID] = obligation
	}

	for index, obligation := range values {
		err = validateReferences(fmt.Sprintf("%s[%d].depends_on", path, index), obligation.DependsOn, known)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateReferences[T any](path string, ids []string, available map[string]T) error {
	err := validateUniqueStrings(path, ids, false)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if _, exists := available[id]; !exists {
			return validationError("book_missing_reference", path, "unknown reference %q", id)
		}
	}

	return nil
}
