package slug

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	DefaultMaxLength = 50
	UUIDPrefixLength = 8
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	multipleHyphens = regexp.MustCompile(`-+`)
)

// Generate creates a URL-safe slug from text and UUID.
// Example: "Cozy Dom Apartment!" + "3995fd11-..." -> "cozy-dom-apartment-3995fd11"
func Generate(text string, id uuid.UUID) string {
	normalized := Normalize(text, DefaultMaxLength)
	uuidPrefix := id.String()[:UUIDPrefixLength]

	if normalized == "" {
		return uuidPrefix
	}

	return normalized + "-" + uuidPrefix
}

// Normalize converts text to kebab-case ASCII.
func Normalize(text string, maxLength int) string {
	if text == "" || maxLength <= 0 {
		return ""
	}

	text = transliterate(text)
	text = strings.ToLower(text)
	text = nonAlphanumeric.ReplaceAllString(text, "-")
	text = multipleHyphens.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")

	if len(text) > maxLength {
		text = text[:maxLength]
		if lastHyphen := strings.LastIndex(text, "-"); lastHyphen > 0 {
			text = text[:lastHyphen]
		}
		text = strings.Trim(text, "-")
	}

	return text
}

func transliterate(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}
