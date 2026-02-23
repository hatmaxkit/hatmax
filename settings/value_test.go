package settings

import "testing"

func TestParseBool(t *testing.T) {
	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"1", true, false},
		{"0", false, false},
		{"", false, false},
		{"invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBool(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("ParseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"42", 42, false},
		{"-10", -10, false},
		{"0", 0, false},
		{"", 0, false},
		{"abc", 0, true},
		{"3.14", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseInt(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("ParseInt(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatBool(t *testing.T) {
	tests := []struct {
		input bool
		want  string
	}{
		{true, "true"},
		{false, "false"},
	}

	for _, tt := range tests {
		got := FormatBool(tt.input)
		if got != tt.want {
			t.Errorf("FormatBool(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{42, "42"},
		{-10, "-10"},
		{0, "0"},
	}

	for _, tt := range tests {
		got := FormatInt(tt.input)
		if got != tt.want {
			t.Errorf("FormatInt(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
