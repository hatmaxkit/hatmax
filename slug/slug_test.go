package slug

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerate(t *testing.T) {
	testID := uuid.MustParse("3995fd11-1234-5678-9abc-def012345678")

	tests := []struct {
		name string
		text string
		id   uuid.UUID
		want string
	}{
		{
			name: "simple text",
			text: "Cozy Apartment",
			id:   testID,
			want: "cozy-apartment-3995fd11",
		},
		{
			name: "text with special characters",
			text: "Cozy Dom Apartment!",
			id:   testID,
			want: "cozy-dom-apartment-3995fd11",
		},
		{
			name: "empty text returns uuid prefix only",
			text: "",
			id:   testID,
			want: "3995fd11",
		},
		{
			name: "text with accents",
			text: "Café résumé",
			id:   testID,
			want: "cafe-resume-3995fd11",
		},
		{
			name: "text with numbers",
			text: "Room 101",
			id:   testID,
			want: "room-101-3995fd11",
		},
		{
			name: "whitespace only returns uuid prefix",
			text: "   ",
			id:   testID,
			want: "3995fd11",
		},
		{
			name: "special chars only returns uuid prefix",
			text: "!!!@@@",
			id:   testID,
			want: "3995fd11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Generate(tt.text, tt.id)
			if got != tt.want {
				t.Errorf("Generate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		want      string
	}{
		{
			name:      "empty string",
			text:      "",
			maxLength: 50,
			want:      "",
		},
		{
			name:      "simple lowercase",
			text:      "hello world",
			maxLength: 50,
			want:      "hello-world",
		},
		{
			name:      "uppercase to lowercase",
			text:      "Hello World",
			maxLength: 50,
			want:      "hello-world",
		},
		{
			name:      "special characters replaced",
			text:      "hello!@#world",
			maxLength: 50,
			want:      "hello-world",
		},
		{
			name:      "multiple spaces collapsed",
			text:      "hello    world",
			maxLength: 50,
			want:      "hello-world",
		},
		{
			name:      "leading trailing special chars trimmed",
			text:      "!!!hello world!!!",
			maxLength: 50,
			want:      "hello-world",
		},
		{
			name:      "accented characters transliterated",
			text:      "café résumé naïve",
			maxLength: 50,
			want:      "cafe-resume-naive",
		},
		{
			name:      "spanish characters",
			text:      "España señor año",
			maxLength: 50,
			want:      "espana-senor-ano",
		},
		{
			name:      "german characters",
			text:      "größe über",
			maxLength: 50,
			want:      "gro-e-uber",
		},
		{
			name:      "truncate to max length at word boundary",
			text:      "this is a very long title that should be truncated",
			maxLength: 20,
			want:      "this-is-a-very-long",
		},
		{
			name:      "truncate without word boundary",
			text:      "abcdefghijklmnopqrstuvwxyz",
			maxLength: 10,
			want:      "abcdefghij",
		},
		{
			name:      "numbers preserved",
			text:      "room 101 floor 2",
			maxLength: 50,
			want:      "room-101-floor-2",
		},
		{
			name:      "mixed content",
			text:      "  ¡Hola! ¿Cómo estás?  ",
			maxLength: 50,
			want:      "hola-como-estas",
		},
		{
			name:      "only special chars returns empty",
			text:      "!@#$%^&*()",
			maxLength: 50,
			want:      "",
		},
		{
			name:      "truncate at word boundary",
			text:      "hello-world-foo-bar-baz",
			maxLength: 15,
			want:      "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.text, tt.maxLength)
			if got != tt.want {
				t.Errorf("Normalize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransliterate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no accents",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "french accents",
			input: "café résumé",
			want:  "cafe resume",
		},
		{
			name:  "spanish tilde",
			input: "señor",
			want:  "senor",
		},
		{
			name:  "german umlaut",
			input: "über",
			want:  "uber",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transliterate(tt.input)
			if got != tt.want {
				t.Errorf("transliterate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		want      string
	}{
		{
			name:      "max length zero",
			text:      "hello",
			maxLength: 0,
			want:      "",
		},
		{
			name:      "max length one",
			text:      "hello",
			maxLength: 1,
			want:      "h",
		},
		{
			name:      "exact max length",
			text:      "hello",
			maxLength: 5,
			want:      "hello",
		},
		{
			name:      "negative max length",
			text:      "hello",
			maxLength: -1,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.text, tt.maxLength)
			if got != tt.want {
				t.Errorf("Normalize() = %q, want %q", got, tt.want)
			}
		})
	}
}
