package intent

import "strings"

func normalizeSemanticNames(value *Intent) {
	featureWords, featureValid := semanticNameWords(value.Feature)
	if featureValid {
		value.Feature = strings.Join(featureWords, "_")
	}

	switch value.Operation {
	case OperationCreateApplication:
		// Application identity and initial features normalize separately.
	case OperationCreateFeature:
		normalizeCreateFeatureNames(value, featureWords, featureValid)
	case OperationAddField:
		if value.Domain.Field != nil {
			normalizeFieldName(value.Domain.Field)
		}
	case OperationAddValidation:
		if value.Domain.Validation != nil {
			value.Domain.Validation.Field = canonicalSnakeName(value.Domain.Validation.Field)
		}
	}

	for index := range value.Domain.Rules {
		value.Domain.Rules[index].Name = canonicalSnakeName(value.Domain.Rules[index].Name)
	}
}

func normalizeApplication(value *Intent) {
	if value.Operation != OperationCreateApplication || value.Application == nil {
		return
	}

	value.Application.DisplayName = strings.TrimSpace(value.Application.DisplayName)
	value.Application.ModulePath = strings.TrimSpace(value.Application.ModulePath)
	value.Application.Description = strings.TrimSpace(value.Application.Description)
	value.Application.Niche = strings.TrimSpace(value.Application.Niche)

	words, valid := semanticNameWords(value.Application.DisplayName)
	if strings.TrimSpace(value.Application.ProjectSlug) == "" && valid {
		value.Application.ProjectSlug = strings.Join(words, "-")
	} else {
		value.Application.ProjectSlug = canonicalKebabName(value.Application.ProjectSlug)
	}

	if value.Target != nil {
		if strings.TrimSpace(value.Target.Base) == "" {
			value.Target.Base = "session_directory"
		}

		if strings.TrimSpace(value.Target.Directory) == "" {
			value.Target.Directory = value.Application.ProjectSlug
		} else {
			value.Target.Directory = canonicalKebabName(value.Target.Directory)
		}
	}

	for index := range value.InitialFeatures {
		featureIntent := Intent{
			Operation: OperationCreateFeature,
			Feature:   value.InitialFeatures[index].Feature,
			Domain:    value.InitialFeatures[index].Domain,
		}
		normalizeSemanticNames(&featureIntent)
		value.InitialFeatures[index].Feature = featureIntent.Feature
		value.InitialFeatures[index].Domain = featureIntent.Domain
	}

	sortInitialFeatures(value.InitialFeatures)
}

func canonicalKebabName(value string) string {
	words, valid := semanticNameWords(value)
	if !valid {
		return strings.TrimSpace(value)
	}

	return strings.Join(words, "-")
}

func normalizeCreateFeatureNames(value *Intent, featureWords []string, featureValid bool) {
	entityWords, entityValid := semanticNameWords(value.Domain.Entity)
	if strings.TrimSpace(value.Domain.Entity) == "" && featureValid {
		entityWords = append([]string{}, featureWords...)
		entityValid = true
	}

	if entityValid {
		value.Domain.Entity = exportedSemanticName(entityWords)
	}

	if strings.TrimSpace(value.Domain.Route) == "" && entityValid {
		value.Domain.Route = "/" + strings.Join(pluralizeSemanticWords(entityWords), "-")
	} else {
		value.Domain.Route = canonicalRoute(value.Domain.Route)
	}

	value.Domain.Label = strings.TrimSpace(value.Domain.Label)
	if value.Domain.Label == "" && entityValid {
		value.Domain.Label = displaySemanticName(pluralizeSemanticWords(entityWords))
	}

	for index := range value.Domain.Fields {
		normalizeFieldName(&value.Domain.Fields[index])
	}
}

func normalizeFieldName(field *Field) {
	words, valid := semanticNameWords(field.Name)
	if valid {
		field.Name = strings.Join(words, "_")
	}

	field.Label = strings.TrimSpace(field.Label)
	if field.Label == "" && valid {
		field.Label = displaySemanticName(words)
	}
}

func canonicalSnakeName(value string) string {
	words, valid := semanticNameWords(value)
	if !valid {
		return strings.TrimSpace(value)
	}

	return strings.Join(words, "_")
}

func canonicalRoute(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimPrefix(trimmed, "/")
	segments := strings.Split(trimmed, "/")
	canonical := make([]string, 0, len(segments))

	for _, segment := range segments {
		words, valid := semanticNameWords(segment)
		if !valid {
			return strings.TrimSpace(value)
		}

		canonical = append(canonical, strings.Join(words, "-"))
	}

	return "/" + strings.Join(canonical, "/")
}

func semanticNameWords(value string) ([]string, bool) {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) == 0 {
		return nil, false
	}

	words := make([]string, 0, 2)
	word := make([]rune, 0, len(runes))

	flush := func() {
		if len(word) == 0 {
			return
		}

		words = append(words, strings.ToLower(string(word)))
		word = word[:0]
	}

	for index, current := range runes {
		if current == '_' || current == '-' || current == ' ' || current == '\t' || current == '\n' || current == '\r' {
			flush()

			continue
		}

		if !isASCIIAlphaNumeric(current) {
			return nil, false
		}

		if isASCIIUpper(current) && len(word) > 0 {
			previous := runes[index-1]

			nextIsLower := index+1 < len(runes) && isASCIILower(runes[index+1])
			if isASCIILower(previous) || isASCIIDigit(previous) || (isASCIIUpper(previous) && nextIsLower) {
				flush()
			}
		}

		word = append(word, current)
	}

	flush()

	return words, len(words) > 0
}

func exportedSemanticName(words []string) string {
	var result strings.Builder

	for _, word := range words {
		if word == "" {
			continue
		}

		result.WriteString(strings.ToUpper(word[:1]))
		result.WriteString(word[1:])
	}

	return result.String()
}

func displaySemanticName(words []string) string {
	result := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" {
			continue
		}

		result = append(result, strings.ToUpper(word[:1])+word[1:])
	}

	return strings.Join(result, " ")
}

func pluralizeSemanticWords(words []string) []string {
	result := append([]string{}, words...)
	if len(result) == 0 {
		return result
	}

	index := len(result) - 1
	result[index] = pluralizeSemanticWord(result[index])

	return result
}

func pluralizeSemanticWord(word string) string {
	if strings.HasSuffix(word, "y") && len(word) > 1 && !isASCIIVowel(rune(word[len(word)-2])) {
		return word[:len(word)-1] + "ies"
	}

	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	return word + "s"
}

func isASCIIAlphaNumeric(value rune) bool {
	return isASCIILower(value) || isASCIIUpper(value) || isASCIIDigit(value)
}

func isASCIILower(value rune) bool {
	return value >= 'a' && value <= 'z'
}

func isASCIIUpper(value rune) bool {
	return value >= 'A' && value <= 'Z'
}

func isASCIIDigit(value rune) bool {
	return value >= '0' && value <= '9'
}

func isASCIIVowel(value rune) bool {
	return value == 'a' || value == 'e' || value == 'i' || value == 'o' || value == 'u'
}
