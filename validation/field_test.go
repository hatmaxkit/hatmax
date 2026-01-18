package validation

import (
	"testing"

	"github.com/google/uuid"
)

func TestFieldValidator(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"empty string fails", "", true},
			{"whitespace only fails", "   ", true},
			{"valid string passes", "hello", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("name", tt.value).Required().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("MinLength", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			min       int
			wantError bool
		}{
			{"too short", "ab", 3, true},
			{"exact length", "abc", 3, false},
			{"longer than min", "abcd", 3, false},
			{"empty string skipped", "", 3, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("name", tt.value).MinLength(tt.min).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("MaxLength", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			max       int
			wantError bool
		}{
			{"too long", "abcd", 3, true},
			{"exact length", "abc", 3, false},
			{"shorter than max", "ab", 3, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("name", tt.value).MaxLength(tt.max).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Email", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"valid email", "test@example.com", false},
			{"invalid email", "not-an-email", true},
			{"empty skipped", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("email", tt.value).Email().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Phone", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"valid phone", "+1234567890", false},
			{"too short", "123", true},
			{"empty skipped", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("phone", tt.value).Phone().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("NoHTML", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"no html", "plain text", false},
			{"has html", "<script>alert('xss')</script>", true},
			{"empty skipped", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("content", tt.value).NoHTML().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("OneOf", func(t *testing.T) {
		allowed := []string{"red", "green", "blue"}
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"valid value", "red", false},
			{"invalid value", "yellow", true},
			{"empty skipped", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("color", tt.value).OneOf(allowed).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Alphanumeric", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"alphanumeric", "abc123", false},
			{"has special char", "abc-123", true},
			{"empty skipped", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := Field("code", tt.value).Alphanumeric().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Chained validations", func(t *testing.T) {
		errs := Field("username", "ab").
			Required().
			MinLength(3).
			MaxLength(20).
			Alphanumeric().
			Errors()

		if !errs.HasErrors() {
			t.Error("expected MinLength error")
		}
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})
}

func TestUUIDFieldValidator(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		tests := []struct {
			name      string
			value     uuid.UUID
			wantError bool
		}{
			{"nil UUID fails", uuid.Nil, true},
			{"valid UUID passes", uuid.New(), false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := UUIDField("id", tt.value).Required().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})
}

func TestIntFieldValidator(t *testing.T) {
	t.Run("Min", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			min       int
			wantError bool
		}{
			{"below min", 5, 10, true},
			{"at min", 10, 10, false},
			{"above min", 15, 10, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := IntField("age", tt.value).Min(tt.min).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Max", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			max       int
			wantError bool
		}{
			{"above max", 15, 10, true},
			{"at max", 10, 10, false},
			{"below max", 5, 10, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := IntField("count", tt.value).Max(tt.max).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("InRange", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			min       int
			max       int
			wantError bool
		}{
			{"in range", 15, 10, 20, false},
			{"below range", 5, 10, 20, true},
			{"above range", 25, 10, 20, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := IntField("score", tt.value).InRange(tt.min, tt.max).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Positive", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			wantError bool
		}{
			{"positive", 5, false},
			{"zero", 0, true},
			{"negative", -5, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := IntField("quantity", tt.value).Positive().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("NonNegative", func(t *testing.T) {
		tests := []struct {
			name      string
			value     int
			wantError bool
		}{
			{"positive", 5, false},
			{"zero", 0, false},
			{"negative", -5, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := IntField("count", tt.value).NonNegative().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})
}

func TestFloatFieldValidator(t *testing.T) {
	t.Run("Min", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			min       float64
			wantError bool
		}{
			{"below min", 5.0, 10.0, true},
			{"at min", 10.0, 10.0, false},
			{"above min", 15.0, 10.0, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := FloatField("price", tt.value).Min(tt.min).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Max", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			max       float64
			wantError bool
		}{
			{"above max", 15.0, 10.0, true},
			{"at max", 10.0, 10.0, false},
			{"below max", 5.0, 10.0, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := FloatField("rate", tt.value).Max(tt.max).Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("Positive", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			wantError bool
		}{
			{"positive", 0.01, false},
			{"zero", 0.0, true},
			{"negative", -0.01, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := FloatField("amount", tt.value).Positive().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})

	t.Run("NonNegative", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float64
			wantError bool
		}{
			{"positive", 0.01, false},
			{"zero", 0.0, false},
			{"negative", -0.01, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				errs := FloatField("balance", tt.value).NonNegative().Errors()
				if tt.wantError && !errs.HasErrors() {
					t.Error("expected error but got none")
				}
				if !tt.wantError && errs.HasErrors() {
					t.Errorf("unexpected error: %v", errs)
				}
			})
		}
	})
}

func TestValidateAll(t *testing.T) {
	errs := ValidateAll(
		Field("name", "").Required(),
		Field("email", "invalid").Email(),
		IntField("age", -5).Positive(),
	)

	if len(errs) != 3 {
		t.Errorf("expected 3 errors, got %d", len(errs))
	}
}
