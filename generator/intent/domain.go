package intent

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"hatmax.adrianpk.com/generator/project"
)

var (
	featureNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	entityNamePattern  = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	routePattern       = regexp.MustCompile(`^/[a-z0-9]+(?:-[a-z0-9]+)*(?:/[a-z0-9]+(?:-[a-z0-9]+)*)*$`)
)

var admittedFieldTypes = map[string]struct{}{
	"boolean":   {},
	"date":      {},
	"decimal":   {},
	"integer":   {},
	"string":    {},
	"text":      {},
	"timestamp": {},
	"uuid":      {},
}

var admittedValidationKinds = map[string]struct{}{
	"maximum":    {},
	"max_length": {},
	"minimum":    {},
	"min_length": {},
	"pattern":    {},
	"required":   {},
	"unique":     {},
}

func validateDomain(value Intent, inventory project.Inventory) ([]Diagnostic, []Clarification) {
	diagnostics := make([]Diagnostic, 0)

	if !featureNamePattern.MatchString(value.Feature) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "HMGEN-FEATURE-NAME",
			Field:   "feature",
			Message: fmt.Sprintf("feature %q must use lower snake case", value.Feature),
		})
	}

	exists := featureExists(inventory, value.Feature)
	if value.Operation == OperationCreateFeature && exists {
		diagnostics = append(diagnostics, Diagnostic{
			Code:    "HMGEN-FEATURE-EXISTS",
			Field:   "feature",
			Message: fmt.Sprintf("feature %q already exists", value.Feature),
		})
	}

	if value.Operation != OperationCreateFeature && !exists {
		diagnostics = append(diagnostics, missingFeatureDiagnostic(value.Feature))
	}

	domainDiagnostics, clarifications := ValidateDomainDecisions(value)
	diagnostics = append(diagnostics, domainDiagnostics...)

	return diagnostics, clarifications
}

// ValidateDomainDecisions checks whether the application-specific decisions
// required by an intent are complete and supported without inspecting a
// project or selecting implementation obligations.
func ValidateDomainDecisions(value Intent) ([]Diagnostic, []Clarification) {
	diagnostics := make([]Diagnostic, 0)
	clarifications := make([]Clarification, 0)

	switch value.Operation {
	case OperationCreateFeature:
		validateCreateFeature(value, &diagnostics, &clarifications)
	case OperationAddField:
		validateAddField(value, &diagnostics, &clarifications)
	case OperationAddValidation:
		validateAddValidation(value, &diagnostics, &clarifications)
	}

	validateBusinessRules(value.Domain.Rules, &diagnostics, &clarifications)
	sortDiagnostics(diagnostics)
	sortClarifications(clarifications)

	return diagnostics, clarifications
}

func validateCreateFeature(value Intent, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	if value.Domain.Entity == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.entity",
			Question: "What singular domain entity should the feature create and manage?",
		})
	} else if !entityNamePattern.MatchString(value.Domain.Entity) {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-ENTITY-NAME",
			Field:   "domain.entity",
			Message: fmt.Sprintf("entity %q must use exported Go identifier form", value.Domain.Entity),
		})
	}

	if value.Domain.Route == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.route",
			Question: "What canonical URL path should own this feature?",
		})
	} else if !routePattern.MatchString(value.Domain.Route) {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-ROUTE-NAME",
			Field:   "domain.route",
			Message: fmt.Sprintf("route %q must be an absolute lower-kebab path", value.Domain.Route),
		})
	}

	if len(value.Domain.Fields) == 0 {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.fields",
			Question: "Which fields belong to the initial domain entity?",
		})
	}

	validateFields("domain.fields", value.Domain.Fields, diagnostics, clarifications)

	if hasRequiredField(value.Domain.Fields) && !containsCapability(value.Capabilities, "runtime_validation") {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-CAPABILITY-REQUIRED",
			Field:   "capabilities",
			Message: "required fields need runtime_validation",
		})
	}

	validateOwnership(value.Domain.Ownership, diagnostics)
}

func validateAddField(value Intent, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	if value.Domain.Entity != "" || value.Domain.Route != "" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-DOMAIN-SCOPE",
			Field:   "domain",
			Message: "add_field cannot redefine the feature entity or route",
		})
	}

	if value.Domain.Field == nil {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.field",
			Question: "Which field should be added to the existing feature?",
		})

		return
	}

	validateFields("domain.field", []Field{*value.Domain.Field}, diagnostics, clarifications)

	if value.Domain.Field.Required && !containsCapability(value.Capabilities, "runtime_validation") {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-CAPABILITY-REQUIRED",
			Field:   "capabilities",
			Message: "a required field needs runtime_validation",
		})
	}
}

