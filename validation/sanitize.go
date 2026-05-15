package validation

import (
	"strings"
	"unicode"
)

// NormalizeText trims text, removes control characters, and collapses
// whitespace runs to a single space.
func NormalizeText(value string) string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, value)

	parts := strings.Fields(strings.TrimSpace(clean))
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// NormalizeOptionalText normalizes optional text pointers and returns nil when
// the resulting value is empty.
func NormalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := NormalizeText(*value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

// SanitizeStringSlice normalizes each item, drops empty values, and deduplicates
// while preserving first occurrence order.
func SanitizeStringSlice(values []string) []string {
	if values == nil {
		return nil
	}

	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, raw := range values {
		value := NormalizeText(raw)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}
