package validation

import (
	"slices"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "trim and collapse spaces",
			input: "   hello    world   ",
			want:  "hello world",
		},
		{
			name:  "control characters become separators",
			input: "hello\tworld\nx",
			want:  "hello world x",
		},
		{
			name:  "whitespace only",
			input: " \t\n ",
			want:  "",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeText(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeOptionalText(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  *string
	}{
		{
			name:  "nil remains nil",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty after normalization becomes nil",
			input: ptr(" \t "),
			want:  nil,
		},
		{
			name:  "normalized value",
			input: ptr("  hello   world "),
			want:  ptr("hello world"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeOptionalText(tt.input)
			switch {
			case got == nil && tt.want == nil:
				return
			case got == nil || tt.want == nil:
				t.Fatalf("NormalizeOptionalText() nil mismatch, got=%v want=%v", got, tt.want)
			case *got != *tt.want:
				t.Fatalf("NormalizeOptionalText() = %q, want %q", *got, *tt.want)
			}
		})
	}
}

func TestSanitizeStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "drop empty and dedupe",
			input: []string{" pool ", "", "pool", " garage", "garage", " \n "},
			want:  []string{"pool", "garage"},
		},
		{
			name:  "preserve first occurrence order",
			input: []string{"b", " a ", "b", "a"},
			want:  []string{"b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeStringSlice(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("SanitizeStringSlice() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func ptr(value string) *string {
	return &value
}