func validateAddValidation(value Intent, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	if value.Domain.Entity != "" || value.Domain.Route != "" || value.Domain.Ownership != "" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-DOMAIN-SCOPE",
			Field:   "domain",
			Message: "add_validation cannot redefine the feature entity, route, or ownership",
		})
	}

	validation := value.Domain.Validation
	if validation == nil {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.validation",
			Question: "What validation should be added to the existing feature?",
		})

		return
	}

	if validation.Field == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.validation.field",
			Question: "Which field does the validation constrain?",
		})
	} else if !featureNamePattern.MatchString(validation.Field) {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-FIELD-NAME",
			Field:   "domain.validation.field",
			Message: fmt.Sprintf("field %q must use lower snake case", validation.Field),
		})
	}

	if validation.Kind == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.validation.kind",
			Question: "Which validation rule should be applied?",
		})
	} else if _, admitted := admittedValidationKinds[validation.Kind]; !admitted {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-VALIDATION-KIND",
			Field:   "domain.validation.kind",
			Message: fmt.Sprintf("validation kind %q is not supported", validation.Kind),
		})
	}

	if validation.Scope == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.validation.scope",
			Question: "Does this rule protect durable state or only client interaction?",
		})
	} else if validation.Scope != ValidationDurable && validation.Scope != ValidationClientOnly {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-VALIDATION-SCOPE",
			Field:   "domain.validation.scope",
			Message: fmt.Sprintf("validation scope %q is not supported", validation.Scope),
		})
	}

	if validationNeedsValue(validation.Kind) && strings.TrimSpace(validation.Value) == "" {
		*clarifications = append(*clarifications, Clarification{
			Field:    "domain.validation.value",
			Question: "What comparison value or pattern should the validation use?",
		})
	}

	if validation.Scope == ValidationClientOnly && validation.Kind == "unique" {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "HMGEN-VALIDATION-DURABLE",
			Field:   "domain.validation.scope",
			Message: "unique validation protects durable state and cannot be client-only",
		})
	}
}

func validateFields(prefix string, fields []Field, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		fieldPath := prefix
		if len(fields) > 1 {
			fieldPath = fmt.Sprintf("%s[%d]", prefix, index)
		}

		if field.Name == "" {
			*clarifications = append(*clarifications, Clarification{
				Field:    fieldPath + ".name",
				Question: "What is the field name?",
			})
		} else if !featureNamePattern.MatchString(field.Name) {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "HMGEN-FIELD-NAME",
				Field:   fieldPath + ".name",
				Message: fmt.Sprintf("field %q must use lower snake case", field.Name),
			})
		}

		if field.Type == "" {
			*clarifications = append(*clarifications, Clarification{
				Field:    fieldPath + ".type",
				Question: fmt.Sprintf("What domain type should field %q use?", field.Name),
			})
		} else if _, admitted := admittedFieldTypes[field.Type]; !admitted {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "HMGEN-FIELD-TYPE",
				Field:   fieldPath + ".type",
				Message: fmt.Sprintf("field type %q is not supported", field.Type),
			})
		}

		if _, exists := seen[field.Name]; field.Name != "" && exists {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "HMGEN-FIELD-DUPLICATE",
				Field:   fieldPath + ".name",
				Message: fmt.Sprintf("field %q is duplicated", field.Name),
			})
		}

		seen[field.Name] = struct{}{}
	}
}

func validateOwnership(ownership string, diagnostics *[]Diagnostic) {
	if ownership == "" || ownership == "global" || ownership == "authenticated" || ownership == "owner" {
		return
	}

	*diagnostics = append(*diagnostics, Diagnostic{
		Code:    "HMGEN-OWNERSHIP",
		Field:   "domain.ownership",
		Message: fmt.Sprintf("ownership %q is not supported", ownership),
	})
}

func validateBusinessRules(rules []BusinessRule, diagnostics *[]Diagnostic, clarifications *[]Clarification) {
	seen := make(map[string]struct{}, len(rules))
	for index, rule := range rules {
		prefix := fmt.Sprintf("domain.rules[%d]", index)
		if rule.Name == "" {
			*clarifications = append(*clarifications, Clarification{
				Field:    prefix + ".name",
				Question: "What stable name identifies this business rule?",
			})
		} else if !featureNamePattern.MatchString(rule.Name) {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "HMGEN-RULE-NAME",
				Field:   prefix + ".name",
				Message: fmt.Sprintf("business rule %q must use lower snake case", rule.Name),
			})
		}

		if strings.TrimSpace(rule.Description) == "" {
			*clarifications = append(*clarifications, Clarification{
				Field:    prefix + ".description",
				Question: fmt.Sprintf("What behavior does business rule %q require?", rule.Name),
			})
		}

		if rule.Owner == "" {
			*clarifications = append(*clarifications, Clarification{
				Field:    prefix + ".owner",
				Question: fmt.Sprintf("Which domain actor or aggregate owns rule %q?", rule.Name),
			})
		}

		if _, exists := seen[rule.Name]; rule.Name != "" && exists {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "HMGEN-RULE-DUPLICATE",
				Field:   prefix + ".name",
				Message: fmt.Sprintf("business rule %q is duplicated", rule.Name),
			})
		}

		seen[rule.Name] = struct{}{}
	}
}

func validationNeedsValue(kind string) bool {
	return kind == "maximum" || kind == "max_length" || kind == "minimum" || kind == "min_length" || kind == "pattern"
}

func hasRequiredField(fields []Field) bool {
	for _, field := range fields {
		if field.Required {
			return true
		}
	}

	return false
}

func featureExists(inventory project.Inventory, feature string) bool {
	for _, featurePath := range inventory.Layout.Features {
		if path.Base(featurePath) == feature {
			return true
		}
	}

	return false
}

func missingFeatureDiagnostic(feature string) Diagnostic {
	return Diagnostic{
		Code:    "HMGEN-FEATURE-MISSING",
		Field:   "feature",
		Message: fmt.Sprintf("feature %q does not exist in the inspected project", feature),
	}
}
