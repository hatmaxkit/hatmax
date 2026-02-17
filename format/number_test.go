package format

import "testing"

func TestNumber(t *testing.T) {
	tests := []struct {
		name string
		n    any
		want string
	}{
		{"integer", 1234, "1,234"},
		{"large integer", 1234567, "1,234,567"},
		{"float", 1234.56, "1,234.56"},
		{"zero", 0, "0"},
		{"negative", -1234, "-1,234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Number(tt.n)
			if got != tt.want {
				t.Errorf("Number(%v) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestInteger(t *testing.T) {
	tests := []struct {
		name string
		n    any
		want string
	}{
		{"integer", 1234, "1,234"},
		{"float rounds", 1234.56, "1,235"},
		{"large", 1234567, "1,234,567"},
		{"zero", 0, "0"},
		{"negative", -1234.5, "-1,234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Integer(tt.n)
			if got != tt.want {
				t.Errorf("Integer(%v) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
